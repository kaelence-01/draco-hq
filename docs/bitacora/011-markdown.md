# 011 — Documentos en Markdown y preservación en el adaptador (4b)

**Fecha:** 2026-10-19
**Rama:** feature/markdown → develop
**Fase del roadmap:** 2 (proveedores, actividad 4b)

## 1. Solicitud

Segunda mitad de la actividad 4: el documento de despliegue y el consolidado QA
se generan en Markdown y el adaptador de Confluence los convierte con el
conversor de la 4a. La preservación de lo escrito a mano pasa al adaptador. El
autor aprobó antes las diferencias del documento publicado (diseño §5) y, de
paso, validó la idea final con una demo interactiva del modo visual.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Los flujos armaban ADF y preservaban secciones leyendo ADF | Ningún destino que no sea Confluence podía publicar; el flujo dependía del formato nativo |
| La «Observación» del consolidado era una celda de tabla | La preservación genérica por secciones no podía alcanzarla |
| Los valores (nombres de archivo, observaciones, repos) se insertaban en tablas | En Markdown, un `\|` o un `<` en un valor rompe la tabla o abre HTML |

## 3. Diseño técnico

- **`document`** genera Markdown: `DeployMarkdown` y `QAMarkdown` con el
  formato aprobado (tablas clave/valor con fila de encabezado vacía y claves en
  negrita, `<br>` en celdas, `<url>` para tarjetas, `✔`/`✖`/`?`). La
  «Observación» del consolidado pasa a ser la sección `### Observación`.
  Se fueron el ADF, `ExtractSection`, `QAPreserved` y el campo sin uso
  `QADoc.QATasks`.
- **Escapado:** todo valor pasa por `mdText` (escapa la puntuación que abre
  marcado, cierra celdas, abre HTML o entidades; los saltos de línea pasan a
  espacio) o `mdURL` (espacios, paréntesis, `<`, `>` y `|`). Un test lo
  verifica con el parser real: con `|` y `<script>` en repo, carpeta y archivo,
  cada fila sigue teniendo sus 5 celdas. **Encontró un error al escribirlo:**
  un `|` dentro de la URL de un enlace también cortaba la celda; `mdURL` ahora
  lo codifica.
- **Puerto `DocStore`:** `Create`/`Update` reciben `provider.Document{Title,
  Markdown, Preserve}`; nuevo `Sections(ref, names)`; se quitaron `Read` y
  `Doc`. El flujo ya no ve el formato nativo.
- **Preservación en el adaptador** (`atlassian/preserve.go`): `Update` lee la
  página actual, extrae en ADF cada sección de `Preserve` y la pone en lugar
  de la de la plantilla, sobre la versión que leyó; si no puede leerla, no
  escribe. **Una sección existe cuando existe su título, aunque esté vacía**:
  una «Observación» nueva no hace preguntar en cada actualización, y si QA la
  dejó vacía se conserva vacía.
- **Migración:** los consolidados publicados antes de 4b tienen la
  «Observación» en la celda de la tabla Resumen; si no hay sección, el
  adaptador lee esa celda.
- **Flujos:** `Draft` lleva `Markdown`; `DraftUpdate` pregunta con `Sections`
  (`deploy`: `Draft.Preserved`; `qa`: `Draft.NotPreserved` con los títulos que
  faltan); `Publish` declara `Preserve`.
- **Goldens:** los de `document` y los de los flujos pasan a Markdown
  (`*.golden.md`, legibles en el diff), con `golden.AssertText`. Lo que recibe
  Confluence queda fijado en el adaptador (`adf-deploy`, `adf-qa`),
  convirtiendo los documentos reales. Se borraron los goldens ADF anteriores:
  el cambio es el aprobado (§5).

## 4. Archivos

- `internal/document/builder.go`, `qa_builder.go`, `markdown.go` (nuevo) —
  Markdown y escapado; `preserve.go` y sus tests, eliminados.
- `internal/document/builder_test.go`, `golden_test.go`,
  `testdata/*.golden.md`.
- `internal/provider/provider.go` — `Document`, `Sections`; fuera `Doc` y
  `Read`.
- `internal/provider/atlassian/preserve.go` (nuevo), `ports.go`,
  `ports_test.go`, `preserve_test.go` (nuevo), `markdown_test.go`,
  `testdata/adf-*.golden.json`.
- `internal/deploy/deploy.go`, `deploy_test.go`, `golden_test.go`,
  `testdata/draft-create.golden.md`.
- `internal/qa/qa.go`, `qa_test.go`, `golden_test.go`,
  `testdata/draft-create-sprint.golden.md`.
- `internal/golden/golden.go` — `AssertText`.
- `CLAUDE.md`, `docs/arquitectura/proveedores.md`, `docs/roadmap.md`.

## 5. Cómo verificar

```bash
go test ./...
cat internal/document/testdata/*.golden.md     # el Markdown que se publica
go test ./internal/provider/atlassian -run 'DocStore|Section|GoldenPublished' -v
```

## 6. Pendientes

- Verificar contra una instancia real de Confluence cuando exista el comando
  (fase 3): que el ADF generado se acepte y se vea como en la demo.
- La fecha del consolidado sigue con el mes en inglés (`21 Sep 2026`): pasa a
  la plantilla en la actividad 5.
- La migración de la celda «Observación» puede quitarse cuando no queden
  consolidados con el formato anterior en uso.
