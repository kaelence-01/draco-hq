# 009 — QA y cambios sobre puertos (3b)

**Fecha:** 2026-10-19
**Rama:** feature/puertos-qa → develop
**Fase del roadmap:** 2 (proveedores, actividad 3b)

## 1. Solicitud

Segunda mitad de la actividad de puertos: la capacidad `QATracker`, `qa` y
`document` sin tipos de Atlassian, y el puerto de cambios (`ChangeSource`) con
los commits recientes para el selector de la TUI.

## 2. Motivación

| Problema | Daño real |
|---|---|
| `qa` llamaba al cliente de Atlassian y conocía el esquema de Jira (`QASchema`) | El consolidado no puede salir de otro tracker ni probarse sin Jira |
| El adaptador decidía convenciones de draco: filtrar «Revisión de Tarea», cruzar claves, ventana de 10 días hábiles, prefijo «Documento de Despliegue» | Un segundo adaptador tendría que copiarlas; cambiarlas obligaría a tocar cada adaptador |
| `document.QADoc` usaba `atlassian.QAIssue` | El builder dependía de un adaptador |
| git se usaba por una función suelta, sin `context` y sin historial | La TUI no puede listar commits ni cancelar un git lento |

## 3. Diseño técnico

- **`provider.QATracker`** amplía `Tracker`: `ReviewTasks` y `AssignedTasks`
  por `QAScope` (sprint + módulo, o Kanban desde una fecha),
  `HasDocumentLink(key, prefix)` y `Observations`. `atlassian.NewQATracker(c,
  qa_jira)` convierte la config al esquema (la conversión salió de `qa`).
- **Convenciones al flujo** (`internal/qa/conventions.go`): `devTasksOnly`,
  `buildReviewMap`, `parseDevTaskKey`, `kanbanWindow`. El adaptador devuelve
  todas las tareas del alcance y el flujo filtra; resultado idéntico. El
  prefijo del enlace lo pasa el flujo (`document.DeployTitlePrefix`).
  «Novedad::» sigue en el adaptador: pasa a la plantilla en la actividad 5.
- **Ubicaciones del consolidado:** `DocStore.SearchTitle` (CQL, todo el
  space), no `Locations`. `Locations` revisa solo las 100 páginas más nuevas;
  en un space con movimiento, un consolidado (uno por sprint) podía no
  aparecer. Se mantiene el comportamiento de gtt. Título desde
  `document.QATitlePrefix`, la misma constante que arma el título.
- **`provider.ChangeSource`** (`ChangedFiles`, `RecentCommits`) e
  `internal/git` → `internal/provider/git` (`git.Local`), con
  `exec.CommandContext` y `LookPath` (P-005). `RecentCommits` lee `git log`
  con un separador de control. `GroupByDirectory` pasa a `deploy`
  (`groupByDirectory`): es armado del documento, no git. Se quitó
  `CommitFiles`, sin uso.
- **«Verifícalo en Bitbucket» → «en tu VCS»** en el mensaje de un hash
  inexistente (pendiente desde 004).
- **Datos de prueba:** los tests movidos de `atlassian` a `qa` usaban claves
  heredadas de gtt (`APP-…`, `ECU-42`); pasan a `OPS-…` / `PAG-…`.
- **Tests:** dobles que embeben `QATracker` y `DocStore`; el esquema se prueba
  en el adaptador (`NewQATracker`); git con un repo temporal local (sin red; se
  salta si no hay git) y el parseo de `git log` como función pura.

## 4. Archivos

- `internal/provider/provider.go` — `QAIssue`, `QAScope`, `QATracker`,
  `Commit`, `ChangeSource`, `DocStore.SearchTitle`.
- `internal/provider/atlassian/jira_qa.go` — sin convenciones de draco; usa
  `provider.QAIssue`; `HasRemoteLink(prefix)`.
- `internal/provider/atlassian/confluence.go` — `SearchPageTitles` (antes
  `FindQAPagesForModule`).
- `internal/provider/atlassian/ports.go`, `ports_test.go` — `QATracker`,
  `SearchTitle`, test del esquema.
- `internal/provider/atlassian/jira_qa_test.go` — tests movidos a `qa`.
- `internal/provider/git/` — movido desde `internal/git`; `Local`,
  `RecentCommits`, tests.
- `internal/qa/qa.go`, `conventions.go`, `conventions_test.go`, `qa_test.go`,
  `golden_test.go`.
- `internal/deploy/deploy.go`, `deploy_test.go` — `ChangeSource`,
  `groupByDirectory`.
- `internal/document/qa_builder.go` (+ tests) — `provider.QAIssue`,
  `QATitlePrefix`.
- `CLAUDE.md`, `docs/arquitectura/proveedores.md`, `docs/roadmap.md`.

## 5. Cómo verificar

```bash
go test ./...
git status --porcelain -- '*/testdata/*'   # vacío: goldens sin regenerar
grep -rl 'provider/atlassian\|provider/git' internal/deploy internal/qa internal/fetch internal/document   # vacío
```

## 6. Pendientes

- Detectar `QATracker` sobre un `Tracker` cualquiera (error centinela cuando
  el tracker no lo soporta): con los perfiles, actividad 7.
- «Novedad::» y «Revisión de Tarea»: a la plantilla, actividad 5.
