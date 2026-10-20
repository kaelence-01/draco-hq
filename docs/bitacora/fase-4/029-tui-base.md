# 029 — Base de la TUI

**Fecha:** 2026-10-20
**Rama:** feature/tui-base → develop
**Fase del roadmap:** 4

## 1. Solicitud

Seguir con la actividad 2 de la fase 4: la base del modo visual.

## 2. Motivación

| Problema | Daño real |
|---|---|
| `draco` sin argumentos solo mostraba la ayuda | El modo visual, la otra mitad de la visión de draco, no tenía dónde nacer: cada pantalla habría tenido que inventar el marco, la navegación y la salida |
| Sin un modelo común probado | Proyectos y Generar (actividades 3 y 4) habrían resuelto cada una la pila, las teclas globales y la prueba sin terminal a su manera |

## 3. Diseño técnico

Sigue `docs/arquitectura/tui.md`:

- **Dependencias:** `charm.land/bubbletea/v2` 2.1.0, `bubbles/v2` 2.2.1 y
  `lipgloss/v2` 2.0.6. Huh se agrega con Proyectos, que es la primera
  pantalla que la usa. `github.com/charmbracelet/x/ansi` pasa a dependencia
  directa: los tests lo usan para quitar los colores de la vista. Ya era
  dependencia de Lip Gloss, así que no suma código. `go.mod` sigue en
  `go 1.26.1`.
- **Entrada:** `App.Run` sin argumentos llama a `runTUI` solo con `App.tty`;
  sin TTY sigue la ayuda. `runTUI` es una variable del paquete (los tests la
  reemplazan) y pasa a `tui.Run` la entrada original sin el `bufio.Reader`
  (`App.rawIn`), que Bubble Tea necesita en modo crudo. Un error de la TUI
  sale con código 1.
- **`tui.Deps`** solo con lo que esta actividad usa (`LoadConfig`,
  `Version`); el diseño ahora dice qué actividad agrega cada campo.
- **Modelo raíz (`app`):** pila de pantallas con una interfaz mínima
  (`update`, `view` con el tamaño disponible, `keys`). `open(s)` apila y `esc`
  vuelve, salvo en la principal. `ctrl+c` sale siempre; `q` sale solo desde la
  principal, porque en las demás pantallas habrá campos de texto. `?` alterna
  la ayuda completa de `bubbles/help`. El marco tiene una cabecera con la
  versión, el perfil (tracker → destino) y el proyecto activo, y un pie con
  las teclas de la pantalla. Usa la pantalla alternativa, que se restaura al
  salir.
- **Estado compartido (`state`):** se lee la config una vez con `LoadLocal`.
  El proyecto activo arranca en `default_project` y vive en memoria. Un
  problema de la config aparece como aviso en la principal en vez de impedir
  abrirla: no se puede leer, hay varios perfiles sin uno por defecto, o el
  perfil usa Jira o Confluence sin credenciales («corre draco init»).
- **Pantalla principal:** DRACO en letras grandes con `█`, o `draco` si el
  ancho no alcanza. Las seis funciones están en el menú; como ninguna tiene
  pantalla todavía, todas aparecen atenuadas con su comando, y `enter` lo
  recuerda («usa draco g»). Las actividades 3 y 4 encienden Proyectos y
  Generar. El menú da la vuelta (subir desde la primera va a la última).
- **Colores ANSI básicos** (cian, amarillo, gris), que siguen la paleta de la
  terminal, clara u oscura; los temas son de la fase 6.

## 4. Archivos

- `internal/tui/tui.go` — `Deps`, `Run` y el estado compartido.
- `internal/tui/app.go` — modelo raíz: pila, teclas globales, marco.
- `internal/tui/home.go` — pantalla principal.
- `internal/tui/logo.go`, `internal/tui/styles.go` — logo y estilos.
- `internal/tui/app_test.go` — pantalla, menú, salida, ayuda, pila y avisos de
  config, sin terminal.
- `cmd/visual.go` — `runTUI`.
- `cmd/app.go` — `rawIn`; `Run` sin argumentos abre la TUI en una terminal.
- `cmd/app_test.go` — la TUI se abre solo con TTY y su error sale con 1.
- `go.mod`, `go.sum` — Charm v2.
- `docs/arquitectura/tui.md` — estado y campos de `Deps` por actividad.
- `docs/roadmap.md`, `CLAUDE.md`, `README.md` — estado, paquete,
  dependencias y `runTUI`.
- `docs/bitacora/fase-4/029-tui-base.md`, índice.

## 5. Cómo verificar

```bash
make check
go test -race ./internal/tui ./cmd
go run . | head -3                 # sin TTY: la ayuda
make run                           # en una terminal: la TUI; ↑↓, enter, ?, q
```

Verificado además en un pty de Linux (WSL) con `script`: la TUI abre, `↓` y
`enter` muestran «usa draco project», `q` sale con código 0 y la terminal sale
de la pantalla alternativa y recupera el cursor.

## 6. Pendientes

- Prueba manual del autor en Windows Terminal (PowerShell) con el binario y
  en una terminal de Linux o WSL, a la vista: tamaño, logo y colores.
