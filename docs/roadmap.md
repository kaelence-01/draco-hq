# Roadmap — draco

## Visión

La misma herramienta que gtt —documento de despliegue, consolidado de QA,
exportar páginas y backlog técnico— con dos formas de uso:

- **Modo comando:** `draco g -i APP-123 -b a1b2c3d`, igual que siempre.
- **Modo visual:** `draco` sin argumentos abre una TUI dentro de la terminal.

```
╭─ 🐉 draco v0.1.0 ──────────────────────────── proyecto: mi-app ─ tu@correo.com ─╮
│                                                                                 │
│   ▸ 📄 Generar documento de despliegue                                          │
│     ✅ Consolidado QA                                                           │
│     📤 Exportar página                                                          │
│     🧭 Backlog técnico                                                          │
│     📁 Proyectos                                                                │
│     ⚙️  Configuración                                                           │
│                                                                                 │
├─────────────────────────────────────────────────────────────────────────────────┤
│ > /g -i APP-123 -b a1b2c3d█                                                     │
│   /generate  Crea o actualiza el documento de despliegue                        │
├─────────────────────────────────────────────────────────────────────────────────┤
│ ↑↓ navegar · enter abrir · / comando · p proyecto · ? ayuda · q salir           │
╰─────────────────────────────────────────────────────────────────────────────────╯
```

Lo que el modo visual mejora frente a gtt:

- **Elegir commits de una lista** (`git log` del repo) en vez de pegar hashes.
- **Vista previa** de los archivos y del documento antes de publicar.
- **Elegir la ubicación en Confluence** con búsqueda, en vez de «Opción (1-12)».
- **Backlog como tabla filtrable** por módulo y severidad, con panel de detalle.
- **Proyectos y configuración en formularios**, sin editar el YAML a mano.
- **QA con progreso en vivo** y resultado en tabla.

No queda atado a Atlassian: Jira y Confluence son los primeros adaptadores, y
la arquitectura admite GitHub, GitLab, Azure DevOps, Linear, Notion o Markdown
local (`docs/arquitectura/proveedores.md`).

Funciona en Windows Terminal, PowerShell, Linux y macOS. En la consola clásica
de Windows (`cmd`) se ve más limitado.

## Estado

| Fase | Contenido | Estado |
|---|---|---|
| 1 | Nombre, identidad y repositorio | ✅ |
| 2 | Core: portar la lógica de gtt, separar los flujos y desacoplar los proveedores | ✅ |
| 3 | Modo comando (`cmd/`, `main.go`), instalador, updater, releases | ✅ `v0.1.0` |
| 4 | Modo visual mínimo: pantalla principal, proyectos, generar documento | ✅ `v0.2.0` |
| 5 | Resto de pantallas: backlog, QA, exportar | ✅ `v0.2.1` |
| 6 | Configuración en formularios, aviso de versión nueva, barra de comandos con autocompletado, ayuda, temas | 🔄 (`v0.2.2`) |
| v1.0 | Instaladores, checksums, migración desde gtt, README completo | ⬜ |

### Fase 2 — hecho

- Portados con sus tests: `atlassian`, `document`, `git`, `backlog`, `config`.
  Mensajes y rutas dicen `draco` (`~/.config/draco/config.yaml`).
- **QA configurable:** el proyecto, los estados y los campos personalizados de
  Jira, que en gtt estaban fijos en el código, salen de `qa_jira` en la config
  (`config.QAJiraConfig`, con `Validate()` que lista todos los problemas). Los
  métodos QA de `atlassian` reciben un `QASchema`; el JQL se arma en funciones
  puras con tests. Se corrigieron tres valores que gtt interpolaba sin
  `quoteLiteral`.
- **Flujo del documento de despliegue** en `internal/deploy`, sin E/S de
  terminal, con 21 tests sobre un doble de Atlassian.
