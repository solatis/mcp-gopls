package mcp

// BestPractices contient des informations sur les bonnes pratiques Go modernes
var BestPractices = map[string]any{
	"latest_go_version": "1.22.1",
	"recommended_features": []map[string]any{
		{
			"name":        "Generics",
			"since":       "1.18",
			"description": "Permet d'écrire du code polymorphe avec des types paramétrés",
			"example":     "func Min[T constraints.Ordered](x, y T) T { if x < y { return x }; return y }",
			"docs_url":    "https://go.dev/doc/tutorial/generics",
		},
		{
			"name":        "Workspaces",
			"since":       "1.18",
			"description": "Permet de travailler avec plusieurs modules dans un même espace de travail",
			"example":     "go.work file: use ./moduleA\nuse ./moduleB",
			"docs_url":    "https://go.dev/doc/tutorial/workspaces",
		},
		{
			"name":        "Error Wrapping",
			"since":       "1.13",
			"description": "Permet d'encapsuler des erreurs avec du contexte supplémentaire",
			"example":     "if err != nil { return fmt.Errorf(\"failed to read file: %w\", err) }",
			"docs_url":    "https://go.dev/blog/go1.13-errors",
		},
		{
			"name":        "Structured Logging",
			"since":       "1.21",
			"description": "Bibliothèque standard pour la journalisation structurée",
			"example":     "logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))\nlogger.Info(\"user logged in\", \"user_id\", user.ID)",
			"docs_url":    "https://pkg.go.dev/log/slog",
		},
		{
			"name":        "Fuzzing",
			"since":       "1.18",
			"description": "Test par génération aléatoire d'entrées pour trouver des bugs",
			"example":     "func FuzzReverse(f *testing.F) { f.Add(\"hello\"); f.Fuzz(func(t *testing.T, s string) { Reverse(s) }) }",
			"docs_url":    "https://go.dev/doc/tutorial/fuzz",
		},
		{
			"name":        "Embed Files",
			"since":       "1.16",
			"description": "Incorporer des fichiers statiques dans le binaire",
			"example":     "//go:embed templates/*.html\nvar templates embed.FS",
			"docs_url":    "https://pkg.go.dev/embed",
		},
		{
			"name":        "Go Modules",
			"since":       "1.11",
			"description": "Système de gestion de dépendances officiel",
			"example":     "go mod init example.com/mymodule",
			"docs_url":    "https://go.dev/doc/modules/gomod-ref",
		},
	},
	"deprecated_features": []map[string]any{
		{
			"name":          "iota dans une const sans valeur explicite",
			"deprecated_in": "N/A",
			"reason":        "Moins lisible et source d'erreurs",
			"replacement":   "Toujours spécifier const MyConst = iota pour clarté",
			"example_old":   "const ( MyConst iota; OtherConst )",
			"example_new":   "const ( MyConst = iota; OtherConst )",
		},
		{
			"name":          "var u *User = &User{}",
			"deprecated_in": "N/A",
			"reason":        "Verbeux et redondant",
			"replacement":   "Utiliser l'inférence de type: u := &User{}",
			"example_old":   "var u *User = &User{Name: \"John\"}",
			"example_new":   "u := &User{Name: \"John\"}",
		},
		{
			"name":          "panic/recover pour gestion d'erreurs",
			"deprecated_in": "N/A",
			"reason":        "Contre les bonnes pratiques Go (explicite > implicite)",
			"replacement":   "Retourner et vérifier les erreurs explicitement",
			"example_old":   "func DoThing() { defer func() { recover() }(); panic(\"error\") }",
			"example_new":   "func DoThing() error { return errors.New(\"error\") }",
		},
		{
			"name":          "time.Sleep pour synchronisation",
			"deprecated_in": "N/A",
			"reason":        "Code fragile et non-déterministe",
			"replacement":   "Utiliser les primitives de sync ou les channels",
			"example_old":   "go doWork(); time.Sleep(100 * time.Millisecond)",
			"example_new":   "var wg sync.WaitGroup; wg.Add(1); go func() { defer wg.Done(); doWork() }(); wg.Wait()",
		},
		{
			"name":          "gofmt (outil séparé)",
			"deprecated_in": "N/A",
			"reason":        "Remplacé par un outil plus complet",
			"replacement":   "goimports (formatage + gestion des imports)",
			"example_old":   "gofmt -w file.go",
			"example_new":   "goimports -w file.go",
		},
		{
			"name":          "erreurs non typées",
			"deprecated_in": "1.13",
			"reason":        "Difficulté à encapsuler/examiner les erreurs",
			"replacement":   "errors.New(), fmt.Errorf() avec %w, ou erreurs personnalisées",
			"example_old":   "return errors.New(\"failed to connect\")",
			"example_new":   "var ErrConnection = errors.New(\"failed to connect\")\nreturn fmt.Errorf(\"db error: %w\", ErrConnection)",
		},
	},
	"code_style": []map[string]any{
		{
			"name":        "Gestion d'erreurs",
			"description": "Vérifier et gérer les erreurs immédiatement",
			"example":     "if err != nil {\n    return fmt.Errorf(\"context: %w\", err)\n}",
		},
		{
			"name":        "Receivers nommés",
			"description": "Utiliser des noms descriptifs (1-2 lettres) pour les receivers de méthodes",
			"example":     "func (u *User) FullName() string { return u.FirstName + \" \" + u.LastName }",
		},
		{
			"name":        "Interfaces petites",
			"description": "Préférer de nombreuses petites interfaces aux grandes interfaces monolithiques",
			"example":     "type Reader interface { Read(p []byte) (n int, err error) }",
		},
		{
			"name":        "Commentaires de doc",
			"description": "Commencer les commentaires par le nom de l'élément documenté",
			"example":     "// User represents a system user with authentication information.",
		},
		{
			"name":        "Noms des variables d'erreur",
			"description": "Utiliser 'err' comme nom de variable pour les erreurs",
			"example":     "if err := doThing(); err != nil { return err }",
		},
		{
			"name":        "Valeurs retournées nommées",
			"description": "Utiliser des valeurs retournées nommées pour améliorer la lisibilité dans les fonctions complexes",
			"example":     "func divide(a, b int) (result int, err error) { if b == 0 { err = errors.New(\"division by zero\") } else { result = a / b }; return }",
		},
	},
}

// GetBestPractices retourne les meilleures pratiques Go pour un aspect spécifique
func GetBestPractices(aspect string) (any, error) {
	switch aspect {
	case "all":
		return BestPractices, nil
	case "latest_version":
		return BestPractices["latest_go_version"], nil
	case "recommended_features":
		return BestPractices["recommended_features"], nil
	case "deprecated_features":
		return BestPractices["deprecated_features"], nil
	case "code_style":
		return BestPractices["code_style"], nil
	default:
		return nil, ErrUnsupportedFeature
	}
}
