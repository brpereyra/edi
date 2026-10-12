package version

import (
	"fmt"
	"runtime"
)

// Version almacena la versión semántica actual de edi.
// Puede ser sobreescrita durante la compilación con:
// -ldflags="-X edi/internal/version.Version=vX.Y.Z"
var Version = "v0.2.0"

// FullVersion retorna la cadena informativa de versión con SO y arquitectura.
func FullVersion() string {
	return fmt.Sprintf("edi %s (%s/%s)", Version, runtime.GOOS, runtime.GOARCH)
}