- `docs/security/patrones-seguros.md` reescrito con ejemplos genéricos.
- Datos de prueba heredados de gtt reemplazados por ejemplos genéricos.
- **Flujo de QA** en `internal/qa`, sin E/S de terminal: `Resolve` valida
  `qa_jira` y lo convierte a `QASchema`; la `Session` va por `CheckAccount`
  (`ErrNotQAAccount`), `Collect` (Sprint o Kanban), `EvaluateTask` (una
  llamada por tarea, para mostrar progreso; una consulta fallida queda como
  **desconocida**), `FindExisting`, `DraftCreate`/`DraftUpdate`, `Locations`
  (`ErrNoLocations`) y `Publish`.
  - El nombre del sprint sale de `qa_jira.sprint_name` (por defecto
    `"{module}_Sprint {sprint}"`, la convención de gtt).
  - **Mejora frente a gtt:** al actualizar un consolidado se preserva lo que QA
    escribió a mano (la celda «Observación» del Resumen y «Consideraciones»,
    `document.ExtractQAPreserved`). Si la página no se puede leer no se
    sobrescribe; lo que no se pudo leer viaja en `Draft.NotPreserved` para que
    la UI pregunte.
  - `KanbanWindow(now)` recibe la fecha; se quitaron `FindQAPage` y
    `FindQAKanbanPage`, que repetían el formato del título (el flujo usa
    `FindPageByTitle` con `document.BuildQA*Title`).
- **Flujo de exportar** en `internal/fetch`: `Resolve`, `Search` (`ErrNoPages`)
  y `Export` (texto con `BuildIssueTxt`, nombre de archivo propuesto,
  `Export.Save`). El nombre translitera tildes y eñes en vez de borrarlas.

### Fase 2 — proveedores y plantillas (cerrada)

draco deja de estar atado a Atlassian: modelo propio, puertos y un adaptador por
plataforma; la estructura del documento pasa a plantillas. Diseño completo en
**`docs/arquitectura/proveedores.md`**. Cada actividad deja draco funcionando y
se cierra con un criterio verificable.

| # | Actividad | Rama | Criterio de cierre |
|---|---|---|---|
| 1 | ✅ Diseño de proveedores, roadmap y decisiones | `docs/arquitectura` | Diseño publicado en el repo |
| 2 | ✅ Snapshot (golden) del ADF actual de deploy y QA | `feature/golden` | Tests que fijan el documento de hoy byte a byte |
| 3a | ✅ Modelo neutro, puertos `Tracker` y `DocStore`, Atlassian como adaptador (`internal/provider`), `context` en todo; `deploy` y `fetch` sobre puertos | `feature/puertos` | `deploy` y `fetch` no importan el adaptador; goldens sin regenerar |
| 3b | ✅ Capacidad `QATracker`, `qa` y `document` sin tipos de Atlassian, puerto de cambios (`ChangeSource`, con commits recientes) | `feature/puertos-qa` | Ningún flujo importa un adaptador; goldens sin regenerar |
| 4a | ✅ Conversor Markdown → ADF en el adaptador de Confluence (`goldmark`) con las convenciones del diseño (§5) | `feature/markdown-adf` | Tests por construcción y convención, golden del conversor, fuzzing sin pánicos; nada publicado cambia |
| 4b | ✅ El builder genera Markdown; preservación de secciones dentro del adaptador (`Sections`); «Observación» del consolidado como sección | `feature/markdown` | Goldens regenerados a **contenido equivalente**; solo las diferencias aceptadas en el diseño (§5) |
| 5 | ✅ Plantillas (`text/template`, manifiesto, resolución por fuentes, plantilla incluida) | `feature/plantillas` | La plantilla incluida reproduce el documento; otra plantilla lo cambia sin recompilar. Absorbe las convenciones de texto («Documento de Despliegue», «Revisión de Tarea», «Novedad::», título del consolidado) |
| 6 | ✅ Destino Markdown local (`internal/provider/markdown`) | `feature/salida-markdown` | Un documento de despliegue completo publicado sin Confluence |
| 7 | ✅ Config con perfiles y migración del `config.yaml` actual | `feature/perfiles` | Un `config.yaml` de hoy se migra solo; absorbe «proyectos y configuración» como funciones puras |

Listar commits recientes para el selector de la TUI entra en el puerto de
cambios (`ChangeSource.RecentCommits`) en la actividad 3b.

Después de la fase 3 (modo comando): segundo tracker (GitHub Issues o Linear),
segundo destino remoto (Notion o SharePoint), PRs de Bitbucket, notificaciones
y plantillas remotas, en ese orden según demanda (diseño, §9).

### Fase 3 — modo comando (cerrada)

draco se vuelve ejecutable. Cada comando arma sus adaptadores con
`profile.Open` y recorre los pasos de su flujo preguntando por texto; todo lo
que publica tiene `--dry-run`. Lo que toca la red se prueba a mano contra una
instancia propia, empezando por `--dry-run`.

