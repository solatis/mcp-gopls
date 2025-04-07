package client

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hloiseaufcms/mcp-gopls/pkg/lsp/protocol"
)

type GoplsClient struct {
	cmd         *exec.Cmd
	transport   *protocol.Transport
	nextID      int64
	closed      atomic.Bool
	mutex       sync.Mutex
	initialized bool
}

func NewGoplsClient() (*GoplsClient, error) {
	goplsPath, err := exec.LookPath("gopls")
	if err != nil {
		return nil, fmt.Errorf("gopls is not installed or not in PATH: %w", err)
	}

	cmd := exec.Command(goplsPath, "serve", "-rpc.trace", "-logfile=auto")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			log.Printf("gopls stderr: %s", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Printf("error reading stderr: %v", err)
		}
	}()

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to start gopls: %w", err)
	}

	var client *GoplsClient
	defer func() {
		if client == nil && cmd.Process != nil {
			log.Printf("Cleaning up gopls process after initialization failure")
			_ = cmd.Process.Kill()
		}
	}()

	bufferedStdout := bufio.NewReader(stdout)
	bufferedStdin := bufio.NewWriter(stdin)

	transport := protocol.NewTransport(bufferedStdout, bufferedStdin)

	client = &GoplsClient{
		cmd:         cmd,
		transport:   transport,
		nextID:      1,
		initialized: false,
	}

	client.closed.Store(false)

	log.Printf("✅ Gopls client created successfully")
	return client, nil
}

func (c *GoplsClient) call(method string, params any) (*protocol.JSONRPCMessage, error) {
	c.mutex.Lock()
	log.Printf("⏳ Calling method: %s", method)
	if c.closed.Load() {
		c.mutex.Unlock()
		log.Printf("❌ Client closed, cannot call %s", method)
		return nil, fmt.Errorf("client closed")
	}

	if method != "initialize" && !c.initialized && method != "shutdown" {
		c.mutex.Unlock()
		log.Printf("❌ Client not initialized, cannot call %s", method)
		return nil, fmt.Errorf("client not initialized")
	}

	id := atomic.AddInt64(&c.nextID, 1)
	req, err := protocol.NewRequest(id, method, params)
	if err != nil {
		c.mutex.Unlock()
		log.Printf("❌ Error creating request: %v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	log.Println("✓ Request created")

	if err := c.transport.SendMessage(req); err != nil {
		c.closed.Store(true)
		c.mutex.Unlock()
		log.Printf("❌ Error sending request: %v", err)
		return nil, fmt.Errorf("failed to send request (client closed): %w", err)
	}
	c.mutex.Unlock()

	startTime := time.Now()
	maxWaitTime := 30 * time.Second
	for time.Since(startTime) < maxWaitTime {
		resp, err := c.transport.ReceiveMessage()
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				return nil, fmt.Errorf("timeout receiving response: %w", err)
			}
			c.closed.Store(true)
			return nil, fmt.Errorf("failed to receive response (client closed): %w", err)
		}

		var respID int64
		switch v := resp.ID.(type) {
		case float64:
			respID = int64(v)
		case int64:
			respID = v
		case json.Number:
			respID64, err := v.Int64()
			if err != nil {
				log.Printf("⚠️ Invalid ID format in response: %v", resp.ID)
				continue
			}
			respID = respID64
		default:
			log.Printf("⚠️ Unsupported ID type in response: %T", resp.ID)
			continue
		}

		if respID != id {
			log.Printf("⚠️ Response ID (%v) does not match request ID (%d), ignored", resp.ID, id)
			continue
		}

		respBytes, _ := json.MarshalIndent(resp, "", "  ")
		log.Printf("📥 Response content: %s", string(respBytes))

		if resp.Error != nil {
			return nil, fmt.Errorf("LSP error: %s (code: %d)", resp.Error.Message, resp.Error.Code)
		}

		return resp, nil
	}

	return nil, fmt.Errorf("no response with matching ID after %v seconds", maxWaitTime.Seconds())
}

func (c *GoplsClient) notify(method string, params any) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed.Load() {
		return fmt.Errorf("client closed")
	}

	notif, err := protocol.NewNotification(method, params)
	if err != nil {
		return err
	}

	if err := c.transport.SendMessage(notif); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}

