# Guía de Contribución de edi

¡Gracias por tu interés en contribuir a **edi**! Este documento contiene las pautas técnicas, convenciones de desarrollo y flujo de trabajo para colaborar en el proyecto.

---

## 1. Requisitos Previos

- **Go**: Versión 1.22 o superior recomendada.
- **Git**: Para control de versiones.
- **Terminal con soporte TrueColor (24 bits)**: Alacritty, iTerm2, Kitty, Ghostty, WezTerm o terminal de VS Code.

---

## 2. Flujo de Trabajo y Comandos de Makefile

El proyecto cuenta con un `Makefile` estandarizado para simplificar todas las operaciones comunes:

```bash
# Compilar el binario local para tu arquitectura
make build

# Ejecutar toda la suite de pruebas unitarias e integración
make test

# Limpiar ejecutables y binarios generados
make clean

# Compilar para todas las plataformas objetivo (Linux, macOS, Windows en dist/)
make cross-compile
```

---

## 3. Principios de Código y Estilo

1. **Go Puro sin CGO**:
   - `edi` debe compilarse siempre con `CGO_ENABLED=0`. No se permiten dependencias que requieran compiladores de C/C++ ni librerías dinámicas del sistema.
2. **Estética y Caracteres de Terminal**:
   - **Cero emojis**: No utilices emojis en cadenas de interfaz, insignias ni registros. Los emojis provocan desalineaciones en rejillas monoespaciadas según el emulador de terminal.
   - Usa exclusivamente caracteres Unicode monoespaciados estándar o ASCII (ejemplos: `▸`, `▾`, `╭`, `╰`, `│`, `─`, `▶`, `•`, `M`, `?`, `+`, `D`).
3. **Manejo de Errores Idiomático**:
   - Nunca uses `panic()` en tiempo de ejecución de la interfaz. Los errores de apertura de archivos o ejecución de comandos deben reportarse con `e.SetStatus(errMsg, duration)` de forma amigable.
4. **Preservación de Comentarios y Documentación**:
   - Mantén los comentarios explicativos y la documentación de funciones públicas en español claro.

---

## 4. Cómo Escribir Tests en edi

### 4.1 Pruebas de Interfaz sin Terminal Real
Para probar componentes visuales y eventos del editor, utiliza la pantalla simulada de `tcell` (`SimulationScreen`):

```go
func TestMiNuevaCaracteristica(t *testing.T) {
    screen := tcell.NewSimulationScreen("UTF-8")
    if err := screen.Init(); err != nil {
        t.Fatalf("error inicializando pantalla de simulación: %v", err)
    }
    defer screen.Fini()

    buf := buffer.NewEmptyBuffer()
    cfg := config.DefaultConfig()
    ed, err := NewEditor(screen, buf, &cfg)
    if err != nil {
        t.Fatalf("error instanciando editor: %v", err)
    }

    // Ejecutar acción o enviar evento
    ed.executeAction("mi_accion")

    // Verificar estado interno
    if ed.ActiveBufferIdx != 0 {
        t.Errorf("estado inesperado")
    }
}
```

### 4.2 Aislamiento del Sistema de Archivos
Siempre utiliza `t.TempDir()` en lugar de crear archivos temporales en el directorio del repositorio o en `/tmp`:

```go
tmpDir := t.TempDir()
testFile := filepath.Join(tmpDir, "test.txt")
os.WriteFile(testFile, []byte("contenido"), 0644)
```

---

## 5. Proceso de Envío de Cambios (Pull Requests)

1. Crea una rama descriptiva para tu característica o corrección:
   ```bash
   git checkout -b feature/nombre-de-la-mejora
   ```
2. Asegúrate de que todos los tests pasen y no haya advertencias de compilación:
   ```bash
   go test -v ./...
   ```
3. Si agregaste un atajo de teclado o una configuración nueva, actualiza:
   - `defaults/help.md`
   - `defaults/keybindings.json` o `defaults/config.json`
   - `README.md` y `docs/user_guide.md`
4. Realiza commits claros siguiendo la convención de Conventional Commits (`feat: ...`, `fix: ...`, `docs: ...`, `refactor: ...`).