| # | Actividad | Rama | Criterio de cierre |
|---|---|---|---|
| 1 | ✅ README | `docs/readme` | El repo explica qué es draco, su estado y cómo se trabaja |
| 2 | ✅ Base del comando: `main.go`, router `cmd/`, flags cortos y largos (portar `parseFlagsWithShorts` y sus tests), ayuda en dos niveles, `version` por ldflags | `feature/cli-base` | `go run . help` y `go run . help <cmd>`; `-h` nunca ejecuta |
| 3 | ✅ `draco g` (falta la prueba manual del autor contra Confluence): despliegue sobre el flujo, ubicación y secciones no conservadas preguntadas por texto, `--dry-run`, `--profile`, `--space`, `--template` | `feature/cli-generate` | `--dry-run` muestra el Markdown; prueba manual contra Confluence y contra Markdown local |
| 4 | ✅ `draco qa` y `draco f` (falta la prueba manual del autor contra Jira y Confluence) (con los diálogos nativos para guardar) | `feature/cli-qa-fetch` | `--dry-run` del consolidado; exportar a un `.txt` |
| 5 | ✅ `draco backlog scan` | `feature/cli-backlog` | Mismo informe y flags que gtt |
| 6a | ✅ `draco init`, `draco project` y `draco profile` sobre las funciones puras de `config`; token oculto con `golang.org/x/term` | `feature/cli-init` | Una config nueva se crea sin editar YAML |
| 6b | ✅ `draco template init/list/validate/preview` sobre el motor de plantillas | `feature/cli-template` | Una plantilla se copia, se edita y se previsualiza sin publicar |
| 7a | ✅ Instalador, updater y `draco update` (SHA-256 contra `checksums.txt`, aviso con caché de 24 h); binarios linux y darwin amd64/arm64 y windows amd64 | `feature/updater` | Tests contra un GitHub simulado: instala, actualiza y rechaza un binario alterado |
| 7b | ✅ Objetivos de release del Makefile, `checksums.txt`, guía de release y primer release `v0.1.0` | `feature/release` | Un release de prueba se instala y se actualiza |

Detalle:

- `main.go` + `cmd/` con router de mapa (sin Cobra), parser de flags cortos y
  largos (portar `parseFlagsWithShorts` de gtt y sus tests), ayuda en dos
  niveles (`draco help`, `draco help <comando>`, `-h` que nunca ejecuta).
- Comandos: `init`, `project`, `g`/`gen`/`generate`, `qa`, `f`/`fetch`,
  `backlog scan`, `update`, `version`, `help`. Mismos flags que gtt.
- `draco` sin argumentos: en una terminal interactiva abrirá la TUI (fase 4);
  sin TTY muestra la ayuda.
- Cada comando arma sus adaptadores con `profile.Open` (flag `--profile`,
  `--space`); `draco init` crea o edita perfiles y proyectos con las funciones
  puras de `config` y guarda con `config.Save`. Una config sin perfiles sigue
  funcionando como el perfil `atlassian`.
- `fetch`: portar de gtt los diálogos nativos para guardar (zenity,
  PowerShell también desde WSL, osascript), con la carpeta actual como respaldo.
- Portar y adaptar de gtt: `installer` (auto-instalación al primer uso, PATH en
  Windows), `updater` (GitHub Releases de este repo, verificación SHA-256,
  aviso cacheado cada 24 h, variable `DRACO_NO_UPDATE_CHECK`), `build` (versión
  por ldflags).
- Plantillas desde el modo comando: flag `--template` y comandos `draco
  template init/list/validate/preview` (el motor ya existe, actividad 5).
- Objetivos de release en el Makefile (ya existe para el día a día) y proceso
  de release con `checksums.txt` (P-006). Releases solo
  desde `main` actualizado y con el árbol limpio.
- Pendiente de decidir: si se porta la skill `backlog-tareas` para Claude Code
  que gtt trae embebida.

**Cierre (bitácora 027):** `v0.1.0` publicado (5 binarios y `checksums.txt`)
e instalado desde el release: el binario descargado se instaló en el PATH y
`draco version` responde `v0.1.0`. La actualización se verifica publicando
esta actividad como `v0.1.1` (`draco update`). Quedan para el autor las
pruebas manuales de `g`, `qa` y `f` contra su instancia.

### Fases 4 a 6 — modo visual

Bubble Tea + Bubbles + Lip Gloss + Huh, en su línea v2 (`charm.land/…`). Cada
pantalla usa el mismo flujo que el comando equivalente; los pasos de la
`Session` se ejecutan como comandos de Bubble Tea para no bloquear la
interfaz. Diseño completo en **`docs/arquitectura/tui.md`**.