func (c *GoplsClient) Initialize() error {
	if c.initialized {
		return nil
	}
	log.Println("Initializing LSP client...")

	if c.closed.Load() {
		return fmt.Errorf("cannot initialize: client closed")
	}
	log.Println("Client LSP not closed")

	initParams := map[string]any{
		"processId": nil,
		"clientInfo": map[string]any{
			"name":    "mcp-gopls",
			"version": "1.0.0",
		},
		"rootUri": "file:///",
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"synchronization": map[string]any{
					"dynamicRegistration": true,
					"willSave":            true,
					"willSaveWaitUntil":   true,
					"didSave":             true,
				},
				"completion": map[string]any{
					"dynamicRegistration": true,
					"completionItem": map[string]any{
						"snippetSupport": true,
					},
				},
				"hover": map[string]any{
					"dynamicRegistration": true,
					"contentFormat":       []string{"markdown", "plaintext"},
				},
				"signatureHelp": map[string]any{
					"dynamicRegistration": true,
				},
				"definition": map[string]any{
					"dynamicRegistration": true,
				},
				"references": map[string]any{
					"dynamicRegistration": true,
				},
				"documentSymbol": map[string]any{
					"dynamicRegistration": true,
				},
				"formatting": map[string]any{
					"dynamicRegistration": true,
				},
				"documentHighlight": map[string]any{
					"dynamicRegistration": true,
				},
				"publishDiagnostics": map[string]any{
					"relatedInformation": true,
				},
			},
			"workspace": map[string]any{
				"applyEdit": true,
				"didChangeConfiguration": map[string]any{
					"dynamicRegistration": true,
				},
				"symbol": map[string]any{
					"dynamicRegistration": true,
				},
			},
		},
		"trace": "verbose",
	}

	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		log.Printf("Initialization attempt %d/3", attempt)
		_, err = c.call("initialize", initParams)
		if err == nil {
			break
		}

		if strings.Contains(err.Error(), "timeout") {
			log.Printf("Timeout during initialization (attempt %d): %v", attempt, err)
			if attempt < 3 {
				time.Sleep(500 * time.Millisecond)
				continue
			}
		} else {
			return fmt.Errorf("failed to initialize (attempt %d): %w", attempt, err)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to initialize after 3 attempts: %w", err)
	}

	log.Println("Initialization succeeded")
	c.initialized = true
	log.Println("LSP client initialized")

	initNotif := map[string]any{}
	if err := c.notify("initialized", initNotif); err != nil {
		c.initialized = false
		return fmt.Errorf("failed to send notification 'initialized': %w", err)
	}
	log.Println("Notification 'initialized' sent")

	return nil
}

func (c *GoplsClient) Shutdown() error {
	_, err := c.call("shutdown", nil)
	if err != nil {
		return fmt.Errorf("failed to shutdown: %w", err)
	}

	return nil
}

func (c *GoplsClient) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}

	var errs []error

	if c.initialized {
		if err := c.Shutdown(); err != nil {
			errs = append(errs, fmt.Errorf("error during shutdown: %w", err))
		}

		if err := c.notify("exit", nil); err != nil {
			errs = append(errs, fmt.Errorf("error sending exit notification: %w", err))
		}
		c.initialized = false
	}

	if c.cmd != nil && c.cmd.Process != nil {
		if err := c.cmd.Process.Kill(); err != nil {
			errs = append(errs, fmt.Errorf("error killing process: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}

	return nil
}

func (c *GoplsClient) GoToDefinition(uri string, line, character int) ([]protocol.Location, error) {
	params := protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: uri,
		},
		Position: protocol.Position{
			Line:      line,
			Character: character,
		},
	}

	resp, err := c.call("textDocument/definition", params)
	if err != nil {
		return nil, err
	}

	var locations []protocol.Location
	if err := resp.ParseResult(&locations); err != nil {
		return nil, fmt.Errorf("failed to decode definition results: %w", err)
	}

	return locations, nil
}

func (c *GoplsClient) FindReferences(uri string, line, character int, includeDeclaration bool) ([]protocol.Location, error) {
	params := protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: uri,
			},
			Position: protocol.Position{
				Line:      line,
				Character: character,
			},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: includeDeclaration,
		},
	}

	resp, err := c.call("textDocument/references", params)
	if err != nil {
		return nil, err
	}

	var locations []protocol.Location
	if err := resp.ParseResult(&locations); err != nil {
		return nil, fmt.Errorf("failed to decode reference results: %w", err)
	}

	return locations, nil
}

