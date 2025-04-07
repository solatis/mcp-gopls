package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/hloiseaufcms/mcp-gopls/pkg/lsp/client"
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
	t.registerHover(s)
	t.registerCompletion(s)
	t.registerCoverageAnalysis(s)
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
		mcp.WithDescription("Navigate to the definition of a symbol"),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("URI of the file"),
		),
		mcp.WithObject("position",
			mcp.Required(),
			mcp.Description("Position of the symbol"),
		),
	)

	s.AddTool(definitionTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileURI, ok := request.Params.Arguments["file_uri"].(string)
		if !ok {
			return nil, errors.New("file_uri must be a string")
		}

		positionObj, ok := request.Params.Arguments["position"].(map[string]any)
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
		mcp.WithDescription("Find all references to a symbol"),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("URI of the file"),
		),
		mcp.WithObject("position",
			mcp.Required(),
			mcp.Description("Position of the symbol"),
		),
	)

	s.AddTool(referencesTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileURI, ok := request.Params.Arguments["file_uri"].(string)
		if !ok {
			return nil, errors.New("file_uri must be a string")
		}

		positionObj, ok := request.Params.Arguments["position"].(map[string]any)
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
		mcp.WithDescription("Get diagnostics for a file"),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("URI of the file"),
		),
	)

	s.AddTool(diagnosticsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileURI, ok := request.Params.Arguments["file_uri"].(string)
		if !ok {
			return nil, errors.New("file_uri must be a string")
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

func (t *LSPTools) registerHover(s *server.MCPServer) {
	hoverTool := mcp.NewTool("get_hover_info",
		mcp.WithDescription("Get hover information for a symbol"),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("URI of the file"),
		),
		mcp.WithObject("position",
			mcp.Required(),
			mcp.Description("Position of the symbol"),
		),
	)

	s.AddTool(hoverTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileURI, ok := request.Params.Arguments["file_uri"].(string)
		if !ok {
			return nil, errors.New("file_uri must be a string")
		}

		positionObj, ok := request.Params.Arguments["position"].(map[string]any)
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

		if t.client == nil {
			return nil, errors.New("LSP client not initialized")
		}

		info, err := t.client.GetHover(fileURI, int(line), int(character))
		if err != nil {
			if strings.Contains(err.Error(), "client closed") {
				return nil, fmt.Errorf("LSP service not available, please restart the server: %w", err)
			}
			return nil, fmt.Errorf("failed to get hover info: %w", err)
		}

		return mcp.NewToolResultText(info), nil
	})
}

func (t *LSPTools) registerCompletion(s *server.MCPServer) {
	completionTool := mcp.NewTool("get_completion",
		mcp.WithDescription("Get completion suggestions at a position"),
		mcp.WithString("file_uri",
			mcp.Required(),
			mcp.Description("URI of the file"),
		),
		mcp.WithObject("position",
			mcp.Required(),
			mcp.Description("Position where to get completion"),
		),
	)

	s.AddTool(completionTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fileURI, ok := request.Params.Arguments["file_uri"].(string)
		if !ok {
			return nil, errors.New("file_uri must be a string")
		}

		positionObj, ok := request.Params.Arguments["position"].(map[string]any)
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

		if t.client == nil {
			return nil, errors.New("LSP client not initialized")
		}

		completions, err := t.client.GetCompletion(fileURI, int(line), int(character))
		if err != nil {
			if strings.Contains(err.Error(), "client closed") {
				return nil, fmt.Errorf("LSP service not available, please restart the server: %w", err)
			}
			return nil, fmt.Errorf("failed to get completions: %w", err)
		}

		result, err := json.Marshal(completions)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}

		return mcp.NewToolResultText(string(result)), nil
	})
}

func (t *LSPTools) registerCoverageAnalysis(s *server.MCPServer) {
	coverageTool := mcp.NewTool("analyze_coverage",
		mcp.WithDescription("Analyze test coverage for Go code"),
		mcp.WithString("path",
			mcp.Description("Path to the package or directory to analyze. If not provided, analyzes the entire project."),
		),
		mcp.WithString("output_format",
			mcp.Description("Format of the coverage output: 'summary' (default) or 'func' (per function)"),
		),
	)

	s.AddTool(coverageTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		packagePath, _ := request.Params.Arguments["path"].(string)
		outputFormat, _ := request.Params.Arguments["output_format"].(string)

		if outputFormat == "" {
			outputFormat = "summary"
		}

		targetPath := "./..."
		if packagePath != "" {
			targetPath = packagePath
		}

		var result strings.Builder

		if outputFormat == "func" {
			cmd := exec.Command("go", "test", targetPath, "-coverprofile=/tmp/go_coverage_temp.out")
			var testOut, testErr bytes.Buffer
			cmd.Stdout = &testOut
			cmd.Stderr = &testErr
			err := cmd.Run()

			if testErr.Len() > 0 {
				result.WriteString("Error:\n")
				result.WriteString(testErr.String())
				result.WriteString("\n")
			}

			if testOut.Len() > 0 {
				result.WriteString("Test output:\n")
				result.WriteString(testOut.String())
				result.WriteString("\n")
			}

			if err == nil {
				coverCmd := exec.Command("go", "tool", "cover", "-func=/tmp/go_coverage_temp.out")
				var coverOut bytes.Buffer
				coverCmd.Stdout = &coverOut

				if err := coverCmd.Run(); err == nil && coverOut.Len() > 0 {
					result.WriteString("\nFunction coverage:\n")
					result.WriteString(coverOut.String())
				} else {
					result.WriteString("\nNo function coverage data available")
				}

				os.Remove("/tmp/go_coverage_temp.out")
			}
		} else {
			cmd := exec.Command("go", "test", targetPath, "-cover")
			var out, errOut bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &errOut
			cmd.Run()

			if errOut.Len() > 0 {
				result.WriteString("Error:\n")
				result.WriteString(errOut.String())
				result.WriteString("\n")
			}

			if out.Len() > 0 {
				result.WriteString("Coverage summary:\n")
				result.WriteString(out.String())
			} else {
				result.WriteString("No coverage data available")
			}
		}

		return mcp.NewToolResultText(result.String()), nil
	})
}
