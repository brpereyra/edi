package syntax

import (
	"testing"
)

func TestDetectSyntax(t *testing.T) {
	syntaxes := LoadAllSyntaxes()
	if len(syntaxes) == 0 {
		t.Fatalf("se esperaban sintaxis cargadas, se obtuvieron 0")
	}

	goSyn := DetectSyntax("main.go", syntaxes)
	if goSyn == nil || goSyn.Name != "Go" {
		t.Fatalf("se esperaba sintaxis Go para main.go")
	}

	pySyn := DetectSyntax("script.py", syntaxes)
	if pySyn == nil || pySyn.Name != "Python" {
		t.Fatalf("se esperaba sintaxis Python para script.py")
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
