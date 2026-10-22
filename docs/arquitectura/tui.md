# Modo visual (TUI)

`draco` sin argumentos, en una terminal interactiva, abre una interfaz dentro
de la terminal: menús, listas, formularios y vista previa. Es otra forma de
recorrer **los mismos flujos** que el modo comando; no tiene lógica propia de
negocio.

Estado: **fases 4, 5 y 6 cerradas**; el diseño de cada pantalla está en el
§6. Implementado: la base (entrada, marco, pila de pantallas, pantalla
principal), Proyectos, Generar y QA completos, de la tarea o el sprint a la
publicación, Exportar, Backlog, Configuración (conexión, perfiles y QA), el
aviso de versión nueva, la barra de comandos (que abre cada pantalla con los
datos de la línea), la Ayuda y los temas. Se
actualiza al implementar cada actividad. Las reglas para escribir una
pantalla están en el §10 y son obligatorias.

## 1. Alcance por fase

| Fase | Pantallas |
|---|---|
| 4 | Principal, Proyectos, Generar documento (selector de commits, vista previa, ubicación con búsqueda, confirmación) |
| 5 | Backlog (tabla filtrable), QA (progreso en vivo), Exportar |
| 6 | Configuración (conexión, perfiles, QA, tema) en formularios, aviso de versión nueva, barra de comandos con autocompletado, ayuda, temas |

Mientras una pantalla no existe, su entrada del menú aparece atenuada con el
comando equivalente: quien abre la TUI sabe que la función existe y cómo
usarla hoy. Desde la fase 6 todas las entradas tienen pantalla; la regla
queda para las funciones que lleguen después.

## 2. Librerías

Charm v2, estable desde 2025, con rutas `charm.land/…`:

| Módulo | Para qué | Versión al planear |
|---|---|---|
| `charm.land/bubbletea/v2` | Bucle de eventos, modelo `Init`/`Update`/`View` | 2.1.0 |
| `charm.land/bubbles/v2` | `list` (con filtro), `viewport`, `spinner`, `textinput`, `help`, `key` | 2.2.1 |
| `charm.land/lipgloss/v2` | Estilos, bordes, composición de la pantalla | 2.0.6 |
| `charm.land/huh/v2` | Formularios de Proyectos (fase 4) y de configuración (fase 6) | 2.0.3 |

Las cuatro están aprobadas (CLAUDE.md). Se agregan cuando la primera actividad
las usa, no antes: Huh llega con Proyectos.

Descartado: **Bubble Tea v1** (`github.com/charmbracelet/…`). Empezar en v1
obliga a migrar después; v2 ya es la línea mantenida. **tview** y otras: no
aprobadas y con otro modelo de programación.

## 3. Entrada

```
draco                       terminal interactiva  →  TUI
draco                       sin TTY (pipe, CI)    →  ayuda, como hoy
draco <comando> …                                 →  modo comando, sin cambios
```

- `App.Run` sin argumentos abre la TUI solo si `App.tty` (entrada y salida
  son una terminal). La TUI se llama a través de una variable del paquete
  `cmd` (`runTUI`), que los tests reemplazan: un test nunca abre la TUI real.
- `Start` no cambia: si el binario no está instalado, la oferta de instalación
  va primero y, sin argumentos, termina ahí (doble clic en Windows).
- Sin argumentos, `checksUpdates` omite el aviso de texto de versión nueva:
  desde la fase 6 lo muestra la TUI en su marco, con las mismas reglas (§6,
  «Aviso de versión nueva»).
- `NO_COLOR` se respeta (Lip Gloss v2 lo detecta).

## 4. Paquete `internal/tui`

```
internal/tui/
  tui.go        Run(ctx, Deps) error; Deps: las dependencias que pasa cmd
  app.go        modelo raíz: pila de pantallas, tamaño, cabecera, pie, teclas globales
  styles.go     estilos de Lip Gloss en un solo lugar y los temas (applyTheme)
  logo.go       DRACO en letras grandes
  home.go       pantalla principal (menú y panel de la función elegida)
  projects.go   Proyectos: lista con filtro, activo, por defecto, borrar
  projectform.go  formulario de Huh para crear y editar un proyecto
  generate.go   Generar: tarea y proyecto (formulario); abre el perfil
  commits.go    Generar: selector de commits por lado, con hash a mano
  progress.go   Generar: pasos de deploy.Session con progreso y cancelación
  preview.go    Generar y QA: vista previa (Markdown y archivos) y secciones no conservadas
  locations.go  Generar y QA: ubicación del documento nuevo, con filtro
  publish.go    Generar y QA: confirmación, publicación y resultado
  filter.go     filtro de una lista (/), compartido por Proyectos, Ubicación y Exportar
  export.go     Exportar: tarea, documentos que la mencionan, texto y guardado
  backlog.go    Backlog: formulario y progreso por repo
  findings.go   Backlog: hallazgos con filtro y detalle
  qa.go         QA: formulario y evaluación en vivo
  docrun.go     docRun: lo que vista previa, ubicación y publicar piden a Generar y a QA
  settings.go   Configuración: secciones con su estado y asistente sin configuración
  connection.go Configuración: conexión con Atlassian y verificación de la cuenta
  profiles.go   Configuración: perfiles, con filtro y detalle
  profileform.go  formulario de Huh para crear y editar un perfil
  qaconfig.go   Configuración: qa_email, qa_report y qa_jira, en páginas
  cmdbar.go     barra de comandos de la principal: sugerencias y lectura de la línea
  barrun.go     barra de comandos: cada comando abre su pantalla con los datos
  themes.go     Configuración: tema, con vista previa en vivo
  helpscreen.go Ayuda: la general y la de cada comando
```

`cmd` importa `tui`; `tui` nunca importa `cmd`. Lo que `cmd` reemplaza en los
tests (`loadConfig`, `openProfile`, `now`) llega a la TUI en `Deps`, con lo
que los tests de la TUI usan los mismos dobles:

