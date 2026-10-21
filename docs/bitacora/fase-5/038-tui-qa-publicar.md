# 038 — QA en la TUI (2): vista previa, ubicación y publicar compartidos

**Fecha:** 2026-10-20
**Rama:** feature/tui-qa-publicar → develop
**Fase del roadmap:** 5

## 1. Solicitud

Seguir con la siguiente actividad de la fase 5: QA (2), actividad 4b del
plan de la bitácora 034.

## 2. Motivación

| Problema | Daño real |
|---|---|
| La pantalla de QA terminaba en el comando para publicar | QA tenía que salir de la TUI y repetir la evaluación entera con `draco qa` para publicar lo que acababa de revisar |
| Vista previa, ubicación y publicar recibían tipos de `deploy` | Para QA había que copiarlas, con sus reglas: aceptar con `s` lo que no se conserva, no salir con `esc` mientras publica |
| El pie de teclas podía medir una columna más que la terminal | Con las teclas de la evaluación de QA en 80 columnas, la pantalla entera se pasaba de ancho |

## 3. Diseño técnico

- **`docRun`** (`internal/tui/docrun.go`): la interfaz con lo que piden las
  tres pantallas:
  - `about()`: sustantivo, qué busca Ubicación, destino, tarea del enlace y
    comando equivalente;
  - `updating()`, `draft(ctx)`, `locations(ctx)` y `publish(ctx, sibling)`.

  Las pantallas trabajan con tipos propios de la TUI (`docDraft`,
  `docLocations`, `docResult`), no con los de un flujo.
- **Dos adaptadores que solo traducen tipos:**
  - `genRun`: el de Generar, que ya existía; pierde `st`, que no usaba.
  - `qaRun`: actualiza si `Session.Existing` existe, como `draco qa` (el
    título es la clave del consolidado).

  Cada uno guarda su borrador tipado entre `draft` y `publish`, y explica
  sus errores: `deploy.ErrNoLocations` propone un perfil Markdown;
  `qa.ErrNoLocations` agrega el título, como el comando.
- **Lo propio de Generar es opcional:** la pestaña Archivos (`files`),
  `TeamWide`/`OwnerUnknown`, la línea «Enlace» (`issue`) y `LinkErr`. Para
  QA no aparecen. El resultado dice «Consolidado creado/actualizado».
- **La vista previa arma el borrador siempre en un `tea.Cmd`**, también al
  crear: antes `DraftCreate` iba directo en `init`. Una sola vía para los
  dos casos; el texto del spinner depende de `updating`.
- **QA:** `enter` al terminar la evaluación abre la vista previa (la línea
  del comando para publicar se reemplaza por esa indicación). El comando
  equivalente sigue en Publicar.
- **Pie de teclas:** la ayuda se mide con `ancho - 1`, porque el marco le
  antepone un espacio. Error de la fase 4 que aparecía solo cuando las
  teclas medían justo el ancho, como en QA en 80 columnas.
- Los tests de Generar (`generate_test.go`, `publish_test.go`) pasaron sin
  cambios.

## 4. Archivos

- `internal/tui/docrun.go` — nuevo: `docRun`, sus tipos, `genRun` y
  `qaRun`.
- `internal/tui/preview.go`, `locations.go`, `publish.go` — sobre `docRun`.
- `internal/tui/progress.go` — crea `genRun` sin `st`.
- `internal/tui/qa.go` — `enter` abre la vista previa.
- `internal/tui/app.go` — ancho del pie de teclas.
- `internal/tui/qa_test.go` — consolidado nuevo de punta a punta en una
  carpeta Markdown, actualización que conserva Observación y
  Consideraciones, `s` cuando falta una sección, sin consolidado previo.
- `docs/arquitectura/tui.md`, `docs/roadmap.md`, `README.md` — estado y
  detalle.
- `docs/bitacora/fase-5/038-tui-qa-publicar.md`, índice.

## 5. Cómo verificar

```bash
make test pkg=./internal/tui run='QA|Preview|Create|Publish' v=1
make check
make run            # Consolidado QA hasta publicar, en una carpeta Markdown
```

## 6. Pendientes

- Pruebas manuales del autor: consolidado nuevo y actualizado contra su Jira
  y su Confluence, y Generar otra vez de punta a punta (sus pantallas
  cambiaron por dentro).
- Actividad 5: cierre de la fase 5 y release `v0.3.0`.
