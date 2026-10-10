# Reglas y Directivas para Agentes de IA (AGENTS.md)

Este repositorio utiliza una guía técnica y arquitectónica estructurada para asistentes de IA y agentes de desarrollo autónomo.

Por favor, consulta la documentación detallada en:
- **[Especificación e Instrucciones para IA](docs/ai_context.md)**: Reglas de diseño, arquitectura, manejo de concurrencia y pruebas.
- **[Arquitectura Técnica](docs/architecture.md)**: Diagrama de módulos y flujo de eventos.
- **[Hoja de Ruta (Roadmap)](docs/roadmap.md)**: Hitos completados y tareas pendientes.

## Reglas Rápidas No Negociables
1. **Cero CGO**: Compilación estricta con `CGO_ENABLED=0` y Go puro sin dependencias externas en C.
2. **Cero Emojis**: Prohibido usar emojis en la terminal TUI para evitar desalineaciones en rejillas monoespaciadas.
3. **UI Asíncrona**: Cualquier operación externa o de Git debe ejecutarse en segundo plano con notificación mediante `tcell.NewEventInterrupt(nil)`.
4. **Tests con Simulación**: Probar cambios con `tcell.NewSimulationScreen("UTF-8")` y verificar siempre con `go test -v ./...`.