```go
type Deps struct {
    LoadConfig  func() (*config.Config, error)                                 // config.LoadLocal (base)
    Version     string                                                         // base
    LoadFile    func() (*config.Config, error)                                 // config.LoadFile (Proyectos)
    SaveConfig  func(*config.Config) error                                     // config.Save (Proyectos)
    OpenProfile func(*config.Config, profile.Options) (*profile.Bundle, error) // Generar (4a)
    Scan        func(context.Context, string, backlog.ScanOptions) (*backlog.Report, error) // Backlog
    Now         func() time.Time                                               // QA: fecha y ventana Kanban
    CheckUpdate func(context.Context) string                                   // aviso de versión nueva
    Commands    []cli.Command                                                  // barra de comandos (commandSpecs)
    Help        func(name string) (string, error)                              // Ayuda (helpText)
}
```

Cada campo llega con la primera pantalla que lo usa; su diseño está en el
§6.

Igual que el modo comando, la TUI arma los adaptadores con `profile.Open` y
habla con los flujos; nunca con un adaptador.

## 5. Modelo

- **Pila de pantallas.** El modelo raíz (`app`) guarda una pila; cada
  pantalla implementa la interfaz `screen`:

  ```go
  type screen interface {
      title() string                        // nombre en la ruta
      update(msg tea.Msg) (screen, tea.Cmd)
      view(width, height int) string        // solo el cuerpo, en ese tamaño
      keys() []key.Binding                  // teclas del pie y de la ayuda
  }
  ```

  Y, si las necesita, estas partes opcionales:

  | Interfaz | Para qué |
  |---|---|
  | `captures(k tea.KeyPressMsg) bool` | Quitarle una tecla global (`?`, `esc`) al marco mientras la pantalla la necesita: un campo de texto, un filtro, una confirmación. `ctrl+c` nunca se captura |
  | `init() tea.Cmd` | Arrancar al mostrarse: al abrirse y al volver a quedar arriba. El formulario de Huh enfoca su primer campo (y, si ya se envió, se rehace con los mismos valores); una pantalla con pasos los lanza una sola vez |
  | `close()` | Soltar lo que la pantalla lanzó al cerrarse: cancela el `context` de un paso en curso |

  `open(s)` apila; `esc` (o el mensaje `back`) cierra la de arriba, llama a
  su `close()` y vuelve a la anterior. `backTo(s)` cierra las que están
  sobre `s` (al terminar el asistente de Configuración), y `toHome` cierra
  todas hasta la principal (al terminar de publicar).
- **Teclas globales:** `ctrl+c` sale siempre; `q` sale desde la principal;
  `?` muestra la ayuda completa de teclas de la pantalla (`bubbles/help`);
  `esc` vuelve. Las tres últimas ceden ante `captures`, y el pie no ofrece
  una tecla que la pantalla captura en ese momento.
- **Marco común:** cabecera con versión, perfil y proyecto activos; debajo,
  fuera de la principal, la **ruta** con los títulos de la pila
  (`Inicio › Proyectos › Nuevo proyecto`); pie con las teclas de la pantalla.
  Las pantallas solo dibujan su cuerpo.
- **Estado compartido (`state`):** la config leída con `LoadLocal` y el
  proyecto activo. `state.save(change)` es la única forma de guardar:
  `LoadFile` → `change` (funciones puras de `config`) → `SaveConfig` → vuelve
  a leer la config; el proyecto activo se mantiene mientras exista.
  `state.useProject` cambia el proyecto activo y, con él, el perfil de la
  cabecera (`config.ProfileFor`: el del proyecto, o el por defecto). Generar
  abre el perfil del proyecto que eligió; QA y Exportar, el por defecto,
  como `draco qa` y `draco f` (bitácora 055).
- **Pasos del flujo como comandos.** Cada paso de una `Session` corre en un
  `tea.Cmd` y vuelve como mensaje (`stepDone{step, err}`). La pantalla muestra
  un spinner mientras tanto y **no lee la `Session` hasta recibir el mensaje**:
  el paso escribe sus campos desde otra goroutine.
- **Cancelación.** Cada pantalla con pasos tiene su `context` y lo cancela en
  `close()`; así `esc` durante un paso lo cancela y vuelve atrás. Un mensaje
  que llega de un paso cancelado se descarta (cada paso lleva un número de
  secuencia).
- **Sin emojis en el marco.** Su ancho varía entre terminales y rompe bordes y
  columnas (sobre todo en Windows). Se usan símbolos de un ancho: `▸ ✓ ⚠ ✗ ●`.

## 6. Pantallas

Fase 4: Principal, Proyectos y Generar. Fase 5: Exportar, Backlog y
Consolidado QA, y lo que QA comparte con Generar. Fase 6: Configuración,
aviso de versión nueva, barra de comandos, Ayuda y temas.

### Principal

- DRACO en letras grandes con caracteres de bloque (ninguna terminal común
  muestra el logo como imagen). Si el ancho no alcanza, solo `draco`.
- Menú: Generar documento, Proyectos, Consolidado QA, Exportar página,
  Backlog técnico, Configuración. Todas tienen pantalla desde la fase 6.
- **Panel de la función elegida**, a la derecha del menú: qué hace y el mismo
  trabajo en modo comando con un ejemplo (`$ draco g -i OPS-7 -b a1b2c3d`), y
  si aún no tiene pantalla, lo dice. Enseña que los dos modos son uno. Si el
  ancho no alcanza (menos de 96 columnas), no hay panel y cada función sin
  pantalla muestra su comando en el menú.
- Sin configuración: la principal lo dice y propone abrir Configuración (o
  `draco init`), que arranca como asistente (§6, «Configuración»).

### Proyectos

- Lista de proyectos con filtro (`/`) y, al lado, el detalle del elegido:
  rutas y repos, `vcs_host`, `vcs_org`, space y checklist. Marca el activo y
  el por defecto. La lista es propia, con el mismo cursor `▸` del menú:
  `bubbles/list` trae paginación, barra de estado y colores fijos que aquí
  sobran, y su filtro sería otro campo con cursor parpadeante en los tests.
- Teclas: `enter` usa el proyecto en esta sesión, `d` lo guarda como por
  defecto, `n` nuevo, `e` editar, `x` eliminar (pide `s`), `/` filtrar; `esc`
  quita el filtro antes de volver.
- **Proyecto activo**: el que usa Generar en esta sesión. Se elige aquí, vive
  en memoria y no se guarda. **Por defecto** (`default_project`) sí se guarda.
