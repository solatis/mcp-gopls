package mcp

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/hloiseaufcms/MCPLSPGO/pkg/lsp"
)

// Server représente le serveur MCP qui expose les fonctionnalités LSP
type Server struct {
	lspClient *lsp.Client
}

// NewServer crée une nouvelle instance du serveur MCP
func NewServer() *Server {
	client, err := lsp.NewClient()
	if err != nil {
		log.Printf("Warning: LSP client initialization failed: %v", err)
		return &Server{lspClient: nil}
	}

	return &Server{
		lspClient: client,
	}
}

// ServeStdio sert le MCP sur l'entrée/sortie standard
func (s *Server) ServeStdio() error {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	// Boucle principale de traitement des messages MCP
	for {
		var request struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      interface{}     `json:"id,omitempty"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params,omitempty"`
		}

		if err := decoder.Decode(&request); err != nil {
			if err == io.EOF {
				return nil // Fin normale
			}
			log.Printf("Error decoding request: %v", err)
			return err
		}

		log.Printf("Received request: %+v", request)

		// Gérer les différentes méthodes du protocole MCP
		var response interface{}

		switch request.Method {
		case "initialize":
			response = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result": map[string]interface{}{
					"serverInfo": map[string]interface{}{
						"name":    "go-lsp-mcp",
						"version": "1.0.0",
					},
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]interface{}{
						"tools": map[string]interface{}{},
					},
				},
			}

		case "listTools":
			// Retourner la liste des outils disponibles
			tools := []map[string]interface{}{
				{
					"name":        "get_definition",
					"description": "Obtient la définition d'un symbole à la position donnée",
					"parameters": map[string]interface{}{
						"type":     "object",
						"required": []string{"file_path", "line", "column"},
						"properties": map[string]interface{}{
							"file_path": map[string]interface{}{
								"type":        "string",
								"description": "Chemin du fichier",
							},
							"line": map[string]interface{}{
								"type":        "integer",
								"description": "Numéro de ligne (1-indexé)",
							},
							"column": map[string]interface{}{
								"type":        "integer",
								"description": "Numéro de colonne (1-indexé)",
							},
						},
					},
				},
				{
					"name":        "get_references",
					"description": "Trouve toutes les références à un symbole",
					"parameters": map[string]interface{}{
						"type":     "object",
						"required": []string{"file_path", "line", "column"},
						"properties": map[string]interface{}{
							"file_path": map[string]interface{}{
								"type":        "string",
								"description": "Chemin du fichier",
							},
							"line": map[string]interface{}{
								"type":        "integer",
								"description": "Numéro de ligne (1-indexé)",
							},
							"column": map[string]interface{}{
								"type":        "integer",
								"description": "Numéro de colonne (1-indexé)",
							},
						},
					},
				},
				{
					"name":        "check_diagnostics",
					"description": "Vérifie les diagnostics (erreurs, avertissements) dans un fichier",
					"parameters": map[string]interface{}{
						"type":     "object",
						"required": []string{"file_path"},
						"properties": map[string]interface{}{
							"file_path": map[string]interface{}{
								"type":        "string",
								"description": "Chemin du fichier",
							},
						},
					},
				},
				{
					"name":        "get_go_version",
					"description": "Obtient des informations sur la dernière version de Go",
					"parameters": map[string]interface{}{
						"type":       "object",
						"properties": map[string]interface{}{},
					},
				},
				{
					"name":        "check_deprecated_features",
					"description": "Vérifie les fonctionnalités obsolètes dans un fichier",
					"parameters": map[string]interface{}{
						"type":     "object",
						"required": []string{"file_path"},
						"properties": map[string]interface{}{
							"file_path": map[string]interface{}{
								"type":        "string",
								"description": "Chemin du fichier",
							},
						},
					},
				},
				{
					"name":        "get_best_practices",
					"description": "Récupère les meilleures pratiques Go",
					"parameters": map[string]interface{}{
						"type":     "object",
						"required": []string{"aspect"},
						"properties": map[string]interface{}{
							"aspect": map[string]interface{}{
								"type":        "string",
								"description": "Aspect des meilleures pratiques (all, latest_version, recommended_features, deprecated_features, code_style)",
							},
						},
					},
				},
				{
					"name":        "search_documentation",
					"description": "Recherche dans la documentation Go",
					"parameters": map[string]interface{}{
						"type":     "object",
						"required": []string{"query"},
						"properties": map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "string",
								"description": "Terme de recherche",
							},
						},
					},
				},
				{
					"name":        "format_code",
					"description": "Formate un morceau de code Go",
					"parameters": map[string]interface{}{
						"type":     "object",
						"required": []string{"code"},
						"properties": map[string]interface{}{
							"code": map[string]interface{}{
								"type":        "string",
								"description": "Code Go à formater",
							},
						},
					},
				},
			}

			response = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      request.ID,
				"result": map[string]interface{}{
					"tools": tools,
				},
			}

		case "callTool":
			var params struct {
				Tool      string          `json:"tool"`
				Arguments json.RawMessage `json:"arguments"`
			}
			if err := json.Unmarshal(request.Params, &params); err != nil {
				// Erreur de décodage des paramètres
				response = s.createErrorResponse(request.ID, -32602, "Invalid params: "+err.Error())
				break
			}

			var result interface{}
			var err error

			switch params.Tool {
			case "get_definition":
				var args struct {
					FilePath string `json:"file_path"`
					Line     int    `json:"line"`
					Column   int    `json:"column"`
				}
				if err := json.Unmarshal(params.Arguments, &args); err != nil {
					// Erreur de décodage des arguments
					response = s.createErrorResponse(request.ID, -32602, "Invalid arguments: "+err.Error())
					break
				}
				result, err = s.handleGetDefinition(args.FilePath, args.Line, args.Column)

			case "get_references":
				var args struct {
					FilePath string `json:"file_path"`
					Line     int    `json:"line"`
					Column   int    `json:"column"`
				}
				if err := json.Unmarshal(params.Arguments, &args); err != nil {
					response = s.createErrorResponse(request.ID, -32602, "Invalid arguments: "+err.Error())
					break
				}
				result, err = s.handleGetReferences(args.FilePath, args.Line, args.Column)

			case "check_diagnostics":
				var args struct {
					FilePath string `json:"file_path"`
				}
				if err := json.Unmarshal(params.Arguments, &args); err != nil {
					response = s.createErrorResponse(request.ID, -32602, "Invalid arguments: "+err.Error())
					break
				}
				result, err = s.handleCheckDiagnostics(args.FilePath)

			case "get_go_version":
				result, err = s.handleGetGoVersion()

			case "check_deprecated_features":
				var args struct {
					FilePath string `json:"file_path"`
				}
				if err := json.Unmarshal(params.Arguments, &args); err != nil {
					response = s.createErrorResponse(request.ID, -32602, "Invalid arguments: "+err.Error())
					break
				}
				result, err = s.handleCheckDeprecatedFeatures(args.FilePath)

			case "get_best_practices":
				var args struct {
					Aspect string `json:"aspect"`
				}
				if err := json.Unmarshal(params.Arguments, &args); err != nil {
					response = s.createErrorResponse(request.ID, -32602, "Invalid arguments: "+err.Error())
					break
				}
				result, err = s.handleGetBestPractices(args.Aspect)

			case "search_documentation":
				var args struct {
					Query string `json:"query"`
				}
				if err := json.Unmarshal(params.Arguments, &args); err != nil {
					response = s.createErrorResponse(request.ID, -32602, "Invalid arguments: "+err.Error())
					break
				}
				result, err = s.handleSearchDocumentation(args.Query)

			case "format_code":
				var args struct {
					Code string `json:"code"`
				}
				if err := json.Unmarshal(params.Arguments, &args); err != nil {
					response = s.createErrorResponse(request.ID, -32602, "Invalid arguments: "+err.Error())
					break
				}
				result, err = s.handleFormatCode(args.Code)

			default:
				response = s.createErrorResponse(request.ID, -32601, "Method not found: "+params.Tool)
			}

			if err != nil {
				response = s.createErrorResponse(request.ID, -32000, err.Error())
			} else if response == nil {
				response = map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      request.ID,
					"result": map[string]interface{}{
						"content": result,
					},
				}
			}

		default:
			// Méthode inconnue
			response = s.createErrorResponse(request.ID, -32601, "Method not found: "+request.Method)
		}

		// S'assurer que l'ID n'est jamais null (utiliser 0 comme valeur par défaut)
		if response != nil {
			respMap, isMap := response.(map[string]interface{})
			if isMap {
				if respMap["id"] == nil {
					respMap["id"] = 0
				}
			}

			log.Printf("Sending response: %+v", response)
			if err := encoder.Encode(response); err != nil {
				log.Printf("Error encoding response: %v", err)
				return err
			}
		}
	}
}

