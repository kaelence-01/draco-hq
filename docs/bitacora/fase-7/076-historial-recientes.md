# 076 — Historial de documentos y Recientes en la principal

**Fecha:** 2026-11-09
**Rama:** feature/historial-recientes → develop
**Fase del roadmap:** 7 (mejoras sobre la fase cerrada)

## 1. Solicitud

El autor vio espacio libre bajo el menú de la principal y propuso «un tipo
dashboard de bitácora»: los últimos 5 a 10 despliegues con un enlace a cada
uno, mapeados por la tarea (`OPS-7`). Si alguno quedó mal, ir directo a él
para borrarlo o actualizarlo, porque la tarea cambió y el documento debe
cambiar con ella.

Tras iterar, el autor decidió:

- historial **local**, no consultado a Confluence;
- **borrar en una segunda actividad**;
- historial **global**, con la opción de filtrar por proyecto **en la misma
  vista**, sin ir a otra pantalla;
- Historial como opción **7** del menú, para no mover los números que ya
  usa, y todo en una sola actividad.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Para volver a un documento había que recordar la tarea y buscarla | Corregir un documento de hace días obligaba a ir a Confluence o a Jira a buscarlo |
| Un enlace fallido solo se avisaba en el momento de publicar | Si se pasaba por alto, la tarea quedaba sin documentación para el consolidado QA semanas después, sin rastro en draco |
| El espacio bajo el menú de la principal estaba vacío en terminales anchas | Lo que más se repite (volver a un documento propio) estaba a varias pantallas |

## 3. Diseño técnico

- **Fuente: historial local** (decisión del autor). Se descartó consultar a
  Confluence (`Locations` con el autor):
  - costaba red cada vez que se abría la principal;
  - ordena por fecha de creación, así que no ve las actualizaciones;
  - no sabe si el enlace desde la tarea falló.

  Costo de lo local: solo ve lo publicado con draco desde este equipo.
- **`internal/history`**, paquete nuevo sin red:
  - `Entry` guarda tarea, título, dirección, id, destino, proyecto, perfil,
    creado o actualizado, fecha, `LinkFailed` y `NotPreserved`.
  - `Record` deja **una entrada por documento** (mismo destino e id): la
    sube al primer lugar y conserva las 200 más recientes. Así, cinco
    actualizaciones de la misma tarea no tapan las demás.
  - Escribe en un archivo temporal que después se renombra: un corte nunca
    deja el historial a medias.
  - Un historial dañado **no se sobrescribe**: `Load` lo informa con qué
    hacer.
  - `Deploy(target, draft, res, …)` arma la entrada para que el modo comando
    y la TUI no la armen cada uno por su lado.
- **Dónde vive:** `history.json` junto a `config.yaml`, como
  `version_check.json`. En la propuesta inicial iba en `~/.local/state`; se
  cambió para seguir la convención que el repo ya tenía, y porque
  desinstalar ya dice que se borre `~/.config/draco/`.
- **Quién escribe:**
  - `draco g` registra después de `Publish`, nunca con `--dry-run`.
  - La TUI registra en `genRun.publish`.
  - El consolidado QA no entra: el pedido era sobre despliegues.
  - Si guardar falla, el documento ya está publicado, así que es un `⚠` (en
    la TUI, `docResult.historyErr`, sobre el resultado). Es un error
    descartado cuyo efecto aparecería después: por eso se reporta.
- **`draco history [-p] [-n]`** (alias `hist`): una entrada por documento,
  con fecha absoluta (es para scripts) y el `⚠` del enlace. Es el comando
  equivalente que la norma §10.8 pide para el panel.
- **TUI** (`history.go`): un `recentList` compartido entre la caja y la
  pantalla, con lista, pestañas, detalle y acciones.
  - **Pestañas por proyecto:** `[Todos]` y un proyecto por cada uno que
    aparece en el historial (no en la config: uno borrado sigue
    filtrable). Se cambian con `←→`. Con un solo proyecto no hay pestañas.
    Así se cumple «elegir entre global o específico ahí mismo».
  - **Fecha como se dice:** `hoy 11:00`, `ayer 15:30`, `04/11`,
    `01/12/2025`.
  - **`enter` abre el documento** con `Deps.OpenURL`. En `cmd` es
    `openURL`: `LookPath` antes de ejecutar (P-005); en WSL, `wslview` o
    `explorer.exe`; `xdg-open`, `open` o `rundll32` en los demás. Solo
    acepta `http(s)://` o una ruta absoluta, porque el historial es texto
    que cualquiera puede editar. Los tests lo reemplazan y nunca abren
    nada.
  - **`u` actualiza:** abre Generar con la tarea y su proyecto
    (`openGenerate`, el mismo camino que la barra), y el progreso encuentra
    el documento y propone actualizarlo, lo que también lo vuelve a enlazar.
    No se reimplementa nada del flujo. Si el proyecto ya no está en la
    config, Generar abre con la tarea y lo avisa.
  - **`e` exporta:** abre Exportar con la tarea.
