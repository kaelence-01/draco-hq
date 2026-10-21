# CLAUDE.md

Guía para Claude Code (claude.ai/code) al trabajar en este repositorio.

## Qué es draco

**DRACO** es una herramienta de terminal open source (Go) para documentar
despliegues. Hoy trabaja con Jira y Confluence; la arquitectura de proveedores
(`docs/arquitectura/proveedores.md`, en implementación) la abre a GitHub,
GitLab, Azure DevOps, Linear, Notion o Markdown local:

- Genera y actualiza el **documento de despliegue** de una tarea en Confluence a
  partir del issue de Jira y sus commits de git.
- Publica el **consolidado de QA** (modo Sprint o Kanban) y verifica qué tareas
  tienen su documento.
- **Exporta** a texto la página de Confluence de una tarea.
- Detecta **deuda técnica** en repos locales y la propone como tareas.

Tiene dos formas de uso: **modo comando** (`draco g -i APP-123 -b a1b2c3d`), para
scripts, CI y quien ya conoce los comandos, y **modo visual** (`draco` sin
argumentos en una terminal interactiva), una TUI con menús, listas, formularios
y una barra de comandos con autocompletado. **Sin IA.**

draco nace de **gtt** (`github.com/geomark27/deploy-doc`), un CLI que queda
congelado como está. La lógica de gtt se copió aquí y se adapta; gtt no se
modifica desde este proyecto.

El estado, las fases y las decisiones están en **`docs/roadmap.md`**: léelo antes
de empezar una tarea.

## Comandos

El modo comando está en construcción (fase 3): `main.go` y `cmd/` existen, con
`help` y `version`; cada actividad suma un comando. Los comandos de todos los
días están en el **Makefile** (`make` lista todo; las opciones van como
variables):

```bash
make start name=<tema> [type=feature|fix|docs]   # nueva actividad desde develop al día
make run args="help"      # draco con go run, sin instalarlo
make test [pkg=./internal/qa] [run=Golden] [v=1]  # tests (todos puros, sin red)
make lint                 # gofmt sin diferencias + go vet
make golden [pkg=...]     # regenerar goldens tras un cambio intencional
make check                # verificaciones de cerrar-actividad
make push [force=1]       # sube la rama actual (lo corre el autor: SSH)
make pr [web=1]           # PR de la rama actual hacia develop
make finish               # tras fusionar: develop al día y ramas fusionadas borradas
make release-build ver=vX.Y.Z    # binarios de todas las plataformas + checksums.txt en dist/
make promote              # develop → main: PR con merge commit, fusión confirmada, main y develop al día (lo corre el autor)
make release bump=patch|minor|major  # tag y GitHub Release desde main (lo corre el autor)
```

`make sync` actualiza `develop` solo con *fast-forward*: si `develop` local se
desvió de `origin/develop`, falla en vez de mezclar. Ningún objetivo escribe
commits locales en `main`: `make promote` fusiona en GitHub, siempre con
*merge commit* y después de preguntar, y `make release` solo etiqueta `main`
ya integrada, limpia e igual a `origin/main` (guía: `docs/release.md`). Las fases 5 y 6 se publican
como *patch* de `v0.2.x` (`bump=patch`: `v0.2.1`, `v0.2.2`…); el *minor* no
sube hasta que el autor lo decida (roadmap). Las plataformas del release
(`PLATFORMS` del Makefile) y `updater.Platforms` deben coincidir; un test lo
verifica.

Go: `go.mod` declara `go 1.26.1`. No cambiarlo sin verificar qué versión tienen
instalada los colaboradores y el CI.

## Arquitectura

