# 072 — Configurar QA desde Jira, sin escribir ids

**Fecha:** 2026-11-04
**Rama:** feature/qa-detectar-jira → develop
**Fase del roadmap:** 7 (mejoras sobre la fase cerrada)

## 1. Solicitud

Después de la 071, el autor seguía sin poder configurar los estados de QA para
su proyecto: el buscador de Jira no le cargaba en el tablero Kanban y los tableros no
son todos iguales. Pidió dejar los ids de estado «por defecto en el código»,
para que el usuario no tenga que configurar casi nada.

Se le explicó por qué eso no conviene (ver Diseño) y propuse leerlos de Jira.
El autor aceptó y eligió **la lista de estados del proyecto** en vez de las
columnas del tablero.

## 2. Motivación

| Problema | Daño real |
|---|---|
| `qa_jira` pedía ids de estados y `customfield_*` escritos a mano | La QA no sabía de dónde sacarlos. gtt los tenía fijos en el código y nunca se los pidió a nadie |
| Para ver los ids o nombres había que usar el buscador de Jira o la configuración del tablero | En el tablero Kanban del proyecto el buscador no cargaba, y no todos tienen permiso para ver la configuración del tablero |
| Ponerlos por defecto en el código | En otra instancia de Jira el mismo id es otro estado: el consolidado saldría con tareas equivocadas y sin error. Además publicaría datos de la organización en un repo público (P-008) |

## 3. Diseño técnico

- **Puerto nuevo `provider.QASetup`**, capacidad opcional del tracker:
  - `ProjectStatuses(project)`: estados con `Done` (categoría «finalizado»),
    uno por id y los finalizados al final.
  - `QAFields()`: campos personalizados numéricos (`FieldNumber`) y el de
    estado del PR (`FieldPRStatus`).

  Modelo neutro: `Status`, `Field`, `FieldKind`. Se separó de `QATracker`
  porque este se construye con un `qa_jira` válido, y el setup existe
  justamente para crearlo. `Bundle.QASetup()` no lo pide.
- **Adaptador de Atlassian (`jira_setup.go`):**
  - Lee `/rest/api/3/project/{key}/statuses`: junta los estados de todos los
    tipos de tarea sin repetir y los ordena por categoría (por hacer, en
    curso, finalizado). Jira no da el orden del flujo por esta API; la
    categoría es lo más cercano sin depender de un tablero.
  - Lee `/rest/api/3/field`: solo campos personalizados, los de tipo
    `number` y el de desarrollo.
  - **El campo Development se reconoce por su tipo** (`devsummarycf`), no por
    su nombre, que cambia con el idioma del sitio. Ese tipo es de Jira Cloud,
    no de una organización.
- **Descartado: elegir por columna del tablero** (decisión del autor). Los
  tableros de un mismo proyecto no son todos iguales y no siempre se pueden
  ver. Los estados son del proyecto (de su flujo de trabajo), así que la lista
  sirve igual para el Kanban y para los tableros de módulo con sprints.
- **Flujo `qa.Setup`** (sin E/S, como los demás):
  - `Read(ctx, project)` trae estados y campos. Un proyecto sin estados es
    `ErrNoStatuses`.
  - `Propose(current)` arma la elección inicial. Toma lo que `qa_jira` ya
    tiene para ese proyecto y sigue existiendo en Jira (los estados por id o
    por nombre). Si no hay nada, sugiere: el estado llamado como el Testing
    por defecto, ese más los finalizados como «ya llegó a QA», y el campo de
    desarrollo.
  - Para los contadores (errores, devoluciones) **no sugiere nada**: sus
    nombres son de cada equipo y adivinarlos sería un dato de organización
    disfrazado.
  - `Apply` escribe la elección en una copia de `qa_jira` y la valida.
- **`config.SetProject`:**
  - Escribe arriba si el proyecto es el por defecto (o si todavía no hay
    ninguno) y quita lo que lo pisaba en `projects`.
  - Para otro proyecto, guarda en `projects` solo lo que difiere del por
    defecto y conserva su `sprint_name`.
  - `Lookup` es `ForProject` sin validar: el setup necesita leer un
    `qa_jira` a medias.
  - `Testing()` reemplaza el cálculo repetido del Testing por defecto.
