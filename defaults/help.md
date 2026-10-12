# Guía y Documentación de edi

Bienvenido a **edi**, un editor de texto para consola (TUI) rápido, portable y extensible creado en Go.

---

## 1. Atajos de Teclado Esenciales

### Gestión de Archivos y Pestañas
- `Ctrl+O` : **Buscador Rápido de Archivos (Fuzzy Finder)** del proyecto.
- `Ctrl+B` : **Alternar Explorador de Archivos** en el panel lateral.
- `Ctrl+S` / `F2` : **Guardar archivo**. Si es un archivo nuevo sin nombre, abrirá una ventana modal interactiva directamente en la terminal para ingresar el nombre y guardarlo automáticamente. También ejecuta formateadores `on_save`. *(Nota para macOS: usa Control ⌃ + S o F2; si usas Cmd ⌘ + S tu terminal o sistema operativo interceptará la pulsación).*
- `Ctrl+N` : **Nueva pestaña** en blanco.
- `Ctrl+W` : **Cerrar pestaña** actual (con confirmación si hay cambios sin guardar).
- `Ctrl+Tab` / `Alt+Right` : **Pestaña siguiente**.
- `Alt+Left` : **Pestaña anterior**.
- `Alt+1` a `Alt+9` : **Ir directamente a la pestaña 1..9**.
- `Ctrl+Q` : **Salir de edi**.

### Paleta de Comandos y Búsqueda
- `Ctrl+P` : **Abrir la Paleta de Comandos** (modal flotante interactivo).
- `Ctrl+Shift+F` / `Alt+Shift+F` / `Alt+F` : **Buscar en Todos los Archivos (Grep)**. Ventana modal dividida en dos paneles (archivos e incidencias con número de línea y fragmento resaltado). Incluye alternadores de búsqueda: `[Aa]` Case sensitive (`Alt+C`), `[.*]` Expresiones regulares (`Alt+R`) y `[~]` Búsqueda difusa (`Alt+Z`). Navega con `Tab` y abre la coincidencia directamente con `Enter`.
- `Ctrl+F` : **Buscar texto** en el documento activo (búsqueda cíclica).
- `Ctrl+G` : **Ir a número de línea / offset** (`goto line`).
- `F1` : **Ver esta documentación**.

### Modo Binario y Visor Hexadecimal (Imágenes y Binarios)
Cualquier archivo binario o de imagen (`.png`, `.jpg`, `.gif`, `.wasm`, `.exe`, `.bin`, `.pdf`, etc.) o archivo con bytes nulos (`0x00`) se abre automáticamente en **modo Hexadecimal canónico**:
- **Estructura**: 16 bytes por línea divididos en bloques de 8 bytes con offset en hexadecimal (`00000000`), bytes hexadecimales y representación en caracteres imprimibles ASCII (`|...|`).
- **Resaltado y Color**: Los offsets, bytes y texto ASCII se colorean con la sintaxis Hex integrada según el tema activo.
- **Protección contra Corrupción**: El buffer se abre en **solo lectura** para evitar sobreescrituras accidentales en archivos binarios.
- **Alternar Vista**: Puedes alternar en cualquier momento entre modo hexadecimal y texto plano abriendo la paleta de comandos (`Ctrl+P` -> *Alternar vista Hexadecimal (Modo Binario)*).

### Edición y Portapapeles
- `Shift + Flechas` : **Seleccionar texto** (multi-línea).
- `Ctrl+C` : **Copiar** texto seleccionado (o línea completa si no hay selección).
- `Ctrl+X` : **Cortar** selección o línea.
- `Ctrl+V` : **Pegar** desde el portapapeles.
- `Ctrl+Z` : **Deshacer** (Undo multinivel).
- `Ctrl+Y` : **Rehacer** (Redo multinivel).
- `Ctrl+D` : **Duplicar** línea actual.
- `Ctrl+K` : **Eliminar** línea actual.
- `Alt+Up` / `Ctrl+Up` : **Mover línea** hacia arriba (o selección completa).
- `Alt+Down` / `Ctrl+Down` : **Mover línea** hacia abajo (o selección completa).
- `Tab` : Insertar tabulación o espacios según configuración.

### Navegación en el Explorador Lateral (`Ctrl+B`)
- `↑` / `↓` : Moverse por archivos y carpetas.
- `Enter` : Expandir/contraer carpeta o abrir archivo en una pestaña.
- `→` / `←` : Expandir o contraer carpeta.
- `a` / `n` : Crear un nuevo archivo dentro de la carpeta seleccionada.
- `Tab` o `Esc` : Devolver el foco al editor de texto.

### Integración con Git (Automática y Zero-Config)
Si `edi` se ejecuta dentro de un repositorio Git, se activa automáticamente sin necesidad de configuración:
- **Barra de estado**: Muestra la rama activa y el resumen de cambios (`git: main [~1 +2]`).
- **Insignias en el explorador (`Ctrl+B`)**:
  - `M` : Archivo modificado
  - `?` : Archivo nuevo sin seguimiento (untracked)
  - `+` : Archivo en área de preparación (staged)
  - `D` : Archivo eliminado
  - `•` : Carpeta con archivos modificados en su interior
