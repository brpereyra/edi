package version

import (
	"strings"
	"testing"
)

func TestFullVersion(t *testing.T) {
	v := FullVersion()
	if !strings.HasPrefix(v, "edi ") {
		t.Errorf("FullVersion debe comenzar con 'edi ', obtenido %s", v)
	}
	if !strings.Contains(v, Version) {
		t.Errorf("FullVersion debe contener %s, obtenido %s", Version, v)
	}
}