- Crear, editar y borrar con un formulario de Huh (nombre, rutas y repos de
  backend y frontend, `vcs_host`, `vcs_org`, space) y confirmación al borrar.
- Guardar repite la regla del modo comando: se lee con `LoadFile` (solo el
  archivo), se modifica con las funciones puras de `config` (`SetProject`,
  `RemoveProject`, `SetDefaultProject`) y se guarda con `config.Save`. Nunca
  `LoadLocal`: un token de `ATLASSIAN_*` terminaría escrito en el disco.
  Editar conserva lo que el formulario no muestra (checklist, backlog).
- Mismas reglas que `draco project add`: el nombre se valida con
  `config.ValidName` y no puede existir; un repo vacío toma el nombre de la
  carpeta; sin rutas de backend ni de frontend no se guarda; el primer
  proyecto queda por defecto. `esc` cierra el formulario sin guardar.

### Generar documento

Los pasos de `deploy.Session`, en el orden en que la TUI los necesita, con las
mismas decisiones que pregunta `draco g`:

| # | Pantalla | Paso del flujo | Decisión del usuario |
|---|---|---|---|
| 1 | Tarea y proyecto (formulario) | `deploy.ProjectSides`, `profile.Open` | Clave de la tarea, proyecto (por defecto, el activo) |
| 2 | Commits | `ChangeSource.CurrentBranch` y `RecentCommits` por lado | Marcar commits de backend y de frontend; `h` agrega un hash a mano |
| 3 | Preparar documento (progreso) | `Resolve`, `FetchIssue`, `FindExisting` | Si ya existe: `1` actualizar, `2` crear otro, `3` cancelar |
| 4 | Preparar documento (progreso) | `ReadCommits` | Si falla un lado, aviso y seguir con el otro |
| 5 | Vista previa | `DraftCreate` / `DraftUpdate` | Pestañas: archivos por carpeta y documento (Markdown en `viewport`). Si `NotPreserved`, confirmar |
| 6 | Ubicación (solo al crear) | `Locations` | Elegir de la lista con búsqueda; aviso si `TeamWide` u `OwnerUnknown` |
| 7 | Confirmación y resultado | `Publish` | Confirmar; resultado con la URL y `LinkErr` si falló el enlace |

- **Los repos antes del `Resolve`.** `deploy.Resolve` exige los hashes, pero
  el selector de commits necesita antes las carpetas de los repos del
  proyecto. `deploy.ProjectSides` (función pura) devuelve los lados del
  proyecto y su space, y `Sides.Deployable` dice qué lados tienen un clon
  (sin ruta de frontend, el backend siempre: su clon o el directorio actual).
  `Resolve` usa la misma función; la TUI no repite esa lógica.
- **El perfil se abre en el paso 1**, con el space del proyecto como alcance:
  una credencial o un space que faltan se explican en el formulario, antes de
  elegir commits, y el selector usa el `ChangeSource` del perfil.
- **Orden de los commits.** El documento enlaza el primer hash de cada lado.
  Los marcados se pasan en el orden de la lista (el más reciente primero); el
  hash manual va al final.
- **Rama actual.** `RecentCommits` lee la rama en la que está el clon. Un
  commit de otra rama se agrega con `h`; la pantalla dice en qué rama está
  (`CurrentBranch`). Cada lado es una pestaña (`tab`), para caber en 80
  columnas; la pestaña cuenta los commits marcados.
- El paso 3 termina con los archivos leídos por carpeta; `enter` abre la
  vista previa.
- **Vista previa:** pestañas Documento (el Markdown en un `viewport`, con las
  líneas largas ajustadas) y Archivos. Reemplaza a `--dry-run`: salir ahí no
  publica nada. Al actualizar dice si lo escrito a mano se conserva; si el
  documento actual no tiene una sección a conservar (`NotPreserved`), lo
  avisa y `enter` no sigue hasta que el usuario acepta con `s`.
- **Ubicación** (solo al crear): `Locations` con aviso de `TeamWide` u
  `OwnerUnknown`, filtro con `/` (el mismo de Proyectos) y `enter` elige.
- **Publicar:** título, destino, acción (crear junto a…, actualizar…) y
  enlace; `enter` publica. Mientras publica, `esc` no sale: dejaría un
  documento sin su enlace (`ctrl+c` sí). El resultado muestra la URL y, si
  falló el enlace desde la tarea, `LinkErr` con qué hacer; un error permite
  reintentar. La pantalla muestra también el comando equivalente. Al
  terminar, `enter` o `esc` vuelven a la principal.
- Los centinelas se explican con texto de la pantalla: `ErrNoSpace` y
  `ErrNoCredentials` (al abrir el perfil), `ErrNoLocations` (propone un perfil
  Markdown).

### Exportar página

Los pasos de `fetch.Session`, con las decisiones de `draco f`:

| # | Pantalla | Paso del flujo | Decisión del usuario |
|---|---|---|---|
| 1 | Tarea (formulario) | `fetch.Resolve`, `profile.Open` con `AnyScope` | Clave de la tarea; space o carpeta (vacío: el del perfil, o todos) |
| 2 | Documentos | `Search` | Con uno solo, sigue sin preguntar; con varios, elegir de la lista con filtro (el más reciente primero) |
| 3 | Texto y guardado | `Export`, `Export.Save` | Revisar el texto (en un `viewport`); ruta del archivo, editable |

- `ErrNoPages` se explica en la pantalla, con la tarea y dónde se buscó.
- **Ruta editable, sin diálogo nativo.** El campo trae la carpeta actual y
  el nombre sugerido (`OPS-7_Titulo.txt`) y tiene el foco: `↑↓` y
  `pgup`/`pgdn` desplazan el texto, lo demás se escribe en la ruta (también
  `?`). Si el archivo existe, pide `s` antes de reemplazarlo y `esc` vuelve
  a la ruta. El resultado muestra la ruta y el comando equivalente
  (`draco f -i OPS-7 -o <ruta>`, entre comillas si tiene espacios); `enter`
  o `esc` vuelven a la principal.
- Las filas de documentos (título y URL) son las de Ubicación (`docRows`),
  recortadas al ancho: son filas de una lista.

