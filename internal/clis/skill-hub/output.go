package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	apperrors "github.com/muidea/skill-hub/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// commandOutput is scoped to one CLI invocation. The Cobra command tree is
// already process-local; neither runtime nor the daemon uses this state.
type commandOutput struct {
	command *cobra.Command
	json    bool
	started bool
	value   json.RawMessage
	batch   bool
	results []json.RawMessage
}

var activeOutput *commandOutput

func jsonMode() bool { return activeOutput != nil && activeOutput.json }

func nonInteractiveMode() bool {
	return jsonMode() || (activeOutput != nil && mustGetBoolFlag(activeOutput.command, "non-interactive"))
}

func beginJSONBatch() {
	if jsonMode() {
		activeOutput.batch = true
		activeOutput.results = []json.RawMessage{}
	}
}

func recordBatchError(id string, err error) {
	if jsonMode() {
		_ = writeJSON(map[string]any{"skill_id": id, "error": commandError{Code: apperrors.Code(err), Message: FormatError(err)}})
	}
}

func textOutput() io.Writer {
	if activeOutput != nil {
		if activeOutput.json {
			return activeOutput.command.ErrOrStderr()
		}
		return activeOutput.command.OutOrStdout()
	}
	return os.Stdout
}

func printText(a ...any)                 { _, _ = fmt.Fprint(textOutput(), a...) }
func printlnText(a ...any)               { _, _ = fmt.Fprintln(textOutput(), a...) }
func printfText(format string, a ...any) { _, _ = fmt.Fprintf(textOutput(), format, a...) }

func writeJSON(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if jsonMode() {
		if activeOutput.batch {
			activeOutput.results = append(activeOutput.results, data)
			return nil
		}
		if activeOutput.value != nil {
			return apperrors.NewWithCode("writeJSON", apperrors.ErrSystem, "命令重复生成 JSON 结果")
		}
		activeOutput.value = data
		return nil
	}
	_, err = fmt.Fprintln(os.Stdout, string(data))
	return err
}

type commandError struct {
	Code    apperrors.ErrorCode `json:"code"`
	Message string              `json:"message"`
	Details map[string]any      `json:"details,omitempty"`
}

// ErrorReported tells the executable that Execute has already rendered the
// error as part of the JSON response. Its exit status must still be nonzero.
func ErrorReported(err error) bool {
	var reported *reportedError
	return apperrors.As(err, &reported)
}

type reportedError struct{ error }

func (e *reportedError) Unwrap() error { return e.error }

func executeCommand(root *cobra.Command, args []string) error {
	previous := activeOutput
	selected, _, _ := root.Find(args)
	if selected == nil {
		selected = root
	}
	output := &commandOutput{command: selected, json: requestsJSON(selected, args)}
	activeOutput = output
	defer func() { activeOutput = previous }()
	oldUsage := root.SilenceUsage
	if output.json {
		root.SilenceUsage = true
	}
	defer func() { root.SilenceUsage = oldUsage }()
	oldVersionTemplate := root.VersionTemplate()
	if output.json {
		root.SetVersionTemplate("{{jsonVersion .Version}}")
	}
	defer root.SetVersionTemplate(oldVersionTemplate)
	root.SetArgs(args)
	_, runErr := root.ExecuteC()
	if runErr != nil && !output.started && apperrors.Code(runErr) == apperrors.ErrSystem {
		runErr = apperrors.NewWithCode("arguments", apperrors.ErrInvalidInput, runErr.Error())
	}
	if !output.json {
		return runErr
	}

	data := output.value
	if output.batch {
		data, _ = json.MarshalIndent(map[string]any{"results": output.results}, "", "  ")
	}
	if runErr == nil && (len(data) == 0 || string(data) == "null") {
		runErr = apperrors.NewWithCode("cli", apperrors.ErrSystem, selected.CommandPath()+" 未生成 JSON 结果")
	}
	if runErr != nil {
		fields := map[string]any{}
		if len(data) > 0 && string(data) != "null" {
			if err := json.Unmarshal(data, &fields); err != nil {
				fields = map[string]any{"data": json.RawMessage(data)}
			}
		}
		fields["command_error"] = commandError{
			Code: apperrors.Code(runErr), Message: FormatError(runErr), Details: apperrors.Details(runErr),
		}
		var err error
		data, err = json.MarshalIndent(fields, "", "  ")
		if err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(output.command.OutOrStdout(), string(data)); err != nil {
		return err
	}
	if runErr != nil {
		return &reportedError{runErr}
	}
	return nil
}

// Detect JSON before Cobra parses flags so parse/argument errors also obey the
// output contract. Known flag values and positional arguments after -- must
// not be mistaken for output options.
func requestsJSON(selected *cobra.Command, args []string) bool {
	flags := pflag.NewFlagSet("output", pflag.ContinueOnError)
	for command := selected; command != nil; command = command.Parent() {
		flags.AddFlagSet(command.Flags())
		flags.AddFlagSet(command.PersistentFlags())
	}
	jsonOutput := false
	auditJSON := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if strings.HasPrefix(arg, "--") {
			name, value, hasValue := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
			if name == "json" {
				jsonOutput = true
				if hasValue {
					if parsed, err := strconv.ParseBool(value); err == nil {
						jsonOutput = parsed
					}
				}
				continue
			}
			if flag := flags.Lookup(name); flag != nil && flag.NoOptDefVal == "" && !hasValue {
				i++
				if i < len(args) {
					value = args[i]
				}
			}
			if selected == auditCmd && name == "format" {
				auditJSON = strings.EqualFold(strings.TrimSpace(value), "json")
			}
		} else if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			for j, shorthand := range arg[1:] {
				if flag := flags.ShorthandLookup(string(shorthand)); flag != nil && flag.NoOptDefVal == "" {
					if j == len(arg)-2 {
						i++
					}
					break
				}
			}
		}
	}
	return jsonOutput || auditJSON
}

func inputRequired(message string) error {
	return apperrors.NewWithCode("cli", apperrors.ErrUserInput, message)
}

func jsonResultCount() int {
	if activeOutput == nil {
		return 0
	}
	return len(activeOutput.results)
}

func init() {
	cobra.AddTemplateFunc("jsonVersion", func(version string) string {
		if jsonMode() {
			_ = writeJSON(map[string]string{"version": version})
		}
		return ""
	})
}

func inputSource() io.Reader {
	if activeOutput != nil {
		return activeOutput.command.InOrStdin()
	}
	return os.Stdin
}

func readInputLine(reader *bufio.Reader) (string, error) {
	if nonInteractiveMode() {
		return "", inputRequired("非交互模式不允许读取 stdin，请提供所需参数")
	}
	return reader.ReadString('\n')
}

func scanInputToken(value *string) error {
	if nonInteractiveMode() {
		return inputRequired("非交互模式不允许读取 stdin，请提供确认参数")
	}
	_, err := fmt.Fscan(inputSource(), value)
	return err
}
