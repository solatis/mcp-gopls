# Intégration du MCP LSP Go avec Cursor

Ce document explique comment configurer et utiliser le MCP LSP Go avec l'éditeur Cursor.

## Prérequis

- Cursor installé
- Go 1.18+ installé
- gopls installé (`go install golang.org/x/tools/gopls@latest`)

## Installation du MCP

1. Clonez ce dépôt:
   ```bash
   git clone https://github.com/hloiseaufcms/mcplspgo.git
   cd mcplspgo
   ```

2. Compilez et installez:
   ```bash
   go install ./cmd/mcplspgo
   ```

## Configuration de Cursor

Pour configurer Cursor afin qu'il utilise notre MCP, créez ou modifiez un fichier `.cursor/mcp-config.json` à la racine de votre projet:

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

Alternativement, vous pouvez ajouter la configuration dans les paramètres globaux de Cursor.

## Configuration avec Claude Desktop

Pour Claude Desktop, modifiez le fichier de configuration:
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

Voici un exemple de configuration:

```json
{
  "mcpServers": {
    "go-lsp-mcp": {
      "command": "mcplspgo"
    }
  }
}
```

## Fonctionnalités disponibles

Une fois configuré, le MCP LSP Go fournit les fonctions suivantes à Claude via Cursor:

### `get_definition`

Obtient la définition d'un symbole à une position donnée dans un fichier.

```
get_definition(file_path="/path/to/file.go", line=10, column=15)
```

### `get_references`

Trouve toutes les références à un symbole dans le code.

```
get_references(file_path="/path/to/file.go", line=10, column=15)
```

### `check_diagnostics`

Vérifie les diagnostics (erreurs, avertissements) dans un fichier.

```
check_diagnostics(file_path="/path/to/file.go")
```

### `get_go_version`

Obtient des informations sur la dernière version de Go.

```
get_go_version()
```

### `check_deprecated_features`

Vérifie les fonctionnalités obsolètes dans un fichier.

```
check_deprecated_features(file_path="/path/to/file.go")
```

### `get_best_practices`

Récupère les meilleures pratiques Go selon différents aspects.

```
get_best_practices(aspect="all")  # Peut être "all", "latest_version", "recommended_features", "deprecated_features", "code_style"
```

### `search_documentation`

Recherche dans la documentation Go.

```
search_documentation(query="context.WithTimeout")
```

### `format_code`

Formate un morceau de code Go.

```
format_code(code="func main() { fmt.Println(\"Hello\") }")
```

## Exemple d'utilisation avec Claude

Voici comment utiliser ce MCP avec Claude dans Cursor:

1. Ouvrez un projet Go dans Cursor
2. Interrogez Claude, par exemple:
   - "Quelles sont les nouvelles fonctionnalités dans Go 1.21?"
   - "Montre-moi la définition de cette fonction"
   - "Est-ce que ce code utilise des fonctionnalités obsolètes?"

Claude aura alors accès aux outils du MCP LSP Go pour fournir des réponses précises et adaptées à la version moderne de Go.

## Dépannage

Si vous rencontrez des problèmes:

1. Vérifiez que le chemin vers l'exécutable `mcplspgo` est correctement défini dans votre PATH
2. Assurez-vous que gopls est correctement installé et accessible
3. Vérifiez les logs de Cursor ou Claude Desktop pour les erreurs liées au MCP
4. Pour tester manuellement le MCP, lancez le depuis un terminal puis tapez des commandes JSON-RPC valides 