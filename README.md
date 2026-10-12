# edi

Un editor de texto en consola (TUI) moderno, rápido, altamente personalizable y diseñado en **Go**. Cero dependencias externas en tiempo de ejecución, empaquetado como un único binario estático ideal para desarrollo local y servidores remotos.

---

## Características

- **Arranque instantáneo**: < 10 ms de tiempo de inicio.
- **Binario estático sin CGO**: Distribución sin fricción mediante `scp`, `curl` o `go install` en Linux, macOS y Windows.
- **Paleta TrueColor (24-bit)**: Soporte completo de temas personalizados (`~/.edi-config/themes/*.json`) con Drácula y Monokai incluidos de fábrica.
- **Resaltado de sintaxis declarativo**: Agrega soporte para cualquier lenguaje simplemente colocando un archivo `.json` en `~/.edi-config/syntax/`.
- **Atajos de teclado configurables**: Mapeo completo en `~/.edi-config/keybindings.json` (soporta formato JSONC con comentarios `//`).
- **Explorador de Archivos en Panel Lateral (`Ctrl+B`)**: Árbol jerárquico interactivo con carpetas expandibles (`▸`/`▾`), navegación con flechas y apertura directa en pestañas.
- **Apertura de Carpetas CLI (`edi .`)**: Abre un proyecto completo directamente desde la terminal abriendo el explorador lateral.
- **Paleta de Comandos Modal (`Ctrl+P`)**: Ventana flotante interactiva con navegación `↑`/`↓`, búsqueda difusa y cambio de temas en caliente.
- **Buscador Rápido de Archivos (`Ctrl+O`)**: Selector difuso instantáneo que descubre archivos en el proyecto ignorando `.git` o dependencias.
- **Búsqueda Global en Archivos (`Ctrl+Shift+F` / `Alt+Shift+F` / `Alt+F`)**: Modal de búsqueda tipo Grep en 2 paneles (archivos e incidencias con número de línea y fragmento coloreado), alternador de opciones: `[Aa]` (Case Sensitive), `[.*]` (Expresiones Regulares), `[~]` (Búsqueda Difusa), navegación con `Tab` y apertura instantánea.
- **Modo Binario y Visor Hexadecimal Automático**: Detección inteligente de cualquier archivo binario o imagen (`.png`, `.jpg`, `.gif`, `.wasm`, `.exe`, etc.). Se abre en volcado hexadecimal canónico (16 bytes/línea con offset y ASCII) con protección contra corrupción (solo lectura) y sintaxis Hex coloreada.
- **Pestañas y Múltiples Buffers (`Ctrl+N`, `Ctrl+W`, `Ctrl+Tab`, `Alt+1..9`)**: Barra superior visual con soporte para múltiples archivos simultáneos e indicador de estado `*`.
- **Selección de Texto y Portapapeles (`Ctrl+C`, `Ctrl+X`, `Ctrl+V`)**: Selección multi-línea con `Shift + Flechas`, copiar, cortar y pegar.
- **Historial Undo/Redo**: Deshacer (`Ctrl+Z`) y Rehacer (`Ctrl+Y`) multinivel.
- **Integración Automática con Git**: Detección sin configuración. Muestra la rama y resumen de cambios en la barra de estado (`git: main [~1 +2]`), insignias en el explorador de archivos (`M`, `?`, `+`, `D`, `•`), y diffs coloreados en pestañas virtuales (`Ctrl+P` -> Git Diff). Asíncrono y no bloqueante.
- **Defaults Embebidos**: El binario incluye temas, sintaxis y atajos compilados internamente con `//go:embed`. Funciona de inmediato sin necesidad de crear archivos previos.

---

## Jerarquía de Configuración (`~/.edi-config/`)

El editor busca su configuración en el siguiente orden de prioridad:

