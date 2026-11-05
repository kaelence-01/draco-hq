# 074 — Proyectos en borrador: lo que se deja sin terminar no se pierde

**Fecha:** 2026-11-05
**Rama:** feature/borradores-proyecto → develop
**Fase del roadmap:** 7 (mejoras sobre la fase cerrada)

## 1. Solicitud

El autor pidió que, si alguien está agregando un proyecto y sale sin querer
antes de terminar, no pierda lo que hizo. Lo que quede sin terminar debe
guardarse como borrador hasta completar la configuración. El borrador
aparece como opción, pero deshabilitado, indicando que no se puede usar.
Aceptó las dos decisiones propuestas:

- `esc` a propósito también guarda borrador.
- Solo aplica a proyectos nuevos.

## 2. Motivación

| Problema | Daño real |
|---|---|
| `esc` o `ctrl+c` en «Nuevo proyecto» cerraban el formulario sin guardar nada | Rutas, repos, VCS y patrones del backlog escritos a mano se perdían con una tecla |
| Guardar sin rutas mostraba «El proyecto no se guardó» y descartaba todo | Había que volver a escribirlo entero solo porque faltaba una ruta |

## 3. Diseño técnico

- **Sección aparte, `project_drafts`**, no un campo `draft` dentro de
  `projects`.
  - `Projects` lo leen Generar, Backlog, `-p`, el proyecto por defecto y la
    detección de prefijos de QA. Con un campo, cada uno tendría que acordarse
    de saltarse los borradores.
  - En su propia sección, un borrador no se puede usar por construcción.
  - `GetProject` y `SetDefaultProject` lo reconocen solo para explicarlo:
    «'x' es un borrador: complétalo…».
- **Funciones puras en `config`:**
  - `SetProjectDraft`, `RemoveProjectDraft` (sin borradores, la sección
    desaparece del archivo) y `DraftNames`.
  - `DraftName`: usa el nombre escrito si es válido y libre. Si no, el primer
    `borrador`, `borrador-2`… libre. Un borrador puede no tener nombre, o
    tener uno que ya usa un proyecto.
  - `SetProject` borra el borrador del mismo nombre, porque ya está completo.
- **Cuándo se guarda.** `projectForm.close()` corre cada vez que el
  formulario sale de la pila:
  - Si es un proyecto nuevo, no se guardó y tiene algo escrito, se guarda
    como borrador. No se guarda nada si no se escribió nada.
  - **`ctrl+c` ahora cierra todas las pantallas antes de salir** (`app.go`),
    para que el formulario alcance a guardar. Además, cancela los `context`
    de los pasos en curso, que antes quedaban vivos hasta terminar el
    proceso.
  - Guardar sin rutas deja el borrador y la nota junta los dos motivos: por
    qué no se guardó y que quedó como borrador.
- **Retomar:** `newDraftForm` abre el formulario con lo que tenía, con el
  título «Borrador *nombre*».
  - Al completarlo, se guarda como proyecto y se quita el borrador, aunque
    se haya renombrado.
  - Si se vuelve a dejar sin terminar, el borrador se actualiza.
- **Proyectos (TUI):**
  - La lista son entradas (`entry{name, draft}`): los proyectos primero, los
    borradores después.
  - El borrador aparece atenuado y marcado `borrador`. El detalle dice qué
    le falta, con el mismo error de `Complete` que daría al guardar.
  - `enter` (pie: «retomar») y `e` lo retoman. `x` lo elimina, con su propia
    confirmación.
  - `d` explica que un borrador no se puede usar. `enter` no lo vuelve
    «activo».
- **Configuración** suma «; N borradores sin terminar» al estado de
  Proyectos.
- **Modo comando:**
  - `draco project list` muestra los borradores en un bloque aparte, con lo
    que les falta, también cuando no hay proyectos.
  - `draco project rm` elimina un borrador.
  - `default` explica que es un borrador.
  - `draco project add` no crea borradores: interrumpir el comando no tiene
    dónde guardar.
- **Fuera de alcance:** borradores de perfiles o de la edición de un
  proyecto existente. Al salir de una edición, el proyecto queda como estaba.

## 4. Archivos

- `internal/config/config.go` — `ProjectDrafts`, se mezcla al leer, y `GetProject` explica el borrador.
- `internal/config/profile.go`, `profile_test.go` — `SetProjectDraft`, `RemoveProjectDraft`, `DraftNames`, `DraftName`; `SetProject` cierra el borrador; `SetDefaultProject` lo rechaza.
- `internal/tui/projectform.go` — `newDraftForm`, `load`, `apply`, `close` (guarda el borrador) y `typed`.
- `internal/tui/projects.go` — entradas con borradores, teclas y detalle; eliminar con confirmación.
- `internal/tui/app.go` — `ctrl+c` cierra las pantallas antes de salir.
- `internal/tui/settings.go`, `settings_test.go` — cantidad de borradores.
- `internal/tui/projects_test.go` — `esc`, `ctrl+c`, sin rutas, nada escrito, retomar hasta completarlo y eliminar.
- `cmd/project.go`, `project_test.go` — `list`, `rm` y `default` con borradores; ayuda.
- `docs/guia-usuario.md` — Borradores en Proyectos y la tabla de comandos.
- `docs/roadmap.md` — sección «proyectos y formularios» (074 ✅, 075), la opcional de QA pasa a 076.
- `docs/bitacora/fase-7/074-borradores-proyecto.md`, índice.

## 5. Cómo verificar

```bash
go vet ./... && gofmt -l ./cmd ./internal                  # corridos: limpio
go test ./...                                              # corridos: ok
make check                                                 # lo corre el autor
draco    # Proyectos › n, escribir algo y esc; luego ctrl+c a mitad: lo corre el autor
```

Lista y detalle de Proyectos con un borrador, revisados a 80×24 con un test
temporal.

## 6. Pendientes

- **075:** formularios como en el simulador QA (paginador y línea punteada
  bajo cada campo).
- Borradores de perfiles, si se ve útil.
