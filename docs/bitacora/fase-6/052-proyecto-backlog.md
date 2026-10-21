# 052 — Backlog técnico en el formulario de proyecto

**Fecha:** 2026-10-21
**Rama:** feature/proyecto-backlog → develop (sobre `feature/config-gtt-primero`)
**Fase del roadmap:** 6

## 1. Solicitud

Segunda parte del pedido de la 051: `projects.<nombre>.backlog` (con
`class_globs` y `modules` de backend y frontend) es parte de lo que ya tenía
gtt y debe poder llenarse desde el modo visual, como opcional.

## 2. Motivación

| Problema | Daño real |
|---|---|
| El formulario de proyecto no tenía el backlog; solo se editaba en el YAML | Sin `class_globs`, el detector de clases sin tests nunca corre, y quien usa la TUI no sabe dónde configurarlo |
| Un patrón mal escrito se ignoraba en silencio | Un `class_globs` con un `[` sin cerrar nunca coincide (`matchGlob`), y un módulo ambiguo solo aparece como «omitido» en el reporte del análisis |

## 3. Diseño técnico

- **Una página opcional por repo** al final del formulario de proyecto
  (`projectform.go`): «clases que deben tener tests» y «módulos», como
  listas separadas por comas (`config.SplitList`, igual que en QA). Cada
  página se oculta mientras su repo no tiene ruta. Los placeholders son
  ejemplos genéricos (`app/**/Services/**/*.php`, `src/app/{modulo}/**`).
  - `enter` en blanco salta la página; un proyecto sin patrones no escribe
    `backlog:`.
- **Validación al escribir:** `backlog.CheckPatterns` (nueva, pura, en
  `options.go`) reporta a la vez los globs con un segmento que `path.Match`
  rechaza y los módulos que `parseModulePatterns` no acepta. El análisis
  sigue igual; la función solo adelanta el error al formulario.
- **Guardar sin perder lo demás:** `ProjectConfig.SetBacklogPatterns`
  (nueva, pura, en `config`) reemplaza `class_globs` y `modules` de un repo
  y conserva `since`, `exclude`, `max_lines` y el resto. Un repo que queda
  vacío se quita, y si no queda ninguno, `backlog` también.
  `reflect.DeepEqual` contra la estructura vacía, para que un campo nuevo de
  `BacklogRepoConfig` no se olvide en la comprobación.
  - Solo se toca el repo con ruta: si se borra la ruta del frontend, su
    backlog queda en el archivo como estaba.
- **Detalle del proyecto** (`projects.go`): muestra los patrones de cada repo
  bajo «backlog backend» y «backlog frontend».
- **Space del proyecto:** la descripción dice «Vacío: el de Conexión o el
  del perfil», al día con la 051.
- **Solo en la TUI.** `draco project add` no pregunta el backlog: son
  preguntas opcionales que alargarían `draco init`, y el modo comando es
  para quien ya edita el YAML. Si se pide, sería un flag o una pregunta
  aparte.
- **Rama sobre la 051:** las dos tocan la guía, el índice de la bitácora y
  el test del asistente; partir de `develop` habría dado conflictos al
  fusionar la segunda. Se fusiona después de la 051.

## 4. Archivos

- `internal/backlog/options.go`, `modules_test.go` — `CheckPatterns` y su
  test.
- `internal/config/config.go`, `config_test.go` — `SetBacklogPatterns` y su
  test.
- `internal/tui/projectform.go` — páginas de backlog, validación, guardado;
  descripción del space.
- `internal/tui/projects.go` — backlog en el detalle.
- `internal/tui/projects_test.go` — crear con backlog (patrón inválido,
  listas, detalle); crear y editar pasan por las páginas nuevas.
- `internal/tui/settings_test.go` — el asistente salta la página de backlog.
- `docs/guia-usuario.md` — ejemplo de config con `backlog`, Proyectos y
  Backlog técnico.
- `docs/bitacora/fase-6/052-proyecto-backlog.md`, índice.

## 5. Cómo verificar

```bash
make check
make test pkg=./internal/tui run='Project|Wizard'
make test pkg=./internal/config run=Backlog
make test pkg=./internal/backlog run=CheckPatterns
```

A mano (pendiente del autor): `make run` → Proyectos → `e` en un proyecto
con backend y frontend; llenar los patrones, guardar, ver el detalle y
correr Backlog técnico con el detector de clases sin tests activo.
