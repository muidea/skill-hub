package logging

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// DefaultRunLogMaxBytes caps in-memory detailed logs per CLI invocation.
const DefaultRunLogMaxBytes = 2 * 1024 * 1024

var (
	urlCredentialPattern = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://[^/\s:@]+:)([^/\s@]+)(@)`)
	authHeaderPattern    = regexp.MustCompile(`(?i)(authorization\s*:\s*(?:bearer|basic)\s+)(\S+)`)
	secretKVPattern      = regexp.MustCompile(`(?i)((?:git_token|secret_key|secretkey|x-skill-hub-secret|token)\s*[:=]\s*["']?)([^\s"',;]+)`)
)

// RunLogOptions configures a per-command RunLog collector.
type RunLogOptions struct {
	Command  string
	LogFile  string
	Mirror   io.Writer
	MaxBytes int
}

// RunLog captures scrubbed detailed diagnostics for a single command invocation.
type RunLog struct {
	mu         sync.Mutex
	buf        bytes.Buffer
	opts       RunLogOptions
	startedAt  time.Time
	hasEntries bool
	truncated  bool
}

// NewRunLog creates a RunLog with bounded memory buffering.
func NewRunLog(opts RunLogOptions) *RunLog {
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = DefaultRunLogMaxBytes
	}
	return &RunLog{
		opts:      opts,
		startedAt: time.Now(),
	}
}

func (rl *RunLog) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	scrubbed := ScrubSecrets(string(p))

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if len(scrubbed) > 0 {
		rl.hasEntries = true
	}
	rl.appendBuffered(scrubbed)
	if rl.opts.Mirror != nil {
		_, _ = rl.opts.Mirror.Write([]byte(scrubbed))
	}
	return len(p), nil
}

func (rl *RunLog) appendBuffered(text string) {
	if rl.truncated || len(text) == 0 {
		return
	}
	remaining := rl.opts.MaxBytes - rl.buf.Len()
	if remaining <= 0 {
		rl.buf.WriteString("\n... [log truncated] ...\n")
		rl.truncated = true
		return
	}
	if len(text) <= remaining {
		rl.buf.WriteString(text)
		return
	}
	rl.buf.WriteString(text[:remaining])
	rl.buf.WriteString("\n... [log truncated] ...\n")
	rl.truncated = true
}

// WriteSection records a labeled diagnostic section into the run log.
func (rl *RunLog) WriteSection(header, body string) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "[%s] %s\n", time.Now().Format(time.RFC3339), header)
	if body != "" {
		sb.WriteString(body)
		if !strings.HasSuffix(body, "\n") {
			sb.WriteByte('\n')
		}
	}
	_, _ = rl.Write([]byte(sb.String()))
}

// WriteCommand records a subprocess invocation, its error (if any), and combined output.
func (rl *RunLog) WriteCommand(name string, args []string, output []byte, err error) {
	_, _ = rl.Write([]byte(formatCommandLog(name, args, output, err)))
}

// RecordError appends the final command error to the log buffer without marking
// an otherwise empty run as having detailed entries.
func (rl *RunLog) RecordError(err error) {
	if err == nil {
		return
	}
	line := fmt.Sprintf("[%s] [error] %s\n", time.Now().Format(time.RFC3339), ScrubSecrets(err.Error()))
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.appendBuffered(line)
}

// HasEntries reports whether detailed diagnostics were captured during this run.
func (rl *RunLog) HasEntries() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.hasEntries
}

