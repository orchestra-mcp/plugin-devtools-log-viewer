package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/orchestra-mcp/plugin-devtools-log-viewer/internal"
	"github.com/orchestra-mcp/sdk-go/plugin"
)

func main() {
	builder := plugin.New("devtools.log-viewer").
		Version("0.2.0").
		Description("Log file viewer, search, and background process management tools").
		Author("Orchestra").
		Binary("devtools-log-viewer")

	tp := internal.NewToolsPlugin()
	tp.RegisterTools(builder)

	p := builder.BuildWithTools()
	p.ParseFlags()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		tp.KillAll()
		cancel()
	}()

	if err := p.Run(ctx); err != nil {
		log.Fatalf("devtools.log-viewer: %v", err)
	}
}
