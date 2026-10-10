# edi ⚡

Un editor de texto en consola (TUI) moderno, rápido, altamente personalizable y diseñado en **Go**. Cero dependencias externas en tiempo de ejecución, empaquetado como un único binario estático ideal para desarrollo local y servidores remotos.

---

## ✨ Características

- 🚀 **Arranque instantáneo**: < 10 ms de tiempo de inicio.
- 📦 **Binario estático sin CGO**: Distribución sin fricción mediante `scp`, `curl` o `go install` en Linux, macOS y Windows.
- 🎨 **Paleta TrueColor (24-bit)**: Soporte completo de temas personalizados (`~/.edi-config/themes/*.json`) con Drácula y Monokai incluidos de fábrica.
- 🔍 **Resaltado de sintaxis declarativo**: Agrega soporte para cualquier lenguaje simplemente colocando un archivo `.json` en `~/.edi-config/syntax/`.
- ⌨️ **Atajos de teclado configurables**: Mapeo completo en `~/.edi-config/keybindings.json` (soporta formato JSONC con comentarios `//`).
- 🌳 **Explorador de Archivos en Panel Lateral (`Ctrl+B`)**: Árbol jerárquico interactivo con carpetas expandibles (`▸`/`▾`), navegación con flechas y apertura directa en pestañas.
- 📁 **Apertura de Carpetas CLI (`edi .`)**: Abre un proyecto completo directamente desde la terminal abriendo el explorador lateral.
- 🧭 **Paleta de Comandos Modal (`Ctrl+P`)**: Ventana flotante interactiva con navegación `↑`/`↓`, búsqueda difusa y cambio de temas en caliente.
- 📂 **Buscador Rápido de Archivos (`Ctrl+O`)**: Selector difuso instantáneo que descubre archivos en el proyecto ignorando `.git` o dependencias.
- 📑 **Pestañas y Múltiples Buffers (`Ctrl+N`, `Ctrl+W`, `Ctrl+Tab`)**: Barra superior visual con soporte para múltiples archivos simultáneos e indicador de estado `*`.
- ✂️ **Selección de Texto y Portapapeles (`Ctrl+C`, `Ctrl+X`, `Ctrl+V`)**: Selección multi-línea con `Shift + Flechas`, copiar, cortar y pegar.
- ↩️ **Historial Undo/Redo**: Deshacer (`Ctrl+Z`) y Rehacer (`Ctrl+Y`) multinivel.
- 🗂️ **Defaults Embebidos**: El binario incluye temas, sintaxis y atajos compilados internamente con `//go:embed`. Funciona de inmediato sin necesidad de crear archivos previos.

---

## 📂 Jerarquía de Configuración (`~/.edi-config/`)

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

## 🛠️ Atajos de Teclado Predeterminados

| Atajo | Acción |
| :--- | :--- |
| `Ctrl+P` | Abrir la Paleta de Comandos (Modal flotante) |
| `Ctrl+O` | Buscador rápido de archivos del proyecto |
| `Ctrl+B` | Alternar Explorador de archivos (Panel lateral) |
| `Ctrl+S` | Guardar archivo (ejecuta formateadores si existen) |
| `Ctrl+N` | Abrir nueva pestaña vacía |
| `Ctrl+W` | Cerrar pestaña activa (con advertencia si no está guardada) |
| `Ctrl+Tab` | Alternar entre pestañas abiertas |
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
| `Ctrl+Q` | Salir de edi (con confirmación de seguridad) |

---

## 🧩 Cómo Agregar un Nuevo Lenguaje (`syntax/`)

Para agregar resaltado a un lenguaje nuevo (por ejemplo, Rust), simplemente crea `~/.edi-config/syntax/rust.json`:

```json
{
  "name": "Rust",
  "extensions": [".rs"],
  "rules": [
    {
      "scope": "comment",
      "pattern": "//.*$"
    },
    {
      "scope": "string",
      "pattern": "\"(?:\\\\.|[^\"\\\\])*\""
    },
    {
      "scope": "keyword",
      "pattern": "\\b(fn|let|mut|pub|struct|enum|impl|trait|use|mod|match|if|else|loop|while|for|in|return)\\b"
    },
    {
      "scope": "type",
      "pattern": "\\b(i32|i64|u32|u64|usize|bool|char|str|String|Option|Result|Vec)\\b"
    },
    {
      "scope": "number",
      "pattern": "\\b\\d+(\\.\\d+)?\\b"
    }
  ]
}
```

¡Al abrir cualquier archivo `.rs`, `edi` lo resaltará de inmediato sin necesidad de recompilar!

---

## 🎨 Cómo Agregar un Nuevo Tema (`themes/`)

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

## 🚀 Compilación e Instalación

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
