package client

import (
	"github.com/solatis/mcp-gopls/pkg/lsp/protocol"
)

// LSPClient définit l'interface pour un client LSP
type LSPClient interface {
	// Méthodes de base du protocole
	Initialize() error
	Shutdown() error
	Close() error

	// Méthodes de navigation de code
	GoToDefinition(uri string, line, character int) ([]protocol.Location, error)
	FindReferences(uri string, line, character int, includeDeclaration bool) ([]protocol.Location, error)

	// Méthodes de diagnostic
	GetDiagnostics(uri string) ([]protocol.Diagnostic, error)

	// Méthodes de document
	DidOpen(uri, languageID, text string) error
	DidClose(uri string) error

	// Support avancé
	GetHover(uri string, line, character int) (string, error)
	GetCompletion(uri string, line, character int) ([]string, error)

	// Symbol navigation
	GetDocumentSymbols(uri string) ([]protocol.DocumentSymbol, error)
	GetWorkspaceSymbols(query string) ([]protocol.SymbolInformation, error)
	GetImplementations(uri string, line, character int) ([]protocol.Location, error)
}
