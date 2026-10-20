# 015 — Plantillas del documento

**Fecha:** 2026-10-19
**Rama:** feature/plantillas → develop
**Fase del roadmap:** 2

## 1. Solicitud

Seguir avanzando con la fase 2, sin prisa: la actividad 5 de la secuencia de
proveedores, plantillas. Alcance recomendado y aceptado: plantillas incluidas,
resolución por carpetas, bloques personalizables y manifiesto; los comandos de
plantillas y las plantillas remotas quedan para después.

## 2. Motivación

| Problema | Daño real |
|---|---|
| La estructura del documento estaba en código Go | Cambiar un título, una columna o el checklist exigía recompilar y liberar |
| Título, prefijo de búsqueda y secciones a conservar estaban fijos en los flujos | Un equipo con otra convención no podía usar draco sin forkearlo |
| «Revisión de Tarea» y «Novedad::» fijos en el código | Otro equipo de QA con otras palabras no podía usar el consolidado |
| La fecha del consolidado salía en inglés («21 Sep 2026») | Un documento en español con el mes en inglés |

## 3. Diseño técnico

- **`document.LoadTemplate(tipo, carpetas...)`:** parte de la plantilla
  incluida (`embed.FS`, `internal/document/templates/{deploy,qa}`) y la
  personaliza con la primera carpeta que exista: `template.yaml` reemplaza
  los campos que trae y los `*.md.tmpl` se leen encima, así que un
  `{{define "checklist"}}` reemplaza solo ese `{{block}}` y un `main.md.tmpl`
  reemplaza el documento completo. Una plantilla con error de sintaxis falla
  al cargar; un campo inexistente falla al generar, nunca publica un
  documento roto.
- **Manifiesto:** `title` (plantilla del título), `prefix` y `preserve`. El
  título generado debe empezar con el prefijo: si no, las búsquedas del
  documento existente fallarían y aparecerían duplicados; es un error.
- **Valores ya escapados:** el builder arma un *view* con todo escapado como
  Markdown (`mdText`, `mdURL`, tarjetas, ✔/✖/?) y la plantilla solo los
  coloca; la única función disponible es `join`. Probado con una plantilla
  del usuario que no escapa: un `|` en un nombre de archivo no rompe la tabla.
- **Resolución en los flujos:** carpeta explícita (`Request.Template`, el
  futuro `--template`), `.draco/templates/<tipo>/` del repo (en `deploy`, la
  ruta del backend del proyecto) y `~/.config/draco/templates/<tipo>/`
  (`config.TemplatesDir`). `qa` carga además el prefijo de la plantilla de
  despliegue, que es con lo que reconoce el enlace de una tarea a su
  documento.
- **Los flujos toman del manifiesto** el título (`document.DeployTitle`,
  `document.QATitle`), el prefijo (búsquedas, ubicaciones, enlace) y las
  secciones a conservar. `deploy.Draft.Preserved` pasa a `NotPreserved`,
  como en `qa`, porque la plantilla puede declarar varias. `DraftCreate`
  devuelve error: una plantilla del usuario puede fallar.
- **Criterio de cierre:** las plantillas incluidas reproducen el documento
  anterior. En los 8 goldens solo cambió la fecha del consolidado (4
  archivos), aprobada aparte: `fecha()` con los meses en español.
- **Convenciones de QA a la config:** `qa_jira.review_prefixes` (por defecto
  «Revisión de Tarea», «Revision de Tarea», «QA») y
  `qa_jira.observation_marker` («Novedad::»). No son del documento sino del
  proceso de QA en Jira; el adaptador recibe el marcador en el esquema
  (`GetObservations`).

## 4. Archivos

- `internal/document/template.go` (nuevo), `templates/deploy/*`,
  `templates/qa/*` (nuevos) — motor y plantillas incluidas.
- `internal/document/builder.go`, `qa_builder.go` — *views* y `Deploy`,
  `DeployTitle`, `QA`, `QATitle`; fecha en español.
- `internal/document/template_test.go`, `helpers_test.go` (nuevos); goldens
  del consolidado (fecha).
- `internal/deploy/deploy.go`, `internal/qa/qa.go`, `conventions.go` —
  plantilla en el `Target`, convenciones desde el manifiesto y la config.
- `internal/config/config.go` — `ReviewPrefixes`, `ObservationMarker`,
  `TemplatesDir`.
- `internal/provider/atlassian/jira_qa.go`, `ports.go` — marcador
  configurable.
- Tests y goldens de `deploy`, `qa`, `atlassian` y `markdown` ajustados.
- `docs/arquitectura/proveedores.md` (§7), `docs/roadmap.md`, `CLAUDE.md`,
  `docs/security/patrones-seguros.md`.

## 5. Cómo verificar

```bash
make test
git diff develop -- '*/testdata/*'     # solo la línea de la fecha
make test pkg=./internal/document run=Template v=1
```

## 6. Pendientes

- Fase 3: flag `--template` y comandos `draco template
  init/list/validate/preview`.
- Plantilla elegida por perfil: actividad 7.
- Plantillas remotas (Git con versión fija o URL con checksum): después de
  v1.0, según demanda.