- **4:** pantalla principal, proyectos, generar documento con selector de
  commits, vista previa, selector de ubicación y confirmación.
- **5:** backlog (tabla filtrable), QA (progreso en vivo), exportar.
- **6:** configuración en formularios, aviso de versión nueva, barra de
  comandos con los mismos comandos del modo comando y autocompletado,
  ayuda, temas.

La pantalla de inicio no puede mostrar el logo como imagen (la mayoría de las
terminales no lo soporta): necesita el nombre en letras grandes con caracteres.

### Fase 4 — modo visual mínimo (cerrada)

`draco` sin argumentos, en una terminal interactiva, abre la TUI; sin TTY
sigue mostrando la ayuda. Lo que no tiene pantalla todavía aparece atenuado en
el menú con su comando. Cada actividad se prueba sin terminal ni red (`Update`
y `View` con dobles) y a mano en Windows Terminal y en Linux o WSL.

| # | Actividad | Rama | Criterio de cierre |
|---|---|---|---|
| 1 | ✅ Diseño de la TUI y plan de la fase | `docs/plan-fase-4` | `docs/arquitectura/tui.md` y actividades con criterio de cierre |
| 2 | ✅ Base: `internal/tui` (Bubble Tea, Bubbles, Lip Gloss), entrada desde `draco` sin argumentos, pantalla principal (DRACO en letras grandes, menú, cabecera con perfil y proyecto, pie de teclas), pila de pantallas, salida limpia | `feature/tui-base` | `draco` abre y cierra limpio; `draco \| cat` muestra la ayuda; tests del modelo sin terminal |
| 3 | ✅ Proyectos: lista con filtro, proyecto activo de la sesión, por defecto, crear, editar y borrar con formulario (Huh) | `feature/tui-proyectos` | Un proyecto se crea y edita sin tocar el YAML; lo guardado nunca trae variables `ATLASSIAN_*` |
| 4a | ✅ Generar (1): tarea y proyecto, repos del proyecto como función pura de `deploy`, selector de commits (multi-selección, hash a mano), pasos con progreso y cancelación con `esc` | `feature/tui-generar` | De la tarea a los archivos leídos, con dobles; `esc` cancela un paso en curso |
| 4b | ✅ Generar (2): vista previa (archivos y Markdown), documento existente, secciones no conservadas, ubicación con búsqueda, confirmación y resultado (`LinkErr`) | `feature/tui-publicar` | Un documento publicado de punta a punta en una carpeta Markdown (test y a mano); prueba manual contra Confluence |
| 5 | ✅ Cierre de la fase 4 y release `v0.2.0` | `docs/cierre-fase-4` | `v0.2.0` instalado con `draco update` y la TUI abierta desde el binario instalado |

**Cierre (bitácora 033):** `draco` sin argumentos abre el modo visual con la
pantalla principal, Proyectos y Generar completo (tarea, commits, vista
previa, ubicación y publicación), sobre los mismos flujos que el modo
comando. Las normas para escribir una pantalla quedaron en
`docs/arquitectura/tui.md` §10. `draco help` menciona el modo visual. Quedan
a cargo del autor el release `v0.2.0` (`docs/release.md`), su verificación con
`draco update` desde `v0.1.x` y la TUI abierta desde el binario instalado, y
las pruebas manuales de las bitácoras 029 a 032: Windows Terminal y Linux/WSL,
y Generar contra su Jira y su Confluence.

### Fase 5 — resto de pantallas (cerrada)

Exportar, Backlog y Consolidado QA dejan de estar atenuados en el menú. Cada
pantalla recorre el mismo flujo que su comando, con las normas de
`docs/arquitectura/tui.md` §10; el diseño de cada una está en su §6. Lo que
hoy vive en `cmd` y la TUI necesita (la resolución de repos y opciones del
backlog) se extrae a su paquete antes de usarlo, y lo que Generar ya resolvió
(vista previa, ubicación, publicar) se comparte con QA en vez de copiarse.