1. **Local del proyecto**: `./.edi-config/` (para configuraciones específicas de un repositorio).
2. **Raíz del usuario**: `~/.edi-config/` (`$HOME/.edi-config` en Linux/macOS, `%USERPROFILE%\.edi-config` en Windows).
3. **Estándar XDG**: `~/.config/edi/`.
4. **Recursos Embebidos**: Incorporados dentro del binario.

### Estructura de `~/.edi-config/`
```
~/.edi-config/
├── config.json         # Configuración general (tabs, wrap, números de línea)
├── keybindings.json    # Atajos de teclado personalizados
├── themes/             # Carpeta de temas (drop-in)
│   ├── dracula.json
│   └── monokai.json
├── syntax/             # Definiciones de lenguajes (drop-in)
│   ├── go.json
│   ├── python.json
│   ├── json.json
│   └── markdown.json
└── plugins/            # Complementos declarativos JSON
    └── go-tools.json
```

Para inicializar automáticamente esta estructura en tu máquina:
```bash
./edi --init
```

---

## Atajos de Teclado Predeterminados

| Atajo | Acción |
| :--- | :--- |
| `Ctrl+P` | Abrir la Paleta de Comandos (Modal flotante) |
| `Ctrl+O` | Buscador rápido de archivos del proyecto |
| `Ctrl+Shift+F` / `Alt+F` | Búsqueda global en archivos del proyecto (Grep en 2 paneles) |
| `Ctrl+B` | Alternar Explorador de archivos (Panel lateral) |
| `Ctrl+S` / `F2` | Guardar archivo (pregunta nombre en terminal si es nuevo) |
| `Ctrl+N` | Abrir nueva pestaña vacía (tecla 'a'/'n' en explorador) |
| `Ctrl+W` | Cerrar pestaña activa (con advertencia si no está guardada) |
| `Ctrl+Tab` / `Alt+Right` | Pestaña siguiente |
| `Alt+Left` | Pestaña anterior |
| `Alt+1` a `Alt+9` | Ir a pestaña 1..9 |
| `Shift + Flechas` | Seleccionar texto (multi-línea) |
| `Ctrl+C` | Copiar selección (o línea actual si no hay selección) |
| `Ctrl+X` | Cortar selección (o línea actual) |
| `Ctrl+V` | Pegar desde el portapapeles |
| `Ctrl+F` | Buscar texto en el archivo activo |
| `Ctrl+G` | Ir a un número de línea |
| `Ctrl+Z` | Deshacer (Undo) |
| `Ctrl+Y` | Rehacer (Redo) |
| `Ctrl+D` | Duplicar línea actual |
| `Ctrl+K` | Eliminar línea actual |
| `F1` | Ayuda y documentación integrada |
| `Ctrl+Q` | Salir de edi (con confirmación de seguridad) |

---

## Lenguajes Soportados y Extensiones

`edi` incluye de fábrica resaltado de sintaxis optimizado para **más de 18 lenguajes y formatos**:
- **Sistemas & Backend**: Go, Rust, C, C++, Python, SQL
- **Web & Frontend**: JavaScript, TypeScript, JSX/TSX, HTML, CSS, SCSS
- **Configuración & Infraestructura**: Dockerfile, Makefile, YAML, TOML, JSON, Shell/Bash, XML, Markdown, Git Diff

### Plugins y Formateadores Automáticos
- **Go**: `gofmt` al guardar, comando `go vet`
- **Rust**: `rustfmt` al guardar, comando `cargo check` y `cargo test`
- **Web**: `prettier` al guardar en JS, TS, HTML, CSS, JSON, YAML y Markdown
- **Python**: `ruff format` o `black` al guardar, comando `pytest`
- **Shell**: `shfmt` al guardar, `shellcheck`

### Gestor de Extensiones en Terminal (`edi ext`)
```bash
# Ver todas las extensiones y lenguajes activos
edi ext

# Instalar una sintaxis o plugin desde una URL o archivo local
edi ext install https://raw.githubusercontent.com/.../mi_lenguaje.json
edi ext install ./mi_lenguaje.json
```

---

