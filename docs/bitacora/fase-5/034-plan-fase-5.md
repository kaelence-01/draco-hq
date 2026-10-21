# 034 — Diseño de las pantallas y plan de la fase 5

**Fecha:** 2026-10-20
**Rama:** docs/plan-fase-5 → develop
**Fase del roadmap:** 5

## 1. Solicitud

Proceder con la fase 5 de draco. La bitácora 033 dejó como pendiente hacer
el plan de Backlog, QA y Exportar antes de escribir código, como en la fase 4.

## 2. Motivación

| Problema | Daño real |
|---|---|
| La fase 5 era una línea del roadmap («backlog, QA, exportar») | Sin actividades ni criterio de cierre, no habría forma de saber qué cierra cada una ni la fase |
| La resolución de repos, las opciones y el guardado del reporte del backlog viven en `cmd/backlog.go` | La TUI no importa `cmd`: la pantalla habría copiado esas reglas y los dos modos analizarían distinto |
| `backlog.Scan` no recibe `context` | `esc` no podría detener un análisis largo, contra la norma 6 del modo visual |
| Vista previa, ubicación y publicar de la TUI reciben tipos de `deploy` | QA tiene los mismos pasos; sin decidirlo antes, se habrían copiado con sus reglas (`NotPreserved`, publicar sin salida a medias) |
| El modo comando guarda lo exportado con el diálogo nativo del sistema | Desde la TUI, una ventana externa la deja esperando sin señal visible, y en un test abriría una ventana real |

## 3. Diseño técnico

Diseño en `docs/arquitectura/tui.md` §6 (el §6 pasa de «Pantallas de la
fase 4» a «Pantallas», con las de las dos fases, para no renumerar el §10 que
citan `CLAUDE.md` y `estandares-desarrollo`). Decisiones:

- **Exportar:** formulario (tarea y space opcional), `Search`, lista con
  filtro cuando hay varios documentos y, al final, el texto en un `viewport`
  con un campo de ruta editable: carpeta actual y el nombre sugerido. Pide
  `s` si el archivo existe. Descartado el diálogo nativo (motivo arriba);
  queda como posible mejora si se pide.
- **Backlog:** primero el núcleo sale de `cmd` a `internal/backlog`
  (repos, opciones con `--since`, nombre y guardado del reporte) y `Scan`
  recibe `context` (`git` y `composer` con `exec.CommandContext`); `cmd` no
  cambia su salida y los mensajes que nombran flags se quedan ahí. Pantallas:
  formulario, progreso por repo (un repo que falla no detiene al otro) y
  hallazgos por pestaña de repo, con filtro, detalle y `s` para guardar el
  reporte completo donde lo deja `-o`. Sin `--limit`: la lista se desplaza.
- **QA:** formulario (sprint vacío: Kanban; módulo), y la evaluación en vivo
  con un `tea.Cmd` por `EvaluateTask`, que suma una fila por tarea con sus
  marcas (`?` para `DeployDocUnknown`). El flujo ya tenía el paso por tarea:
  no cambia. Un consolidado existente se actualiza sin preguntar, como el
  comando (el título es su clave).
- **Compartido con Generar:** vista previa, ubicación y publicar reciben una
  interfaz de la TUI que Generar y QA implementan con adaptadores que solo
  traducen tipos; lo propio de Generar (Archivos, `TeamWide`, `LinkErr`) es
  opcional. Criterio: los tests de Generar pasan sin cambios.
- **`Deps` suma** `Scan` (el `scanRepo` de `cmd`) y `Now` (el `now` de
  `cmd`, para la ventana Kanban), con lo que los tests usan los mismos dobles.
- **Plan:** 6 actividades en el roadmap, con criterio de cierre: este plan,
  Exportar, Backlog, QA en dos partes (como Generar) y el cierre con
  `v0.3.0`. Exportar va primero por ser el flujo más corto y sin publicación.

## 4. Archivos

- `docs/arquitectura/tui.md` — estado, archivos y `Deps` de la fase 5;
  §6 con Exportar, Backlog, QA y lo compartido; tests y decisiones.
- `docs/roadmap.md` — fase 5 en curso, sus actividades y criterios.
- `README.md` — la fase 5, en curso.
- `docs/bitacora/fase-5/034-plan-fase-5.md`, `docs/bitacora/README.md` —
  primera entrada de la fase 5 y su sección en el índice.

## 5. Cómo verificar

```bash
make check
```
