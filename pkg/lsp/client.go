package lsp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Client gère la communication avec le serveur LSP (gopls)
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	mutex  sync.Mutex
	nextID int
	closed bool
}

// Message JSON-RPC de base
type jsonRPCMessage struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Method  string `json:"method,omitempty"`
	Params  any    `json:"params,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

// Position dans un document
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// TextDocumentIdentifier identifie un document
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// TextDocumentPositionParams paramètres pour les requêtes de position
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// NewClient crée un nouveau client LSP connecté à gopls
func NewClient() (*Client, error) {
	// Lancement du processus gopls
	cmd := exec.Command("gopls", "serve", "-rpc.trace")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to start gopls: %w", err)
	}

	client := &Client{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		nextID: 1,
	}

	// Initialiser la connexion LSP
	if err := client.initialize(); err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to initialize gopls: %w", err)
	}

	return client, nil
}

// initialize envoie la requête d'initialisation au serveur LSP
func (c *Client) initialize() error {
	initParams := map[string]any{
		"processId": nil,
		"clientInfo": map[string]any{
			"name":    "mcplspgo",
			"version": "0.1.0",
		},
		"rootUri":      nil,
		"capabilities": map[string]any{},
	}

	// Ignorer le résultat car nous ne l'utilisons pas
	_, err := c.call("initialize", initParams)
	if err != nil {
		return err
	}

	// Envoyer la notification "initialized"
	return c.notify("initialized", map[string]any{})
}

// call envoie une requête JSON-RPC au serveur LSP
func (c *Client) call(method string, params any) (json.RawMessage, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return nil, errors.New("client is closed")
	}

	id := c.nextID
	c.nextID++

	message := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Envoyer une requête avec Content-Length
	content := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
	_, err = c.stdin.Write([]byte(content))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// TODO: Lire la réponse correctement
	// Cette implémentation est simplifiée et ne gère pas les réponses complètes

	// Lire l'en-tête Content-Length
	var contentLength int
	var header bytes.Buffer
	for {
		b := make([]byte, 1)
		_, err := c.stdout.Read(b)
		if err != nil {
			return nil, fmt.Errorf("failed to read header: %w", err)
		}
		header.Write(b)

		if bytes.Contains(header.Bytes(), []byte("\r\n\r\n")) {
			fmt.Sscanf(header.String(), "Content-Length: %d\r\n\r\n", &contentLength)
			break
		}
	}

	// Lire le corps du message
	body := make([]byte, contentLength)
	_, err = io.ReadFull(c.stdout, body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var response jsonRPCMessage
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("server error: %v", response.Error)
	}

	result, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return result, nil
}

// notify envoie une notification JSON-RPC au serveur LSP
func (c *Client) notify(method string, params any) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return errors.New("client is closed")
	}

	message := jsonRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Envoyer une notification avec Content-Length
	content := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
	_, err = c.stdin.Write([]byte(content))
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}

// GetDefinition obtient la définition d'un symbole à la position donnée
func (c *Client) GetDefinition(filePath string, line, column int) (any, error) {
	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{
			URI: "file://" + filePath,
		},
		Position: Position{
			Line:      line - 1, // LSP est 0-basé, notre API est 1-basée
			Character: column - 1,
		},
	}

	result, err := c.call("textDocument/definition", params)
	if err != nil {
		return nil, err
	}

	// Désérialiser le résultat
	var locations []any
	if err := json.Unmarshal(result, &locations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal definition result: %w", err)
	}

	return locations, nil
}

// GetReferences trouve toutes les références à un symbole
func (c *Client) GetReferences(filePath string, line, column int) (any, error) {
	params := map[string]any{
		"textDocument": map[string]any{
			"uri": "file://" + filePath,
		},
		"position": map[string]any{
			"line":      line - 1,
			"character": column - 1,
		},
		"context": map[string]any{
			"includeDeclaration": true,
		},
	}

	result, err := c.call("textDocument/references", params)
	if err != nil {
		return nil, err
	}

	// Désérialiser le résultat
	var locations []any
	if err := json.Unmarshal(result, &locations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal references result: %w", err)
	}

	return locations, nil
}

// GetDiagnostics obtient les diagnostics pour un fichier spécifique
func (c *Client) GetDiagnostics(filePath string) (any, error) {
	// Pour obtenir les diagnostics, nous devons d'abord ouvrir le document
	params := map[string]any{
		"textDocument": map[string]any{
			"uri":        "file://" + filePath,
			"languageId": "go",
			"version":    1,
			"text":       "", // Idéalement, il faudrait lire le contenu du fichier
		},
	}

	err := c.notify("textDocument/didOpen", params)
	if err != nil {
		return nil, err
	}

	// Les diagnostics sont normalement envoyés de manière asynchrone par le serveur
	// Cette implémentation est simplifiée et ne capture pas les diagnostics
	return map[string]string{
		"status": "Diagnostics requested, will be published asynchronously",
	}, nil
}

// Close ferme la connexion avec le serveur LSP
func (c *Client) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.closed {
		// Envoyer shutdown puis exit
		c.call("shutdown", nil)
		c.notify("exit", nil)

		if c.stdin != nil {
			c.stdin.Close()
		}

		if c.stdout != nil {
			c.stdout.Close()
		}

		if c.cmd != nil && c.cmd.Process != nil {
			c.cmd.Process.Kill()
			c.cmd.Wait()
		}

		c.closed = true
	}
}
