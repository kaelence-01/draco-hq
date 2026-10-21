# 054 — La regla del proyecto en `config`

**Fecha:** 2026-10-21
**Rama:** feature/regla-proyecto → develop
**Fase del roadmap:** 6

## 1. Solicitud

El autor pidió borrar la rama local `feature/tui-proyectos`, pero solo si
se podía verificar que estaba fusionada sin perder nada. No se podía: tenía
código que no está en `develop`. Eligió rescatarlo antes de borrarla.

## 2. Motivación

| Problema | Daño real |
|---|---|
| La rama local apunta a un borrador de la actividad 030 (`0728146`); el PR #33 fusionó una versión rehecha (`3709787`) sobre el mismo padre | Git no la ve fusionada, y borrarla perdía lo que solo estaba en el borrador |
| Lo único del borrador que no llegó a `develop` es `ProjectConfig.Complete`: al menos una ruta, y el nombre del repo solo junto a su ruta, por defecto el de la carpeta | La regla vive hoy copiada en `draco project add` y en el formulario de la TUI; una corrección en uno no llega al otro (norma de la TUI: «lo que falte se extrae, no se copia») |

## 3. Diseño técnico

- **`ProjectConfig.Complete` y `ErrNoProjectPaths`** en `config`, como en
  el borrador, con su test (`TestProjectComplete`, también del borrador,
  movido a `config_test.go`).
- **`config.DefaultRepoName`:** el nombre por defecto sale aparte, porque
  `draco project add` lo muestra como valor sugerido al preguntar el
  nombre del repo; así tampoco se duplica `filepath.Base`.
- **Llamadores:** `draco project add` y el formulario de proyecto de la TUI
  llaman a `Complete` en lugar de repetir la regla.
- **Mensaje de la TUI:** «El proyecto no se guardó: sin rutas de backend ni
  de frontend» (antes «Sin rutas de backend ni de frontend: el proyecto no
  se guardó»). El del modo comando no cambia. La guía de usuario no cita
  ninguno de los dos, así que no cambia.
- **No se rescata** `github.com/sahilm/fuzzy` como dependencia indirecta en
  `go.mod`: el borrador la arrastraba por una versión anterior de las
  pantallas, y hoy nada la importa (`go mod tidy` no la pide).
- **La rama `feature/tui-proyectos` se borra después de fusionar esta
  actividad**: hasta entonces, su código solo existe en la máquina del
  autor.

## 4. Archivos

- `internal/config/config.go`, `config_test.go` — `Complete`,
  `ErrNoProjectPaths`, `DefaultRepoName` y su test.
- `cmd/project.go` — usa `Complete` y `DefaultRepoName`.
- `internal/tui/projectform.go`, `projects_test.go` — usa `Complete`;
  mensaje.
- `docs/bitacora/fase-6/054-regla-proyecto.md`, índice.

## 5. Cómo verificar

```bash
make check
make test pkg=./internal/config run=ProjectComplete
make test pkg=./cmd run=Project
make test pkg=./internal/tui run=Project
git diff 3709787 0728146 -- internal/config/   # lo que solo tenía el borrador
```
