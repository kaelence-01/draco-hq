# CLAUDE.md

Guía para Claude Code (claude.ai/code) al trabajar en este repositorio.

## Qué es draco

**DRACO** es una herramienta de terminal open source (Go) para equipos que usan
Jira y Confluence:

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

Todavía no hay Makefile ni `main.go`: solo existe el core en `internal/`.

```bash
go build ./...           # compilar
go vet ./...             # análisis estático
gofmt -l internal        # debe salir vacío
go test ./...            # tests (todos puros, sin red)
```

Go: `go.mod` declara `go 1.26.1`. No cambiarlo sin verificar qué versión tienen
instalada los colaboradores y el CI.

## Arquitectura

```
internal/
  atlassian/   Cliente HTTP (Basic Auth) + Jira v3 + Confluence v1/v2,
               escapado JQL/CQL (query.go), conversión storage → texto
  config/      Config YAML + variables de entorno (env > archivo)
  document/    Construcción del ADF (deploy doc y QA) y preservación de secciones
  git/         git show, agrupación por carpeta, errores accionables
  backlog/     Detectores de deuda técnica (sin red)
  deploy/      Flujo del documento de despliegue, sin E/S de terminal
  qa/          Flujo del consolidado de QA, sin E/S de terminal
  fetch/       Flujo de exportar una página a texto, sin E/S de terminal
docs/
  roadmap.md                 Visión, fases, estado y decisiones
  security/patrones-seguros.md   Patrones P-001..P-008 (autoridad en seguridad)
```

Planeado (ver roadmap): `cmd/` (modo
comando), `internal/tui/` (modo visual), `internal/installer`,
`internal/updater`, `main.go`, `Makefile`.

### El patrón de los flujos: preparar → revisar → publicar

Cada flujo vive en su propio paquete (`internal/deploy` es la referencia) y
**no imprime, no lee stdin y no termina el proceso**:

- `Resolve(cfg, Request)` combina la petición con la config y devuelve un
  `Target` con sus `Warnings`.
- Una `Session` expone **un método por paso** (`FetchIssue`, `FindExisting`,
  `ReadCommits`, `DraftCreate`/`DraftUpdate`, `Locations`, `Publish`). Entre paso
  y paso decide quien llama: el modo comando pregunta por texto y la TUI con
  listas y formularios, sobre los mismos pasos.
- Atlassian entra por una **interfaz** definida en el paquete del flujo, con
  `var _ Atlassian = (*atlassian.Client)(nil)` para que no se desincronice. Los
  tests usan un doble (`fakeAPI`).
- Las situaciones degradadas pero viables se devuelven como `Warnings` o como
  campos (`Draft.Preserved`, `Draft.NotPreserved`, `Result.LinkErr`,
  `Locations.TeamWide`), nunca se
  imprimen desde el flujo.
- Los errores que el llamador debe explicar con su propia UI son centinelas
  (`ErrNoSpace`, `ErrNoLocations`, `ErrNotQAAccount`, `ErrNoPages`).

## Reglas que no se negocian

- **Nada de datos de una organización en el código** — URLs de instancia, space
  keys, project keys, IDs de estados o campos, nombres de repos, workspace del
  VCS, nombres de personas, pasos de despliegue de un stack. Todo va a
  `config.yaml`, con flag de override cuando tiene sentido. Incluye ejemplos en
  textos de ayuda y **datos de prueba**: usa nombres genéricos (`OPS-7`,
  `pagos-api`, `mi-org`, módulos `Pagos`/`Facturacion`). El repositorio es
  público. Autoridad: `docs/security/patrones-seguros.md`.
- **Todo valor interpolado en JQL o CQL pasa por `atlassian.quoteLiteral`.**
  Nunca un valor entre comillas escritas a mano en un format string.
- **Confluence v2 filtra spaces por `space-id` numérico, nunca por `space-key`.**
  Un parámetro desconocido se ignora en silencio y el filtro deja de filtrar.
  Resolver claves con `ResolveSpaceID`.
- **El ADF es `map[string]any` crudo**, sin structs tipados. Se envía como
  `body.value` (string JSON) con `representation: atlas_doc_format`.
- **Actualizar una página reemplaza todo su cuerpo.** Lo que el usuario edita a
  mano (la sección «A considerar») se lee y se vuelve a emitir
  (`document.ExtractSection`); en el consolidado QA, la «Observación» del
  Resumen y «Consideraciones» (`document.ExtractQAPreserved`). Si la página
  actual no se puede leer, no se sobrescribe: `DraftUpdate` devuelve error.
- **Una consulta fallida nunca se publica como un control fallido** en el
  reporte de QA: es evidencia. Se usa el estado desconocido
  (`QAIssue.DeployDocUnknown`).
- **Un error descartado cuyo efecto aparece después se reporta.** Si falla el
  enlace remoto en Jira, el documento existe pero QA lo marcará sin
  documentación semanas después: por eso viaja en `Result.LinkErr`.
- `qa_email` es una protección de UX, no control de acceso: lo lee de la config
  del propio usuario. La autorización real son los permisos de Atlassian.

## Dependencias

Hoy: solo `gopkg.in/yaml.v3`. Están aprobadas para la TUI las librerías de Charm
(Bubble Tea, Bubbles, Lip Gloss, Huh), y `golang.org/x/sys` cuando se porte el
instalador de Windows. Cualquier otra se justifica antes de agregarla. El router
de comandos será un `map[string]func([]string) error` como en gtt, sin Cobra.

## Idioma

- Interfaz y mensajes al usuario: **español**. Internacionalizar está pendiente
  de decisión (roadmap).
- Código, identificadores y comentarios: **inglés**, como en gtt. Los mensajes
  de los tests están en español.

## Tests

- `go test ./...` cubre funciones puras y los flujos con dobles. **Ningún test
  usa la red ni una instancia real de Atlassian.**
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
- Toda actividad lleva su entrada en **`docs/bitacora/NNN-<tema>.md`** en ese
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
