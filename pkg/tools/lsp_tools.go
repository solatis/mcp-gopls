package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/solatis/mcp-gopls/pkg/lsp/client"
)

type LSPTools struct {
	client       client.LSPClient
	clientGetter func() client.LSPClient
	resetFunc    func(error) bool
}

func NewLSPTools(lspClient client.LSPClient) *LSPTools {
	return &LSPTools{
		client:       lspClient,
		clientGetter: func() client.LSPClient { return lspClient },
		resetFunc:    func(error) bool { return false },
	}
}

func (t *LSPTools) SetClientGetter(getter func() client.LSPClient) {
	t.clientGetter = getter
}

func (t *LSPTools) SetResetFunc(resetFunc func(error) bool) {
	t.resetFunc = resetFunc
}

func (t *LSPTools) getClient() client.LSPClient {
	if t.clientGetter != nil {
		return t.clientGetter()
	}
	return t.client
}

func (t *LSPTools) handleLSPError(err error) error {
	if err != nil {
		if t.resetFunc != nil && t.resetFunc(err) {
			return fmt.Errorf("LSP error (client reinitialized, please try again): %w", err)
		}

		return fmt.Errorf("LSP error: %w", err)
	}
	return nil
}

func (t *LSPTools) Register(s *server.MCPServer) {
	t.registerGoToDefinition(s)
	t.registerFindReferences(s)
	t.registerCheckDiagnostics(s)
	t.registerDocumentSymbol(s)
	t.registerWorkspaceSymbol(s)
	t.registerListImplementations(s)
}

func convertPathToURI(path string) string {
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err == nil {
			path = filepath.Join(cwd, path)
		}
	}

	path = filepath.Clean(path)

	if !strings.HasPrefix(path, "file://") {
		if filepath.Separator == '\\' {
			path = "/" + strings.ReplaceAll(path, "\\", "/")
			if path[1] == ':' {
				drive := strings.ToLower(string(path[0]))
				path = "/" + drive + path[2:]
			}
		}
		path = "file://" + path
	}

	return path
}

