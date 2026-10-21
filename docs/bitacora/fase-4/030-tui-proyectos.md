# 030 — Proyectos en la TUI y normas del modo visual

**Fecha:** 2026-10-20
**Rama:** feature/tui-proyectos → develop
**Fase del roadmap:** 4

## 1. Solicitud

Revisar la fase 4 hecha hasta aquí (actividades 1 y 2) contra el diseño y la
maqueta del modo visual. El autor aprobó la base y pidió seguir con la
actividad 3 (Proyectos) en esta misma rama, sumando lo que la revisión
encontró: teclas que una pantalla pueda capturar, el panel lateral y la ruta
de la maqueta. También pidió dejarlo escrito como **norma de desarrollo**,
para que cualquier otra sesión o agente trabaje sobre los mismos parámetros.

## 2. Motivación

| Problema | Daño real |
|---|---|
| El marco atendía `?` y `esc` antes que la pantalla | En el filtro o en un formulario, escribir `?` abría la ayuda y `esc` cerraba la pantalla en vez de quitar el filtro |
| Al cerrar una pantalla no se le avisaba | El diseño cancela con `esc` el paso en curso, pero no había por dónde hacerlo. Generar (4a) habría dejado pasos corriendo |
| La maqueta tenía un panel con la descripción y el comando equivalente, y una ruta bajo la cabecera, pero el diseño no los incluía | Se perdía lo que enseña que los dos modos son uno, y con pantallas apiladas no se sabía dónde estaba uno |
| Proyectos solo se manejaba con `draco project` o editando el YAML | La actividad 3 de la fase no estaba hecha |
| Las reglas de la TUI estaban repartidas en el diseño, sin una lista obligatoria | Cada sesión o agente habría escrito sus pantallas a su manera |

## 3. Diseño técnico

**Marco (`app.go`):**

- `screen` suma `title()` para la **ruta** (`Inicio › Proyectos › Nuevo
  proyecto`), que va debajo de la cabecera fuera de la principal.
- Tres partes opcionales que el marco consulta: `captures(k)` (la pantalla se
  queda con `?` o `esc`; `ctrl+c` nunca), `init()` (al abrirse; el
  formulario de Huh enfoca su primer campo) y `close()` (al cerrarse; ahí
  Generar cancelará su `context`). `close()` llega ahora porque es parte del
  modelo que se fija como norma; su test usa una pantalla de prueba.

**Principal:** cada entrada del menú tiene su descripción y un ejemplo del
comando. Desde 96 columnas, un panel a la derecha muestra la función elegida
y «En modo comando, para scripts y CI: `$ draco g -i OPS-7 -b a1b2c3d`». Si la
función no tiene pantalla, el panel lo dice. Con menos ancho no hay panel y el
menú muestra el comando de cada función sin pantalla, como antes. Proyectos
queda encendido.

**Proyectos (`projects.go`, `projectform.go`):**

- La lista y el detalle (rutas, repos, VCS, space, checklist) están lado a
  lado desde 80 columnas, y uno debajo del otro con menos. Marca el proyecto
  activo y el por defecto.
- `enter` cambia el proyecto activo, solo en memoria. `d` guarda el proyecto
  por defecto. `n` y `e` abren el formulario. `x` elimina, previa
  confirmación con `s`. `/` filtra, y `esc` quita el filtro antes de volver.
- El formulario es de Huh v2 (2.0.3, aprobada en el plan), con el tema
  `ThemeBase16` (colores ANSI) y sin su ayuda propia, porque las teclas van en
  el pie. Aplica las mismas reglas que `draco project add`: el nombre se
  valida y no puede existir; un repo vacío toma el nombre de la carpeta; sin
  rutas no se guarda; el primer proyecto queda por defecto; editar conserva
  checklist y backlog (se parte del proyecto del archivo y solo se cambian
  los campos del formulario).
- **Guardar** pasa por `state.save`: `Deps.LoadFile` → la función pura de
  `config` → `Deps.SaveConfig` → se vuelve a leer la config. El proyecto
  activo se mantiene mientras exista. `cmd` pasa `config.LoadFile` y
  `saveConfig`. Escribir la config local es rápido y va directo en `update`;
  la red y git irán en un `tea.Cmd`.
