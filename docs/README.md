# Documentación de edi

Bienvenido a la documentación oficial y técnica de **edi**, el editor de texto en consola (TUI) moderno, rápido, portable y personalizable desarrollado en **Go**.

---

## Estructura de la Documentación

| Documento | Descripción |
| :--- | :--- |
| [Guía de Usuario](user_guide.md) | Manual completo de uso: atajos de teclado, navegación, buscador de archivos, Git y personalización. |
| [Arquitectura Técnica](architecture.md) | Diagrama de paquetes, flujo de eventos, modelo de buffers, concurrencia y ciclo de vida de la UI. |
| [Hoja de Ruta (Roadmap)](roadmap.md) | Fases completadas, características a corto/medio plazo y visión a largo plazo del proyecto. |
| [Guía de Contribución](contributing.md) | Entorno de desarrollo, ejecución de tests de simulación, compilación cruzada y estándares de código. |
| [Contexto e Instrucciones para IA](ai_context.md) | Especificación técnica exhaustiva, reglas de diseño, invariantes y guía de desarrollo para asistentes IA y agentes. |

---

## Principios de Diseño de edi

1. **Arranque instantáneo (< 10 ms)**: El editor debe iniciar de inmediato en cualquier máquina o servidor, sin dependencias pesadas ni tiempos de espera.
2. **Binario estático único sin CGO (`CGO_ENABLED=0`)**: Máxima portabilidad. Se compila en un único ejecutable sin dependencias externas en tiempo de ejecución.
3. **Cero configuración previa obligatoria**: Incluye temas, sintaxis base y atajos compilados dentro del binario mediante `embed.FS`. Funciona desde el primer segundo.
4. **Estética terminal pulida y pura**: Paleta TrueColor (24 bits) de alta gama, alineación estricta de caracteres monoespaciados y sin emojis para compatibilidad total con emuladores de terminal antiguos y modernos.
5. **UI no bloqueante y asíncrona**: Operaciones de Git, lectura de disco y formateadores externos corren en goroutines en segundo plano, manteniendo 60 FPS estables en la terminal.
