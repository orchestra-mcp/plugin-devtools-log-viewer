package devtoolslogviewer

import (
	"github.com/orchestra-mcp/plugin-devtools-log-viewer/internal"
	"github.com/orchestra-mcp/sdk-go/plugin"
)

// Register adds all log viewer tools to the builder.
// The returned function kills all managed background processes on shutdown.
func Register(builder *plugin.PluginBuilder) func() {
	tp := internal.NewToolsPlugin()
	tp.RegisterTools(builder)
	return tp.KillAll
}
