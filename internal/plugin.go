package internal

import (
	"github.com/orchestra-mcp/sdk-go/plugin"
	"github.com/orchestra-mcp/plugin-devtools-log-viewer/internal/tools"
)

// ToolsPlugin registers all log viewer tools.
type ToolsPlugin struct{}

// RegisterTools registers all 5 log viewer tools with the plugin builder.
func (tp *ToolsPlugin) RegisterTools(builder *plugin.PluginBuilder) {
	builder.RegisterTool("log_tail",
		"Read the last N lines of a log file",
		tools.LogTailSchema(), tools.LogTail())

	builder.RegisterTool("log_watch",
		"Return a snapshot of the last N lines of a log file (streaming not supported)",
		tools.LogWatchSchema(), tools.LogWatch())

	builder.RegisterTool("log_search",
		"Search a log file for lines matching a regular expression, with optional context lines",
		tools.LogSearchSchema(), tools.LogSearch())

	builder.RegisterTool("log_parse",
		"Parse a log file using json, syslog, or auto-detection format",
		tools.LogParseSchema(), tools.LogParse())

	builder.RegisterTool("log_list_sources",
		"List log files in a directory or common log locations",
		tools.LogListSourcesSchema(), tools.LogListSources())
}
