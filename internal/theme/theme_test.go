package theme

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestParseHexColor(t *testing.T) {
	c := ParseHexColor("#282a36")
	r, g, b := c.RGB()
	if r != 0x28 || g != 0x2a || b != 0x36 {
		t.Errorf("color incorrecto: esperado (0x28, 0x2a, 0x36), obtenido (%x, %x, %x)", r, g, b)
	}

	short := ParseHexColor("#fff")
	r, g, b = short.RGB()
	if r != 0xff || g != 0xff || b != 0xff {
		t.Errorf("color corto incorrecto: obtenido (%x, %x, %x)", r, g, b)
	}
}

func TestLoadTheme(t *testing.T) {
	th, err := LoadTheme("dracula")
	if err != nil {
		t.Fatalf("error cargando dracula: %v", err)
	}

	if th.Name != "dracula" {
		t.Errorf("esperado nombre dracula, obtenido %s", th.Name)
	}

	keywordStyle := th.GetSyntaxStyle("keyword")
	if keywordStyle == tcell.StyleDefault {
		t.Errorf("se esperaba estilo definido para keyword")
	}
}