```
main.go        Entrada: cmd.New(os.Stdin, os.Stdout, os.Stderr).Start(os.Args[1:])
               (Start = Run + oferta de instalación y aviso de versión nueva,
               solo en una terminal)
cmd/           Modo comando: router de mapa, flags por comando, ayuda en dos
               niveles, códigos de salida (0 ok, 1 error, 2 uso)
internal/
  build/       Versión (ldflags)
  cli/         Descripción de los comandos (nombres, subcomandos, flags con
               su descripción) y el parser de flags, que usan cmd y la barra
               de comandos de la TUI
  installer/   Instalación en el primer uso: copia a ~/.local/bin o
               %LOCALAPPDATA%\Programs\draco y agrega la carpeta al PATH
  updater/     Releases de GitHub: última versión, descarga verificada con
               checksums.txt (P-006), reemplazo del ejecutable, caché del aviso
  provider/    Modelo neutro y puertos (Tracker, QATracker, DocStore,
               ChangeSource)
    atlassian/ Adaptador: cliente HTTP (Basic Auth) + Jira v3 + Confluence
               v1/v2, escapado JQL/CQL (query.go), conversión storage → texto,
               Markdown → ADF (markdown.go)
    git/       Adaptador: clon local (archivos de un commit, commits recientes)
    markdown/  Adaptador: carpeta local de archivos .md como destino
  profile/     Arma los adaptadores del perfil (config.Profile) y detecta QA
  config/      Config YAML + variables de entorno (env > archivo)
  document/    Documento de despliegue y consolidado QA en Markdown, desde
               plantillas (templates/ incluidas; ver arquitectura §7)
  backlog/     Deuda técnica (sin red): detectores, repos y opciones del
               análisis, reporte
  deploy/      Flujo del documento de despliegue, sin E/S de terminal
  qa/          Flujo del consolidado de QA, sin E/S de terminal
  fetch/       Flujo de exportar una página a texto, sin E/S de terminal
  tui/         Modo visual (Bubble Tea v2): pila de pantallas sobre los flujos;
               recibe sus dependencias de cmd en tui.Deps
docs/
  roadmap.md                 Visión, fases, estado y decisiones
  arquitectura/proveedores.md    Puertos, adaptadores y plantillas (diseño)
  arquitectura/tui.md            Modo visual: pantallas, modelo y tests (diseño)
  bitacora/fase-N/           Una entrada por actividad, en la carpeta de su fase
  security/patrones-seguros.md   Patrones P-001..P-008 (autoridad en seguridad)
```

`draco` sin argumentos abre la TUI solo en una terminal interactiva
(`App.tty`); sin TTY muestra la ayuda.

### El patrón de los flujos: preparar → revisar → publicar

Cada flujo vive en su propio paquete (`internal/deploy` es la referencia) y
**no imprime, no lee stdin y no termina el proceso**:

- `Resolve(cfg, Request)` combina la petición con la config y devuelve un
  `Target` con sus `Warnings`.
- Una `Session` expone **un método por paso** (`FetchIssue`, `FindExisting`,
  `ReadCommits`, `DraftCreate`/`DraftUpdate`, `Locations`, `Publish`). Entre paso
  y paso decide quien llama: el modo comando pregunta por texto y la TUI con
  listas y formularios, sobre los mismos pasos.
- Cada paso recibe un `context.Context` (la TUI cancela con él).
- Los flujos y `document` hablan con los **puertos** de `internal/provider`,
  **nunca con un adaptador**; el llamador construye los adaptadores
  con `profile.Open` (según el perfil: Jira, Confluence, Markdown local, git)
  y se los pasa a la `Session`. El flujo resuelve solo lo que pidió el
  usuario; el space o la carpeta por defecto vienen del perfil. Los tests usan un doble
  (`fakeAPI`) que embebe el puerto y sobrescribe solo lo que el flujo usa.
- Las convenciones de draco (prefijos de título, «Revisión de Tarea», ventana
  Kanban) viven en el flujo o en `document` y se pasan al adaptador como
  parámetro; el adaptador no las conoce.
- Las situaciones degradadas pero viables se devuelven como `Warnings` o como
  campos (`Draft.Preserved`, `Draft.NotPreserved`, `Result.LinkErr`,
  `Locations.TeamWide`), nunca se
  imprimen desde el flujo.
- Los errores que el llamador debe explicar con su propia UI son centinelas
  (`ErrNoLocations`, `ErrNotQAAccount`, `ErrNoPages`; y en `profile`,
  `ErrNoSpace`, `ErrNoQA`, `ErrNoCredentials`).

## Reglas que no se negocian

- **Nada de datos de una organización en el código** — URLs de instancia, space
  keys, project keys, IDs de estados o campos, nombres de repos, workspace del
  VCS, nombres de personas, pasos de despliegue de un stack. Todo va a
  `config.yaml`, con flag de override cuando tiene sentido. Incluye ejemplos en
  textos de ayuda y **datos de prueba**: usa nombres genéricos (`OPS-7`,
  `pagos-api`, `mi-org`, módulos `Pagos`/`Facturacion`). El repositorio es
  público. Autoridad: `docs/security/patrones-seguros.md`.
