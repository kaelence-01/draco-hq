# 053 — `~` en las rutas y space del proyecto con perfil Markdown

**Fecha:** 2026-10-21
**Rama:** fix/rutas-config → develop
**Fase del roadmap:** 6

## 1. Solicitud

Corregir los dos errores anotados como pendientes en la 051, encontrados al
revisar los perfiles: draco no expande `~` en las rutas de `config.yaml`, y
con un perfil Markdown el space de un proyecto se usa como carpeta de
destino.

## 2. Motivación

| Problema | Daño real |
|---|---|
| `~` no se expande en `backend_path`, `frontend_path`, `docs.path` ni `template` | `docs.path: ~/despliegues` crea una carpeta llamada `~` en el directorio actual; `template: ~/plantillas` no se encuentra y draco usa otra plantilla sin avisar; un repo con `~` no se lee. La ayuda de `draco template` sugería justo `template: ~/plantillas/equipo` |
| El space del proyecto llegaba a `profile.Open` como `Scope`, que en Markdown reemplaza a `docs.path` | Un proyecto con `confluence_space_key: PAG` y un perfil Markdown guardaba los documentos en `./PAG`, contra lo que dice la ayuda de `draco g` (carpeta: `--space` > perfil) |

## 3. Diseño técnico

- **`config.ExpandHome`** (pura): reemplaza un `~` inicial (`~`, `~/…`, y
  `~\…` en Windows) por la carpeta personal. `~otro/…` y un `~` en medio
  no se tocan; sin carpeta personal, la ruta queda igual.
  - **Se expande donde se usa, no al cargar.** Si `LoadLocal` expandiera,
    los formularios de la TUI (que parten de esa config) guardarían la ruta
    expandida y cambiarían el archivo del usuario sin que lo pida.
  - Puntos: `deploy.ProjectSides` (repos de Generar), `backlog.Targets`
    (repos y `--path`, que la barra de la TUI no pasa por un shell),
    `Profile.TemplateDir`, `profile.Open` (`docs.path` y la carpeta de
    plantillas del `Bundle`) y el repo de `draco template`.
  - Los flags del modo comando ya llegan expandidos por el shell.
- **El space del proyecto viaja aparte:** `profile.Options.ProjectSpace`.
  `Scope` queda para lo explícito de la ejecución (`--space`: un space o una
  carpeta). Confluence usa `Scope`, luego `ProjectSpace`, luego
  `docs.space`; Markdown usa `Scope` o `docs.path` e ignora `ProjectSpace`.
  - `deploy.Target` separa `SpaceKey` (el `--space`) y `ProjectSpace` (el
    del proyecto). Antes `Resolve` los mezclaba en `SpaceKey` y el llamador
    ya no podía distinguirlos.
  - `draco g` y Generar en la TUI pasan cada uno en su campo. QA y Exportar
    no cambian: no usan el space del proyecto.
- **Descartado:** que `profile.Open` adivinara si `Scope` era un space o
  una carpeta por su forma. Un campo por significado es explícito.
- La precedencia que ya documentaba `draco g --help` no cambia; ahora el
  código la cumple. La guía la explica también para Markdown y deja de decir
  que `~` no se expande (lo había escrito la 051).

## 4. Archivos

- `internal/config/config.go`, `config_test.go` — `ExpandHome` y su test.
- `internal/config/profile.go` — `TemplateDir` expande `~`.
- `internal/profile/profile.go`, `profile_test.go` — `ProjectSpace`;
  `docs.path` y plantillas con `~`; test de Markdown con space de proyecto.
- `internal/deploy/deploy.go`, `deploy_test.go` — `Target.ProjectSpace`;
  repos con `~`.
- `internal/backlog/targets.go` — repos y `--path` con `~`.
- `cmd/generate.go` — pasa `ProjectSpace`.
- `cmd/template.go` — repo del proyecto con `~`.
- `internal/tui/generate.go` — pasa `ProjectSpace`.
- `internal/tui/generate_test.go`, `barrun_test.go` — el space del proyecto
  llega como `ProjectSpace`; rutas de prueba absolutas (la expansión se
  prueba en `config` y `deploy`).
- `docs/guia-usuario.md` — `~` en las rutas; carpeta de un perfil Markdown
  en `draco g`.
- `docs/bitacora/fase-6/053-rutas-config.md`, índice.

## 5. Cómo verificar

```bash
make check
make test pkg=./internal/config run=ExpandHome
make test pkg=./internal/profile run=ProjectSpace
make test pkg=./internal/deploy run='SpacePrecedence|ProjectSides'
```

A mano (pendiente del autor): un perfil Markdown con `docs.path:
~/despliegues` y un proyecto con `confluence_space_key`; `draco g …` escribe
en `~/despliegues` y no crea `./<SPACE>` ni `./~`.