- **Descartado: `bubbles/list`** (lo pedía el diseño). Trae paginación,
  barra de estado y colores fijos que sobran para unas pocas filas, y su
  filtro es otro campo con cursor parpadeante. La lista es propia, con el
  mismo `▸` del menú.

**Tests:** `send` hace ahora de bucle de Bubble Tea. Pasa el mensaje por
`Update`, ejecuta los comandos que devuelve, abre `Batch` y `Sequence`, y
descarta el que no responde en 20 ms: el parpadeo del cursor de Huh espera
medio segundo, y Huh no permite apagarlo. `fakeFile` guarda en memoria, y su
`LoadConfig` agrega un token como haría `ATLASSIAN_TOKEN`. Cada test que
guarda comprueba que el token no llega al archivo, que es el criterio de
cierre de la actividad.

**Normas:** `docs/arquitectura/tui.md` §10 fija 11 normas obligatorias para
escribir una pantalla: sin lógica de negocio, una pantalla por paso revisado,
teclas y `captures`, pasos en `tea.Cmd` y cancelación en `close()`, guardar
solo con `state.save`, menú con descripción y comando, estilos de `styles.go`
sin emojis y con 80 columnas como mínimo, mensajes y tests. `CLAUDE.md` las
resume en «El modo visual», dentro de las reglas que no se negocian, y la
skill `estandares-desarrollo` las suma a su lista de revisión. Así llegan a
cualquier sesión o agente: `CLAUDE.md` se carga siempre y la skill es
obligatoria al tocar código.

## 4. Archivos

- `internal/tui/app.go`: `title`, `captures`, `init` y `close`; la ruta en la
  cabecera.
- `internal/tui/tui.go`: `Deps.LoadFile` y `Deps.SaveConfig`; `state` guarda
  la config y suma `load` y `save`.
- `internal/tui/home.go`: descripción y ejemplo por función, el panel y
  Proyectos encendido.
- `internal/tui/projects.go`: nuevo, la pantalla Proyectos.
- `internal/tui/projectform.go`: nuevo, el formulario de Huh.
- `internal/tui/styles.go`: `okStyle`, `errStyle` y `boxStyle`.
- `internal/tui/app_test.go`: `send` con comandos, `typeText`, la ruta, el
  panel, `captures` y `close`.
- `internal/tui/projects_test.go`: nuevo, con la lista, el activo, el por
  defecto, el filtro, el borrado, crear, editar, el nombre repetido, sin rutas
  y `esc`.
- `cmd/visual.go`: pasa `LoadFile` y `SaveConfig` a la TUI.
- `go.mod`, `go.sum`: `charm.land/huh/v2` 2.0.3.
- `docs/arquitectura/tui.md`: el modelo con las partes opcionales, el panel,
  Proyectos, los tests, las decisiones y las normas (§10).
- `CLAUDE.md`: «El modo visual» en las reglas; Huh ya en uso.
- `.claude/skills/estandares-desarrollo/SKILL.md`: la lista de revisión del
  modo visual.
- `docs/roadmap.md`: actividad 3 ✅. `README.md`: el estado.
- `docs/bitacora/fase-4/030-tui-proyectos.md`, índice.

## 5. Cómo verificar

```bash
make check
go test -race -count=3 ./internal/tui ./cmd
make run        # en una terminal: ↓ enter (Proyectos), n, /, e, x, d, esc
```

Verificado además en un pty de Linux (WSL) con `script`, con `HOME` en una
carpeta temporal y una config de prueba: se abrió Proyectos, se creó
`reportes` con el formulario, el archivo quedó con `backend_repo:
reportes-api`, se vieron la ruta `Proyectos › Nuevo proyecto` y la
confirmación, y al salir con `q` la terminal dejó la pantalla alternativa.

## 6. Pendientes

- Prueba manual del autor en Windows Terminal (PowerShell) y en Linux/WSL:
  el panel, los bordes del formulario de Huh y los colores. Sigue pendiente
  también la de la 029.
- `bubbles/list` sigue en el diseño para la ubicación con búsqueda de
  Generar (4b). Se decide ahí con el mismo criterio.
