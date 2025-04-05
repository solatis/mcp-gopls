package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: simple_client <command>")
		fmt.Println("Available commands: version, best_practices")
		os.Exit(1)
	}

	command := os.Args[1]

	// Créer une requête JSON-RPC 2.0 appropriée
	var request map[string]any

	switch command {
	case "version":
		request = map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "callTool",
			"params": map[string]any{
				"tool":      "get_go_version",
				"arguments": map[string]any{},
			},
		}
	case "best_practices":
		request = map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "callTool",
			"params": map[string]any{
				"tool": "get_best_practices",
				"arguments": map[string]any{
					"aspect": "all",
				},
			},
		}
	case "list_tools":
		request = map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "listTools",
		}
	case "init":
		request = map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
			"params": map[string]any{
				"capabilities": map[string]any{},
				"clientInfo": map[string]any{
					"name":    "simple-client",
					"version": "1.0.0",
				},
				"protocolVersion": "2024-11-05",
			},
		}
	default:
		fmt.Printf("Unknown command: %s\n", command)
		os.Exit(1)
	}

	// Exécuter le MCP server comme un processus externe et communiquer via STDIO
	cmd := exec.Command("mcplspgo")
	cmd.Stderr = os.Stderr

	// Préparer stdin/stdout
	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Printf("Error creating stdin pipe: %v\n", err)
		os.Exit(1)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Printf("Error creating stdout pipe: %v\n", err)
		os.Exit(1)
	}

	// Démarrer le processus
	if err := cmd.Start(); err != nil {
		fmt.Printf("Error starting mcplspgo: %v\n", err)
		os.Exit(1)
	}

	// Si ce n'est pas la commande "init", envoyer d'abord une requête d'initialisation
	if command != "init" {
		initRequest := map[string]any{
			"jsonrpc": "2.0",
			"id":      0,
			"method":  "initialize",
			"params": map[string]any{
				"capabilities": map[string]any{},
				"clientInfo": map[string]any{
					"name":    "simple-client",
					"version": "1.0.0",
				},
				"protocolVersion": "2024-11-05",
			},
		}

		initData, err := json.Marshal(initRequest)
		if err != nil {
			fmt.Printf("Error serializing init request: %v\n", err)
			os.Exit(1)
		}

		// Envoyer la requête d'initialisation
		if _, err := stdin.Write(initData); err != nil {
			fmt.Printf("Error sending init request: %v\n", err)
			os.Exit(1)
		}
		if _, err := stdin.Write([]byte("\n")); err != nil {
			fmt.Printf("Error sending newline: %v\n", err)
			os.Exit(1)
		}

		// Lire la réponse d'initialisation
		var initResponseBuffer bytes.Buffer
		buffer := make([]byte, 4096)

		n, err := stdout.Read(buffer)
		if err != nil {
			fmt.Printf("Error reading init response: %v\n", err)
			os.Exit(1)
		}

		initResponseBuffer.Write(buffer[:n])

		// Afficher la réponse d'initialisation en mode verbeux
		fmt.Println("== Initialization Response ==")
		fmt.Println(strings.TrimSpace(initResponseBuffer.String()))
		fmt.Println("============================")
	}

	// Sérialiser la requête principale
	data, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("Error serializing request: %v\n", err)
		os.Exit(1)
	}

	// Envoyer la requête principale
	if _, err := stdin.Write(data); err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		os.Exit(1)
	}
	if _, err := stdin.Write([]byte("\n")); err != nil {
		fmt.Printf("Error sending newline: %v\n", err)
		os.Exit(1)
	}

	// Lire la réponse
	var responseBuffer bytes.Buffer
	buffer := make([]byte, 4096)

	n, err := stdout.Read(buffer)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		os.Exit(1)
	}

	responseBuffer.Write(buffer[:n])

	// Fermer le processus proprement
	stdin.Close()
	cmd.Wait()

	// Afficher le résultat en JSON formaté
	var prettyJSON bytes.Buffer
	responseStr := strings.TrimSpace(responseBuffer.String())
	if err := json.Indent(&prettyJSON, []byte(responseStr), "", "  "); err != nil {
		fmt.Printf("Error formatting response: %v\n", err)
		fmt.Println("Raw response:", responseStr)
		os.Exit(1)
	}

	fmt.Println(prettyJSON.String())
}
