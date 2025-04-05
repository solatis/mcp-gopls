# MCP LSP Go

Un Model Control Protocol (MCP) pour fournir aux IA comme Claude des outils pour interagir avec le LSP (Language Server Protocol) de Go.

## Objectif

Le but de ce MCP est d'aider les assistants d'IA à:
- Utiliser les versions récentes et modernes de Go
- Éviter l'utilisation de syntaxes ou fonctionnalités dépréciées
- Suivre les meilleures pratiques actuelles de Go
- Obtenir des informations précises sur le code Go via le LSP

## Fonctionnalités

- Connexion au serveur LSP de Go (gopls)
- Interrogation du LSP pour obtenir:
  - Définitions de fonctions/types
  - Références à des symboles
  - Diagnostics et erreurs de code
  - Suggestions de refactoring
- Vérification de compatibilité avec les versions récentes de Go
- Détection de patterns obsolètes

## Structure du projet

```
.
├── cmd
│   └── mcplspgo        # Point d'entrée de l'application
├── pkg
│   ├── lsp             # Client LSP pour communiquer avec gopls
│   └── mcp             # Implémentation du protocole MCP
```

## Installation

```bash
go install github.com/hloiseaufcms/mcplspgo/cmd/mcplspgo@latest
```

## Configuration avec Cursor

Pour utiliser ce MCP avec Cursor, créez ou modifiez un fichier `.cursor/mcp-config.json` à la racine de votre projet:

```json
{
  "mcpServers": {
    "go-lsp-mcp": {
      "command": "mcplspgo",
      "args": []
    }
  }
}
```

## Configuration avec Claude Desktop

Pour Claude Desktop, modifiez le fichier de configuration:
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

Exemple de configuration:

```json
{
  "mcpServers": {
    "go-lsp-mcp": {
      "command": "mcplspgo"
    }
  }
}
```

## Utilisation avec le client d'exemple

Le projet inclut un client exemple pour tester les fonctionnalités:

```bash
# Compiler le client d'exemple
go build -o simple_client examples/simple_client.go

# Afficher la version de Go et les fonctionnalités recommandées
./simple_client version

# Obtenir toutes les meilleures pratiques Go
./simple_client best_practices

# Lister tous les outils disponibles
./simple_client list_tools
```

## Développement

```bash
git clone https://github.com/hloiseaufcms/mcplspgo.git
cd mcplspgo
go mod tidy
go run cmd/mcplspgo/main.go
```

Pour plus de détails sur l'intégration avec Cursor, consultez le fichier [docs/cursor_integration.md](docs/cursor_integration.md).

## Licence

MIT 