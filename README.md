# MCP LSP Go

A Model Context Protocol (MCP) server that allows AI assistants like Claude to interact with Go's Language Server Protocol (LSP) and benefit from advanced Go code analysis features.

## Overview

This MCP server helps AI assistants to:

- Use LSP to analyze Go code with minimal context usage
- Navigate to definitions and find references instantly
- Check code diagnostics without running builds

## Architecture

This project uses the [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) library to implement the Model Context Protocol. The MCP integration enables seamless communication between AI assistants and Go tools.

The server communicates with [gopls](https://github.com/golang/tools/tree/master/gopls), the official language server for Go, via the Language Server Protocol (LSP).

## Features

- **LSP Integration**: Connection to Go's Language Server Protocol for code analysis
- **Code Navigation**: Finding definitions and references in the code
- **Code Quality**: Getting diagnostics and errors in real-time

## Project Structure

```bash
.
├── cmd
│   └── mcp-gopls        # Application entry point
├── pkg
│   ├── lsp             # LSP client to communicate with gopls
│   │   ├── client      # LSP client implementation
│   │   └── protocol    # LSP protocol types and features
│   ├── server          # MCP server
│   └── tools           # MCP tools exposing LSP features
```

## Installation

```bash
go install github.com/solatis/mcp-gopls/cmd/mcp-gopls@latest
```

## Add to Cursor

```json
{
  "mcpServers": {
    "mcp-gopls": {
      "command": "mcp-gopls"
    }
  }
} 
```

## MCP Tools

The MCP server provides the following LSP-powered tools for efficient Go code analysis:

| Tool | Description |
|-------|-------------|
| `go_to_definition` | Navigate instantly to where any symbol (function, type, variable) is defined. Much faster and more accurate than text search. |
| `find_references` | Find all usages of a symbol across the entire codebase. Essential for understanding code impact before making changes. |
| `check_diagnostics` | Get all compile errors, type errors, and linting issues without running builds. The fastest way to verify code correctness. |
| `document_symbol` | Get a complete hierarchical outline of all symbols in a file. 10-100x faster than reading the entire file. |
| `workspace_symbol` | Search for any symbol across the entire project instantly. Supports fuzzy matching and understands Go syntax. |
| `list_interface_implementation` | Find all types that implement an interface, or find which interface a method implements. Critical for Go's interface-based design. |

## Usage Example

Using the server with AI assistants that support MCP:

```Markdown
# Navigate to definitions instantly
Where is the ServeStdio function defined?

# Find all usages efficiently  
Show me everywhere the Context type is used in this project

# Check for errors without building
Are there any errors in my main.go file?

# Understand code structure
What's the structure of the server.go file?

# Search across the project
Where is the MCPServer type defined in this codebase?

# Understand interfaces
What types implement the LSPClient interface?
```

## Why Use These LSP Tools?

1. **Reduced Token Usage**: LSP tools provide precise results without reading entire files
2. **Better Accuracy**: Understands Go's type system, imports, and syntax
3. **Faster Analysis**: Instant results compared to multiple grep/search operations
4. **Lower Costs**: Less context means fewer tokens and reduced API costs

## Development

```bash
git clone https://github.com/solatis/mcp-gopls.git
cd mcp-gopls
go mod tidy
go build -o mcp-gopls cmd/mcp-gopls/main.go
./mcp-gopls
```

## Prerequisites

- Go 1.21 or higher
- gopls installed (`go install golang.org/x/tools/gopls@latest`)

## Integration with Ollama

This MCP server can be used with any tool that supports the MCP protocol. For Ollama integration:

1. Make sure Ollama is running
2. The MCP server runs independently and communicates through stdin/stdout
3. Configure your client to use the MCP server as a tool provider

## License

Apache License 2.0