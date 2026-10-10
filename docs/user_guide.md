# Guía de Usuario de edi

Manual completo y detallado para dominar el editor de texto en consola **edi**.

---

## 1. Instalación y Primeros Pasos

### 1.1 Ejecución Básica
`edi` es un único binario ejecutable sin dependencias externas:

```bash
# Abrir un archivo específico
edi main.go

# Abrir la carpeta actual como proyecto en el explorador lateral
edi .

# Abrir una carpeta específica
edi /ruta/a/mi-proyecto

# Crear un archivo nuevo
edi nuevo_archivo.txt
```

### 1.2 Banderas de Línea de Comandos
- `edi -h` / `edi --help` : Muestra el resumen de uso y atajos de teclado.
- `edi -v` / `edi --version` : Muestra la versión actual de `edi`.
- `edi --init` : Genera la estructura de configuración en `~/.edi-config/`.
- `edi --config-dir` : Imprime la ruta de configuración activa en tu sistema.

---

## 2. Anatomía de la Interfaz

```text
 ╭──────────────────────────────────────────────────────────────────────────╮
 │ [1: main.go*]  [2: config.json]  [3: help.md]                    (+)     │ ◄── Barra de Pestañas
 ├───────────────────┬──────────────────────────────────────────────────────┤
 │ ▸ defaults/       │  1 package main                                      │
 │ ▾ internal/       │  2                                                   │
 │   ▸ buffer/       │  3 import "fmt"                                      │ ◄── Área de Edición
 │   ▸ config/       │  4                                                   │
 │   ▾ editor/     • │  5 func main() {                                     │
 │     editor.go   M │  6     fmt.Println("Hola edi")                       │
 │     tree.go       │  7 }                                                 │
 │   main.go       M │                                                      │
 ├───────────────────┴──────────────────────────────────────────────────────┤
 │  edi | main.go* | git: main [~2 +1] (tab 1/3)   Ln 5, Col 4 | Go | dracula│ ◄── Barra de Estado
 ╰──────────────────────────────────────────────────────────────────────────╯
```

1. **Barra Superior de Pestañas**: Muestra todos los buffers abiertos, el número de pestaña y un asterisco `*` si hay cambios sin guardar.
2. **Panel Lateral (Explorador `Ctrl+B`)**: Árbol jerárquico de archivos con carpetas desplegables y las insignias de estado de Git a la derecha.
3. **Área de Edición**: Numeración de línea, cursor destacado y coloreado de sintaxis TrueColor.
4. **Barra de Estado Inferior**: Muestra el archivo activo, estado de Git (`git: rama [cambios]`), número de pestaña, posición `Ln/Col`, lenguaje detectado y tema en uso.

---

## 3. Atajos de Teclado Esenciales

### 3.1 Gestión de Archivos y Espacio de Trabajo
| Atajo | Acción | Descripción |
| :--- | :--- | :--- |
| `Ctrl+O` | **Buscador Rápido (Fuzzy Finder)** | Modal flotante para buscar y saltar a cualquier archivo del proyecto. |
| `Ctrl+P` | **Paleta de Comandos** | Modal interactivo para acceder a todas las funciones y comandos. |
| `Ctrl+B` | **Alternar Explorador** | Abre o cierra el panel lateral del explorador de archivos. |
| `Ctrl+S` / `F2` | **Guardar archivo** | Guarda los cambios. Si es un archivo nuevo, solicita el nombre en la terminal. |
| `Ctrl+N` | **Nueva pestaña** | Crea un buffer vacío sin título. |
| `Ctrl+W` | **Cerrar pestaña** | Cierra la pestaña activa (advierte si no se ha guardado). |
| `Ctrl+Tab` / `Alt+Right` | **Pestaña siguiente** | Pasa a la siguiente pestaña abierta. |
| `Alt+Left` | **Pestaña anterior** | Vuelve a la pestaña anterior. |
| `Alt+1` a `Alt+9` | **Ir a pestaña N** | Salta directamente a la pestaña número 1 a 9. |
| `Ctrl+Q` | **Salir de edi** | Sale del editor con confirmación de seguridad si hay cambios sin guardar. |

> [!NOTE]
> **Usuarios de macOS**: Asegúrate de usar la tecla `Control` física (`⌃ + S`, `⌃ + O`) o las teclas de función como `F2`. Si usas `Cmd ⌘ + S`, tu terminal interceptará la pulsación.