- **Todo valor interpolado en JQL o CQL pasa por `quoteLiteral`** (adaptador de Atlassian).
  Nunca un valor entre comillas escritas a mano en un format string.
- **Confluence v2 filtra spaces por `space-id` numérico, nunca por `space-key`.**
  Un parámetro desconocido se ignora en silencio y el filtro deja de filtrar.
  Resolver claves con `ResolveSpaceID`.
- **Los documentos son Markdown generado desde plantillas**
  (`internal/document`); cada adaptador de destino lo convierte a su formato.
  Los valores llegan a la plantilla ya escapados por `mdText` / `mdURL`: un
  `|` o un `<` en un nombre de archivo no puede romper una tabla ni abrir
  HTML, aunque la plantilla sea del usuario. Título, prefijo de búsqueda y
  secciones a conservar salen del manifiesto (`template.yaml`), no del flujo.
- **El ADF es `map[string]any` crudo**, sin structs tipados, y solo existe en el
  adaptador de Atlassian. Se envía como `body.value` (string JSON) con
  `representation: atlas_doc_format`.
- **Actualizar un documento reemplaza todo su cuerpo.** Lo que la gente edita a
  mano («A considerar»; «Observación» y «Consideraciones» del consolidado) lo
  declara el flujo en `Document.Preserve` y lo conserva el adaptador en su
  formato nativo. Antes, el flujo pregunta con `DocStore.Sections` y la UI
  avisa si falta alguna. Si el documento actual no se puede leer, no se
  sobrescribe.
- **Una consulta fallida nunca se publica como un control fallido** en el
  reporte de QA: es evidencia. Se usa el estado desconocido
  (`QAIssue.DeployDocUnknown`).
- **Un error descartado cuyo efecto aparece después se reporta.** Si falla el
  enlace remoto en Jira, el documento existe pero QA lo marcará sin
  documentación semanas después: por eso viaja en `Result.LinkErr`.
- `qa_email` es una protección de UX, no control de acceso: lo lee de la config
  del propio usuario. La autorización real son los permisos de Atlassian.

### El modo comando (`cmd/`)

- Un comando se registra con `register(&command{run, help, summary, flags,
  subs}, nombres…)`; `TestEveryCommandHasHelpAndSummary` exige las tres
  primeras. `flags` y `subs` lo describen para la barra de comandos de la TUI
  (`commandSpecs`).
- Los flags se declaran por comando (`[]cli.Flag`, cada uno con `Desc`) y se
  leen con `cli.Parse`: un flag desconocido o sin valor es un error de uso
  (sale con 2), nunca se ignora; solo un flag `Optional` (`-o` de
  `backlog scan`) puede ir sin valor. `TestEveryFlagIsDescribedAndInItsHelp`
  exige que cada flag y subcomando tenga descripción y aparezca en la ayuda.
  `-h`/`--help` muestra la ayuda y **nunca ejecuta** el comando.
- Los comandos que **guardan** la config la leen con `config.LoadFile` (solo
  el archivo), nunca con `LoadLocal`: este mezcla las variables `ATLASSIAN_*`
  y un token del entorno terminaría escrito en el disco.
- Todo se escribe en `App.Out`/`App.Err` (los tests usan buffers); el color
  solo aparece en una terminal real y sin `NO_COLOR`.
- Un comando recorre los pasos de su flujo y pregunta por texto donde el flujo
  deja la decisión al usuario; nunca reimplementa lógica del flujo.
- Los tests de comandos no tocan la red ni el sistema: `loadConfig`,
  `openProfile`, `saveDialog`, `now`, `scanRepo`, `newUpdater`, `executable`,
  `installDir` y `runTUI` son variables del paquete
  que los tests reemplazan (config en memoria, tracker y git simulados, adaptador Markdown
  real en una carpeta temporal). El diálogo nativo de guardar nunca se abre en
  un test: en WSL abriría una ventana de Windows. La TUI tampoco: sus tests
  llaman a `Update` y `View` sin terminal.

### El modo visual (`internal/tui`)

Normas completas en `docs/arquitectura/tui.md` §10; son obligatorias. Lo
esencial:

- **Sin lógica de negocio**: una pantalla recorre los pasos de un flujo o usa
  funciones puras de `config`; lo que falte se extrae al flujo, no se copia.
- Cada pantalla implementa `screen` (`title`, `update`, `view`, `keys`) y
  dibuja solo su cuerpo: cabecera, **ruta** (`Inicio › Proyectos`) y pie son
  del marco. Un paso que el usuario revisa es una pantalla de la pila; `esc`
  vuelve.
