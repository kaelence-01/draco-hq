# 055 — Cada proyecto con su perfil

**Fecha:** 2026-10-21
**Rama:** feature/perfil-del-proyecto → develop
**Fase del roadmap:** 6

## 1. Solicitud

El autor, y otros desarrolladores de su célula, trabajan a la vez en
proyectos distintos: un sistema grande con backend y frontend, y servicios
sueltos (APIs, workers) en otros workspaces del VCS, documentados en otros
spaces de Confluence y quizá con otro formato. Preguntó si los perfiles ya
permitían dejar eso configurado, para cambiar de contexto sin repetir
opciones. Pidió la actividad para la `v0.3.2`.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Repos, workspace y space ya eran por proyecto, pero el perfil (destino y plantillas) solo se elegía con `--profile` o con el perfil por defecto | Para documentar un servicio con otra plantilla había que acordarse de `--profile` en cada `draco g`, o cambiar el perfil por defecto y volver a cambiarlo después; un olvido publica con la plantilla equivocada |
| La TUI solo usaba el perfil por defecto | En el modo visual no había forma de generar con otro perfil |

## 3. Diseño técnico

- **`projects.<n>.profile`** (`ProjectConfig.Profile`), opcional.
- **`Config.ProfileFor(flag, project)`** (pura): `--profile`, si no el
  perfil del proyecto (vacío: `default_project`), si no `""`, que
  `Config.Profile` resuelve como hasta ahora (`default_profile` o el único).
  Un proyecto que no existe devuelve `""`: el error lo da después
  `deploy.Resolve`, con su mensaje de siempre.
- **Dónde se usa:**
  - `draco g` y `draco template` (que previsualiza la plantilla que usaría
    `draco g -p`);
  - Generar en la TUI, con el proyecto elegido en su formulario;
  - la cabecera de la TUI: `state.useProject` cambia el proyecto activo y,
    con él, el perfil. Elegir un proyecto en Proyectos (`enter`) dice con
    qué perfil queda.
- **QA y Exportar siguen con el perfil por defecto**, en los dos modos. No
  trabajan sobre un proyecto: el consolidado es del equipo, y publicarlo en
  el space de un servicio porque ese era el proyecto activo sería un error.
  En la TUI usaban el perfil de la cabecera; ahora piden el por defecto
  explícitamente, como `draco qa` y `draco f`.
- **Eliminar un perfil en uso falla** (`RemoveProfile`) y dice qué
  proyectos lo usan. Quitarlo en silencio de esos proyectos cambiaría a qué
  perfil publican sin que nadie lo pida; dejar la referencia rota haría
  fallar `draco g` después.
- **Formularios:**
  - TUI: una página «Perfil del proyecto» (lista con «El perfil por
    defecto» y los perfiles), solo con dos o más perfiles: con uno, elegir
    no cambia nada. El detalle del proyecto muestra `perfil`.
  - `draco project add` pregunta el perfil cuando hay perfiles y rechaza
    uno que no existe. `draco project list` suma la columna PERFIL. Ayudas
    de `draco project` y `--profile` de `draco g` al día.
- **Descartado:** `projects.<n>.template`. La plantilla ya puede venir del
  repo (`.draco/templates`) o del perfil; con el perfil del proyecto, la
  del perfil alcanza para lo que se pidió. Si hiciera falta, se suma.

## 4. Archivos

- `internal/config/config.go` — `ProjectConfig.Profile`.
- `internal/config/profile.go`, `profile_test.go` — `ProfileFor`;
  `RemoveProfile` rechaza un perfil en uso.
- `cmd/generate.go`, `generate_test.go` — perfil del proyecto; ayuda de
  `--profile`.
- `cmd/template.go` — perfil del proyecto.
- `cmd/project.go`, `project_test.go` — pregunta el perfil, columna PERFIL,
  ayuda.
- `internal/tui/tui.go` — `state.useProject`.
- `internal/tui/projects.go`, `projectform.go`, `projects_test.go` —
  `enter` cambia también el perfil; página del perfil; detalle.
- `internal/tui/generate.go`, `generate_test.go` — perfil del proyecto
  elegido.
- `internal/tui/qa.go`, `export.go`, `export_test.go` — perfil por defecto
  explícito.
- `docs/guia-usuario.md`, `docs/arquitectura/tui.md`.
- `docs/bitacora/fase-6/055-perfil-del-proyecto.md`, índice.

## 5. Cómo verificar

```bash
make check
make test pkg=./internal/config run=ProfileForAProject
make test pkg=./cmd run='GenerateUsesTheProjectProfile|ProjectAddAsksItsProfile'
make test pkg=./internal/tui run='ProjectBringsItsProfile|GenerateUsesTheProjectProfile'
```

A mano (pendiente del autor): dos perfiles con plantillas distintas y dos
proyectos, cada uno con el suyo; `draco g … -p <cada uno> --dry-run` muestra
«Perfil:» y el documento de cada plantilla; en la TUI, elegir cada proyecto
en Proyectos cambia la cabecera y Generar usa ese perfil.

## 6. Pendientes

- En la TUI, la cabecera muestra el perfil del proyecto activo también en
  QA y Exportar, que usan el por defecto. Si confunde en el uso, esas
  pantallas pueden decir con qué perfil trabajan.
