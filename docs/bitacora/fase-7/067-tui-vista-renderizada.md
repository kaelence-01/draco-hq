# 067 — Vista previa renderizada en Generar y QA

**Fecha:** 2026-10-28
**Rama:** feature/tui-vista-renderizada → develop
**Fase del roadmap:** 7

## 1. Solicitud

El autor pidió seguir con la actividad 5 de la fase 7: la vista previa del
documento dibujada, con `m` para el Markdown crudo, para Generar y para QA.

## 2. Motivación

| Problema | Daño real |
|---|---|
| La vista previa mostraba el Markdown crudo, con los comentarios `draco:tabla` y `draco:recuadro`, filas de guiones y `<br>` dentro de las celdas | Se revisaba un documento que no se parece al que se publica; un error de contenido se notaba recién en Confluence |
| Nada decía qué pasaría al publicar (crear o actualizar) cuando el documento era nuevo | La pantalla solo hablaba cuando había algo que conservar |
| Los pasos del flujo no se veían en la vista previa | Quien llega a ella no sabe cuánto falta |

## 3. Diseño técnico

- **Renderizador propio, sin dependencia nueva** (`mdrender.go`), como decidió
  el §11: `goldmark` sigue solo en los adaptadores. Lee líneas, no un árbol:
  el Markdown de draco es un subconjunto pequeño y conocido.
  - Dibuja títulos, párrafos, listas con casillas, tablas, el recuadro y las
    marcas `✔ ✖ ?`; negrita, cursiva, código y enlaces; y quita los escapes
    (`\|`, `\*`) que pone `mdText`.
  - Sigue las convenciones del conversor de Confluence: los comentarios
    `draco:` no se muestran, una fila de encabezado vacía se descarta, una
    celda toda en negrita es una etiqueta, `<br>` con cada línea en `- ` son
    viñetas.
  - **Lo que no reconoce se muestra tal cual** (citas, código, listas
    numeradas, líneas horizontales, HTML, una línea con `|` sin su fila de
    guiones). Una plantilla del usuario con otro Markdown no se rompe ni se
    traga texto.
- **Anchos de las tablas:** `anchos=` son proporciones del ancho disponible,
  con un piso por columna: no baja de su palabra más larga (hasta 14
  columnas) mientras haya sitio, y solo si no hay se parte. Sin `anchos=`,
  cada columna es tan ancha como su contenido y se encoge si no cabe. En el
  primer intento, a 56 columnas «Épica» y «Tarea(s)» se partían letra a
  letra; el piso lo corrige.
- **El recuadro** se arma 4 columnas más estrecho y mide lo que su contenido
  (como el prototipo): en el primer intento se salía del ancho y se perdía su
  borde derecho.
- **Celdas que son solo una URL:** si no caben muestran su cola
  (`…/browse/OPS-7`), como la tarjeta de Confluence muestra la clave de la
  tarea. Sin eso, la columna de tareas del consolidado partía las URL en
  trozos.
- **Vista previa** (`preview.go`): el documento dibujado por defecto, `m`
  alterna con el Markdown (en la ayuda completa, y recordado a la derecha de
  la pestaña), `tab` con los archivos (solo Generar). `m` solo vale en la
  pestaña del documento. El contenido del `viewport` se rearma solo cuando
  cambia el ancho o el modo.
- **Línea de título y de estado:** el título lleva los pasos a la derecha
  (`docAbout.steps`, de la actividad 4). Si el título no deja sitio (a 110
  columnas, con el título largo de un despliegue), los pasos pasan a la línea
  de estado. La línea de estado dice qué pasará: «Se creará un documento
  nuevo», o lo que ya había (se conserva lo escrito a mano / lo que se
  perdería).
- **No entra:** el recuento «2 commits · 5 archivos» del prototipo en la línea
  de estado (la vista previa no tiene esos números: viven en la pantalla de
  Commits) ni la tabla de archivos con commit y letra A/M/D (el flujo
  agrupa los archivos por carpeta, sin commit ni estado).
- **Descartado:** probar el renderizador contra `goldmark`. Dos
  implementaciones del mismo subconjunto se comparan para encontrar
  diferencias que no importan; se prueba por construcción y contra los
  goldens reales (`deploy` y `qa`), a 56, 80, 100 y 160 columnas, que ninguna
  línea pase del ancho y que no se vea ninguna marca de draco.

## 4. Archivos

- `internal/tui/mdrender.go` — nuevo: `renderMarkdown`, el reparto de columnas, `inline`, `shortURL`.
- `internal/tui/preview.go` — documento dibujado, `m`, pasos, línea de estado y pestañas.
- `internal/tui/mdrender_test.go` — nuevo: por construcción (títulos, directivas, tabla con anchos, encabezado vacío, marcas, viñetas, casillas, recuadro, inline, escapes, líneas desconocidas), `columnWidths`, `shortURL` y los goldens reales a cuatro anchos.
- `internal/tui/preview_test.go` — nuevo: `m`, `tab`, ayuda, pasos a tres anchos, ancho cambiante y QA.
- `docs/arquitectura/tui.md`, `docs/guia-usuario.md` — la vista previa y el renderizador.
- `docs/roadmap.md` — actividad 5 con ✅.
- `docs/bitacora/fase-7/067-tui-vista-renderizada.md`, índice.

## 5. Cómo verificar

```bash
go vet ./internal/tui && go test ./internal/tui   # corridos: ok
make check                                        # lo corre el autor
make run                                          # Generar hasta la vista previa: se ve dibujada; m el Markdown; tab los archivos
```

## 6. Pendientes

- `make check` completo y la mirada en Windows Terminal, a cargo del autor,
  con un documento real de su plantilla (la prueba que más importa: el
  renderizador solo conoce el Markdown de las plantillas incluidas y de los
  goldens).
- QA con la tabla consolidada real y muchos sprints: no se probó contra
  Jira real (heredado de las bitácoras 037 y 038).
- Siguiente: QA, contadores y tabla a todo el ancho (actividad 6).