// createErrorResponse crée une réponse d'erreur JSON-RPC
func (s *Server) createErrorResponse(id interface{}, code int, message string) map[string]interface{} {
	// Assurer que l'ID n'est jamais null
	if id == nil {
		id = 0
	}

	return map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
}

// HandleRequest gère les requêtes entrantes au MCP via HTTP (fonction maintenue pour compatibilité)
func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Function string          `json:"function"`
		Args     json.RawMessage `json:"args"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	var response interface{}
	var err error

	switch request.Function {
	case "get_definition":
		var args struct {
			FilePath string `json:"file_path"`
			Line     int    `json:"line"`
			Column   int    `json:"column"`
		}
		if err := json.Unmarshal(request.Args, &args); err != nil {
			http.Error(w, "Invalid arguments", http.StatusBadRequest)
			return
		}
		response, err = s.handleGetDefinition(args.FilePath, args.Line, args.Column)

	case "get_references":
		var args struct {
			FilePath string `json:"file_path"`
			Line     int    `json:"line"`
			Column   int    `json:"column"`
		}
		if err := json.Unmarshal(request.Args, &args); err != nil {
			http.Error(w, "Invalid arguments", http.StatusBadRequest)
			return
		}
		response, err = s.handleGetReferences(args.FilePath, args.Line, args.Column)

	case "check_diagnostics":
		var args struct {
			FilePath string `json:"file_path"`
		}
		if err := json.Unmarshal(request.Args, &args); err != nil {
			http.Error(w, "Invalid arguments", http.StatusBadRequest)
			return
		}
		response, err = s.handleCheckDiagnostics(args.FilePath)

	case "get_go_version":
		response, err = s.handleGetGoVersion()

	case "check_deprecated_features":
		var args struct {
			FilePath string `json:"file_path"`
		}
		if err := json.Unmarshal(request.Args, &args); err != nil {
			http.Error(w, "Invalid arguments", http.StatusBadRequest)
			return
		}
		response, err = s.handleCheckDeprecatedFeatures(args.FilePath)

	case "get_best_practices":
		var args struct {
			Aspect string `json:"aspect"`
		}
		if err := json.Unmarshal(request.Args, &args); err != nil {
			http.Error(w, "Invalid arguments", http.StatusBadRequest)
			return
		}
		response, err = s.handleGetBestPractices(args.Aspect)

	case "search_documentation":
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(request.Args, &args); err != nil {
			http.Error(w, "Invalid arguments", http.StatusBadRequest)
			return
		}
		response, err = s.handleSearchDocumentation(args.Query)

	case "format_code":
		var args struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal(request.Args, &args); err != nil {
			http.Error(w, "Invalid arguments", http.StatusBadRequest)
			return
		}
		response, err = s.handleFormatCode(args.Code)

	default:
		http.Error(w, "Unknown function", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// Close ferme proprement les ressources
func (s *Server) Close() {
	if s.lspClient != nil {
		s.lspClient.Close()
	}
}

// Handlers pour les différentes fonctions

func (s *Server) handleGetDefinition(filePath string, line, column int) (interface{}, error) {
	if s.lspClient == nil {
		return nil, ErrLSPClientNotInitialized
	}
	return s.lspClient.GetDefinition(filePath, line, column)
}

func (s *Server) handleGetReferences(filePath string, line, column int) (interface{}, error) {
	if s.lspClient == nil {
		return nil, ErrLSPClientNotInitialized
	}
	return s.lspClient.GetReferences(filePath, line, column)
}

func (s *Server) handleCheckDiagnostics(filePath string) (interface{}, error) {
	if s.lspClient == nil {
		return nil, ErrLSPClientNotInitialized
	}
	return s.lspClient.GetDiagnostics(filePath)
}

func (s *Server) handleGetGoVersion() (interface{}, error) {
	// Utilise notre base de connaissances pour retourner la dernière version et les fonctionnalités récentes
	return GetBestPractices("latest_version")
}

func (s *Server) handleCheckDeprecatedFeatures(filePath string) (interface{}, error) {
	if s.lspClient == nil {
		return nil, ErrLSPClientNotInitialized
	}

	// Pour l'instant, retourne simplement la liste des fonctionnalités obsolètes
	// À terme, analyserait le code pour détecter des utilisations
	deprecated, err := GetBestPractices("deprecated_features")
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"file_path":           filePath,
		"deprecated_features": deprecated,
		"suggestions":         []string{},
	}, nil
}

func (s *Server) handleGetBestPractices(aspect string) (interface{}, error) {
	return GetBestPractices(aspect)
}

func (s *Server) handleSearchDocumentation(query string) (interface{}, error) {
	// Simuler une recherche dans la documentation Go
	// Une véritable implémentation pourrait interroger pkg.go.dev ou une base locale

	// Résultats fictifs pour démonstration
	return map[string]interface{}{
		"query": query,
		"results": []map[string]interface{}{
			{
				"title":       "Documentation Go pour " + query,
				"description": "Documentation officielle pour " + query,
				"url":         "https://pkg.go.dev/search?q=" + query,
			},
		},
	}, nil
}

func (s *Server) handleFormatCode(code string) (interface{}, error) {
	// Dans une véritable implémentation, utiliserait gofmt ou goimports
	// Pour cet exemple, retourne simplement le code inchangé

	return map[string]interface{}{
		"formatted_code": code,
		"message":        "Code formatting not yet implemented. Would use goimports in a full implementation.",
	}, nil
}