### Backlog técnico

No usa red (salvo dependencias) ni perfil, como `draco backlog scan`.

- **El núcleo vive en `internal/backlog`**, no en `cmd`: `Targets` (carpeta,
  un repo del proyecto, todos o la carpeta actual), `OptionsFor` (la config
  del repo con el período: `90d`, `12w`, `3m`), `ApplyModules` (con
  `ErrNoModules`), `SplitModules`, `ReportName`, `SaveReport` y
  `ModuleSummary`. `cmd` los usa sin cambiar su salida; los mensajes que
  nombran un flag quedan en `cmd` (`ErrNoRepoPath` se explica con
  `--path`).
- **`Scan` recibe un `context`** y corre `git` y `composer` con él: `esc`
  detiene un análisis largo en vez de dejarlo corriendo de fondo.

| # | Pantalla | Paso | Decisión del usuario |
|---|---|---|---|
| 1 | Analizar (formulario, dos páginas) | `Targets` | Proyecto (por defecto, el activo), repos y carpeta; luego período, módulos y dependencias (usa internet) |
| 2 | Progreso | `Scan` y `ApplyModules` por repo, uno tras otro | Un repo que falla se marca y se sigue con el otro, como el comando; muestra el comando equivalente (`-l 0`) |
| 3 | Hallazgos | — | Recorrer, filtrar, ver el detalle, guardar |

- **Hallazgos:** una pestaña por repo (`tab`), como el selector de commits.
  Arriba, el repo y su período, los detectores omitidos como aviso, y el
  total, los puntos y la cuenta por módulo. Cada fila: severidad (en su
  color), puntos, módulo y título; `/` filtra por texto (título, tipo,
  severidad, módulo, archivo). El detalle del elegido (tipo, ID, archivos
  con el separador local, evidencia) va al lado desde 96 columnas y debajo
  en 80.
- **Sin `--limit`:** la lista se desplaza; el límite existe porque la salida
  de texto no lo hace.
- `s` guarda el reporte completo de cada repo donde lo deja `-o` sin ruta
  (junto a `config.yaml`, `0600`, con el nombre que suma los módulos
  filtrados) y muestra las rutas.

### Consolidado QA

Los pasos de `qa.Session`, con las decisiones de `draco qa`:

| # | Pantalla | Paso del flujo | Decisión del usuario |
|---|---|---|---|
| 1 | Consolidado (formulario) | `profile.Open`, `Bundle.QA`, `qa.Resolve` (con `Deps.Now` y la plantilla del perfil) | Sprint (vacío: Kanban) y módulo (obligatorio con sprint) |
| 2 | Evaluación (progreso en vivo) | `CheckAccount`, `Collect`, `EvaluateTask` por tarea, `FindExisting` | Ninguna: ver cómo se evalúa cada tarea |
| 3 | Vista previa | `DraftCreate` / `DraftUpdate` | Si `NotPreserved`, aceptar con `s` |
| 4 | Ubicación (solo al crear) | `Locations` | Elegir de la lista con filtro |
| 5 | Publicar | `Publish` | Confirmar |

- **En vivo:** cada `EvaluateTask(i)` es un `tea.Cmd`, uno tras otro; la
  tabla suma la fila de la tarea al llegar su mensaje (clave, marcas de
  errores, devoluciones, documento y PR: `✓`, `✗`, y `?` en amarillo para
  `DeployDocUnknown`; resumen). Arriba, el período o módulo y sprint, los
  `Warnings` del target, la cuenta y las tareas; debajo, los avisos de cada
  tarea. `esc` cancela entre tareas.
- **La pantalla guarda su copia de las filas:** al terminar `Collect` copia
  `Session.Tasks`, y de cada `EvaluateTask(i)` copia solo la fila `i`. Así
  nunca lee la `Session` mientras el paso siguiente la escribe.
- Al terminar, `↑↓` recorre las filas; debajo, la tarea de revisión y las
  observaciones de la elegida, y qué hará publicar (crear o actualizar).
- `ErrNotQAAccount` muestra la cuenta autenticada, el `qa_email` del config
  y dónde ajustarlo; `ErrNoQA`, que el tracker del perfil no soporta QA.
- Un consolidado con el mismo título se actualiza conservando lo escrito a
  mano, como el comando: no hay pregunta de crear otro, porque el título es
  la clave del consolidado.

### Vista previa, ubicación y publicar: compartidos

Las tres pantallas reciben `docRun` (`docrun.go`), no tipos de un flujo:

```go
type docRun interface {
    about() docAbout        // sustantivo, qué busca Ubicación, destino, tarea, comando
    updating() bool         // el borrador reemplaza un documento publicado
    draft(ctx) (*docDraft, error)
    locations(ctx) (*docLocations, error)
    publish(ctx, sibling) (*docResult, error)  // publica el borrador de draft
}
```

- `genRun` (Generar) y `qaRun` (QA) la implementan con adaptadores que solo
  traducen tipos y guardan el borrador tipado entre `draft` y `publish`.
- Lo propio de Generar es opcional: la pestaña Archivos (`docDraft.files`),
  `TeamWide`/`OwnerUnknown`, el enlace desde la tarea (`docAbout.issue`) y
  `LinkErr`. Sin ellos, la pantalla no los muestra.
- Cada adaptador explica sus errores: `deploy.ErrNoLocations` propone un
  perfil Markdown; `qa.ErrNoLocations` agrega el título que falta.
- La vista previa arma el borrador siempre en un `tea.Cmd`, también al
  crear (es instantáneo); el texto del spinner depende de `updating`.
- En QA, `enter` al terminar la evaluación abre la vista previa; el
  resultado dice «Consolidado creado» o «actualizado».
- Los tests de Generar pasaron sin cambios.

### Configuración

Lo que hacen `draco init` y `draco profile`, con sus mismas reglas, en
formularios.

- **Primero, el núcleo sale de `cmd`** (como el del backlog en la fase 5).
  Tres reglas que vivían ahí son ahora funciones puras, y `draco init` y
  `draco profile` las usan sin cambiar su salida (sus tests pasaron sin
  cambios):
  - `config.KeepImplicitProfile`: agregar el primer perfil también guarda
    el implícito (`atlassian`) y lo deja por defecto, para que los comandos
    que lo usaban sigan igual;
  - `config.NormalizeBaseURL`: la URL del sitio empieza con `https://` y se
    guarda sin espacios ni `/` final;
  - `profile.SameAccount`: el email guardado es el de la cuenta autenticada
    (una cuenta que oculta su email cuenta como la misma).
