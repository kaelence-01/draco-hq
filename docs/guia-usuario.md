# Guía de usuario de draco

draco documenta despliegues desde la terminal:

- arma el **documento de despliegue** de una tarea a partir del issue y de sus
  commits;
- publica el **consolidado de QA** de un sprint o de los últimos días;
- **exporta** a texto el documento de una tarea;
- detecta **deuda técnica** en tus repos locales.

Trabaja con Jira y Confluence, o con una carpeta de archivos Markdown en lugar
de Confluence. No usa IA.

Esta guía corresponde a la versión de draco del mismo commit: la de `main` es
la del último release. Para cualquier comando, `draco help <comando>` muestra
su ayuda completa.

**Contenido**

1. [Instalación](#1-instalación)
2. [Primeros pasos](#2-primeros-pasos)
3. [Conceptos: proyectos, perfiles y configuración](#3-conceptos-proyectos-perfiles-y-configuración)
4. [Modo visual](#4-modo-visual)
5. [Modo comando](#5-modo-comando)
6. [Plantillas](#6-plantillas)
7. [Solución de problemas](#7-solución-de-problemas)
8. [Desinstalar](#8-desinstalar)

---

## 1. Instalación

### Con un comando

**Windows (PowerShell):**

```powershell
iwr -useb https://raw.githubusercontent.com/kaelence-01/draco-hq/main/scripts/install.ps1 | iex
```

**Linux, macOS y WSL:**

```bash
curl -fsSL https://raw.githubusercontent.com/kaelence-01/draco-hq/main/scripts/install.sh | sh
```

El script:

1. descarga el binario de tu sistema del último release;
2. verifica su SHA-256 contra el `checksums.txt` del mismo release, y si no
   coincide no instala nada;
3. lo copia a `%LOCALAPPDATA%\Programs\draco` (Windows) o `~/.local/bin`
   (Linux y macOS);
4. agrega esa carpeta a tu PATH si no está (Windows: el PATH de tu usuario;
   Linux y macOS: una línea en `~/.bashrc` o `~/.zshrc`).

Abre una terminal nueva y comprueba con `draco version`.

Opciones:

| Variable | Para qué |
|---|---|
| `DRACO_VERSION=v0.3.0` | Instala esa versión en lugar de la última |
| `DRACO_INSTALL_DIR=<carpeta>` | Otra carpeta de instalación (solo `install.sh`) |

En PowerShell, la variable va antes del comando: `$env:DRACO_VERSION = "v0.3.0"`.

WSL y Windows son sistemas distintos: si usas draco en los dos, instálalo en
cada uno.

### A mano

1. Descarga de [Releases](https://github.com/kaelence-01/draco-hq/releases/latest)
   el binario de tu sistema (`draco-windows-amd64.exe`, `draco-linux-amd64`,
   `draco-linux-arm64`, `draco-darwin-amd64` o `draco-darwin-arm64`) y
   `checksums.txt`.
2. Verifica el hash:
   - Windows: `Get-FileHash .\draco-windows-amd64.exe` y compáralo con su
     línea de `checksums.txt`;
   - Linux: `sha256sum -c checksums.txt --ignore-missing`;
   - macOS: `shasum -a 256 -c checksums.txt --ignore-missing`.
3. Ejecútalo. La primera vez ofrece instalarse en la misma carpeta que el
   script y agregarla al PATH.

Los binarios no están firmados:

- **Windows SmartScreen** puede pedir «Más información → Ejecutar de todas
  formas»;
- **macOS** puede pedir `xattr -d com.apple.quarantine <binario>`.

### Actualizar

```bash
draco update            # descarga, verifica el SHA-256 y reemplaza el ejecutable
draco update --check    # solo consulta si hay una versión nueva
```

draco avisa cuando hay una versión nueva: en el modo comando, al terminar un
comando; en el modo visual, en la cabecera. Consulta GitHub como mucho una vez
al día y nunca en pipes ni en CI. `DRACO_NO_UPDATE_CHECK=1` lo apaga.

---

## 2. Primeros pasos

Necesitas:

- tu **email** de Atlassian;
- un **API token**, que se crea en
  <https://id.atlassian.com/manage-profile/security/api-tokens>;
- la **URL de tu sitio** (`https://<tu-sitio>.atlassian.net`);
- el **space de Confluence** donde van los documentos;
- las **rutas de tus repos** (backend y frontend) y su organización o
  workspace en el VCS: el proyecto.

Es lo mismo que pedía gtt. Tienes dos caminos:

- **Modo visual:** corre `draco` y abre **Configuración**. Si no hay
  configuración, arranca sola como asistente: conexión (con el space),
  verificación de la cuenta y el primer proyecto.
- **Modo comando:** `draco init` pregunta lo mismo por texto.

Los perfiles son opcionales: no hace falta crear uno para publicar en
Confluence.

### ¿Vienes de gtt?

No vuelvas a configurar nada. Si draco todavía no tiene configuración y
existe la de gtt (`~/.config/gtt/config.yaml`, o `~/.config/deploy-doc/`
en versiones viejas), los dos caminos ofrecen importarla:

- `draco init` pregunta «¿Importarla?»; con Enter la importa y solo
  verifica la cuenta;
- el asistente de Configuración abre primero **Importar de gtt**: `enter`
  la importa, `n` configura desde cero.

Credenciales, space, proyectos (con su checklist y su backlog) y el
proyecto por defecto pasan tal cual. Se muestra un resumen sin el token, y
la configuración de gtt no se modifica: puedes seguir usando los dos.

- **Consolidado QA:** gtt guardaba en su código el proyecto, los estados y
  los campos de Jira; draco los lee de `qa_jira`. Si usabas el consolidado,
  complétalo una vez en Configuración › QA.
- **gtt y draco en sistemas distintos** (uno en Windows y otro en WSL):
  cada uno tiene su carpeta personal y draco no encuentra la de gtt. Copia
  el archivo a la carpeta de draco del otro sistema, o configura desde cero.

El token no se muestra al escribirlo. Todo se guarda en
`~/.config/draco/config.yaml` (en Windows,
`%USERPROFILE%\.config\draco\config.yaml`), legible solo por tu usuario.

Después, prueba sin publicar nada:

```bash
draco g -i OPS-7 -b a1b2c3d --dry-run
```

Si prefieres no publicar en Confluence mientras pruebas, crea un perfil con
destino **Carpeta Markdown local**: los documentos quedan como archivos `.md`.

---

## 3. Conceptos: proyectos, perfiles y configuración

### Proyecto

Un proyecto describe dónde están tus repos y cómo enlazarlos. Generar un
documento lo necesita:

- `backend_path` y `frontend_path`: las carpetas de los clones locales;
- `backend_repo` y `frontend_repo`: sus nombres en el VCS (si se dejan
  vacíos, el nombre de la carpeta);
- `vcs_host` y `vcs_org`: el host y la organización o workspace del VCS;
- `confluence_space_key`: el space de sus documentos, si no es el de
  Conexión (sin perfiles) o el del perfil. Solo aplica cuando el destino es
  Confluence: un perfil Markdown lo ignora;
- `profile`: su perfil, si no es el por defecto (ver abajo).

El proyecto por defecto lo usan Generar, Backlog y `draco template` (para el
perfil y el `.draco/templates` del repo) cuando no indicas otro con `-p`.

Si trabajas en varios proyectos (un sistema grande y servicios sueltos, en
otros workspaces y spaces), crea un proyecto por cada uno: cada uno lleva sus
repos, su workspace, su space y, si hace falta, su perfil. Cambiar de
proyecto (`-p`, o elegirlo en Proyectos) cambia todo junto.

### Perfil (opcional)

Un perfil dice **de dónde salen las tareas** (el tracker, hoy Jira) y **a
dónde van los documentos** (Confluence con su space, o una carpeta Markdown).
Opcionalmente, también qué plantillas usar.

- Una configuración sin perfiles funciona como el perfil implícito
  `atlassian`: Jira y Confluence con `confluence_space_key`, como gtt.
- Sirve cuando necesitas otro destino (una carpeta Markdown para revisar
  antes de publicar), otro space para todo un equipo u otra plantilla.
- Al generar, el perfil sale de `--profile`, si no del proyecto
  (`profile:`) y si no del perfil por defecto. Así un proyecto puede
  documentarse con otra plantilla u otro destino sin pasar `--profile`
  cada vez.
- El consolidado QA y Exportar no trabajan sobre un proyecto: usan el
  perfil por defecto, u otro con `--profile`.
- Un perfil que usa algún proyecto no se puede eliminar: primero cámbiale
  el perfil a esos proyectos.
- Al agregar el primer perfil, el implícito también se guarda, para que
  nada cambie. Con perfiles, el space por defecto sale de cada perfil
  (`docs.space`) y `confluence_space_key` del nivel principal deja de usarse;
  `--space` y el `confluence_space_key` de un proyecto siguen ganando sobre
  el del perfil.

### El archivo de configuración

Ejemplo con los campos principales y datos ficticios. No hace falta
escribirlo a mano:
Configuración (modo visual), `draco init`, `draco project` y `draco profile`
lo crean. En las rutas, `~` al inicio es tu carpeta personal (`~/code`, o
`~\code` en Windows).

```yaml
# Lo necesario, lo mismo que gtt
atlassian_email: tu@example.test
atlassian_token: <tu token>
base_url: https://empresa.atlassian.net
confluence_space_key: DEV                # el space de los documentos (sin perfiles)

default_project: pagos
projects:
  pagos:
    backend_path: /home/tu-usuario/code/pagos-api
    backend_repo: pagos-api
    frontend_path: /home/tu-usuario/code/pagos-web
    frontend_repo: pagos-web
    vcs_host: https://github.com
    vcs_org: mi-org
    confluence_space_key: PAG            # opcional: su space; si no, el del perfil o el general
    profile: equipo                      # opcional: su perfil; si no, el por defecto
    deploy_checklist:                    # opcional: reemplaza al global
      - Ejecutar migraciones
    backlog:                             # opcional: ajusta draco backlog scan
      backend:
        class_globs: ["app/**/Services/**/*.php"]   # clases que deben tener tests
        modules: ["app/Http/{modulo}/**"]           # {modulo}: la carpeta del módulo
      frontend:
        class_globs: ["src/app/**/*.service.ts"]
        modules: ["src/app/{modulo}/**"]
        # más opciones (since, exclude, max_lines…): draco help backlog

deploy_checklist:                        # «A considerar» de un documento nuevo
  - Respaldo de la base de datos

# Opcional: perfiles. Con perfiles, el space por defecto sale de cada uno
default_profile: equipo
profiles:
  equipo:
    tracker: { provider: jira }
    docs:    { provider: confluence, space: DEV }
  local:
    tracker: { provider: jira }
    docs:    { provider: markdown, path: /home/tu-usuario/documentos/despliegues }
    template: /home/tu-usuario/plantillas/equipo   # carpeta con deploy/ y qa/

# Opcional: solo para el consolidado QA
qa_email: qa@example.test                # la cuenta que publica el consolidado
qa_report: { lider_tecnico: Ana, pmo: Luis, qa: Sofía }
qa_jira:
  project: OPS
  statuses: ["10001", "En QA"]           # estados de una tarea que ya llegó a QA
  testing_status: Testing                # abre la ventana Kanban
  sprint_name: "{module}_Sprint {sprint}"
  fields:
    coding_errors: customfield_10001
    dev_returns: customfield_10002
    dev_status: customfield_10003
  review_prefixes: ["Revisión de Tarea"]
  observation_marker: "Novedad::"

theme: ansi                              # ansi, alto-contraste o mono
```

Guardar desde draco reescribe el archivo, así que los comentarios escritos a
mano se pierden.

### Variables de entorno

| Variable | Efecto |
|---|---|
| `ATLASSIAN_EMAIL`, `ATLASSIAN_TOKEN`, `ATLASSIAN_BASE_URL` | Ganan sobre el archivo. draco nunca las copia al archivo |
| `CONFLUENCE_SPACE_KEY` | Reemplaza el `confluence_space_key` del nivel principal, así que solo tiene efecto sin perfiles. El de un proyecto y `--space` ganan sobre ella. Nunca se copia al archivo |
| `DRACO_NO_UPDATE_CHECK=1` | Sin aviso de versión nueva |
| `NO_COLOR=1` | Sin colores, en los dos modos |

---

## 4. Modo visual

`draco` sin argumentos, en una terminal, abre el modo visual. En un pipe o en
CI muestra la ayuda. Funciona en Windows Terminal, PowerShell, Linux y macOS;
en la consola clásica de Windows (`cmd`) se ve más limitado.

### La pantalla principal

- A la izquierda, el **menú**. A la derecha (desde 96 columnas), un **panel**
  con qué hace la función elegida y el mismo trabajo en modo comando.
- Arriba, una línea de cabecera: la versión y el camino por donde llegaste
  (`Inicio › Proyectos`), y a la derecha el proyecto activo y su perfil (el
  que usa Generar; QA y Exportar usan el por defecto). Si la terminal es
  angosta, primero se quita el perfil, después el proyecto y, al final, las
  primeras pantallas del camino. Si hay una versión nueva, el aviso va junto
  a la versión.
- Abajo, el pie: las teclas principales de la pantalla actual y `? más`.
- Sobre el pie, tras publicar un documento, queda una línea con el resultado
  (`✓ OPS-7 publicado en «Despliegues»`) hasta que salgas de draco. Si el
  documento no se pudo enlazar desde la tarea, la línea lo avisa con `⚠`.

### Teclas en todas las pantallas

| Tecla | Acción |
|---|---|
| `ctrl+c` | Sale de draco, siempre |
| `esc` | Vuelve a la pantalla anterior |
| `?` | Muestra todas las teclas de la pantalla, también las que el pie compacto no trae (de nuevo, las oculta) |
| `q` | Sale, desde la principal |
| `enter` / `tab` / `shift+tab` | En un formulario: siguiente campo / siguiente / anterior |

### Barra de comandos

En la principal, `/` abre una línea para escribir un comando como en la
terminal, con `draco` o sin él: `/g -i OPS-7 -b a1b2c3d`.

- Mientras escribes, sugiere comandos, sus flags (con descripción) y los
  valores que conoce tu config: proyectos tras `-p`, perfiles tras
  `--profile` (sirve para copiar el comando: en el modo visual, Generar usa
  el perfil del proyecto), `backend` o `frontend` tras `-r`.
- `tab` completa la sugerencia, `↑↓` la elige, `enter` ejecuta y `esc`
  cierra.
- **Ejecutar abre la pantalla del comando con los datos cargados**, en el
  primer paso que tienes que revisar. `/g -i OPS-7 -b a1b2c3d` llega a
  Commits con el hash marcado, y `esc` vuelve al formulario con lo que
  escribiste.
- Un flag desconocido o sin valor es un error, como en la terminal. Los flags
  que el modo visual no necesita (`--dry-run`, `--template`, `--profile`,
  `-o`…) se
  explican sin abrir nada.
- `/help` muestra la ayuda general y `/help g`, la de un comando.

### Generar documento

1. **Tarea y proyecto.** Aquí se abre el perfil: si falta una credencial o un
   space, lo dice antes de seguir.
2. **Commits.** Muestra la lista de commits recientes de cada repo, con la
   rama en la que está el clon.
   - `espacio` marca un commit;
   - `tab` cambia entre backend y frontend;
   - `h` agrega un hash a mano (por ejemplo, de otra rama);
   - `enter` continúa.
3. **Preparar documento.** Lee la tarea, busca si ya hay un documento y lee
   los archivos de los commits. Si el documento existe, pregunta si
   actualizarlo, crear otro o cancelar. `esc` cancela el paso.
4. **Vista previa.** `tab` cambia entre el documento y los archivos. Salir de
   aquí no publica nada. Al actualizar, si el documento actual no tiene una
   sección que se conserva, lo avisa y pide `s` para seguir.
5. **Ubicación** (solo al crear). `/` filtra la lista y `enter` elige dónde
   crear el documento.
6. **Publicar.** `enter` publica. Mientras publica, `esc` no sale, para no
   dejar el documento sin su enlace en la tarea. Al terminar muestra la URL;
   si falló el enlace en Jira, lo dice.

Lo que el equipo escribió a mano en «A considerar» se conserva al actualizar.

### Consolidado QA

1. **Sprint y módulo.** Sin sprint, modo Kanban: las tareas que pasaron a
   Testing en los últimos 10 días hábiles.
2. **Evaluación.** Se suma una fila por tarea a medida que se evalúa. `esc`
   cancela entre tareas. En las columnas, `✓` es un control que pasa y `✗`
   uno que falla. `?` significa que la consulta falló: nunca se publica como
   un control fallido.
3. `enter` abre la **vista previa**, y después **Ubicación** y **Publicar**,
   como en Generar.

Solo la cuenta de `qa_email` publica el consolidado. Es una protección contra
errores, no un permiso: los permisos reales son los de Atlassian. Un
consolidado con el mismo título se actualiza, y se conserva lo escrito a mano
en «Observación» y «Consideraciones».

### Exportar página

1. **Tarea y space o carpeta** (vacío: el del perfil; en Confluence sin
   space, todos).
2. **Documentos** que mencionan la tarea, el más reciente primero. `/`
   filtra; con uno solo, sigue sin preguntar.
3. **Texto.** `↑↓` desplaza, y el campo de abajo tiene la ruta donde se
   guarda (la carpeta actual y un nombre sugerido). `enter` guarda; si el
   archivo existe, pide `s` para reemplazarlo.

### Backlog técnico

1. **Formulario:** proyecto, repos, carpeta y, en la segunda página,
   período, módulos y dependencias.
2. **Análisis**, repo por repo. `esc` lo cancela, y un repo que falla no
   detiene al otro.
3. **Hallazgos:**
   - `tab` cambia de repo;
   - `/` filtra por texto;
   - el detalle del elegido aparece al lado o debajo;
   - `s` guarda el reporte completo en JSON.

No usa internet (salvo la revisión de dependencias) ni credenciales.

### Proyectos

Muestra la lista con filtro y el detalle del proyecto elegido. Se abre desde
la principal o desde Configuración.

| Tecla | Acción |
|---|---|
| `enter` | Lo usa en esta sesión, con su perfil, que la cabecera muestra (no cambia la config) |
| `d` | Lo guarda como proyecto por defecto |
| `n` | Crea un proyecto |
| `e` | Edita el proyecto (conserva el checklist y el resto del backlog) |
| `x` | Elimina el proyecto (pide `s`) |
| `/` | Filtra la lista |

El pie muestra `enter`, `n`, `e` y `/`; `d` y `x` aparecen al pulsar `?`.

Si tienes dos o más perfiles, el formulario pregunta el **perfil del
proyecto** («El perfil por defecto» o uno de la lista). Termina con una
página opcional por repo con ruta, **Backlog técnico**: las clases que deben tener tests (`class_globs`) y los módulos
(`modules`), separados por comas. Dejarla en blanco (`enter` en cada campo)
la salta. Un patrón mal
escrito se marca ahí mismo y no se guarda. El resto del backlog (`since`,
`exclude`, `max_lines`…) se edita en el YAML y se conserva.

### Configuración

Arriba, lo que necesita el documento de despliegue; debajo de **Opcional**,
lo demás. Sin configuración arranca como asistente; si encuentra la de gtt,
empieza por **Importar de gtt** (ver «¿Vienes de gtt?» en la sección 2).

| Sección | Qué contiene |
|---|---|
| **Conexión** | Email, token, URL y, sin perfiles, el space de Confluence. Al guardar, verifica la cuenta y ofrece corregir el email si no es el de la cuenta; si no hay proyectos, `enter` sigue con el primero. Un token vacío conserva el actual. Si una variable `ATLASSIAN_*` o `CONFLUENCE_SPACE_KEY` gana sobre el archivo, lo avisa |
| **Proyectos** | La misma pantalla de Proyectos |
| **Perfiles** (opcional) | Lista con `d`, `n`, `e` y `x`, como Proyectos. Sin perfiles, dice qué se usa: Jira → Confluence con el space de Conexión |
| **QA** (opcional) | `qa_email`, los nombres del encabezado y `qa_jira`, en cuatro páginas. Las listas van separadas por comas. Si falta algo, muestra todos los problemas a la vez y no guarda |
| **Tema** (opcional) | `↑↓` prueba cada tema en toda la pantalla, `enter` guarda y `esc` vuelve al anterior |

Los temas son tres:

- `ansi`: los colores de tu terminal, por defecto;
- `alto-contraste`: colores brillantes y negrita en lo importante;
- `mono`: sin colores.

---

## 5. Modo comando

Para scripts, CI y quien ya conoce los comandos. Los flags aceptan
`-i OPS-7`, `-i=OPS-7` y `--issue OPS-7`. Un flag desconocido es un error,
nunca se ignora.

| Código de salida | Significado |
|---|---|
| `0` | Bien |
| `1` | Error |
| `2` | Uso incorrecto (flag desconocido, falta un valor) |

| Comando | Para qué |
|---|---|
| `draco g`, `gen`, `generate` | Documento de despliegue |
| `draco qa` | Consolidado QA |
| `draco f`, `fetch` | Exportar un documento a texto |
| `draco backlog scan` | Deuda técnica |
| `draco init` | Configuración inicial |
| `draco project` | Proyectos: `list`, `add`, `default`, `remove` |
| `draco profile` | Perfiles: `list`, `add`, `default`, `remove` |
| `draco template` | Plantillas: `init`, `list`, `validate`, `preview` |
| `draco update` | Actualizar draco |
| `draco version` | Versión instalada |

### Documento de despliegue

```bash
draco g -i OPS-7 -b a1b2c3d -f e4f5a6b              # backend y frontend
draco g -i OPS-7 -b a1b2c3d,9f8e7d6                 # varios commits
draco g -i OPS-7 -b a1b2c3d -p pagos --space DEV    # otro proyecto y space
draco g -i OPS-7 -b a1b2c3d --profile local         # a la carpeta Markdown del perfil
draco g -i OPS-7 -b a1b2c3d --dry-run               # vista previa sin publicar
```

Hace falta al menos uno de `-b` o `-f`. El space sale de `--space`, si no del
proyecto y si no del perfil. Con un perfil Markdown, la carpeta sale de
`--space` o del perfil: el space del proyecto es solo para Confluence. Si el
documento existe, pregunta si actualizarlo. El perfil sale de `--profile`,
si no del proyecto y si no del perfil por defecto.

### Consolidado QA

```bash
draco qa                          # pregunta el sprint; Enter: Kanban (últimos 10 días hábiles)
draco qa -s 12 -m Pagos           # Sprint 12 del módulo Pagos
draco qa -s 12 -m Pagos --dry-run
```

Necesita `qa_email` y `qa_jira` en la config (ver la sección 3).

### Exportar

```bash
draco f -i OPS-7                  # abre el diálogo de guardar del sistema
draco f -i OPS-7 -o despliegue.txt
draco f -i OPS-7 --space DEV
```

### Backlog técnico

```bash
draco backlog scan                          # los repos del proyecto por defecto
draco backlog scan -r frontend --since 30d  # solo el frontend, último mes
draco backlog scan -m pagos                 # solo un módulo
draco backlog scan -l 0                     # todos los hallazgos
draco backlog scan -o                       # guarda el reporte completo en JSON
draco backlog scan --path ~/repos/pagos-api # una carpeta que no es un proyecto
```

Detecta:

- **hotspots:** archivos con muchos commits;
- **clases sin tests:** requiere `class_globs` en la config del proyecto
  (en el modo visual, la página Backlog técnico del formulario de proyecto);
- **TODO/FIXME/HACK** pendientes;
- **archivos grandes;**
- **dependencias vulnerables:** con `--deps`, usa internet.

La configuración opcional por repo (`backlog:` dentro del proyecto) está en
`draco help backlog`.

---

## 6. Plantillas

La estructura de los documentos sale de plantillas editables. Puedes cambiar
el checklist, el título o el documento entero sin recompilar.

```bash
draco template init deploy          # a .draco/templates/deploy de la carpeta actual: córrelo en el repo backend
draco template init qa --user       # a ~/.config/draco/templates/qa, para todos tus repos
draco template list                 # qué carpetas se buscan y cuál está en uso
draco template validate             # la llena con datos de ejemplo y avisa si falta algo
draco template preview qa           # muestra el resultado con datos de ejemplo
```

Gana la primera carpeta que exista:

1. `--template`, o la carpeta de plantillas del perfil;
2. `.draco/templates/<tipo>` del repo: en `deploy`, el backend del proyecto
   (`backend_path`); en `qa`, la carpeta actual;
3. `~/.config/draco/templates/<tipo>`.

La plantilla incluida es la base: una carpeta solo reemplaza lo que trae.

Si cambias el `prefix` de la plantilla de despliegue (el inicio fijo de
todos los títulos), el consolidado QA lo sigue reconociendo: busca el enlace
al documento con el prefijo de cada plantilla que tu configuración puede
usar (la de cada proyecto, con su perfil y su repo) y con el de siempre, así
que también cuentan los documentos publicados antes del cambio.

Las plantillas son Markdown. Para lo que Markdown no expresa y Confluence sí,
hay tres convenciones (en un destino Markdown no se ven):

| Para | Escribe | En Confluence |
|---|---|---|
| Fijar el ancho de las columnas | `<!-- draco:tabla anchos=63,697 -->` justo antes de la tabla (opcional: `ancho=1800`, `layout=full-width`) | Columnas con esos anchos, como en gtt |
| Viñetas dentro de una celda | Cada línea con `- ` al inicio, separadas por `<br>`: `- {{join .Files "<br>- "}}` | Lista con viñetas |
| Un bloque dentro de un recuadro | `<!-- draco:recuadro -->` justo antes del bloque | Tabla de una celda con el bloque dentro |

Un comentario `draco:` mal escrito se publica como texto, para que lo veas y
lo corrijas.

---

## 7. Solución de problemas

| Mensaje | Qué hacer |
|---|---|
| «configuración incompleta de Atlassian. Corre: draco init» | Faltan el email, el token o la URL: completa Configuración › Conexión o corre `draco init` |
| «No se pudo validar las credenciales» | Revisa el email, el token (que no haya expirado) y que la URL empiece con `https://` |
| «Confluence space key no configurado» | Sin perfiles, pon el space en Configuración › Conexión; con perfiles, en el perfil (`docs.space`). Al generar también sirven el space del proyecto (Proyectos, `e`) o `--space`; en `draco qa`, `--space` o el del perfil |
| «este comando está pensado para la cuenta de QA y no coincide con la tuya» | Te autenticaste con otra cuenta. Si te toca publicar, ajusta `qa_email` en Configuración › QA |
| «no se encontraron documentos de despliegue previos» (o «ningún consolidado QA previo») | draco crea cada documento junto a uno anterior del mismo tipo en el space. La primera vez, crea uno a mano en Confluence como referencia, o usa un perfil con carpeta Markdown |
| «configuración de QA incompleta» o «falta la sección 'qa_jira'» | Completa Configuración › QA: lista todos los campos que faltan |
| «proyecto 'x' no encontrado» | Revisa el nombre con `draco project list` |
| «flag desconocido» | Revisa `draco help <comando>`. En la barra, las sugerencias muestran los flags válidos |
| `draco` no se encuentra tras instalar | Abre una terminal nueva: el PATH se actualiza ahí. En Linux y macOS, también `source ~/.bashrc` (o `~/.zshrc` con zsh) |
| «el SHA-256 … no coincide» al instalar o actualizar | La descarga está incompleta o alterada. Vuelve a intentarlo; si se repite, abre un issue |
| La TUI se ve con caracteres raros | Usa Windows Terminal en lugar de `cmd`, con una fuente con Unicode. Si el problema son los colores, prueba el tema `mono` o `NO_COLOR=1` |

Para reportar un error:
[issues del repo](https://github.com/kaelence-01/draco-hq/issues). Incluye
la versión (`draco version`), el comando o la pantalla y el mensaje. **Nunca
pegues tu token** ni datos de tu empresa.

---

## 8. Desinstalar

1. Borra el ejecutable:
   - Windows: `%LOCALAPPDATA%\Programs\draco`;
   - Linux y macOS: `~/.local/bin/draco`.
2. Quita esa carpeta del PATH:
   - Windows: Configuración › Variables de entorno de tu usuario;
   - Linux y macOS: la línea bajo `# draco` en `~/.bashrc` o `~/.zshrc`.
3. Si no la vas a usar más, borra `~/.config/draco/`, que tiene tu
   configuración, tu token y las plantillas propias.