func (t *LSPTools) registerGoToDefinition(s *server.MCPServer) {
	definitionTool := mcp.NewTool("go_to_definition",
		mcp.WithDescription("CRITICAL FOR CODE NAVIGATION: Use this LSP-powered tool instead of grep/search when you need to find where a function, type, variable, or interface is actually defined. This tool understands Go's type system and import paths, providing the EXACT location where a symbol is declared. Much faster and more accurate than text search. Use this when: 1) User asks 'where is X defined?', 2) You need to understand what a function/type actually does, 3) You're debugging and need to trace back to source definitions. Returns the file URI and exact line/character position of the definition."),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("URI or absolute path of the file containing the symbol. Can be a file:// URI or absolute path like /path/to/file.go"),
		),
		mcp.WithObject("position",
			mcp.Required(),
			mcp.Description("Position of the symbol to look up. Must contain 'line' (0-indexed line number) and 'character' (0-indexed column number) keys"),
		),
	)

	s.AddTool(definitionTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileURI := request.GetString("file_uri", "")
		if fileURI == "" {
			return nil, errors.New("file_uri is required")
		}

		args := request.GetArguments()
		positionObj, ok := args["position"].(map[string]any)
		if !ok {
			return nil, errors.New("position must be an object")
		}

		line, ok := positionObj["line"].(float64)
		if !ok {
			return nil, errors.New("line must be a number")
		}

		character, ok := positionObj["character"].(float64)
		if !ok {
			return nil, errors.New("character must be a number")
		}

		if !strings.HasPrefix(fileURI, "file://") {
			fileURI = convertPathToURI(fileURI)
		}

		lspClient := t.getClient()
		if lspClient == nil {
			return nil, errors.New("LSP client not available")
		}

		locations, err := lspClient.GoToDefinition(fileURI, int(line), int(character))
		if err != nil {
			return nil, t.handleLSPError(err)
		}

		result, err := json.Marshal(locations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}

func (t *LSPTools) registerFindReferences(s *server.MCPServer) {
	referencesTool := mcp.NewTool("find_references",
		mcp.WithDescription("ESSENTIAL FOR CODE IMPACT ANALYSIS: Use this LSP tool to instantly find ALL places where a function, type, method, or variable is used across the entire codebase. Dramatically faster and more accurate than grep because it understands Go's syntax, imports, and type system. Use this when: 1) User asks 'where is X used?', 2) Before modifying any function/type to understand impact, 3) Analyzing code dependencies and relationships, 4) Refactoring or renaming considerations. This tool saves significant time and context by providing a complete, accurate list of usages rather than requiring multiple file reads. Returns all locations with file URI and line/character positions."),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("URI or absolute path of the file containing the symbol. Can be a file:// URI or absolute path like /path/to/file.go"),
		),
		mcp.WithObject("position",
			mcp.Required(),
			mcp.Description("Position of the symbol to find references for. Must contain 'line' (0-indexed line number) and 'character' (0-indexed column number) keys"),
		),
	)

	s.AddTool(referencesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileURI := request.GetString("file_uri", "")
		if fileURI == "" {
			return nil, errors.New("file_uri is required")
		}

		args := request.GetArguments()
		positionObj, ok := args["position"].(map[string]any)
		if !ok {
			return nil, errors.New("position must be an object")
		}

		line, ok := positionObj["line"].(float64)
		if !ok {
			return nil, errors.New("line must be a number")
		}

		character, ok := positionObj["character"].(float64)
		if !ok {
			return nil, errors.New("character must be a number")
		}

		if !strings.HasPrefix(fileURI, "file://") {
			fileURI = convertPathToURI(fileURI)
		}

		lspClient := t.getClient()
		if lspClient == nil {
			return nil, errors.New("LSP client not available")
		}

		locations, err := lspClient.FindReferences(fileURI, int(line), int(character), true)
		if err != nil {
			if strings.Contains(err.Error(), "client closed") {
				return nil, fmt.Errorf("LSP client not available, please restart the server: %w", err)
			}
			return nil, t.handleLSPError(err)
		}

		result, err := json.Marshal(locations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}

func (t *LSPTools) registerCheckDiagnostics(s *server.MCPServer) {
	diagnosticsTool := mcp.NewTool("check_diagnostics",
		mcp.WithDescription("INSTANT CODE VALIDATION: Use this LSP tool to immediately get all compile errors, type errors, and linting issues for a Go file WITHOUT running 'go build' or reading file contents. This is the fastest way to verify code correctness. Use this when: 1) After making any code changes to verify correctness, 2) User reports errors or asks 'why doesn't this compile?', 3) Before suggesting code fixes to understand current issues, 4) Debugging type mismatches or import problems. Returns a comprehensive list of all problems with exact locations and error messages. Much more efficient than running build commands or manually checking syntax."),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("URI or absolute path of the Go file to check. Can be a file:// URI or absolute path like /path/to/file.go"),
		),
	)

	s.AddTool(diagnosticsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileURI := request.GetString("file_uri", "")
		if fileURI == "" {
			return nil, errors.New("file_uri is required")
		}

		if !strings.HasPrefix(fileURI, "file://") {
			fileURI = convertPathToURI(fileURI)
		}

		if t.client == nil {
			return nil, errors.New("LSP client not initialized")
		}

		diagnostics, err := t.client.GetDiagnostics(fileURI)
		if err != nil {
			if strings.Contains(err.Error(), "client closed") {
				return nil, fmt.Errorf("LSP service not available, please restart the server: %w", err)
			}
			return nil, fmt.Errorf("failed to get diagnostics: %w", err)
		}

		result, err := json.Marshal(diagnostics)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}

func (t *LSPTools) registerDocumentSymbol(s *server.MCPServer) {
	documentSymbolTool := mcp.NewTool("document_symbol",
		mcp.WithDescription("FILE STRUCTURE AT A GLANCE: Use this LSP tool to instantly get a complete hierarchical outline of ALL symbols (functions, types, methods, variables, constants) in a Go file. This is 10-100x faster than reading the entire file and gives you immediate understanding of code structure. Use this when: 1) User asks 'what's in this file?' or 'show me the structure', 2) You need to understand a file's organization before making changes, 3) Looking for specific functions/types in a file, 4) Analyzing code architecture. Returns a tree structure with symbol names, types, and exact locations. Saves massive amounts of context compared to reading entire files."),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("URI or absolute path of the Go file to analyze. Can be a file:// URI or absolute path like /path/to/file.go"),
		),
	)

	s.AddTool(documentSymbolTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileURI := request.GetString("file_uri", "")
		if fileURI == "" {
			return nil, errors.New("file_uri is required")
		}

		if !strings.HasPrefix(fileURI, "file://") {
			fileURI = convertPathToURI(fileURI)
		}

		lspClient := t.getClient()
		if lspClient == nil {
			return nil, errors.New("LSP client not available")
		}

		symbols, err := lspClient.GetDocumentSymbols(fileURI)
		if err != nil {
			return nil, t.handleLSPError(err)
		}

		result, err := json.Marshal(symbols)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}

func (t *LSPTools) registerWorkspaceSymbol(s *server.MCPServer) {
	workspaceSymbolTool := mcp.NewTool("workspace_symbol",
		mcp.WithDescription("PROJECT-WIDE SYMBOL SEARCH: Use this LSP tool to search for any symbol (function, type, interface, method, constant) across the ENTIRE workspace/project instantly. Far superior to grep because it understands Go syntax and only returns actual symbol definitions, not comments or string matches. Use this when: 1) User asks 'where is type X defined in the project?', 2) You need to find a function but don't know which file, 3) Exploring unfamiliar codebases, 4) Understanding project structure and dependencies. Supports fuzzy matching (e.g., 'htpSrv' finds 'httpServer'). Returns symbol names, types, and exact file locations."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Symbol name to search for. Supports partial and fuzzy matching. Examples: 'Server' finds all symbols with Server in name, 'hndlr' might find 'handler', 'Handler', etc."),
		),
	)

	s.AddTool(workspaceSymbolTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query := request.GetString("query", "")
		if query == "" {
			return nil, errors.New("query is required")
		}

		lspClient := t.getClient()
		if lspClient == nil {
			return nil, errors.New("LSP client not available")
		}

		symbols, err := lspClient.GetWorkspaceSymbols(query)
		if err != nil {
			return nil, t.handleLSPError(err)
		}

		result, err := json.Marshal(symbols)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}

func (t *LSPTools) registerListImplementations(s *server.MCPServer) {
	implementationsTool := mcp.NewTool("list_interface_implementation",
		mcp.WithDescription("FIND ALL IMPLEMENTATIONS: Use this LSP tool to instantly find ALL types that implement a specific interface, or find the interface that a method implements. Critical for understanding Go's interface-based design. Use this when: 1) User asks 'what implements interface X?', 2) Understanding which concrete types satisfy an interface, 3) Before modifying interfaces to see impact, 4) Exploring polymorphic code behavior, 5) Finding all handlers/plugins that implement a pattern. Much more accurate than text search as it understands Go's type system. Returns exact locations of all implementing types."),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("URI or absolute path of the file containing the interface or method. Can be a file:// URI or absolute path like /path/to/file.go"),
		),
		mcp.WithObject("position",
			mcp.Required(),
			mcp.Description("Position of the interface name or method to find implementations for. Must contain 'line' (0-indexed line number) and 'character' (0-indexed column number) keys"),
		),
	)

	s.AddTool(implementationsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileURI := request.GetString("file_uri", "")
		if fileURI == "" {
			return nil, errors.New("file_uri is required")
		}

		args := request.GetArguments()
		positionObj, ok := args["position"].(map[string]any)
		if !ok {
			return nil, errors.New("position must be an object")
		}

		line, ok := positionObj["line"].(float64)
		if !ok {
			return nil, errors.New("line must be a number")
		}

		character, ok := positionObj["character"].(float64)
		if !ok {
			return nil, errors.New("character must be a number")
		}

		if !strings.HasPrefix(fileURI, "file://") {
			fileURI = convertPathToURI(fileURI)
		}

		lspClient := t.getClient()
		if lspClient == nil {
			return nil, errors.New("LSP client not available")
		}

		locations, err := lspClient.GetImplementations(fileURI, int(line), int(character))
		if err != nil {
			return nil, t.handleLSPError(err)
		}

		result, err := json.Marshal(locations)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}