- **Pantalla Configuración:** una lista de secciones, cada una con su
  estado (`✓` lo que tiene, `⚠` lo que falta): Conexión y Proyectos, lo
  que necesita el documento; y bajo «Opcional», Perfiles, QA y Tema
  (bitácora 051). `enter` abre la elegida.
- **Conexión** (formulario de Huh): email, API token, URL del sitio y, si
  el archivo no tiene perfiles, `confluence_space_key` (con perfiles, el
  space es de cada uno).
  - El formulario parte del archivo (`LoadFile`), nunca del entorno.
  - El token no se muestra al escribirlo (`EchoModeNone`, como
    `draco init`), así que tampoco se ve su largo. Vacío, conserva el
    actual.
  - Guardar usa `state.save`. Después, un `tea.Cmd` verifica la cuenta con
    lo guardado, como `draco init`: `OpenProfile` sobre la config del
    archivo, con `AnyScope`, y `Tracker.Me`. Muestra con quién se
    autenticó. Si el email no es el de la cuenta, `s` lo corrige con otro
    `state.save`.
  - Una verificación fallida no deshace lo guardado, como en `draco init`:
    dice qué revisar (email, token, URL).
  - **Variables de entorno:** si `ATLASSIAN_EMAIL`, `ATLASSIAN_TOKEN`,
    `ATLASSIAN_BASE_URL` o `CONFLUENCE_SPACE_KEY` definen un valor, la
    pantalla avisa que ese valor
    gana sobre lo que se guarde al usar draco. La TUI no lee el entorno:
    compara la config del estado (`LoadLocal`) con la del archivo
    (`LoadFile`). El valor del entorno nunca se muestra ni se guarda.
- **Perfiles:** como Proyectos, una lista con filtro y el detalle del
  elegido (tracker, destino, space o carpeta, plantillas).
  - Teclas: `d` por defecto, `n` nuevo, `e` editar y `x` eliminar (pide
    `s`, y avisa si era el por defecto y quedan varios).
  - Formulario en páginas: nombre (`ValidName`, no puede existir) y
    destino; el space o la carpeta, según el destino; la carpeta de
    plantillas y si queda por defecto. El tracker es Jira, el único: se
    muestra, no se elige.
  - Sin perfiles, la lista muestra el implícito `atlassian` y avisa que
    agregar el primero también lo guarda.
- **QA** (formulario de cuatro páginas, que parte del archivo):
  1. `qa_email` y los nombres del encabezado (`qa_report`);
  2. de `qa_jira`: proyecto, estados, estado de Testing y nombre del
     sprint;
  3. los tres campos `customfield_*`;
  4. los prefijos de revisión y el marcador de observación.

  Reglas:
  - las listas se escriben separadas por comas (`config.SplitList`); los
    valores por defecto se muestran en la descripción de su campo;
  - con `qa_jira` vacío (`QAJiraConfig.Empty`) no se guarda esa sección,
    porque QA es opcional; tampoco `qa_report` sin nombres;
  - con algún campo de `qa_jira` escrito, `QAJiraConfig.Validate` muestra
    todos los problemas a la vez sobre el formulario, que conserva lo
    escrito y no guarda hasta corregirlos;
  - el estado de la sección distingue sin configurar, incompleta, sin
    `qa_email` y completa;
  - la pantalla del consolidado, ante un `qa_jira` incompleto o un
    `qa_email` que no es la cuenta, ahora remite a Configuración › QA.
- **Sin configuración** (el archivo no tiene credenciales ni perfiles),
  Configuración arranca como asistente, en el orden de `draco init`:
  1. abre Conexión sola, con el space;
  2. tras verificar, si el archivo no tiene proyectos, `enter` abre
     Proyectos con el formulario de uno nuevo. Al guardarlo queda la
     lista; `esc` vuelve a Configuración.

  Como `draco init`, no crea perfiles: sin ellos, la config es el perfil
  implícito (Jira → Confluence con `confluence_space_key`), igual que gtt.

  Si el archivo está vacío (`Config.IsEmpty`) y `Deps.LoadGtt`
  (`config.GttConfig`) encuentra la config de gtt, el asistente abre antes
  **Importar de gtt** (`gttimport.go`): el resumen de `config.ImportNotes`,
  sin el token; `enter` la guarda tal cual con `state.save` y sigue con la
  verificación de la cuenta; `n` sigue con Conexión. Una config de gtt que no
  se puede leer no detiene el asistente: Configuración dice por qué no se
  ofreció (bitácora 059).
- La entrada del menú deja de estar atenuada; su ejemplo sigue siendo
  `draco init`. El aviso de la principal dice «abre Configuración (o corre
  draco init)».

### Aviso de versión nueva

- **`Deps.CheckUpdate func(context.Context) string`** devuelve la versión
  nueva o `""`. En `cmd` es `latestVersion`, con las reglas del aviso de
  texto:
  - sin aviso en `dev` ni con `DRACO_NO_UPDATE_CHECK` (`updatesOff`, que
    comparten los dos modos); la TUI solo abre en una terminal;
  - con la caché de 24 h al día, responde sin red. Si no, consulta GitHub
    con 3 s de límite, y ante un error usa la caché.

  La TUI no conoce `updater`.
- **Corre en un `tea.Cmd` al abrir** (`app.Init`); hasta que responde, no
  se muestra nada.
- **Con versión nueva:** la cabecera lo dice junto a la versión actual
  (`draco v0.3.0 · v0.3.1 disponible`), del lado que nunca se recorta. La
  principal explica cómo instalarla: «sal con q y corre draco update».
- **No se actualiza desde la TUI:** `draco update` reemplaza el binario en
  uso y escribe su progreso en la terminal, así que se corre fuera del modo
  visual, como hoy.

### Barra de comandos