- **Se guardan ids, se muestran nombres.** Los ids sobreviven a que alguien
  renombre un estado; la QA nunca los escribe.
- **Modo comando `draco qa setup [-j KEY]`:** listas numeradas con `*` en lo
  propuesto, Enter acepta y un número fuera de rango se pregunta de nuevo.
  Con un solo campo de desarrollo, lo toma solo. Lee con `config.LoadFile` y
  abre el perfil con la config efectiva (credenciales del entorno), así el
  token del entorno no llega al archivo (test).
- **TUI:** `d` en la sección QA de Configuración (el pie lo muestra solo ahí)
  y `/qa setup [-j KEY]` en la barra. Con `-j` lee de una vez.
  - Una pantalla con tres momentos: pide el proyecto, lee (spinner en un
    `tea.Cmd`, cancelable con `esc`) y ofrece dos páginas.
  - Página 1: el estado con que se entra a QA y los estados marcables, con
    altura acotada.
  - Página 2: los tres campos.
  - Guarda con `state.save`.
- **Arreglo en la barra, descubierto aquí:** con `qa` como primer comando que
  tiene flags propios *y* un subcomando, la barra exigía el subcomando y
  rompía `qa -s 12 -m Pagos` (lo detectó `TestPanelExamplesWorkInTheBar`).
  Ahora, si la primera palabra es un flag, no se busca subcomando
  (`flagsFirst`), y las sugerencias tras `qa ` muestran `setup` y los flags.
- **Lo que no cambia:** ejecutar el consolidado sigue siendo sprint (Enter:
  Kanban) y módulo, como en gtt. El formulario manual sigue para `qa_email`,
  los nombres y quien prefiera escribir.

## 4. Archivos

- `internal/provider/provider.go` — `Status`, `Field`, `FieldKind`, `QASetup`.
- `internal/provider/atlassian/jira_setup.go`, `jira_setup_test.go` — los dos endpoints y sus tests con un Jira simulado; `ports.go` — `Tracker` cumple `QASetup`, `NewQATracker` usa `Testing()`.
- `internal/config/config.go`, `config_test.go` — `Lookup`, `SetProject`, `Testing`.
- `internal/profile/profile.go`, `profile_test.go` — `Bundle.QASetup`.
- `internal/qa/setup.go`, `setup_test.go` — flujo `Setup` (`Read`, `Propose`, `Apply`).
- `cmd/qasetup.go`, `qasetup_test.go` — `draco qa setup`; `cmd/qa.go` — subcomando y ayuda.
- `internal/tui/qasetup.go`, `qasetup_test.go` — pantalla «Detectar desde Jira»; `settings.go` — tecla `d`; `barrun.go` — `/qa setup`; `cmdbar.go`, `cmdbar_test.go`, `barrun_test.go` — comando con flags y subcomando.
- `docs/guia-usuario.md` — detectar desde Jira en la config, Configuración › QA, el comando y problemas frecuentes.
- `docs/arquitectura/proveedores.md` — el puerto `QASetup`.
- `docs/roadmap.md` — 072 ✅, `v0.4.0` publicado, 071 en `v0.4.1`, decisión de no compilar ids.
- `docs/bitacora/fase-7/072-qa-detectar-jira.md`, índice.

## 5. Cómo verificar

```bash
go vet ./... && gofmt -l ./cmd ./internal                  # corridos: limpio
go test ./...                                              # corridos: ok
make check                                                 # lo corre el autor
draco qa setup -j <CLAVE>                                   # contra Jira real: lo corre el autor
```

Pantalla revisada a 80×24 con un test temporal (proyecto, estados y campos
caben sin cortarse).

## 6. Pendientes

- **Probar contra Jira real.** Que `/project/<CLAVE>/statuses` traiga los
  estados esperados y que el campo Development se reconozca por su tipo.
  Comparar lo guardado con los ids que gtt tenía fijos. No se probó:
  los tests no usan la red.
- Con muchos estados (más de 8), la lista se desplaza dentro de su altura.
  Falta verlo con un proyecto real.
- **073 (opcional):** módulo y sprint elegidos de listas.
