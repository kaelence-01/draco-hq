# 071 — QA con varios proyectos de Jira

**Fecha:** 2026-11-04
**Rama:** feature/qa-multiproyecto → develop
**Fase del roadmap:** 7 (mejoras sobre la fase cerrada)

## 1. Solicitud

El autor no entendía de dónde salía tanta configuración de QA comparada con
gtt, ni qué eran los «estados que ya llegaron a QA, id o nombre, separados por
comas». Planteó el caso real: una QA que hace consolidados de dos tableros de
Jira distintos. Pidió revisar si los perfiles lo resolvían y plantear una
actividad que conserve la simplicidad que la QA tenía con gtt, incluido el
consolidado sin sprint (Kanban). Aceptó las dos recomendaciones: el proyecto en
el título del Kanban y el flag `-j/--jira-project`.

## 2. Motivación

| Problema | Daño real |
|---|---|
| `qa_jira` tenía un solo proyecto de Jira | Una QA con dos tableros tenía que editar la config entre un consolidado y otro, y deshacerlo después |
| Los perfiles no guardan el proyecto de Jira: `profile.Open` armaba el tracker de QA con el único `qa_jira` | Dos perfiles, uno por tablero, leían el mismo proyecto; la vía «natural» no servía |
| El título del Kanban era solo el período | Con dos proyectos, el Kanban del segundo encontraba la página del primero por título y la actualizaba |
| El formulario pedía «Id o nombre, separados por comas» sin decir qué eran | Nadie sabía que son las columnas del tablero desde QA hasta el final |

gtt no tenía este problema porque solo servía para un tablero: el proyecto, los
4 ids de estados, la columna de Testing y los 3 campos estaban fijos en el
código, y a la QA solo le preguntaba el sprint (Enter: Kanban) y el módulo.
draco ya hacía esas mismas dos preguntas.

## 3. Diseño técnico

- **`qa_jira.projects`, por clave, que heredan del nivel de arriba.**
  `project`, `statuses`, `testing_status` y `sprint_name` siguen siendo los
  del proyecto por defecto. Cada proyecto de `projects` cambia solo lo que
  define (`QAJiraProject`: estados, Testing, nombre del sprint), y uno vacío
  (`PAG: {}`) usa todo lo de arriba. Si los tableros comparten flujo de trabajo
  (en Jira los ids de estado son de toda la instancia), agregar un proyecto es
  escribir su clave. Los `fields` y las convenciones (prefijos de revisión,
  marcador) son de la instancia y valen para todos.
- **Compatibilidad sin migrar:** una config de un solo proyecto se lee igual.
  No hay `default_project` aparte: el por defecto es `project`, como antes.
- **`ForProject(key)`** valida y devuelve un `QAJiraConfig` de un solo
  proyecto, con sus valores aplicados y sin `Projects`. Lo usan `qa.Resolve` y
  `profile.Open`. `NewQATracker`, `SprintNameFor` y el JQL no cambian: siguen
  recibiendo una config de un proyecto. La clave se compara sin distinguir
  mayúsculas (`-j pag`) y se devuelve como está escrita en `qa_jira`. Si el
  proyecto no está, el error lista los configurados.
- **Proyectos no listados no se aceptan.** Se descartó dejar que `-j` usara
  cualquier clave con los valores por defecto. La lista de la TUI tiene que
  estar completa, y un error de tipeo tiene que fallar antes de consultar.
- **Descartado: un `qa_jira` por perfil.** Los perfiles dicen dónde se
  publica; meter ahí el tablero mezclaría dos ejes. Además, la QA tendría que
  elegir un perfil para elegir un tablero.
- **Simplicidad de gtt:** con un solo proyecto no aparece nada nuevo. Con
  varios:
  - **Modo comando:** pregunta `Proyecto de Jira (OPS, PAG; Enter: OPS)` antes
    del sprint, solo si no hay `-j`. Para eso la config se carga antes de las
    preguntas.
  - **TUI:** muestra una lista antes de Sprint y Módulo.
  - **Lo que se ve:** la línea de modo de Evaluar y el comando equivalente
    muestran el proyecto solo si hay varios.
  - **Modo comando:** imprime `Proyecto:` siempre.
