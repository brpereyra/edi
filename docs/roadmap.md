# Hoja de Ruta (Roadmap) de edi

Este documento traza la evolución, hitos completados y la visión a futuro para **edi**.

---

## Estado Actual: v0.1.0 (Completado)

El núcleo de `edi` se encuentra completamente operativo como editor estático de grado de producción:

- [x] **Arranque ultra-rápido**: Tiempo de inicio inferior a 10 ms.
- [x] **Distribución autónoma**: Binario estático compilado en Go puro (`CGO_ENABLED=0`).
- [x] **Gestión de Pestañas y Buffers**: Soporte para múltiples archivos abiertos simultáneamente con alternancia rápida (`Ctrl+Tab`, `Alt+1..9`).
- [x] **Explorador de Archivos Lateral (`Ctrl+B`)**: Árbol jerárquico con navegación por flechas, colapso/expansión y creación de archivos en caliente.
- [x] **Paleta de Comandos Modal (`Ctrl+P`)**: Acceso interactivo flotante a todas las funciones y comandos del editor.
- [x] **Buscador Rápido de Archivos (`Ctrl+O`)**: Selector difuso (*Fuzzy Finder*) para saltar instantáneamente entre archivos del proyecto.
- [x] **Guardado Seguro en Terminal**: Diálogo interactivo embebido para nombrar archivos nuevos sin invocar herramientas externas.
- [x] **Historial Multinivel**: Deshacer (`Ctrl+Z`) y Rehacer (`Ctrl+Y`) independientes por buffer.
- [x] **Reordenamiento de Líneas**: Movimiento vertical de líneas o selecciones con `Alt+Up` / `Alt+Down`.
- [x] **Integración Automática con Git**: Detección sin configuración, insignias de estado en el explorador (`M`, `?`, `+`, `D`), visor de Diff y rama activa en la barra de estado.
- [x] **Resaltado y Temas TrueColor Declarativos**: Soporte drop-in para temas y lenguajes mediante archivos `.json`.

---

## Fase 2: Edición Avanzada y Productividad (Corto Plazo)

El foco de esta fase es expandir la ergonomía de trabajo en proyectos de mediana y gran envergadura:

### 1. Vistas Divididas (Split Panes)
- [ ] División horizontal (`Ctrl+W s`) y vertical (`Ctrl+W v`).
- [ ] Navegación entre paneles con `Ctrl+W` seguido de flechas.
- [ ] Capacidad de ver el mismo buffer en dos paneles independientes (scroll independiente).

### 2. Búsqueda Global en el Proyecto (Project-wide Search)
- [ ] Modal interactivo (`Ctrl+Shift+F`) para buscar texto en todos los archivos del workspace.
- [ ] Integración con algoritmo de búsqueda en paralelo nativo en Go.
- [ ] Salto directo a la línea y columna del resultado seleccionado.

### 3. Ajuste de Línea (Soft Wrap)
- [ ] Opción en `config.json` (`"word_wrap": true/false`).
- [ ] Alternancia en caliente desde la paleta de comandos (`toggle:word_wrap`).

### 4. Acciones Rápidas de Git en el Explorador
- [ ] Tecla `s` sobre un archivo en el explorador lateral para agregar a staging (`git add`).
- [ ] Tecla `u` para quitar de staging (`git reset HEAD <archivo>`).
- [ ] Tecla `d` para descartar cambios en el archivo de trabajo (con confirmación de seguridad).

---

## Fase 3: Integración de Lenguajes y Herramientas (Medio Plazo)

Dotar al editor de inteligencia de código sin comprometer su velocidad ni su ligereza:

### 1. Cliente LSP (Language Server Protocol) Básico
- [ ] Comunicación estándar vía `stdio` con servidores como `gopls`, `pyright`, `rust-analyzer` o `typescript-language-server`.
- [ ] Salto a definición (`F12` / `gd`).
- [ ] Diagnósticos y advertencias inline en el margen izquierdo.
- [ ] Autocompletado emergente liviano (`Ctrl+Space`).

### 2. Terminal Embebida (Bottom Drawer Terminal)
- [ ] Panel inferior desplegable (`Ctrl+\``) para ejecutar comandos de consola rápidos sin salir de `edi`.
- [ ] PTY nativa sin librerías C externas (usando `creack/pty`).

### 3. Persistencia y Restauración de Sesión
- [ ] Guardar estado de pestañas abiertas, posición de cursor y carpetas expandidas en `.edi/session.json`.
- [ ] Restauración automática al reiniciar el editor si se activa en la configuración (`"restore_session": true`).

---

## Fase 4: Ecosistema y Escalabilidad (Largo Plazo)

- [ ] **Plugins en WebAssembly (WASM)**: Permitir extensiones complejas escritas en Rust, Go o Zig ejecutadas en una sandbox segura y portable.
- [ ] **Modo Headless / Servidor Remoto**: Arquitectura desacoplada para conectarse a una sesión remota mediante túnel seguro con latencia mínima.
- [ ] **Gestor de Paquetes Integrado**: Comando `edi pkg install <theme|syntax|plugin>` para descargar complementos desde repositorios de la comunidad.
