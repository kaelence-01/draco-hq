# 032 — Generar en la TUI (2): vista previa, ubicación y publicación

**Fecha:** 2026-10-20
**Rama:** feature/tui-publicar → develop
**Fase del roadmap:** 4

## 1. Solicitud

Seguir con la 4b de la fase 4 («sigue con la 4b»). Es la segunda parte de
Generar, desde la pantalla de progreso de la 031: vista previa, documento
existente, secciones no conservadas, ubicación con búsqueda, confirmación y
resultado con `LinkErr`.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Generar en la TUI terminaba en los archivos leídos | Para publicar había que volver al modo comando y copiar el comando que mostraba la pantalla |
| Sin vista previa, actualizar era a ciegas | Una actualización que pierde «A considerar» (lo que el equipo escribió a mano) se publicaba sin aviso visible |
| Salir mientras se publica | El documento quedaría creado sin el enlace desde la tarea, y semanas después QA lo marcaría sin documentación |
| Los errores largos se cortaban en el borde | En la prueba en el pty, «dial tcp 127.» perdía justo la causa del error |

## 3. Diseño técnico

**Pantallas nuevas**, una por paso que el usuario revisa:

- **Vista previa** (`preview.go`):
  - Al crear, `DraftCreate` (puro). Al actualizar, `DraftUpdate` en un
    `tea.Cmd`, porque lee el documento actual del destino.
  - Dos pestañas: Documento (el Markdown en un `viewport` de Bubbles con
    `SoftWrap`, para que las tablas anchas no pierdan columnas) y Archivos.
  - Al actualizar dice si lo escrito a mano se conserva. Con `NotPreserved`
    avisa, y `enter` no sigue hasta que el usuario acepta con `s`, como el
    «¿Continuar de todas formas?» de `draco g`, que por defecto dice no.
  - Salir de aquí no publica nada: reemplaza a `--dry-run`.
- **Ubicación** (`locations.go`), solo al crear:
  - `Session.Locations` en un `tea.Cmd`, con los avisos `TeamWide` y
    `OwnerUnknown`.
  - Filtro con `/` y, por documento, su título y su URL.
  - `ErrNoLocations` se explica con la misma sugerencia que el modo
    comando: un perfil Markdown no lo necesita.
- **Publicar** (`publish.go`):
  - Título, destino, acción (crear junto a… / actualizar…) y enlace desde la
    tarea; `enter` publica.
  - Mientras publica, la pantalla captura `esc`; `ctrl+c` sigue saliendo.
  - El resultado muestra la URL y, si falló el enlace, `LinkErr` con qué
    hacer (el mismo texto que `draco g`). Un error deja reintentar.
  - Muestra el comando equivalente, que antes estaba en la pantalla de
    progreso. Al terminar, `enter` o `esc` vuelven a la principal.

**Encadenamiento:** la pantalla de progreso ofrece `enter` (vista previa) al
terminar. Las pantallas siguientes comparten un `genRun` (estado, perfil,
`Session`, si se actualiza y el comando equivalente). No hay estado global ni
segunda `Session`.

**Marco:**

- `toHome` cierra todas las pantallas hasta la principal, llamando a su
  `close()`.
- El pie ya no ofrece `esc volver` ni `? ayuda` cuando la pantalla los
  captura: mientras se publica, ofrecer `esc` sería falso.

**Filtro compartido (`filter.go`):** Proyectos y Ubicación tenían el mismo
código copiado para el filtro (escribir, `enter`, `esc`, borrar). Ahora es un
tipo `filter` con `captures`, `handle`, `matches` y `view`, y el
comportamiento de Proyectos no cambió (sus tests pasan igual).

**Texto que no cabe:** progreso, ubicación, publicar y el error de la vista
previa ya no recortan con `MaxWidth`: el texto pasa a la línea siguiente.
Solo se recortan las filas de las listas, que son de una línea. Queda como
norma en `tui.md` §10, `CLAUDE.md` y `estandares-desarrollo`. El test
`TestLongErrorsWrap` falla con el recorte anterior; se comprobó.

**Tests** (`publish_test.go`), con el adaptador Markdown real:

- La vista previa no publica nada, y `tab` muestra los archivos.
- Crear de punta a punta: la ubicación, la ruta completa, el comando
  equivalente, el archivo publicado con los archivos de los commits, el
  enlace desde la tarea y la vuelta al inicio.
- Mientras publica, `esc` no sale y el pie no ofrece `esc`.
- `LinkErr` se reporta junto al documento creado.
- Actualizar conserva lo escrito a mano en «A considerar» y trae los
  archivos nuevos.
- Actualizar sin «A considerar» exige `s` antes de seguir.
- «Crear otro» pasa por Ubicación.
- El filtro de Ubicación funciona.
- Sin ubicaciones, se explica qué hacer.

## 4. Archivos

- `internal/tui/preview.go`, `locations.go`, `publish.go`: nuevas, las tres
  pantallas.
- `internal/tui/filter.go`: nuevo, el filtro compartido.
- `internal/tui/progress.go`: `enter` abre la vista previa; `readFiles` y
  `plural` («1 archivo»).
- `internal/tui/projects.go`: usa `filter`.
- `internal/tui/app.go`: `toHome`, `pop`, `captured`; el pie respeta las
  teclas capturadas.
- `internal/tui/publish_test.go`: nuevo. `generate_test.go`: dobles con
  enlace, error de enlace y destino reemplazable, y `TestLongErrorsWrap`.
  `app_test.go`: el pie con teclas capturadas.
- `docs/arquitectura/tui.md`: estado, paquete, `toHome`, pasos 5 a 7 de
  Generar, decisiones y la norma 9 (texto que no cabe).
- `CLAUDE.md`, `.claude/skills/estandares-desarrollo/SKILL.md`: la misma
  norma.
- `docs/roadmap.md`: 4b ✅. `README.md`: el estado.
- `docs/bitacora/fase-4/032-tui-publicar.md`, índice.

## 5. Cómo verificar

```bash
make check
go test -race -count=2 ./...
make run        # Generar: tarea, commits, enter, vista previa, ubicación, publicar
```

En un pty de Linux (WSL), con `HOME` temporal, un repo git temporal y un
perfil Markdown, se recorrió Generar con git real: el selector mostró los
commits con la rama `master`. Con `base_url` en `127.0.0.1:1`, para que no
saliera nada a la red, la búsqueda de la tarea falló y el error se mostró.
Ahí se vio el corte en el borde que esta actividad corrige.

## 6. Pendientes

- **Prueba manual del autor contra su Jira y su Confluence**, que es parte
  del criterio de cierre de la 4b: crear y actualizar un documento real,
  ubicación con `TeamWide`, enlace en la tarea. También en una carpeta
  Markdown con su Jira: la publicación de punta a punta solo se probó con
  dobles, porque no hay tracker sin red.
- Prueba en Windows Terminal: el `viewport`, los bordes y el spinner.
- Lo siguiente es la actividad 5: cierre de la fase 4 y `v0.2.0`.