- **Comandos Git en la Paleta (`Ctrl+P`)**:
  - `Git: Ver Diff del archivo actual` : Abre una pestaña virtual con el diff coloreado frente a HEAD.
  - `Git: Ver Diff de todo el proyecto` : Abre un diff unificado de todo el repositorio.
  - `Git: Archivos modificados (Status)` : Abre un filtro rápido para saltar directamente a los archivos cambiados.
  - `Git: Refrescar estado` : Fuerza una actualización inmediata de cambios.

---

## 2. Personalización (`~/.edi-config/`)

Puedes inicializar la carpeta de configuración en tu home ejecutando:
```bash
edi --init
```
Esto creará la siguiente estructura en `~/.edi-config/`:

```
~/.edi-config/
├── config.json         # Preferencias generales (JSON con comentarios)
├── keybindings.json    # Mapeo de atajos de teclado
├── themes/             # Carpeta de temas drop-in (.json)
├── syntax/             # Definiciones de lenguaje drop-in (.json)
└── plugins/            # Complementos declarativos (.json)
```

### 2.1 Configuración General (`config.json`)
```json
{
  "theme": "dracula",
  "tab_size": 4,
  "expand_tabs": true,
  "line_numbers": true,
  "highlight_current_line": true,
  "status_bar": true,
  "auto_indent": true,
  "git_integration": true,
  "sidebar_position": "left" // "left" o "right"
}
```

### 2.2 Agregar Nuevos Lenguajes (`syntax/*.json`)
Basta con crear un archivo en `~/.edi-config/syntax/mi_lenguaje.json`:
```json
{
  "name": "Rust",
  "extensions": [".rs"],
  "rules": [
    { "scope": "comment", "pattern": "//.*$" },
    { "scope": "string", "pattern": "\"(?:\\\\.|[^\"\\\\])*\"" },
    { "scope": "keyword", "pattern": "\\b(fn|let|mut|pub|struct|enum|impl|match|return)\\b" },
    { "scope": "type", "pattern": "\\b(i32|i64|u32|String|bool|Option|Result)\\b" },
    { "scope": "number", "pattern": "\\b\\d+\\b" }
  ]
}
```

### 2.3 Agregar Temas (`themes/*.json`)
Crea `~/.edi-config/themes/mi_tema.json` con colores hexadecimales TrueColor:
```json
{
  "name": "mi_tema",
  "ui": {
    "background": "#1e1e2e",
    "foreground": "#cdd6f4",
    "cursor": "#f5e0dc",
    "cursor_line": "#313244",
    "line_number": "#585b70",
    "line_number_active": "#cdd6f4",
    "status_bar_bg": "#313244",
    "status_bar_fg": "#cdd6f4",
    "selection": "#45475a"
  },
  "syntax": {
    "comment": "#6c7086",
    "keyword": "#cba6f7",
    "string": "#a6e3a1",
    "type": "#89b4fa",
    "number": "#fab387",
    "function": "#89dceb"
  }
}
```

### 2.4 Complementos Declarativos (`plugins/*.json`)
Puedes conectar formateadores automáticos (`gofmt`, `prettier`, `black`, etc.):
```json
{
  "name": "Go Helper",
  "languages": ["go"],
  "hooks": {
    "on_save": [
      {
        "action": "pipe_buffer",
        "command": "gofmt",
        "description": "Formatear automáticamente al guardar"
      }
    ]
  }
}
```

---

### 2.5 Lenguajes y Extensiones de Serie (Built-in)
`edi` incluye de fábrica resaltado de sintaxis optimizado para los lenguajes más populares:
- **Sistemas y Backend**: Go, Rust, C, C++, Python, SQL
- **Web y Frontend**: JavaScript, TypeScript, HTML, CSS
- **Configuración y DevOps**: Dockerfile, Makefile, YAML, TOML, JSON, Shell/Bash, XML, Diff, Markdown

### 2.6 Gestor de Extensiones (`edi ext`)
Puedes inspeccionar o instalar extensiones y plugins desde la terminal:
```bash
# Listar todas las extensiones activas
edi ext

# Instalar una sintaxis o plugin desde una URL o archivo local
edi ext install https://url/a/mi-sintaxis.json
edi ext install ./mi_plugin.json
```

---

## 3. Uso en Línea de Comandos

```bash
# Abrir un archivo específico
edi main.go

# Abrir una carpeta completa en el explorador
edi .
edi /ruta/a/mi-proyecto

# Gestión de extensiones y plugins
edi ext
edi ext install <url-o-archivo.json>

# Ayuda, versión y actualización
edi version
edi --version
edi --update
edi --config-dir
edi --help
```

---
*Presiona `Ctrl+W` para cerrar esta pestaña de ayuda y volver a tu archivo.*
