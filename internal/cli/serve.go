package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	serverservice "github.com/muidea/skill-hub/internal/modules/kernel/server/service"
	"github.com/spf13/cobra"
)

var (
	serveHost        string
	servePort        int
	serveOpenBrowser bool
	serveInputReader io.Reader = os.Stdin
	serveRunServer             = func(ctx context.Context, cfg serverservice.Config) error {
		return serverservice.New().Run(ctx, cfg)
	}
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "以本地服务模式运行 skill-hub",
	Long:  "启动本地 HTTP 服务与 Web 管理界面。",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServe()
	},
}

func init() {
	serveCmd.Flags().StringVar(&serveHost, "host", "127.0.0.1", "监听地址")
	serveCmd.Flags().IntVar(&servePort, "port", 5525, "监听端口")
	serveCmd.Flags().BoolVar(&serveOpenBrowser, "open-browser", false, "启动后自动打开浏览器")
}

func runServe() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	url := fmt.Sprintf("http://%s:%d", serveHost, servePort)
	fmt.Printf("skill-hub service listening on %s\n", url)
	fmt.Println("输入 q 并回车可停止服务")

	if serveOpenBrowser {
		go openBrowser(url)
	}

	go waitForServeStopInput(ctx, serveInputReader, stop)

	return serveRunServer(ctx, serverservice.Config{Host: serveHost, Port: servePort})
}

func waitForServeStopInput(ctx context.Context, reader io.Reader, stop context.CancelFunc) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		switch strings.ToLower(strings.TrimSpace(scanner.Text())) {
		case "q", "quit", "exit", "stop":
			stop()
			return
		}
	}
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	_, _ = exec.Command(cmd, args...).CombinedOutput()
}