- **Cada comando se describe una sola vez,** en `internal/cli`:
  - `cli.Command`: nombres (el primero es el nombre, los demás alias),
    resumen, flags y subcomandos;
  - `cli.Flag`, con `Desc` (una línea para las sugerencias);
  - `cli.Parse`, el parser que antes era `parseFlags` de `cmd`.

  `cmd` registra cada comando con sus `flags` y `subs`, y `commandSpecs()`
  arma la lista que recibe la TUI en `Deps.Commands`. `cmd` sigue siendo
  dueño de `run` y `help`, y la TUI nunca importa `cmd`. Un test de `cmd`
  exige que cada flag y subcomando tenga descripción y aparezca en la ayuda
  de su comando.
- **`help` es propio de la barra** (`helpCommand`): el modo comando lo
  atiende antes de su router, así que no está entre los comandos
  registrados.
- **Se abre con `/`, solo desde la principal** (y solo si hay
  `Deps.Commands`). En las demás pantallas `/` ya filtra listas, y las que
  tienen texto lo escriben. Una tecla global chocaría con los campos y con
  las teclas de Huh (`ctrl+p` y `ctrl+n` recorren sus listas).
- **Escribir:** una línea al pie del cuerpo de la principal, con `draco`
  opcional (`/g …` o `/draco g …`). Mientras está abierta, el menú y el
  panel se ocultan y queda el logo: así la línea, cinco sugerencias y un
  mensaje caben en 80×24. Debajo aparecen sugerencias según lo escrito
  (`suggest`, función pura):
  - los comandos, con todos sus nombres y su resumen; completa el nombre
    más corto que coincide (`g` para generate);
  - después del comando, sus subcomandos y los flags que aún no se usaron,
    con su descripción. Completa el flag corto si existe, o el largo si se
    empezó con `--`;
  - después de un flag con valor, los valores que la config conoce:
    proyectos (`-p`), perfiles (`--profile`), `backend` o `frontend`
    (`-r`). Un flag de valor opcional (`-o`) sigue ofreciendo flags;
  - después de `help`, los nombres de los comandos.

  Teclas: `↑↓` recorre las sugerencias, `tab` completa la elegida, `enter`
  ejecuta y `esc` cierra. La barra se queda con todas las teclas mientras
  está abierta (`q` y `?` se escriben).
- **El campo es texto propio, como el filtro de las listas:** se escribe al
  final y se borra con retroceso, sin mover el cursor dentro de la línea.
  Las palabras se separan por espacios; un valor con espacios no se puede
  escribir entre comillas.
- **Leer la línea:** `enter` la lee con `readLine`, con el mismo parser y
  los mismos flags que el modo comando. Un flag desconocido o sin valor, un
  comando o un subcomando desconocidos son un error en la barra, igual que
  en el comando salen con 2.
- **Ejecutar es abrir la pantalla con los datos cargados** (`barrun.go`).
  Se abre la pantalla del comando en el primer paso que el usuario debe
  revisar. Un formulario con lo necesario se envía, y la pila queda como si
  el usuario lo hubiera llenado: `esc` vuelve al formulario con los
  valores.

  | Comando | Abre |
  |---|---|
  | `g -i OPS-7 [-b …] [-f …] [-p …]` | Generar con el formulario enviado; en Commits, los hashes dados quedan marcados (por prefijo) o agregados a mano si no están en la lista |
  | `qa [-s 12 -m Pagos]` | QA con el formulario enviado: la evaluación (sin `-s`, Kanban, como el comando) |
  | `f -i OPS-7 [-s …]` | Exportar, ya buscando |
  | `backlog scan […]` | Backlog: el análisis con las opciones de la línea |
  | `backlog` | Backlog: el formulario, sin analizar (el comando muestra su ayuda) |
  | `project` (`list`, `add`) | Proyectos, o su formulario nuevo |
  | `init`, `profile …` (`add`) | Configuración: Conexión, Perfiles o su formulario nuevo |
  | `help [comando]` | Ayuda |
  | `version` | La versión, en la línea de la barra |

  - Si falta un dato obligatorio (`g` o `f` sin `-i`), se abre el
    formulario con lo que se dio. Un error del paso (un sprint inválido, un
    repo que no es backend ni frontend) queda sobre el formulario, como si
    se hubiera enviado.
  - **Un `-p` que no existe es un error en la barra** (`cfg.GetProject`,
    como en el modo comando). La lista de proyectos del formulario lo
    reemplazaría por el primero, y Backlog analizaría otra cosa. Por lo
    mismo, Generar y Backlog leen los valores de la línea antes de
    reconstruir su formulario.
  - Un `-b` o `-f` para un lado que el proyecto no tiene se ignora con un
    aviso en Commits.
  - Los flags que cada pantalla no toma (`barFlags`: `--dry-run`,
    `--template`, `--space` de Generar, `--vcs-host`, `--vcs-org`,
    `--profile`, `-o`, `--json`, `-l`) no abren nada. La barra explica que
    se usan en la terminal; `--dry-run` sobra porque la vista previa no
    publica. Con varios, se explica el primero en orden alfabético.
  - Los comandos sin pantalla (`update`, `template`) se explican igual: se
    corren fuera del modo visual.
- **Los ejemplos del panel de la principal funcionan tal cual en la barra**
  (un test lo comprueba).

### Ayuda

- `/help` abre la pantalla Ayuda. Muestra qué es draco, las teclas
  globales, cómo se usa la barra y la lista de comandos con su resumen.
- `/help g` muestra la ayuda del comando en un `viewport` que se desplaza
  con `↑↓`. Es el mismo texto de `draco help generate`: `cmd` lo entrega sin
  color por `Deps.Help` (`helpText`). Ningún texto de ayuda se escribe dos
  veces. El título usa el nombre del comando («Ayuda de generate»), aunque se
  pida por un alias; un comando que no existe es un error en la barra.
- La tecla de la barra en el pie y en `?` dice «/ comando (/help: ayuda)».

### Temas

- **Un tema es el juego de estilos de `styles.go`**, siempre con colores
  ANSI básicos (norma 9). Son tres, sin colores definidos por el usuario:
  - `ansi`: el de hoy, por defecto;
  - `alto-contraste`: colores brillantes y negrita en lo importante;
  - `mono`: sin color, solo negrita, tenue y subrayado.
