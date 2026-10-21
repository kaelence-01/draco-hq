# Modo visual (TUI)

`draco` sin argumentos, en una terminal interactiva, abre una interfaz dentro
de la terminal: menús, listas, formularios y vista previa. Es otra forma de
recorrer **los mismos flujos** que el modo comando; no tiene lógica propia de
negocio.

Estado: **fase 5 en curso** (Exportar, Backlog y Consolidado QA, diseñadas en
el §6). Implementado: la base (entrada, marco, pila de pantallas, pantalla
principal), Proyectos, Generar completo, de la tarea a la publicación,
Exportar y Backlog. Se actualiza al implementar cada actividad. Las reglas para escribir una pantalla están en el
§10 y son obligatorias.

## 1. Alcance por fase

| Fase | Pantallas |
|---|---|
| 4 | Principal, Proyectos, Generar documento (selector de commits, vista previa, ubicación con búsqueda, confirmación) |
| 5 | Backlog (tabla filtrable), QA (progreso en vivo), Exportar |
| 6 | Barra de comandos con autocompletado, configuración (perfiles, credenciales) en formularios, temas, ayuda, aviso de versión nueva |

Mientras una pantalla no existe, su entrada del menú aparece atenuada con el
comando equivalente (`draco qa`, `draco f`, `draco backlog scan`): quien abre
la TUI sabe que la función existe y cómo usarla hoy.

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
- El aviso de versión nueva no se muestra en la TUI hasta la fase 6
  (`checksUpdates` ya lo omite sin argumentos).
- `NO_COLOR` se respeta (Lip Gloss v2 lo detecta).

## 4. Paquete `internal/tui`

```
internal/tui/
  tui.go        Run(ctx, Deps) error; Deps: las dependencias que pasa cmd
  app.go        modelo raíz: pila de pantallas, tamaño, cabecera, pie, teclas globales
  styles.go     estilos de Lip Gloss en un solo lugar (base para temas, fase 6)
  logo.go       DRACO en letras grandes
  home.go       pantalla principal (menú y panel de la función elegida)
  projects.go   Proyectos: lista con filtro, activo, por defecto, borrar
  projectform.go  formulario de Huh para crear y editar un proyecto
  generate.go   Generar: tarea y proyecto (formulario); abre el perfil
  commits.go    Generar: selector de commits por lado, con hash a mano
  progress.go   Generar: pasos de deploy.Session con progreso y cancelación
  preview.go    Generar: vista previa (archivos y Markdown) y secciones no conservadas
  locations.go  Generar: ubicación del documento nuevo, con filtro
  publish.go    Generar: confirmación, publicación y resultado
  filter.go     filtro de una lista (/), compartido por Proyectos, Ubicación y Exportar
  export.go     Exportar: tarea, documentos que la mencionan, texto y guardado
  backlog.go    Backlog: formulario y progreso por repo
  findings.go   Backlog: hallazgos con filtro y detalle
  qa.go         QA: formulario y evaluación en vivo (fase 5)
  docrun.go     lo que Generar y QA comparten: vista previa, ubicación y publicar (fase 5)
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
    Now         func() time.Time                                               // QA: ventana Kanban (fase 5)
}
```

Cada campo llega con la primera pantalla que lo usa.

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
  su `close()` y vuelve a la anterior. `toHome` cierra todas hasta la
  principal (al terminar de publicar).
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
Consolidado QA, y lo que QA comparte con Generar.

### Principal

- DRACO en letras grandes con caracteres de bloque (ninguna terminal común
  muestra el logo como imagen). Si el ancho no alcanza, solo `draco`.
- Menú: Generar documento, Proyectos, Consolidado QA, Exportar página,
  Backlog técnico, Configuración. Atenuadas, las que aún no tienen pantalla:
  cada actividad de la fase 5 activa la suya; Configuración, en la fase 6.
- **Panel de la función elegida**, a la derecha del menú: qué hace y el mismo
  trabajo en modo comando con un ejemplo (`$ draco g -i OPS-7 -b a1b2c3d`), y
  si aún no tiene pantalla, lo dice. Enseña que los dos modos son uno. Si el
  ancho no alcanza (menos de 96 columnas), no hay panel y cada función sin
  pantalla muestra su comando en el menú.
- Sin configuración: la principal lo dice y propone `draco init` (los
  formularios de configuración son de la fase 6).

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
| 1 | Consolidado (formulario) | `qa.Resolve` (con `Deps.Now`), `profile.Open`, `Bundle.QA` | Sprint (vacío: Kanban) y módulo (obligatorio con sprint) |
| 2 | Evaluación (progreso en vivo) | `CheckAccount`, `Collect`, `EvaluateTask` por tarea, `FindExisting` | Ninguna: ver cómo se evalúa cada tarea |
| 3 | Vista previa | `DraftCreate` / `DraftUpdate` | Si `NotPreserved`, aceptar con `s` |
| 4 | Ubicación (solo al crear) | `Locations` | Elegir de la lista con filtro |
| 5 | Publicar | `Publish` | Confirmar |

- **En vivo:** cada `EvaluateTask(i)` es un `tea.Cmd`, uno tras otro; la
  tabla suma la fila de la tarea al llegar su mensaje (clave, resumen y las
  marcas de errores, devoluciones, documento y PR: `✓`, `✗`, y `?` en
  amarillo para `DeployDocUnknown`). Arriba, el período o módulo y sprint,
  y los `Warnings` del target y de cada tarea. `esc` cancela entre tareas.
- `ErrNotQAAccount` muestra la cuenta autenticada, el `qa_email` del config
  y dónde ajustarlo; `ErrNoQA`, que el tracker del perfil no soporta QA.
- Un consolidado con el mismo título se actualiza conservando lo escrito a
  mano, como el comando: no hay pregunta de crear otro, porque el título es
  la clave del consolidado.

### Vista previa, ubicación y publicar: compartidos

Las pantallas de Generar reciben hoy tipos de `deploy`. Para QA no se copian:
reciben una interfaz de la TUI (`docrun.go`) con lo que cada una usa (el
borrador, las ubicaciones, publicar y el comando equivalente), y Generar y QA
la implementan con adaptadores que solo traducen tipos. Lo propio de Generar
(pestaña Archivos, `TeamWide`/`OwnerUnknown`, `LinkErr`) es opcional en la
interfaz. Los tests de Generar pasan sin cambios.

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
| Formulario de Backlog en dos páginas (`newPagedForm`) | Sus seis campos no caben en 24 líneas: la pregunta de dependencias quedaba fuera de la vista |
| Hallazgos sin límite, en una lista desplazable | `--limit` existe porque la salida de texto no se desplaza |
| QA evalúa una tarea por comando | Progreso en vivo y cancelación entre tareas, sin tocar el flujo |
| Vista previa, ubicación y publicar compartidos por una interfaz de la TUI | QA tiene los mismos pasos que Generar; copiarlas duplicaría las reglas de `NotPreserved` y de publicar sin salida a medias |

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
   `OPS-7`, `pagos`, `mi-org`). Sin pantalla, queda atenuada.
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