### 3.2 Edición y Navegación
| Atajo | Acción | Descripción |
| :--- | :--- | :--- |
| `Shift + Flechas` | **Seleccionar texto** | Selección continua multi-línea. |
| `Ctrl+C` | **Copiar** | Copia la selección o la línea actual al portapapeles. |
| `Ctrl+X` | **Cortar** | Corta la selección o la línea actual. |
| `Ctrl+V` | **Pegar** | Pega el texto almacenado en el portapapeles. |
| `Ctrl+Z` | **Deshacer (Undo)** | Revierte la última acción en el buffer activo. |
| `Ctrl+Y` | **Rehacer (Redo)** | Re-aplica la última acción deshecha. |
| `Ctrl+D` | **Duplicar línea** | Clona la línea actual inmediatamente debajo. |
| `Ctrl+K` | **Eliminar línea** | Borra por completo la línea donde está el cursor. |
| `Alt+Up` / `Ctrl+Up` | **Mover línea arriba** | Desplaza la línea o bloque seleccionado una fila arriba. |
| `Alt+Down` / `Ctrl+Down` | **Mover línea abajo** | Desplaza la línea o bloque seleccionado una fila abajo. |
| `Ctrl+F` | **Buscar texto** | Búsqueda cíclica dentro del archivo actual. |
| `Ctrl+G` | **Ir a línea** | Salto numérico directo a una fila específica. |
| `F1` | **Documentación interna** | Abre la guía de ayuda integrada en una pestaña virtual. |

### 3.3 Explorador de Archivos Lateral (`Ctrl+B`)
- `↑` / `↓` : Desplazarse verticalmente por la lista de archivos.
- `Enter` : Expandir o colapsar carpeta, o abrir el archivo seleccionado en una pestaña.
- `→` / `←` : Expandir o colapsar carpetas.
- `a` / `n` : Crear un nuevo archivo directamente dentro del directorio seleccionado.
- `Tab` / `Esc` : Devolver el foco al editor de texto.

---

## 4. Integración Automática con Git

Si el proyecto actual es un repositorio Git, `edi` se activa automáticamente:

### 4.1 Insignias en el Explorador
Alineadas a la derecha de cada elemento del árbol:
- `M` *(amarillo)* : Archivo modificado.
- `?` *(verde)* : Archivo sin seguimiento (*untracked*).
- `+` *(verde)* : Archivo en área de preparación (*staged*).
- `D` *(rojo)* : Archivo eliminado.
- `R` *(cian)* : Archivo renombrado.
- `•` *(indicador)* : Carpeta que contiene archivos modificados en su interior.

### 4.2 Comandos Git en la Paleta (`Ctrl+P`)
- **`Git: Ver Diff del archivo actual`**: Abre una pestaña virtual con el diff coloreado frente a `HEAD` (utiliza sintaxis `.diff` con líneas `+` en verde y `-` en rojo).
- **`Git: Ver Diff de todo el proyecto`**: Abre un archivo virtual con todos los cambios unificados del repositorio.
- **`Git: Archivos modificados (Status)`**: Abre un filtro rápido para saltar instantáneamente a cualquiera de los archivos con cambios.
- **`Git: Refrescar estado`**: Fuerza la actualización inmediata de Git.

---

## 5. Personalización (`~/.edi-config/`)

Ejecuta `edi --init` para crear la carpeta de configuración con temas y sintaxis base:

```
~/.edi-config/
├── config.json         # Opciones globales (JSONC con comentarios)
├── keybindings.json    # Atajos de teclado personalizados
├── themes/             # Temas personalizados TrueColor (.json)
├── syntax/             # Definiciones de lenguaje (.json)
└── plugins/            # Formateadores y herramientas (.json)
```

### 5.1 Opciones de `config.json`
```json
{
  "theme": "dracula",
  "tab_size": 4,
  "expand_tabs": true,
  "line_numbers": true,
  "highlight_current_line": true,
  "status_bar": true,
  "auto_indent": true,
  "git_integration": true
}
```

### 5.2 Agregar Nuevos Lenguajes (`syntax/*.json`)
Crea un archivo como `~/.edi-config/syntax/rust.json`:
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

### 5.3 Plugins de Formateo Automático (`plugins/*.json`)
Conecta formateadores automáticos al guardar:
```json
{
  "name": "Go Tools",
  "languages": ["go"],
  "hooks": {
    "on_save": [
      {
        "action": "pipe_buffer",
        "command": "gofmt",
        "description": "Formateo automático al guardar"
      }
    ]
  }
}
```