// Persist writes the captured log to opts.LogFile or a 0600 temp file and returns its path.
func (rl *RunLog) Persist() (string, error) {
	rl.mu.Lock()
	command := rl.opts.Command
	if strings.TrimSpace(command) == "" {
		command = "skill-hub"
	}
	var payload strings.Builder
	payload.WriteString("# skill-hub detailed log\n")
	fmt.Fprintf(&payload, "# command: %s\n", command)
	fmt.Fprintf(&payload, "# started_at: %s\n", rl.startedAt.Format(time.RFC3339))
	fmt.Fprintf(&payload, "# finished_at: %s\n\n", time.Now().Format(time.RFC3339))
	payload.WriteString(rl.buf.String())
	logFile := rl.opts.LogFile
	rl.mu.Unlock()

	data := []byte(payload.String())
	if logFile != "" {
		if dir := filepath.Dir(logFile); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", err
			}
		}
		if err := os.WriteFile(logFile, data, 0600); err != nil {
			return "", err
		}
		if err := os.Chmod(logFile, 0600); err != nil {
			return "", err
		}
		return logFile, nil
	}

	slug := sanitizeCommandSlug(command)
	f, err := os.CreateTemp("", "skill-hub-"+slug+"-*.log")
	if err != nil {
		return "", err
	}
	path := f.Name()
	if err := f.Chmod(0600); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func sanitizeCommandSlug(command string) string {
	lower := strings.ToLower(strings.TrimSpace(command))
	lower = strings.TrimPrefix(lower, "skill-hub ")
	if lower == "skill-hub" || lower == "" {
		return "cli"
	}
	var sb strings.Builder
	lastDash := false
	for _, r := range lower {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			sb.WriteRune(r)
			lastDash = false
		case !lastDash && sb.Len() > 0:
			sb.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(sb.String(), "-")
	if slug == "" {
		return "cli"
	}
	return slug
}

// ScrubSecrets masks common credential patterns and known token environment variables.
func ScrubSecrets(input string) string {
	if input == "" {
		return ""
	}
	out := urlCredentialPattern.ReplaceAllString(input, "${1}***${3}")
	out = authHeaderPattern.ReplaceAllString(out, "${1}***")
	out = secretKVPattern.ReplaceAllString(out, "${1}***")
	for _, envKey := range []string{"SKILL_HUB_SECRET_KEY", "GITHUB_TOKEN", "GH_TOKEN"} {
		if val := os.Getenv(envKey); len(val) >= 6 {
			out = strings.ReplaceAll(out, val, "***")
		}
	}
	return out
}

func formatCommandLog(name string, args []string, output []byte, err error) string {
	var sb strings.Builder
	cmdLine := name
	if len(args) > 0 {
		cmdLine += " " + strings.Join(args, " ")
	}
	fmt.Fprintf(&sb, "[%s] $ %s\n", time.Now().Format(time.RFC3339), ScrubSecrets(cmdLine))
	if len(output) > 0 {
		text := ScrubSecrets(string(output))
		sb.WriteString(text)
		if !strings.HasSuffix(text, "\n") {
			sb.WriteByte('\n')
		}
	}
	if err != nil {
		fmt.Fprintf(&sb, "[exit error] %s\n", ScrubSecrets(err.Error()))
	}
	return sb.String()
}

var (
	detailWriterMu     sync.RWMutex
	activeDetailWriter io.Writer = io.Discard
)

// SetDetailWriter sets the active writer for detailed subprocess/internal diagnostics
// and returns a restore function.
func SetDetailWriter(w io.Writer) func() {
	if w == nil {
		w = io.Discard
	}
	detailWriterMu.Lock()
	prev := activeDetailWriter
	activeDetailWriter = w
	detailWriterMu.Unlock()
	return func() {
		detailWriterMu.Lock()
		activeDetailWriter = prev
		detailWriterMu.Unlock()
	}
}

type activeDetailWriterProxy struct{}

func (activeDetailWriterProxy) Write(p []byte) (int, error) {
	detailWriterMu.RLock()
	w := activeDetailWriter
	detailWriterMu.RUnlock()
	if w == nil || w == io.Discard {
		return len(p), nil
	}
	return w.Write([]byte(ScrubSecrets(string(p))))
}

// DetailWriter returns an io.Writer that forwards scrubbed diagnostics to the active detail writer.
func DetailWriter() io.Writer {
	return activeDetailWriterProxy{}
}

// DetailPrintf writes formatted diagnostic output to the active detail writer.
func DetailPrintf(format string, args ...any) {
	_, _ = fmt.Fprintf(DetailWriter(), format, args...)
}

// DetailPrintln writes a diagnostic line to the active detail writer.
func DetailPrintln(args ...any) {
	_, _ = fmt.Fprintln(DetailWriter(), args...)
}

// LogCommandOutput records a subprocess command and its output to the active detail writer.
func LogCommandOutput(name string, args []string, output []byte, err error) {
	detailWriterMu.RLock()
	w := activeDetailWriter
	detailWriterMu.RUnlock()
	if w == nil || w == io.Discard {
		return
	}
	if rl, ok := w.(*RunLog); ok {
		rl.WriteCommand(name, args, output, err)
		return
	}
	_, _ = io.WriteString(w, formatCommandLog(name, args, output, err))
}
