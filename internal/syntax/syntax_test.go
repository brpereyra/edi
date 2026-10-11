package syntax

import (
	"testing"
)

func TestDetectSyntax(t *testing.T) {
	syntaxes := LoadAllSyntaxes()
	if len(syntaxes) == 0 {
		t.Fatalf("se esperaban sintaxis cargadas, se obtuvieron 0")
	}

	testCases := []struct {
		file     string
		expected string
	}{
		{"main.go", "Go"},
		{"script.py", "Python"},
		{"main.rs", "Rust"},
		{"app.js", "JavaScript"},
		{"index.ts", "TypeScript"},
		{"component.tsx", "TypeScript"},
		{"main.c", "C"},
		{"engine.cpp", "C++"},
		{"setup.sh", "Shell"},
		{".bashrc", "Shell"},
		{"index.html", "HTML"},
		{"style.css", "CSS"},
		{"config.yaml", "YAML"},
		{"deploy.yml", "YAML"},
		{"Cargo.toml", "TOML"},
		{"schema.sql", "SQL"},
		{"Dockerfile", "Dockerfile"},
		{"Makefile", "Makefile"},
		{"data.xml", "XML"},
		{"patch.diff", "Diff"},
		{"package.json", "JSON"},
	}

	for _, tc := range testCases {
		syn := DetectSyntax(tc.file, syntaxes)
		if syn == nil {
			t.Errorf("no se detectó sintaxis para %s (se esperaba %s)", tc.file, tc.expected)
		} else if syn.Name != tc.expected {
			t.Errorf("para %s se detectó %s, se esperaba %s", tc.file, syn.Name, tc.expected)
		}
	}
}

func TestHighlightLine(t *testing.T) {
	syntaxes := LoadAllSyntaxes()
	goSyn := DetectSyntax("main.go", syntaxes)

	line := `func main() { // package comment`
	spans := HighlightLine(goSyn, line)

	if len(spans) == 0 {
		t.Fatalf("se esperaban spans resaltados para línea Go")
	}

	foundKeyword := false
	foundComment := false
	for _, span := range spans {
		if span.Scope == "keyword" {
			foundKeyword = true
		}
		if span.Scope == "comment" {
			foundComment = true
		}
	}

	if !foundKeyword {
		t.Errorf("no se encontró keyword 'func'")
	}
	if !foundComment {
		t.Errorf("no se encontró comment '// package comment'")
	}
}