func (c *GoplsClient) GetDiagnostics(uri string) ([]protocol.Diagnostic, error) {
	if err := c.DidOpen(uri, "go", ""); err != nil {
		return nil, err
	}

	return []protocol.Diagnostic{}, nil
}

func (c *GoplsClient) DidOpen(uri, languageID, text string) error {
	log.Printf("📝 Opening document: %s", uri)

	if text == "" {
		filePath := strings.TrimPrefix(uri, "file://")
		if runtime.GOOS == "windows" {
			filePath = strings.TrimPrefix(filePath, "/")
			filePath = strings.ReplaceAll(filePath, "/", "\\")
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("⚠️ Unable to read file content: %v", err)
			text = ""
		} else {
			text = string(content)
			log.Printf("✓ File read successfully (%d bytes)", len(text))
		}
	}

	params := map[string]any{
		"textDocument": map[string]any{
			"uri":        uri,
			"languageId": languageID,
			"version":    1,
			"text":       text,
		},
	}

	err := c.notify("textDocument/didOpen", params)
	if err != nil {
		log.Printf("❌ Error opening document: %v", err)
		return fmt.Errorf("failed to open document: %w", err)
	}

	log.Printf("✓ Document opened successfully: %s", uri)
	return nil
}

func (c *GoplsClient) DidClose(uri string) error {
	params := map[string]any{
		"textDocument": map[string]any{
			"uri": uri,
		},
	}

	return c.notify("textDocument/didClose", params)
}

func (c *GoplsClient) GetHover(uri string, line, character int) (string, error) {
	log.Printf("🔍 Requesting hover information for %s position L%d:C%d", uri, line, character)

	if err := c.DidOpen(uri, "go", ""); err != nil {
		log.Printf("⚠️ Warning opening document: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	params := protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: uri,
		},
		Position: protocol.Position{
			Line:      line,
			Character: character,
		},
	}

	resp, err := c.call("textDocument/hover", params)
	if err != nil {
		return "", fmt.Errorf("failed to request hover: %w", err)
	}

	if resp == nil {
		return "", fmt.Errorf("no response received for hover")
	}

	if len(resp.Result) == 0 || string(resp.Result) == "null" {
		return "", fmt.Errorf("no hover information available for this position")
	}

	var result map[string]any
	if err := resp.ParseResult(&result); err != nil {
		return "", fmt.Errorf("failed to decode hover result: %w", err)
	}

	log.Printf("📋 Decoded hover response: %+v", result)

	if len(result) == 0 {
		return "", fmt.Errorf("no hover information available for this position")
	}

	if contents, ok := result["contents"].(map[string]any); ok {
		if value, ok := contents["value"].(string); ok {
			return value, nil
		}
		if kind, ok := contents["kind"].(string); ok && kind == "markdown" {
			if value, ok := contents["value"].(string); ok {
				return value, nil
			}
		}
	}

	if contents, ok := result["contents"].(string); ok {
		return contents, nil
	}

	if contentsArray, ok := result["contents"].([]any); ok && len(contentsArray) > 0 {
		if firstItem, ok := contentsArray[0].(map[string]any); ok {
			if value, ok := firstItem["value"].(string); ok {
				return value, nil
			}
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	if len(data) == 2 && string(data) == "{}" {
		return "", fmt.Errorf("no hover information available for this position")
	}

	return string(data), nil
}

func (c *GoplsClient) GetCompletion(uri string, line, character int) ([]string, error) {
	params := protocol.TextDocumentPositionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: uri,
		},
		Position: protocol.Position{
			Line:      line,
			Character: character,
		},
	}

	resp, err := c.call("textDocument/completion", params)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := resp.ParseResult(&result); err != nil {
		return nil, fmt.Errorf("failed to decode completion result: %w", err)
	}

	var completions []string
	if items, ok := result["items"].([]any); ok {
		for _, item := range items {
			if itemMap, ok := item.(map[string]any); ok {
				if label, ok := itemMap["label"].(string); ok {
					completions = append(completions, label)
				}
			}
		}
	}

	return completions, nil
}
