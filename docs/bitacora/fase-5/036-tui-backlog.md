# 036 — Backlog en la TUI

**Fecha:** 2026-10-20
**Rama:** feature/tui-backlog → develop
**Fase del roadmap:** 5

## 1. Solicitud

Seguir con la siguiente actividad de la fase 5: Backlog (actividad 3 del
plan de la bitácora 034).

## 2. Motivación

| Problema | Daño real |
|---|---|
| «Backlog técnico» estaba atenuado en el menú | Para ver la deuda técnica había que salir a `draco backlog scan` y leer una lista de texto recortada a 10 hallazgos |
| La resolución de repos, las opciones, el filtro de módulos y el guardado del reporte vivían en `cmd/backlog.go` | La TUI no importa `cmd`: copiarlos daría dos backlogs que analizan distinto |
| `backlog.Scan` corría `git` y `composer` sin `context` | `esc` no podía detener un análisis largo; seguía de fondo |

## 3. Diseño técnico

- **Núcleo a `internal/backlog`** (`targets.go`, `report.go`), con los mismos
  algoritmos que tenía `cmd`:
  - `Targets(proj, repo, path)`: la carpeta, un repo del proyecto, todos los
    que tienen ruta o la carpeta actual. Un repo sin ruta es `ErrNoRepoPath`.
  - `OptionsFor(rc, since)`: la config del repo con el período encima
    (`90d`, `12w`, `3m` → `git log --since`).
  - `ApplyModules(rep, modules)`: filtra, o `ErrNoModules` si el repo no tiene
    `modules`.
  - `SplitModules`, `ReportName`, `SaveReport`, `ModuleSummary`.
- **`cmd/backlog.go`** las usa sin cambiar su salida. Se quedan ahí:
  - los mensajes que nombran flags: `--repo` inválido se valida antes, y
    `ErrNoRepoPath`/`ErrNoModules` se explican con `--path` y `-m`;
  - lo propio del texto: `-o`, `--limit`, la impresión.

  Los tests de las funciones movidas se movieron con ellas; en `cmd` queda
  el de los mensajes.
- **`Scan(ctx, dir, opts)`**: `git` y `composer` corren con
  `exec.CommandContext`. El `repo` interno guarda el `context` del análisis:
  vive lo que dura un `Scan` y evita pasarlo a cada detector. Si se cancela,
  `Scan` devuelve `ctx.Err()`, nunca un reporte con detectores «omitidos»
  por la cancelación ni un «no es un repositorio git». `draco backlog scan`
  usa `a.context()`: `ctrl+c` también lo corta.
- **Pantallas** (`internal/tui/backlog.go`, `findings.go`):
  1. **Formulario en dos páginas** (`newPagedForm`): proyecto, repos y
     carpeta; luego período, módulos y dependencias. En una sola página no
     cabía en 24 líneas: la pregunta de dependencias quedaba fuera de la
     vista. Funciona sin config: analiza la carpeta escrita o la actual.
  2. **Progreso:** `Deps.Scan` (el `scanRepo` de `cmd`) y `ApplyModules` por
     repo, uno tras otro en un `tea.Cmd`. Un repo que falla se marca `✗` y
     se sigue con el otro. `esc` cancela el `context`. Al terminar, el
     comando equivalente con `-l 0`.
  3. **Hallazgos:** pestaña por repo, filas de una línea (severidad en su
     color, puntos, módulo, título) con filtro por texto y detalle (al lado
     desde 96 columnas, debajo en 80). Arriba, los detectores omitidos y el
     resumen. `s` guarda el reporte completo de cada repo donde lo deja `-o`
     sin ruta. Sin límite: la lista se desplaza.
- **`window`** (en `filter.go`): la cuenta de qué filas de una lista se ven
  alrededor del cursor, que repetían Commits y las filas de documentos y
  ahora usa también Hallazgos.

## 4. Archivos

- `internal/backlog/targets.go`, `report.go` — nuevos: el núcleo que estaba
  en `cmd`.
- `internal/backlog/targets_test.go` — nuevo: sus tests (los de `cmd`, más
  `ApplyModules`).
- `internal/backlog/scan.go`, `repo.go` — `Scan` con `context`.
- `internal/backlog/backlog_test.go` — repo de prueba en un helper y
  `TestScanStopsWhenCanceled`.
- `cmd/backlog.go`, `cmd/backlog_test.go` — usan el núcleo; tests de
  mensajes con flags.
- `cmd/visual.go`, `internal/tui/tui.go` — `Deps.Scan`.
- `internal/tui/backlog.go`, `findings.go`, `backlog_test.go` — nuevos:
  formulario, progreso, hallazgos y sus tests.
- `internal/tui/generate.go` — `newPagedForm`.
- `internal/tui/filter.go`, `commits.go`, `locations.go` — `window`.
- `internal/tui/home.go`, `app_test.go` — «Backlog técnico» con pantalla.
- `docs/arquitectura/tui.md`, `docs/roadmap.md`, `README.md`, `CLAUDE.md` —
  estado y detalle.
- `docs/bitacora/fase-5/036-tui-backlog.md`, índice.

## 5. Cómo verificar

```bash
make test pkg=./internal/backlog
make test pkg=./internal/tui run=Backlog v=1
make test pkg=./cmd run=Backlog
make check
make run            # Backlog técnico sobre un proyecto con clones reales
```

## 6. Pendientes

- Pruebas manuales del autor: Backlog en Windows Terminal y Linux/WSL sobre
  repos reales, con `esc` durante un análisis largo y `--deps`.
- Los selectores de proyecto (Generar y Backlog) muestran todas las opciones
  a la vez: con muchos proyectos, la primera página del formulario no cabría
  en 24 líneas.