- **Título del Kanban** (decisión del autor):
  `Consolidado de Pruebas QA - {{.Project}} - {{.Period}}`. `QATitle` recibe
  el proyecto y las plantillas propias pueden usar `.Project` en el título.
  El título por sprint no cambia. Costo: el Kanban del período en curso se
  crea una vez como página nueva.
- **Ubicación del Kanban:** `Locations` busca primero los consolidados del
  proyecto (`… - OPS`). Si no hay, busca cualquiera, como antes. Así el primer
  Kanban con el título nuevo encuentra dónde crearse.
- **`-j/--jira-project`** (decisión del autor): `-p` ya es el proyecto de
  draco en `g`, `backlog` y `template`. Se suma a la barra (`barFlags`). Si el
  proyecto no existe, es un error en la barra, como `--project`.
- **Configuración › QA:**
  - Una página nueva, «Otros proyectos de Jira» (claves separadas por comas).
  - Lo que cada proyecto define en el archivo (sus estados) no se edita ahí,
    pero se conserva al guardar.
  - La página de Jira tiene cuatro campos con descripción. Un quinto no
    entraba en 24 líneas, por eso la página aparte.
  - «Estados que ya llegaron a QA» ahora explica que son las columnas del
    tablero desde QA hasta el final, con un ejemplo por nombre.
- **Inicio y Configuración** muestran `proyectos OPS, PAG` con varios.

## 4. Archivos

- `internal/config/config.go` — `QAJiraProject`, `Projects`, `ProjectKeys`, `ForProject`; `Validate` y `Empty` los cubren.
- `internal/config/config_test.go` — herencia, caja de la clave, error con la lista, validación, YAML (`PAG:` y `PAG: {}`).
- `internal/qa/qa.go` — `Request.Project`, `Target.Project`, títulos con el proyecto, `Locations` del Kanban.
- `internal/qa/qa_test.go` — proyecto elegido, título Kanban, proyecto desconocido, orden de búsqueda de ubicaciones.
- `internal/profile/profile.go`, `profile_test.go` — `Options.QAProject`.
- `internal/document/qa_builder.go`, `templates/qa/template.yaml`, `helpers_test.go`, `testdata/qa-kanban.golden.md` — `.Project` en el título; golden regenerado (solo cambia el título).
- `cmd/qa.go`, `qa_test.go` — `-j`, la pregunta con varios proyectos, `Proyecto:`, ayuda.
- `cmd/template.go` — vista previa con proyecto de ejemplo.
- `internal/tui/qa.go`, `barrun.go`, `qaconfig.go`, `homecontext.go`, `settings.go` — lista de proyectos, barra, página nueva, estados.
- `internal/tui/qa_test.go`, `qa_layout_test.go`, `barrun_test.go`, `qaconfig_test.go` — casos nuevos y el título Kanban nuevo.
- `docs/guia-usuario.md` — qué son los estados, `projects`, `-j`, la lista en la TUI, el título del Kanban.
- `docs/roadmap.md` — la mejora (071 a 073) y tres decisiones.
- `docs/bitacora/fase-7/071-qa-multiproyecto.md`, índice.

## 5. Cómo verificar

```bash
go vet ./... && gofmt -l ./cmd ./internal                  # corridos: limpio
go test ./internal/... ./cmd/                              # corridos: ok
make check                                                 # lo corre el autor
draco qa -j <OTRO> -s <N> -m <MÓDULO> --dry-run            # contra Jira real: lo corre el autor
```

## 6. Pendientes

- **Probar contra Jira real** con dos proyectos: un Sprint y un Kanban de cada
  uno, y que el primer Kanban con el título nuevo se cree junto a los
  anteriores. No se probó: los tests no usan la red.
- **072:** configurar sin escribir ids. draco consulta los proyectos, las
  columnas del tablero y los campos por nombre, y la QA elige. «¿En qué
  columna empieza QA?» reemplaza la lista de estados y `testing_status`.
- **073 (opcional):** módulo y sprint elegidos de listas.
- Los valores propios de un proyecto (sus estados) solo se escriben en
  `config.yaml` hasta la 072.