- `ctrl+c` sale siempre. Una pantalla con texto, filtro o confirmación
  implementa `captures` para quedarse con `?` y `esc`; la que lanza pasos
  cancela su `context` en `close()`.
- Red y git en un `tea.Cmd`, nunca en `update`; la `Session` no se lee hasta
  recibir el mensaje del paso.
- Guardar solo con `state.save` (`LoadFile` + funciones puras + `SaveConfig`);
  nunca lo leído con `LoadLocal`.
- Toda función entra en el menú de la principal con su descripción y un
  ejemplo del comando equivalente, que el **panel** muestra; sin pantalla,
  atenuada.
- Estilos solo de `styles.go` (ANSI básicos), sin emojis, símbolos de un
  ancho; cada vista funciona en 80 columnas, y un texto largo (error, URL)
  pasa a la línea siguiente, nunca se corta.
- Tests sin terminal con `send`/`typeText` y `View` sin color; si guarda,
  el token del entorno no llega al archivo.

## Dependencias

Hoy: `gopkg.in/yaml.v3`, `github.com/yuin/goldmark` (Markdown → formato de cada
destino; solo en los adaptadores) y `golang.org/x/term` (leer el token sin
mostrarlo en `draco init`) y `golang.org/x/sys` (el PATH de Windows en el
registro, en el instalador). Para la TUI, Charm v2 (`charm.land/…`):
Bubble Tea, Bubbles, Lip Gloss y Huh (formularios). `github.com/charmbracelet/x/ansi` (ya dependencia de Lip Gloss)
quita los colores en los tests de la TUI. Cualquier otra se justifica antes de
agregarla. El router de comandos es un mapa de nombre a comando, como
en gtt, sin Cobra.

## Idioma

- Interfaz y mensajes al usuario: **español**. Internacionalizar está pendiente
  de decisión (roadmap).
- Código, identificadores y comentarios: **inglés**, como en gtt. Los mensajes
  de los tests están en español.

## Tests

- `go test ./...` cubre funciones puras y los flujos con dobles. **Ningún test
  usa la red ni una instancia real de Atlassian.**
- **Goldens** (`internal/golden`): el documento generado por el builder y por
  los flujos se fija en `testdata/*.golden.json`. Un cambio intencional se
  regenera con `DRACO_UPDATE_GOLDEN=1 go test ./...` y el diff de esos archivos
  se revisa en el PR como parte del cambio.
- Lo que toca la red se verifica a mano contra una instancia propia; los
  comandos que publican tendrán `--dry-run`.
- Cuando exista el instalador: **nunca ejecutar un binario recién compilado**
  (`go build -o x && ./x`). Fuera de su carpeta de instalación ofrecerá
  instalarse y puede reemplazar el `draco` instalado. Usar `go run .`, que el
  instalador debe detectar y saltar (`/go-build/` en la ruta), como en gtt.

## Flujo de trabajo (regla)

```
main ── develop ── feature/<tema>   (PR → develop, nunca a main)
```

- Toda actividad nace de `develop` actualizada en `feature/<tema>` (o `fix/`,
  `docs/`), termina en **un solo commit** y llega a `develop` por PR. El push y
  el PR los hace el autor.
- **`develop` → `main` siempre con *merge commit***, nunca con *rebase and
  merge* ni *squash*: esos copian los commits con otros hashes, `main` y
  `develop` dejan de compartir historia y el siguiente PR choca en cada
  archivo (bitácora 026).
- Toda actividad lleva su entrada en **`docs/bitacora/fase-N/NNN-<tema>.md`**
  (N: fase del roadmap; NNN: número global) en ese
  mismo commit, y su fila en el índice. Las entradas de actividades ya
  fusionadas no se reescriben.
- La historia y el porqué de las decisiones van a la bitácora, **no a
  comentarios del código**: comentarios mínimos (ver `estandares-desarrollo`).

Skills obligatorias (`.claude/skills/`):

| Skill | Cuándo |
|---|---|
| `nueva-actividad` | Antes de cambiar archivos para una tarea nueva |
| `estandares-desarrollo` | Al escribir o editar código y al revisar el diff propio |
| `bitacora` | En toda actividad, antes del commit |
| `cerrar-actividad` | Al terminar: verificaciones (`check.sh`), commit único, push y PR |
