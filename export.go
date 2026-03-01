package devtoolslogviewer

import (
	"github.com/orchestra-mcp/plugin-devtools-log-viewer/internal"
	"github.com/orchestra-mcp/sdk-go/plugin"
)

// Register adds all log viewer tools to the builder.
func Register(builder *plugin.PluginBuilder) {
	tp := &internal.ToolsPlugin{}
	tp.RegisterTools(builder)
}