- **Principal:**
  - La caja Recientes (10 filas) va bajo el menú, del ancho del menú, desde
    96 columnas.
  - Se prueba con cada forma del menú (con espacios, con descripciones,
    sola) y se necesitan al menos tres filas. Si no cabe con ninguna, la
    principal queda como antes. En 80 columnas no hay caja: se llega por
    Historial.
  - `tab` le pasa las teclas a la caja. El panel de la derecha muestra
    entonces el documento elegido, con su comando (`draco history -p …`).
- **Las teclas dependen de lo que cupo.** La principal solo sabe si la caja
  entró al dibujarse, y el marco armaba el pie antes que el cuerpo. Por eso
  el primer dibujo no ofrecía `tab`.
  - El marco ahora vuelve a pedir el pie después del cuerpo, si ocupa el
    mismo alto. Es un cambio general: sirve a cualquier pantalla cuyas
    teclas dependan del espacio.
  - El marco también llama a `init` de la principal al volver con
    `toHome`, como ya hacía con `back`. Así, lo recién publicado aparece al
    volver al inicio.
- **En 80 columnas el pie no cabía.** `e exportar` pasó a la ayuda completa
  (`?`); el detalle ya dice «enter lo abre · u lo actualiza · e lo exporta».
- **Tests que llegaban a Configuración con `↑` desde la primera opción**
  (la última del menú) ahora usan `6`. Los de las teclas numéricas incluyen
  el `7`.
- **Borrar** quedó para la 077, por decisión del autor. Borrar la página no
  basta: hay que quitar el enlace desde la tarea, o `HasDocumentLink` lo
  seguiría viendo y QA daría la tarea por documentada. El diseño está en el
  roadmap.

## 4. Archivos

- `internal/history/history.go` — nuevo: `Entry`, `Deploy`, `Path`, `Load`, `Record` (atómico, no sobrescribe uno dañado), `Add`, `Of`, `Projects`.
- `internal/history/history_test.go` — nuevo: orden, una entrada por documento, destinos distintos, límite, historial dañado, filtros, entrada de un despliegue.
- `cmd/history.go` — nuevo: `draco history`/`hist` con `-p` y `-n`, su ayuda, `historyPath` y `App.record`.
- `cmd/history_test.go` — nuevo: crear y actualizar dejan una entrada, `--dry-run` no registra, enlace fallido, historial dañado no frena la publicación, filtro, límite, vacío.
- `cmd/openurl.go` — nuevo: `openURL` (P-005, solo web o ruta absoluta).
- `cmd/generate.go` — registra tras publicar.
- `cmd/generate_test.go` — `fakeEnvWith` lleva el historial a una carpeta temporal.
- `cmd/visual.go` — `HistoryPath` y `OpenURL` a la TUI.
- `internal/tui/history.go` — nuevo: `recentList`, `historyScreen`, `when`.
- `internal/tui/history_test.go` — nuevo: caja bajo el menú, `tab` y panel, `←→`, abrir, actualizar (y proyecto que ya no existe), exportar, sin alto, 80 columnas, vacío y dañado, pantalla en 110, 80 y 60 columnas, barra, publicar registra, fallo al guardar, `when`.
- `internal/tui/home.go` — opción 7 Historial, foco con `tab`, caja Recientes, panel del documento, `init`.
- `internal/tui/tui.go` — `Deps.HistoryPath` y `Deps.OpenURL`; `state.recent`, `loadRecent`, `record`, `now`.
- `internal/tui/app.go` — el pie se arma de nuevo tras el cuerpo; `init` al volver al inicio.
- `internal/tui/docrun.go`, `progress.go`, `publish.go` — `genRun` registra; `historyErr` en el resultado.
- `internal/tui/barrun.go` — `/history -p` abre Historial en esa pestaña.
- `internal/tui/app_test.go`, `home_test.go`, `settings_test.go`, `barrun_test.go` — el menú con 7 opciones.
- `docs/arquitectura/tui.md` — mapa de archivos, `Deps`, principal y sección Historial.
- `docs/guia-usuario.md` — Recientes, teclas, Historial en los dos modos y desinstalar.
- `docs/roadmap.md` — sección «historial de documentos»: 076 ✅, 077 (borrar) ⬜; `qa-listas` pasa a 078.
- `docs/bitacora/fase-7/076-historial-recientes.md`, índice.

## 5. Cómo verificar

```bash
gofmt -l ./cmd ./internal && go vet ./...                      # corridos: limpio
go test ./cmd ./internal/tui ./internal/history                # corridos: ok
make check                                                     # lo corre el autor
draco g -i OPS-7 -b <hash>   y después   draco history         # lo corre el autor
draco    # Recientes bajo el menú, tab, ←→, enter, u, e; y 7 Historial: lo corre el autor
```

La principal en 120×36, con y sin foco en Recientes, se revisó con un test
temporal.

## 6. Pendientes

- **077:** borrar un documento desde el historial, quitando también el
  enlace desde la tarea.
- Abrir en el navegador no se probó en una terminal real (WSL con `wslview`
  o `explorer.exe`, Linux, macOS, Windows). En WSL sin `wslview`, un
  documento Markdown se abre con `explorer.exe` y una ruta de Linux, que
  puede no abrirse.
- El historial solo conoce lo publicado con draco desde este equipo.
  Sincronizarlo con Confluence queda para cuando haga falta.
- **078 (opcional):** módulo y sprint de QA elegidos de listas.
