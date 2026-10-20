# Arquitectura de proveedores y plantillas

draco no queda atado a Atlassian: el núcleo trabaja con un modelo propio y cada
plataforma (Jira, Confluence, GitHub, GitLab, Azure DevOps, Linear, Notion…) se
conecta con un adaptador. La estructura del documento deja de estar en código:
es una plantilla que cada equipo elige y ajusta.

Estado: **diseño aprobado, en implementación** (roadmap, fase 2). Los nombres de
tipos e interfaces son un borrador: se fijan al implementarlos y este documento
se actualiza con lo que quede.

## 1. Roles

draco usa las plataformas en roles distintos que se combinan libremente (Jira
con Notion, GitHub Issues con Markdown en el repo):

| Rol | Qué hace draco con él | Hoy | Otros previstos |
|---|---|---|---|
| **Tracker** | Leer un issue, identificar al usuario, enlazar el documento, leer comentarios | Jira | GitHub Issues, GitLab Issues, Azure Boards, Linear |
| **Destino de documentos** | Buscar, leer, crear y actualizar páginas; elegir dónde crearlas | Confluence | Markdown local, wiki de GitHub/GitLab, Notion, Azure Wiki, SharePoint |
| **Fuente de cambios** | Commits y archivos tocados | git local | PRs de Bitbucket, GitHub, GitLab, Azure Repos |
| **Plantillas** | De dónde sale la estructura del documento | (en código) | Incluida, carpeta del repo, carpeta del usuario, Git con versión fija |
| **Notificación** | Avisar que se publicó | — | Teams, Slack, webhook genérico |

## 2. Paquetes

```
internal/
  provider/            modelo neutro + interfaces de los puertos
    atlassian/         adaptador Jira + Confluence (hoy internal/atlassian)
    markdown/          destino Markdown local (sin red)
  template/            resolución y render de plantillas → Markdown
  deploy/ qa/ fetch/   flujos: solo importan provider y template
```

Regla: **un flujo nunca importa un adaptador.** Los adaptadores se compilan
dentro del binario y se eligen por nombre desde la config.

## 3. Modelo neutro (borrador)

```go
type Issue struct {
    Key, Title, URL, Status string
}

type User struct {
    ID, Email, Name string
}

// DocRef identifies a document in a destination.
type DocRef struct {
    ID, Title, URL string
    Version        int // optimistic concurrency where the destination has it
}

// Location is where a new document can be created: a Confluence parent page,
// a directory, a Notion database.
type Location struct {
    ID, Title, URL string
}

// Document is what a flow publishes: Markdown rendered from a template.
type Document struct {
    Title    string
    Markdown string
    Preserve []string // headings of sections the user edits by hand
}
```

## 4. Puertos (borrador)

```go
type Tracker interface {
    Issue(ctx context.Context, key string) (*Issue, error)
    Me(ctx context.Context) (*User, error)
    LinkDocument(ctx context.Context, issueKey string, doc DocRef) error
    HasDocumentLink(ctx context.Context, issueKey string) (bool, error)
}

type DocStore interface {
    FindByTitle(ctx context.Context, title string) (*DocRef, error)
    Search(ctx context.Context, text string) ([]DocRef, error)
    Locations(ctx context.Context, hint LocationHint) ([]Location, error)
    Export(ctx context.Context, ref DocRef) (*Exported, error) // text + metadata
    // Sections reports which of the named hand-edited sections the current
    // page has, so the caller can ask before an update that would lose them.
    Sections(ctx context.Context, ref DocRef, names []string) (found []string, err error)
    Create(ctx context.Context, at Location, doc Document) (*DocRef, error)
    Update(ctx context.Context, ref DocRef, doc Document) (*DocRef, error)
}

type ChangeSource interface {
    ChangedFiles(ctx context.Context, repo string, hashes []string) ([]string, error)
    RecentCommits(ctx context.Context, repo string, n int) ([]Commit, error)
}
```

- `context.Context` en todos los puertos: la TUI necesita cancelar.
- **Capacidades opcionales** como interfaces aparte, detectadas con una
  aserción de tipo: `QATracker` (consultas del consolidado QA, solo Jira por
  ahora), `CommentReader` (observaciones «Novedad::»). Un flujo que necesita una
  capacidad que el tracker no tiene devuelve un error centinela que la UI
  explica.
- Cada adaptador pasa la misma batería de **tests de contrato** con servidores
  simulados (`httptest`); ningún test usa una instancia real.

## 5. Documento: plantilla → Markdown → formato del destino

1. La plantilla (`text/template` de Go) produce **Markdown**.
2. El adaptador de destino lo convierte a su formato (ADF en Confluence, bloques
   en Notion; Markdown local lo escribe tal cual). Para leer el Markdown se usa
   `goldmark` con extensiones GFM.

