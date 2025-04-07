package protocol

// Position représente une position dans un document texte
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range représente une plage dans un document texte
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Location représente un emplacement dans un document texte
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentIdentifier identifie un document texte
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// TextDocumentPositionParams paramètres pour les requêtes basées sur la position
type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

// ReferenceContext contexte pour une requête de références
type ReferenceContext struct {
	IncludeDeclaration bool `json:"includeDeclaration"`
}

// ReferenceParams paramètres pour les requêtes de références
type ReferenceParams struct {
	TextDocumentPositionParams
	Context ReferenceContext `json:"context"`
}

// Diagnostic représente un diagnostic comme un problème de code
type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity,omitempty"` // 1=Error, 2=Warning, 3=Info, 4=Hint
	Code     string `json:"code,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

// DiagnosticSeverity énumère les niveaux de sévérité des diagnostics
type DiagnosticSeverity int

const (
	SeverityError   DiagnosticSeverity = 1
	SeverityWarning DiagnosticSeverity = 2
	SeverityInfo    DiagnosticSeverity = 3
	SeverityHint    DiagnosticSeverity = 4
)