| # | Actividad | Rama | Criterio de cierre |
|---|---|---|---|
| 1 | ✅ Diseño de las pantallas y plan de la fase | `docs/plan-fase-5` | `docs/arquitectura/tui.md` §6 y actividades con criterio de cierre |
| 2 | ✅ Exportar: tarea y space, búsqueda, lista de documentos con filtro, vista del texto y guardado en una ruta editable (faltan las pruebas manuales del autor) | `feature/tui-exportar` | Un documento exportado de punta a punta desde una carpeta Markdown (test y a mano); prueba manual contra Confluence |
| 3 | ✅ Backlog: resolución de repos, opciones y guardado del reporte de `cmd` a `internal/backlog`, `Scan` con `context`; formulario, progreso por repo, tabla con filtro y detalle, guardar el JSON (faltan las pruebas manuales del autor) | `feature/tui-backlog` | `draco backlog scan` sin cambios (sus tests pasan igual); un análisis en la TUI con `esc` que lo cancela; el reporte guardado igual al de `-o` |
| 4a | ✅ QA (1): sprint o Kanban, cuenta de QA, tareas y evaluación en vivo, una fila por tarea (faltan las pruebas manuales del autor) | `feature/tui-qa` | De la cuenta a la tabla evaluada, con dobles; `ErrNotQAAccount` y `ErrNoQA` explicados; `esc` cancela |
| 4b | ✅ QA (2): vista previa, ubicación y publicar compartidos con Generar; consolidado existente y `NotPreserved` (faltan las pruebas manuales del autor) | `feature/tui-qa-publicar` | Generar sin cambios (sus tests pasan igual); un consolidado publicado de punta a punta en una carpeta Markdown; prueba manual contra Jira y Confluence |
| 5 | ✅ Cierre de la fase 5 y release `v0.2.1` (planeado como `v0.3.0`; ver decisiones) | `docs/cierre-fase-5` | `v0.2.1` instalado con `draco update` y las tres pantallas abiertas desde el binario instalado |

Exportar va primero: es el flujo más corto y no publica. Backlog no usa red
ni perfil. QA se divide como Generar: primero hasta la tabla evaluada,
después la publicación.

**Cierre (bitácora 039):** el modo visual cubre todas las funciones de draco
salvo Configuración (fase 6): Generar, Proyectos, Consolidado QA hasta la
publicación, Exportar y Backlog, sobre los mismos flujos que el modo comando.
El núcleo del backlog salió de `cmd` a `internal/backlog`, `Scan` se cancela
con `context`, y vista previa, ubicación y publicar son las mismas pantallas
para Generar y QA (`docRun`). El autor decidió saltar las pruebas manuales de
las bitácoras 035 a 038 en este cierre: Exportar, Backlog y QA no se probaron
contra Jira y Confluence reales, ni Generar de punta a punta después de la
038. Quedan a cargo del autor el release `v0.2.1` (`docs/release.md`) y su
verificación con `draco update` desde `v0.2.0`.

### Fase 6 — configuración, barra de comandos y temas (en curso)

Configuración deja de estar atenuada en el menú, y el modo visual suma lo
que no es una función nueva sino una forma de usar las que ya tiene: la
barra de comandos, la ayuda, el aviso de versión nueva y los temas. El
diseño está en `docs/arquitectura/tui.md` §6, con las normas del §10. Como
en la fase 5, lo que hoy vive en `cmd` y la TUI necesita se extrae antes de
usarlo: las reglas de `init` y `profile`, y la descripción de los comandos y
su parser. El release de la fase es `v0.2.2` (ver decisiones).

| # | Actividad | Rama | Criterio de cierre |
|---|---|---|---|
| 1 | ✅ Diseño de las pantallas y plan de la fase | `docs/plan-fase-6` | `docs/arquitectura/tui.md` §6 y actividades con criterio de cierre |
| 2 | ⬜ Configuración (1): reglas de `init` y `profile` de `cmd` a funciones puras; Conexión (token oculto, verificación de la cuenta), Perfiles y asistente sin configuración | `feature/tui-config` | Una config nueva creada desde la TUI sin tocar el YAML; el token del entorno nunca llega al archivo; `draco init` y `draco profile` sin cambios (sus tests pasan igual) |
| 3 | ⬜ Configuración (2): QA (`qa_email`, `qa_report`, `qa_jira`) | `feature/tui-config-qa` | `qa.Resolve` acepta una config de QA creada solo desde la TUI; `Validate` muestra todos los problemas en la pantalla |
| 4 | ⬜ Aviso de versión nueva en el marco (`Deps.CheckUpdate`) | `feature/tui-aviso` | Con un aviso simulado, la cabecera y la principal lo muestran; sin aviso en `dev` ni con `DRACO_NO_UPDATE_CHECK`; ningún test usa la red |
| 5a | ⬜ Barra (1): descripción de los comandos y parser de `cmd` a un paquete neutro, flags con descripción; barra con autocompletado y pantalla Ayuda | `feature/tui-barra` | `draco help` y los tests de `cmd` sin cambios; cada flag en la ayuda de su comando; sugerencias de comandos, flags y valores probadas con tablas |
| 5b | ⬜ Barra (2): cada comando abre su pantalla con los datos cargados | `feature/tui-barra-ejecutar` | `/g -i OPS-7 -b a1b2c3d` llega a Commits con el hash marcado (test); un flag sin pantalla se explica; los ejemplos del panel funcionan en la barra |
| 6 | ⬜ Temas: `ansi`, `alto-contraste` y `mono`, elegidos en Configuración | `feature/tui-temas` | El tema se guarda en `config.yaml` y se aplica al abrir; cada tema funciona en 80 columnas; `mono` sin color |
| 7 | ⬜ Cierre de la fase 6 y release `v0.2.2` | `docs/cierre-fase-6` | `v0.2.2` instalado con `draco update`; Configuración y la barra abiertas desde el binario instalado |