**Markdown mínimo garantizado:** títulos, párrafos, negrita, cursiva, código,
enlaces, listas, listas de tareas (`- [ ]`) y tablas. `draco template validate`
avisa lo que un destino no soporta.

**Lo que Markdown no expresa y hoy está en el ADF**, con cómo lo recupera el
adaptador de Confluence:

| Hoy en ADF | Recuperación |
|---|---|
| Tarjeta de enlace (inlineCard) a un issue | Un enlace cuyo texto es la propia URL se convierte en tarjeta |
| Emoji ✔ / ✖ de los controles QA | `✔`/`✖` en el Markdown se mapean al emoji de Confluence |
| Ancho de tablas y columnas | Valores por defecto del adaptador (configurables por plantilla en el manifiesto) |
| Texto gris en cursiva de los marcadores | Se pierde: queda en cursiva normal |

Por eso el criterio de «documento idéntico» aplica al pasar a puertos, cuando
el builder no cambia; desde el paso a Markdown el criterio es **contenido
equivalente**, con las diferencias de esta tabla aceptadas.

## 6. Secciones escritas a mano

«A considerar» del documento de despliegue, y «Observación» y
«Consideraciones» del consolidado QA, las escribe la gente, no draco. La
preservación es **responsabilidad del adaptador de destino**, en su formato
nativo: así el estado de las tareas marcadas y el formato se copian tal cual,
sin pasar por Markdown.

- La plantilla declara qué secciones son del usuario (`preserve:` en el
  manifiesto).
- Antes de actualizar, el flujo pregunta al destino cuáles encuentra
  (`Sections`) y, si falta alguna, la UI pregunta.
- En `Update`, el adaptador lee la página actual, extrae esas secciones en su
  formato y las inserta en el cuerpo nuevo. Si no puede leer la página, no
  actualiza.

## 7. Plantillas

- Una plantilla es una carpeta: `template.yaml` (nombre, versión, datos que
  requiere, secciones a preservar, variables) y `*.md.tmpl`.
- Resolución, la primera que exista gana: flag `--template`, perfil activo,
  `.draco/templates/` del repo, `~/.config/draco/templates/`, plantilla incluida
  en el binario (`embed.FS`).
- La plantilla incluida reproduce el documento actual. Reemplaza el punto
  «convenciones de texto configurables»: títulos, «Revisión de Tarea»,
  «Novedad::» y el título del consolidado pasan a la plantilla o a su manifiesto.
- Herencia con `{{ block }}` / `{{ define }}`: una plantilla extiende otra y
  redefine solo lo que cambia.
- Seguridad: solo funciones registradas por draco; plantillas remotas solo con
  versión fija o checksum SHA-256; los secretos nunca son variables de
  plantilla.

## 8. Configuración con perfiles

```yaml
version: 2
default_profile: equipo

profiles:
  equipo:
    tracker: { provider: jira, url: https://empresa.atlassian.net }
    docs:    { provider: confluence, space: DEV }
    template: deploy-estandar
  externo:
    tracker: { provider: github, repo: mi-org/pagos-api }
    docs:    { provider: markdown, path: ./docs/despliegues }
```

- El `config.yaml` actual se migra solo a un perfil equivalente.
- Credenciales: variables de entorno o el archivo con permisos 0600, como hoy.
  Llevarlas al gestor de secretos del sistema necesita una dependencia y se
  decide en la actividad de perfiles.

## 9. Prioridad de plataformas

| Plataforma | Rol | Prioridad |
|---|---|---|
| Jira, Confluence, git local | Tracker, destino, cambios | Actual |
| Markdown local | Destino | Siguiente: valida el diseño sin red |
| GitHub Issues, Linear | Tracker | P1 |
| Bitbucket (PRs) | Cambios | P1 |
| GitLab Issues, Azure Boards | Tracker | P2 |
| GitHub / GitLab / Azure Repos | Cambios | P2 |
| Notion, SharePoint | Destino | P2 |
| Teams, Slack, webhook | Notificación | P2 |
| Wiki de GitHub/GitLab, Google Docs | Destino | P3 |

## 10. Decisiones

| Decisión | Motivo |
|---|---|
| Solo multiplataforma, sin multi-tenant | draco es una herramienta local; cada usuario trae sus credenciales |
| Adaptadores compilados en el binario | Un solo ejecutable; plugins externos (gRPC) solo si aparece la necesidad |
| `text/template` como motor | Biblioteca estándar, sin dependencias |
| `goldmark` para leer Markdown | Go puro, MIT, mantenido, soporta tablas y tareas GFM |
| La unidad sigue siendo el issue | Es lo que hace draco hoy; un documento por release queda como extensión |
| QA solo con Jira al inicio | Depende de campos personalizados sin equivalente directo en otros trackers |

Pendientes: mapeo de estados entre trackers (tabla fija o por perfil), gestor
de secretos del sistema, plantillas remotas.
