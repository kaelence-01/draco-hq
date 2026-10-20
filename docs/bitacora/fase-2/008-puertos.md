# 008 — Puertos y Atlassian como adaptador (3a)

**Fecha:** 2026-10-19
**Rama:** feature/puertos → develop
**Fase del roadmap:** 2 (proveedores, actividad 3a)

## 1. Solicitud

Actividad 3 de la secuencia de proveedores: modelo neutro, puertos y Atlassian
como adaptador. El autor aprobó dividirla en dos PRs (3a: puertos, `deploy` y
`fetch`; 3b: QA y cambios) y agregar `context.Context` ahora.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Los flujos usan `atlassian.Page`, `atlassian.Issue`, space IDs y páginas padre | Ninguna otra plataforma puede implementar lo que un flujo pide |
| Sin `context` en el cliente ni en los pasos | La TUI no podría cancelar una consulta lenta; agregarlo después cambiaría las firmas del comando y la TUI ya escritos |
| El formato del `.txt` exportado vivía en el adaptador y decía «Confluence» fijo | El texto exportado de otra plataforma diría que viene de Confluence |

## 3. Diseño técnico

- **`internal/provider`**: modelo (`Issue`, `User`, `DocRef`, `Doc`,
  `Locations`, `Exported`) y puertos `Tracker` y `DocStore`.
- **`internal/atlassian` → `internal/provider/atlassian`** (mismo nombre de
  paquete). `ports.go` implementa los puertos sobre el cliente existente:
  `NewTracker(c)`, `NewDocStore(c, space)`. El adaptador esconde spaces y
  páginas padre: `Create` recibe el documento junto al que va el nuevo y
  resuelve space y padre con `GetPage`, como hacía el flujo.
- **Convenciones fuera del adaptador:** `FindDeployDocByIssue` y
  `FindLastDeployDoc` tenían «Documento de Despliegue» fijo; ahora son
  `FindPageByTitleKey(prefix, key)` y `FindRecentPages(prefix, author)` y el
  flujo pasa `document.DeployTitlePrefix`, la misma constante que usa
  `BuildTitle`. Una sola fuente del prefijo.
- **Transición:** el cuerpo que cruza `DocStore` es el formato nativo (ADF)
  hasta la actividad 4, y la preservación de «A considerar» sigue en el flujo.
  Es lo que permite el criterio «documento idéntico».
- **`context.Context`** en `Get`/`Post`/`Put` (`NewRequestWithContext`), en
  todos los métodos del cliente y en los pasos de las `Session` de `deploy`,
  `fetch` y `qa`. Excepción: `deploy.ReadCommits`, que corre git local; su
  `context` llega con `ChangeSource` en la 3b.
- **`fetch`**: el formato del `.txt` (`BuildIssueTxt`) pasa al flujo
  (`issueText`) con salida idéntica; «Fuente: Confluence» sale de
  `Exported.Source`, que pone el adaptador. `ErrNoPages` ya no nombra
  Confluence.
- **`qa`** solo recibe `context` (sigue con su interfaz propia sobre el
  cliente); pasa a puertos en la 3b.
- **Tests:** los dobles embeben el puerto y sobrescriben solo lo que el flujo
  usa (si el flujo llamara a otro método, el test entra en pánico). Lo que
  pasó del flujo al adaptador (space y padre al crear, filtro por prefijo y
  autor, exportación a texto) se prueba con `httptest`, sin red.

## 4. Archivos

- `internal/provider/provider.go` — modelo y puertos (nuevo).
- `internal/provider/atlassian/` — paquete movido; `ports.go` y
  `ports_test.go` nuevos; `context` en el cliente; búsquedas por prefijo
  parametrizadas; `BuildIssueTxt` fuera.
- `internal/deploy/deploy.go`, `deploy_test.go` — sobre `Tracker` y
  `DocStore`, con `context`; `Locations.Pages` pasa a `Locations.Docs`.
- `internal/fetch/fetch.go`, `fetch_test.go` — sobre `DocStore`; `issueText` y
  sus tests.
- `internal/qa/qa.go`, `qa_test.go`, `golden_test.go` — `context`.
- `internal/document/builder.go` — `DeployTitlePrefix`.
- `internal/document/*_test.go`, `qa_builder.go` — ruta del import.
- `CLAUDE.md`, `docs/arquitectura/proveedores.md` (secciones 3 y 4 con lo
  implementado), `docs/roadmap.md` (3a y 3b).

## 5. Cómo verificar

```bash
go test ./...
git status --porcelain -- '*/testdata/*'   # vacío: goldens sin regenerar
grep -l provider/atlassian internal/deploy/*.go internal/fetch/*.go   # vacío
```

## 6. Pendientes

- 3b: `QATracker`, `qa` y `document` sin tipos de Atlassian, `ChangeSource`.
- `document.QADoc` todavía usa `atlassian.QAIssue`.
