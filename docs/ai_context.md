# Contexto e Instrucciones para Inteligencias Artificiales (AI Context)

Este documento está diseñado como referencia técnica y guía de comportamiento para **asistentes de Inteligencia Artificial y agentes autónomos de código** (como Antigravity, Claude, Gemini, Cursor, Copilot o ChatGPT) que colaboren en el repositorio de **edi**.

---

## 1. Misión y Filosofía del Proyecto

**edi** es un editor de texto en terminal (TUI) moderno, hiper-ligero y extensible escrito en **Go puro**.

### Invariantes No Negociables
1. **Cero CGO (`CGO_ENABLED=0`)**: El proyecto debe compilar siempre como un único binario estático transportable sin bibliotecas compartidas de C/C++.
2. **Arranque Sub-10ms**: Nunca introduzcas dependencias ni análisis síncronos pesados en la inicialización que degraden el tiempo de arranque.
3. **Cero Emojis en la TUI**: Las interfaces de terminal sufren graves problemas de desalineación de columnas cuando se emplean emojis (debido a diferencias en anchos de glifos en diferentes emuladores). Usa únicamente caracteres ASCII o runas Unicode estándar con ancho 1 (`▸`, `▾`, `╭`, `╰`, `│`, `─`, `▶`, `•`, `M`, `?`, `+`, `D`).
4. **UI No Bloqueante (60 FPS)**: Cualquier operación de entrada/salida externa (Git, escaneo de disco, formateadores o plugins) DEBE ejecutarse en goroutines asíncronas en segundo plano. Nunca bloquees el hilo del bucle de eventos (`HandleEvent` / `Draw`).
5. **Autocontenido (*Embedded by Default*)**: Todo recurso indispensable (temas base, definiciones de sintaxis, atajos y ayuda) debe compilarse dentro del ejecutable usando `defaults/defaults.go` con `//go:embed`.

---

## 2. Mapa Arquitectónico de Paquetes

| Paquete | Responsabilidad Principal | Reglas de Modificación |
| :--- | :--- | :--- |
| `cmd/edi` | Entrada de CLI, banderas, setup inicial de pantalla. | Mantener mínimo y limpio. No meter lógica de edición aquí. |
| `internal/buffer` | Representación en memoria del texto, cursor, clamp, undo/redo, mover líneas. | Debe ser completamente agnóstico de `tcell` y de la interfaz visual. |
| `internal/editor` | Renderizado TUI, modales flotantes, captura de teclas y orquestación. | El estado de modales se maneja con `PromptMode`. Los eventos despiertan con `tcell.NewEventInterrupt`. |
| `internal/git` | Detección de repositorios, cálculo de estado, diffs y ramas. | Concurrencia protegida con `sync.RWMutex`. Toda consulta pesada debe ser asíncrona (`RefreshAsync`). |
| `internal/tree` | Árbol jerárquico de archivos del panel lateral (`Ctrl+B`). | Maneja profundidad (`Depth`), colapso/expansión y creación de archivos en caliente. |
| `internal/syntax` | Coloreado sintáctico declarativo basado en Regex. | Definido en JSON. Los regex se compilan una vez al cargar. |
| `internal/theme` | Parseo de paletas TrueColor (24 bits) en formato JSON. | Provee mapeo de colores HEX a estilos `tcell.Style`. |
| `internal/config` | Carga jerárquica de configuración y atajos de teclado. | Soporta JSONC eliminando comentarios mediante `StripJSONComments`. |
| `internal/plugin` | Hooks declarativos (`on_save`) y acciones en tubería (`pipe_buffer`). | Ejecuta comandos externos pasando el contenido del búfer por stdin/stdout. |
| `defaults` | Activos estáticos embebidos en el binario (`//go:embed`). | Modificar aquí temas, sintaxis o atajos base cuando se agreguen de fábrica. |

---

## 3. Máquina de Estados y Modales del Editor

El editor controla las ventanas emergentes a través del campo `e.PromptMode`:

```go
const (
    PromptNone       // Edición normal en el buffer activo
    PromptFind       // Búsqueda de texto dentro del archivo (Ctrl+F)
    PromptGoto       // Ir a línea específica (Ctrl+G)
    PromptPalette    // Paleta de comandos modal (Ctrl+P)
    PromptFileFinder // Buscador rápido de archivos / Fuzzy Finder (Ctrl+O)
    PromptSaveAs     // Cuadro modal para ingresar nombre al guardar archivo nuevo
)
```

### Reglas de Manejo de Teclas en Modales
- Cuando `PromptMode != PromptNone`, el método `HandleEvent` desvía las pulsaciones a `handlePromptKey`.
- `Esc`: Siempre debe cancelar el modal actual y regresar a `PromptNone`.
- `Enter`: Confirma la acción mediante `submitPrompt()`.
- `↑` / `↓`: Navega los elementos filtrados dentro del modal.

---

## 4. Buffers Virtuales (Diffs, Ayuda y Vistas de Lectura)

Cuando se requiera mostrar información generada dinámicamente (como la salida de `git diff` o la ayuda de `help.md`):
- Usa `e.openVirtualBuffer(nombre, contenido)`.
- El nombre debe tener la extensión adecuada (por ejemplo `archivo.diff` para diffs o `help.md` para markdown). Esto permite que el motor de sintaxis detecte automáticamente el resaltado sin código especial.
- Un buffer virtual no escribe en disco hasta que el usuario decida explícitamente ejecutar `Guardar` (`Ctrl+S`).

---

## 5. Patrón Asíncrono y Redibujado Reactivo

Para tareas asíncronas en segundo plano (como Git):
```go
go func() {
    // 1. Ejecutar operación pesada en goroutine
    resultado := operacionPesada()

    // 2. Actualizar estado bajo mutex si aplica
    // 3. Despertar el bucle de la terminal enviando un evento de interrupción
    e.Screen.PostEvent(tcell.NewEventInterrupt(nil))
}()
```
En `editor.go`, `HandleEvent` captura `*tcell.EventInterrupt` e inmediatamente desencadena un redibujado (`Draw()`) en el siguiente ciclo, logrando actualización instantánea sin consumo de CPU.

---

## 6. Pautas para Escribir Tests

1. **Pantalla Simulada**: Usa siempre `tcell.NewSimulationScreen("UTF-8")`.
2. **Aislamiento**: Crea entornos temporales con `t.TempDir()`.
3. **Determinismo**: No confíes en tiempos de espera fijos (`time.Sleep`) a menos que sea para verificar expiraciones de mensajes de estado. Prefiere llamadas sincrónicas en los tests (e.g. `GitManager.RefreshSync`).
4. **Verificación de Regresiones**: Siempre ejecuta `go test -v ./...` antes de proponer cambios para asegurar que todos los paquetes compilen y pasen al 100%.

---

## 7. Errores Comunes y Cosas a Evitar (Gotchas)

- **Intercepción en macOS**: `Cmd+S` y `Cmd+O` son interceptados por el emulador de terminal del sistema operativo. Por ello, `edi` soporta `F2` y combinaciones `Ctrl` estándar.
- **Rutas Relativas vs Absolutas**: Al comparar si un archivo ya está abierto en una pestaña, normaliza siempre con `filepath.Clean()` y verifica equivalencia absoluta con `filepath.Abs()` para evitar pestañas duplicadas.
- **Normalización de Saltos de Línea**: El buffer de `edi` maneja líneas como `[]string` sin el carácter `\n` al final de cada elemento. Al serializar a disco o a plugins, únelas con `strings.Join(lines, "\n")`.
