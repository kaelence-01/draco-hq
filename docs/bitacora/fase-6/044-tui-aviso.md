# 044 — Aviso de versión nueva en la TUI

**Fecha:** 2026-10-21
**Rama:** feature/tui-aviso → develop
**Fase del roadmap:** 6

## 1. Solicitud

Seguir con la actividad 4 del plan de la fase 6 (bitácora 041): el aviso de
versión nueva en el modo visual.

## 2. Motivación

| Problema | Daño real |
|---|---|
| `draco` sin argumentos omite el aviso de texto (`checksUpdates`), y la TUI no lo mostraba | Quien usa sobre todo el modo visual nunca se enteraba de una versión nueva |
| Las condiciones del aviso estaban dentro de `updateNotice` | Un aviso para la TUI las habría repetido, y los dos modos podían terminar avisando distinto |

## 3. Diseño técnico

- **`updatesOff`** (`cmd/startup.go`): versión `dev` o
  `DRACO_NO_UPDATE_CHECK`. La usan el aviso de texto (`updateNotice`) y el
  de la TUI.
- **`latestVersion(ctx)`**: lo que la TUI recibe como `Deps.CheckUpdate`.
  Usa la misma caché que el aviso de texto (`version_check.json`, 24 h):
  - al día, responde sin red;
  - vencida, consulta GitHub con 3 s de límite y, ante un error, usa la
    caché.

  A diferencia de `updateNotice`, puede esperar: la TUI pregunta en
  segundo plano.
- **TUI:** `app.Init` lanza la consulta en un `tea.Cmd`. El resultado
  (`newVersionMsg`) queda en el estado y se ve en dos lugares:
  - la cabecera agrega `· v0.2.3 disponible` junto a la versión actual, del
    lado que no se recorta en terminales angostas;
  - la principal dice «Nueva versión v0.2.3 (tienes v0.2.2): sal con q y
    corre draco update».

  Hasta que responde, no se muestra nada. Sin `CheckUpdate` (los tests
  existentes), `Init` no lanza nada.
- **No se actualiza desde la TUI** (decisión del plan): `draco update`
  reemplaza el binario en uso y escribe su progreso en la terminal.

## 4. Archivos

- `cmd/startup.go` — `updatesOff` y `latestVersion`; `updateNotice` usa
  `updatesOff`.
- `cmd/visual.go` — `CheckUpdate: latestVersion` en `tui.Deps`.
- `cmd/update_test.go` — la TUI recibe la versión nueva; con la caché al
  día no se consulta GitHub (un servidor que falla el test si recibe una
  petición); `DRACO_NO_UPDATE_CHECK` y `dev` lo apagan.
- `internal/tui/tui.go` — `Deps.CheckUpdate` y `state.newVersion`.
- `internal/tui/app.go` — `Init`, `newVersionMsg` y la cabecera.
- `internal/tui/home.go` — la línea de la principal.
- `internal/tui/app_test.go` — aviso con y sin versión nueva, y sin
  consulta.
- `docs/arquitectura/tui.md`, `docs/roadmap.md`, `README.md` — estado y
  diseño ajustado a lo implementado.
- `docs/bitacora/fase-6/044-tui-aviso.md`, índice.

## 5. Cómo verificar

```bash
make test pkg=./cmd run='VisualMode|StartShows|ChecksUpdates' v=1
make test pkg=./internal/tui run=NewVersion v=1
make check
```

Con `go run` la versión es `dev` y no hay aviso; se ve con el binario
instalado cuando haya un release más nuevo.

## 6. Pendientes

- Prueba manual del autor: con `v0.2.1` instalado y `v0.2.2` publicado,
  `draco` muestra el aviso en la cabecera y en la principal.
- Actividad 5a: la barra de comandos (1).