Configuración va primero: es la única entrada atenuada del menú, y sin ella
quien instala draco tiene que salir a `draco init`. El aviso de versión
nueva es corto y no depende de nada. La barra va después porque abre todas
las pantallas, Configuración incluida, y se divide como Generar y QA:
primero escribir y sugerir, después ejecutar. Los temas van al final porque
son lo de menor valor.

### v1.0

- Instaladores (script, Homebrew tap, Scoop/winget). El paquete `draco` de
  Homebrew, Debian y MacPorts es Google Draco (compresión 3D): publicar con otro
  nombre de paquete; el comando sigue siendo `draco`.
- Migración desde gtt: leer `~/.config/gtt/config.yaml` para no repetir el
  asistente (pendiente de confirmar).
- Guía de usuario y completar el README con la instalación (la base existe
  desde la fase 3); bitácora enlazada por versión
  (Solicitud → Motivación → Diseño técnico → Archivos → Cómo verificar).

## Decisiones tomadas

| Decisión | Motivo |
|---|---|
| Proyecto nuevo, gtt queda congelado y solo CLI | Es un cambio grande; no contaminar el estilo ni el comportamiento de gtt |
| Nombre DRACO, comando `draco` | `gtt` se confundía con `git`; corto, pronunciable, sin choque en la terminal |
| Sin IA | Se evaluó y se descartó para esta versión |
| La lógica se **copia** de gtt, no se importa | Go no permite importar `internal/` de otro módulo, y gtt no evoluciona |
| Flujos preparar → revisar → publicar sin E/S | Un solo núcleo para el modo comando y el visual, testeable sin red |
| Datos de instancia de Jira del QA en config (`qa_jira`) | El proyecto es abierto: lo propio de una instancia no se compila |
| Licencia MIT | La misma de gtt |
| Multiplataforma por puertos y adaptadores, adaptadores compilados en el binario | Que draco sirva a equipos sin Jira ni Confluence; un solo ejecutable |
| Estructura del documento en plantillas `text/template` → Markdown → formato del destino | Cada equipo ajusta su documento sin recompilar; una plantilla sirve para cualquier destino |
| `goldmark` para leer Markdown | Go puro, MIT, mantenido; soporta tablas y tareas GFM |
| Ramas `main` → `develop` → `feature/<tema>`, un commit por actividad, PR a `develop` | Historial corto y legible; `main` solo recibe lo integrado |
| Bitácora por actividad, agrupada por fase (`docs/bitacora/fase-N/`), y comentarios mínimos en el código | La historia de cada decisión en un solo lugar, el código limpio |
| Las fases 5 y 6 se publican como *patch* de `v0.2.x` (`make release bump=patch`): `v0.2.1` cierra la fase 5 y la fase 6 sigue con `v0.2.2` en adelante | Decisión del autor (bitácora 039); el *minor* no cambia hasta que el autor lo decida |

## Decisiones pendientes

- Significado del acrónimo: *Deployment Records, Audits & Code Overview*, la
  propuesta original *Deployment Records And Change Orchestrator* («orchestrator»
  sugiere que ejecuta despliegues, y no lo hace) o ninguno.
- Nombre del paquete en Homebrew/Scoop (el repo es `draco-hq`).
- Solo español o español e inglés desde el inicio.
- Leer el config de gtt para migrar.
- Portar o no la skill `backlog-tareas`.
- Proveedores: mapeo de estados entre trackers, gestor de secretos del sistema
  para las credenciales, plantillas remotas (`docs/arquitectura/proveedores.md`).
- Logo: confirmar origen y licencia de la imagen, quitar el ©, vectorizar,
  versión para fondo claro y versión para 16–32 px.