- **Se elige en Configuración › Tema** (`themes.go`):
  - moverse con `↑↓` aplica el tema a toda la pantalla, con una muestra de
    cada estilo;
  - `enter` lo guarda en `config.yaml` (`theme: mono`) con `state.save`.
    `ansi` se guarda vacío, porque es el valor por defecto;
  - salir sin guardar (`close()`) vuelve al tema guardado.

  `theme` es un campo nuevo de `config.Config`, y `mergeMissing` lo copia
  del archivo. `NO_COLOR` sigue ganando: Bubble Tea quita el color al
  escribir en la terminal.
- **`applyTheme` reasigna las variables de `styles.go`:** ninguna pantalla
  cambia, y los tests sin color quedan iguales.
  - `state.load` lo aplica al leer la config, así que el tema rige desde
    que se abre la TUI.
  - Un tema desconocido usa `ansi`, y la sección Tema avisa que no existe.
- **El tema también cubre lo que no dibuja `styles.go`:** el pie de teclas
  (`helpStyles`, que el marco asigna al dibujar) y los formularios de Huh
  (`formTheme`, que usa `newPagedForm`). `mono` usa la base de Huh sin sus
  pocos colores (botones en vídeo inverso, *placeholder* tenue).
- En `ansi`, el pie conserva los estilos por defecto de Bubbles, que usan
  grises en hexadecimal. Venía así desde la fase 4; pasarlo a ANSI básicos
  cambiaría su aspecto y queda como mejora.

## 7. Tests

- Sin terminal ni red: los tests llaman a `Update` con mensajes (teclas,
  tamaño, resultados de pasos) y revisan el estado y el texto de `View` con
  un tamaño fijo y sin color.
- `send` (en `app_test.go`) hace de bucle de Bubble Tea: pasa el mensaje por
  `Update` y luego lo que devuelven sus comandos, abriendo `Batch` y
  `Sequence`. Un comando que no responde enseguida (el parpadeo del cursor de
  un campo de texto, que espera medio segundo) se descarta. `typeText` escribe
  un texto tecla por tecla.
- Guardar se prueba con un archivo en memoria (`fakeFile`) cuyo `LoadConfig`
  agrega un token, como haría `ATLASSIAN_TOKEN`: cada test que guarda
  comprueba que el token no llegó al archivo.
- La cancelación se prueba con un tracker que espera al `context` del paso:
  `esc` debe cancelarlo, y un resultado que llega de un paso cancelado no
  cambia la pantalla nueva.
- Los pasos del flujo usan los dobles del modo comando: config en memoria,
  tracker y git simulados, adaptador Markdown real en una carpeta temporal.
- Un test recorre Generar de punta a punta y comprueba el archivo publicado;
  otro, QA. Exportar guarda en una carpeta temporal y comprueba el texto.
- Backlog usa un `Scan` simulado (`Deps.Scan`) con reportes fijos, y un test
  real de `internal/backlog` cubre la cancelación del `context`.
- **Configuración:** cada test que guarda comprueba que el token del
  entorno no llega al archivo y que el token escrito sí llega, pero nunca
  aparece en `View`. La verificación de la cuenta usa un tracker simulado
  (`Me`).
- **Barra:** el análisis de la línea y las sugerencias son funciones puras,
  probadas con tablas. Hay un test por fila de la tabla de comandos (cada
  uno abre su pantalla con los datos), y otro para el flag desconocido.
  `cmd` suma el test de que cada flag está en la ayuda de su comando.
- **Aviso:** un `CheckUpdate` simulado; ningún test usa la red.
- **Temas:** la vista de cada tema en 80 columnas; `mono` no emite color.
- **Nunca ejecutar un binario recién compilado** (instalador); `make run`
  usa `go run`.

## 8. Terminales

Windows Terminal, PowerShell, Linux y macOS. En la consola clásica de Windows
(`cmd`) se ve más limitado: sin colores de 24 bits y con los caracteres de
bloque según la fuente. La prueba manual de cada actividad incluye Windows
Terminal y una terminal de Linux o WSL.

## 9. Decisiones

