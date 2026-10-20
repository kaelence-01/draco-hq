# 010 — Conversor Markdown → ADF (4a)

**Fecha:** 2026-10-19
**Rama:** feature/markdown-adf → develop
**Fase del roadmap:** 2 (proveedores, actividad 4a)

## 1. Solicitud

Actividad 4 de la secuencia de proveedores: el documento pasa a Markdown y el
adaptador de Confluence lo convierte a ADF. Antes de empezar, el autor aprobó
las diferencias visuales del documento publicado (diseño, §5) y dividir la
actividad en 4a (conversor) y 4b (documentos, preservación y goldens).

## 2. Motivación

| Problema | Daño real |
|---|---|
| El documento se arma directamente en ADF | Solo Confluence puede publicarlo; las plantillas (actividad 5) no tendrían un formato común |
| Markdown no expresa tarjetas de enlace, emojis de Confluence ni tablas clave/valor | Convertir sin convenciones empobrecería el documento que hoy recibe la gente |
| El conversor recibe texto que escribe el usuario (plantillas, observaciones) | Un pánico o HTML interpretado rompería la publicación o abriría una inyección |

## 3. Diseño técnico

- **`atlassian.MarkdownToADF(md)`** (`markdown.go`) recorre el AST de
  `goldmark` v1.8.6 con las extensiones GFM de tablas, tareas y tachado.
  Soporta títulos, párrafos, negrita, cursiva, código, tachado, enlaces,
  listas, listas de tareas, tablas, bloques de código, citas y separadores.
- **Convenciones** (diseño §5, aprobadas): enlace cuyo texto es la URL o
  `<https://…>` → inlineCard; `✔`/`✖` → emoji de Confluence; fila de
  encabezado vacía → se omite; celda entera en negrita → celda de encabezado
  sin la marca; `<br>` en una celda → salto de línea; tablas a todo el ancho.
- **Texto:** goldmark guarda el texto crudo y el renderer HTML resuelve
  escapes y entidades; el conversor aplica las mismas funciones
  (`UnescapePunctuations`, `ResolveNumericReferences`, `ResolveEntityNames`).
  Sin esto, `\*` o `&amp;` llegarían literales a Confluence.
- **HTML:** salvo `<br>`, todo HTML queda como texto literal, nunca se
  interpreta.
- **Casillas:** `localId` únicos por lista y por ítem (`tasklist-N`,
  `task-N-M`), estado `TODO`/`DONE` desde `[ ]`/`[x]`.
- **4a no cambia nada publicado:** el conversor no se usa todavía desde los
  flujos; la 4b lo conecta.
- **Tests:** uno por construcción y convención; golden del conversor con el
  Markdown de ejemplo del documento de despliegue (deja a la vista el ADF que
  generará la 4b); fuzzing: ~360.000 entradas aleatorias en 20 s sin pánicos y
  siempre ADF serializable.

## 4. Archivos

- `internal/provider/atlassian/markdown.go` — conversor (nuevo).
- `internal/provider/atlassian/markdown_test.go`,
  `testdata/markdown-deploy.golden.json` — tests, golden y fuzzing.
- `go.mod`, `go.sum` — `github.com/yuin/goldmark v1.8.6`.
- `docs/arquitectura/proveedores.md` — §5 con las convenciones implementadas.
- `docs/roadmap.md` — actividad 4 dividida en 4a y 4b.
- `CLAUDE.md` — dependencias y `markdown.go` en la arquitectura.

## 5. Cómo verificar

```bash
go test ./internal/provider/atlassian -run 'Markdown|Golden' -v
go test ./internal/provider/atlassian -run '^$' -fuzz FuzzMarkdownToADF -fuzztime 20s
```

## 6. Pendientes

- Ítems de una lista de tareas con sublistas: solo se convierte la línea de la
  tarea (draco no los genera).
- 4b: builder a Markdown, `DocStore.Sections` y preservación en el adaptador,
  «Observación» como sección con lectura de la celda vieja, goldens
  regenerados.