## Cómo Crear un Nuevo Lenguaje Personalizado (`syntax/`)

Para agregar resaltado a un lenguaje propietario o no incluido, crea `~/.edi-config/syntax/mi_lenguaje.json`:

```json
{
  "name": "MiLenguaje",
  "extensions": [".ext"],
  "rules": [
    { "scope": "comment", "pattern": "//.*$" },
    { "scope": "string", "pattern": "\"(?:\\\\.|[^\"\\\\])*\"" },
    { "scope": "keyword", "pattern": "\\b(fn|let|return|if|else)\\b" },
    { "scope": "type", "pattern": "\\b(Int|String|Bool)\\b" },
    { "scope": "number", "pattern": "\\b\\d+\\b" }
  ]
}
```

¡Al abrir cualquier archivo `.ext`, `edi` lo resaltará de inmediato sin necesidad de recompilar!

---

## Cómo Agregar un Nuevo Tema (`themes/`)

Crea `~/.edi-config/themes/nord.json`:

```json
{
  "name": "nord",
  "ui": {
    "background": "#2e3440",
    "foreground": "#d8dee9",
    "cursor": "#eceff4",
    "cursor_line": "#3b4252",
    "line_number": "#4c566a",
    "line_number_active": "#88c0d0",
    "status_bar_bg": "#3b4252",
    "status_bar_fg": "#eceff4",
    "selection": "#434c5e"
  },
  "syntax": {
    "comment": "#4c566a",
    "keyword": "#81a1c1",
    "string": "#a3be8c",
    "type": "#8fbcbb",
    "number": "#b48ead",
    "function": "#88c0d0"
  }
}
```

---

## Instalación Rápida (Linux y macOS)

Instala o actualiza `edi` con un solo comando:

```bash
curl -fsSL https://raw.githubusercontent.com/brpereyra/edi/main/install.sh | bash
```

> El instalador detecta automáticamente tu sistema (Linux/macOS) y arquitectura (`amd64` o `arm64`), instala el binario en `~/.local/bin` o `/usr/local/bin` y genera la configuración inicial en `~/.edi-config/`.

### Instalación vía `go install` (cualquier plataforma):
```bash
go install github.com/brpereyra/edi/cmd/edi@latest
```

---

## Verificación de Versión y Actualizaciones

### Consultar la versión instalada:
```bash
edi version
# o también:
edi --version
edi -v
```

### Actualizar `edi` sin tocar tus configuraciones:
```bash
edi update
# o también:
edi --update
```
> **Preservación total de configuración**: El comando de actualización reemplaza únicamente el binario ejecutable. Todas tus configuraciones, atajos y temas personalizados en `~/.edi-config/` se conservan **100% intactos**.

---

## Compilación Local desde el Código Fuente

### Compilar binario local:
```bash
make build
```

### Ejecutar tests:
```bash
make test
```

### Compilar para todas las plataformas (Linux x86/ARM, macOS Intel/Apple Silicon, Windows):
```bash
make cross-compile
```
Los ejecutables se guardarán en `dist/`.

### Uso básico:
```bash
./edi main.go
```

---

## Documentación Detallada

Para consultar guías técnicas y documentación adicional, visita la carpeta [`docs/`](docs/):

- 📖 **[Guía Completa de Usuario](docs/user_guide.md)**: Atajos, buscador difuso, explorador de archivos y personalización.
- 🏗️ **[Arquitectura Técnica](docs/architecture.md)**: Diseño de módulos, concurrencia de Git y ciclo de vida de la TUI.
- 🗺️ **[Hoja de Ruta (Roadmap)](docs/roadmap.md)**: Hitos completados y planificación de nuevas características.
- 🤝 **[Guía de Contribución](docs/contributing.md)**: Pautas de desarrollo, tests de simulación y estándares.
- 🤖 **[Instrucciones para IA](docs/ai_context.md)**: Especificación técnica y reglas para asistentes de Inteligencia Artificial.