| Decisión | Motivo |
|---|---|
| Charm v2 (`charm.land/…`) | Línea estable y mantenida; empezar en v1 obligaría a migrar |
| `cmd` llama a la TUI por una variable; `tui` recibe `Deps` | Sin ciclo de imports; los tests reemplazan la TUI y la TUI usa los mismos dobles |
| Pantallas en una pila, `esc` vuelve | Modelo simple de navegación; cada pantalla se prueba sola |
| La `Session` no se lee mientras un paso corre | El paso escribe desde otra goroutine |
| Proyecto activo en memoria; «por defecto» en la config | Cambiar de proyecto para una tarea no debe cambiar la config |
| Entradas futuras del menú atenuadas con su comando | Muestra lo que existe y cómo usarlo hoy |
| Función pura de repos del proyecto en `deploy` (`ProjectSides`) | El selector de commits la necesita antes de `Resolve`; la TUI no reimplementa el flujo |
| El perfil se abre al enviar la tarea, no después de los commits | Un error de credenciales o de space se ve antes de elegir commits, y el selector necesita el `ChangeSource` del perfil |
| Publicar no se abandona con `esc` mientras corre | Se crearía el documento sin el enlace desde la tarea, y QA lo marcaría sin documentación |
| Filtro de lista compartido (`filter.go`) | Proyectos y Ubicación filtraban igual con código copiado |
| Texto que no cabe se ajusta a la línea siguiente | Un error o una URL cortados en el borde pierden justo lo que explica el problema |
| `CurrentBranch` en el puerto `ChangeSource` | La pantalla dice de qué rama son los commits; sin eso, un commit de otra rama parecería no existir |
| Sin emojis en el marco | Ancho variable entre terminales |
| Panel con la descripción y el comando equivalente en la principal | El modo visual enseña el modo comando; viene de la maqueta del modo visual |
| Ruta con los títulos de la pila | Con pantallas apiladas (Proyectos › formulario) se pierde de vista dónde se está |
| Teclas globales que la pantalla puede capturar (`captures`) | Un campo de texto o un filtro necesita `?` y `esc`; sin esto, escribir `?` abre la ayuda |
| `close()` al cerrar una pantalla | Es lo que cancela un paso en curso al pulsar `esc` |
| Lista de Proyectos propia, no `bubbles/list` | Pocas filas; `bubbles/list` trae paginación, barra de estado y colores fijos, y su filtro complica los tests |
| Exportar con ruta editable, sin el diálogo nativo | Una ventana del sistema abierta desde la TUI la deja esperando sin señal visible, y en un test abriría una ventana real (en WSL, de Windows) |
| Resolución de repos y opciones del backlog de `cmd` a `internal/backlog` | La TUI no importa `cmd`; copiar esas reglas daría dos backlogs que analizan distinto |
| `backlog.Scan` con `context` | Un análisis largo debe detenerse con `esc`, no seguir de fondo |
| El pie de teclas se mide con un espacio menos que la terminal | El marco antepone un espacio a la ayuda; con teclas que medían justo el ancho, la línea se pasaba una columna |
| Formulario de Backlog en dos páginas (`newPagedForm`) | Sus seis campos no caben en 24 líneas: la pregunta de dependencias quedaba fuera de la vista |
| Hallazgos sin límite, en una lista desplazable | `--limit` existe porque la salida de texto no se desplaza |
| QA evalúa una tarea por comando | Progreso en vivo y cancelación entre tareas, sin tocar el flujo |
| Vista previa, ubicación y publicar compartidos por una interfaz de la TUI | QA tiene los mismos pasos que Generar; copiarlas duplicaría las reglas de `NotPreserved` y de publicar sin salida a medias |
| Reglas de `init` y `profile` de `cmd` a funciones puras (perfil implícito, URL, cuenta) | La TUI no importa `cmd`; copiarlas daría dos formas de configurar que guardan distinto |
| El entorno se detecta comparando la config del estado con la del archivo | La TUI no lee variables de entorno ni puede mostrar o guardar su valor por error |
| El token sin eco (`EchoModeNone`) y la verificación con lo guardado | Lo mismo que `draco init`: ni el largo del token queda a la vista, y se verifica lo que se escribió, no un valor del entorno |
| La barra abre la pantalla con los datos, no ejecuta el comando de texto | Lo que el comando pregunta por texto, la TUI ya lo pregunta con sus pantallas; dos interfaces en una pantalla confundirían |
| La barra solo desde la principal | En las demás `/` filtra o se escribe, y una tecla global choca con Huh |
| Descripción de los comandos y parser en un paquete neutro, pasados por `Deps` | Una sola fuente para la ayuda, el parser y el autocompletado; la TUI no importa `cmd` |
| Ayuda de un comando desde `cmd` (`Deps.Help`) | Sin texto de ayuda duplicado que se desactualice |
| Campo de la barra propio, como el filtro, y no `textinput` | Mismo comportamiento que el filtro de las listas y tests sin el cursor parpadeante; mover el cursor dentro de la línea queda como mejora |
| Con la barra abierta, la principal oculta el menú y el panel | El menú no se usa mientras se escribe, y sin él la línea, las sugerencias y el mensaje caben en 80×24 |
| Aviso de versión en el marco; actualizar fuera de la TUI | `draco update` reemplaza el binario en uso y escribe en la terminal |
| Tres temas incluidos, solo con colores ANSI básicos | Siguen la paleta de la terminal (norma 9); colores del usuario romperían el contraste en terminales claras u oscuras |

## 10. Normas para escribir una pantalla

Obligatorias para toda pantalla nueva o modificada (las resume `CLAUDE.md`):

1. **Sin lógica de negocio.** La pantalla recorre los pasos de un flujo o
   llama a funciones puras de `config`. Si necesita algo que el flujo
   calcula, se extrae una función pura en el flujo y la usan los dos modos.
2. **Una pantalla, un `screen`.** Implementa `title`, `update`, `view` y
   `keys`; dibuja solo su cuerpo en el tamaño que recibe. Nada de cabecera,
   ruta ni pie propios: los pone el marco.
3. **Un paso que se revisa es una pantalla.** Un flujo de varios pasos apila
   una pantalla por paso, y `esc` vuelve al anterior sin lógica extra.
4. **Teclas:** `ctrl+c` es del marco y nunca se captura. Si la pantalla tiene
   un campo de texto, un filtro o una confirmación, implementa `captures` y
   reclama `?` y `esc` mientras los necesita. Cada tecla propia va en
   `keys()` con su ayuda en español; ninguna tecla oculta.
5. **Red, git y disco lento en un `tea.Cmd`**, nunca en `update`: la pantalla
   muestra un spinner y no lee la `Session` hasta recibir el mensaje. Leer y
   guardar la config local puede ir directo en `update`.
6. **Lo que se lanza se cancela.** Una pantalla con pasos crea su `context` y
   lo cancela en `close()`; descarta los mensajes de pasos cancelados.
7. **Guardar solo con `state.save`**: `LoadFile` + funciones puras de
   `config` + `SaveConfig`. Nunca se guarda lo leído con `LoadLocal`.
8. **Menú y panel:** toda función nueva entra en el menú de la principal con
   su descripción y un ejemplo del comando equivalente (datos genéricos:
   `OPS-7`, `pagos`, `mi-org`). Sin pantalla, queda atenuada. Si tiene
   comando, la barra la abre (`openFromBar`, con sus flags en `barFlags`),
   y su ejemplo funciona en la barra.
9. **Estilos solo de `styles.go`**, con colores ANSI básicos. Sin emojis;
   símbolos de un ancho (`▸ ✓ ⚠ ✗ ●`). Cada vista funciona en 80 columnas y
   aprovecha las anchas (como el panel desde 96). Un texto que no cabe
   (errores, URLs, párrafos) pasa a la línea siguiente; solo se recortan las
   filas de una lista, que son de una línea.
10. **Mensajes:** el resultado de una acción se muestra en la pantalla
    (`✓` en verde, `✗` en rojo con qué hacer); un problema de config no
    impide abrir la TUI, se muestra como aviso.
11. **Tests sin terminal** para cada pantalla: teclas por `send`/`typeText`,
    texto de `View` sin color con un tamaño fijo, y, si guarda, que el token
    del entorno no llega al archivo. Nunca `tea.NewProgram` en un test.

