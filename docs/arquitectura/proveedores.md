# Arquitectura de proveedores y plantillas

draco no queda atado a Atlassian: el núcleo trabaja con un modelo propio y cada
plataforma (Jira, Confluence, GitHub, GitLab, Azure DevOps, Linear, Notion…) se
conecta con un adaptador. La estructura del documento deja de estar en código:
es una plantilla que cada equipo elige y ajusta.

Estado: **en implementación** (roadmap, fase 2). Las secciones 3 y 4 reflejan
lo implementado; el resto es diseño y se actualiza al implementarlo.

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

## 3. Modelo neutro

Implementado en `internal/provider/provider.go` (actividades 3a y 3b): `Issue`,
`User`, `DocRef`, `Document` (título, Markdown y secciones a conservar),
`LocationQuery` / `Locations`, `Exported` (texto + metadatos, con `Source`, el
nombre de la plataforma), `QAIssue` / `QAScope` y `Commit`. Desde la actividad
4b el flujo nunca ve el formato nativo del destino.

## 4. Puertos

```go
type Tracker interface {
    Issue(ctx context.Context, key string) (*Issue, error)
    Me(ctx context.Context) (*User, error)
    LinkDocument(ctx context.Context, issueKey string, doc DocRef) error
}

type DocStore interface {
    FindByTitle(ctx context.Context, title string) (*DocRef, error)
    FindByTitleKey(ctx context.Context, prefix, key string) (*DocRef, error)
    Sections(ctx context.Context, ref DocRef, names []string) ([]string, error)
    Locations(ctx context.Context, q LocationQuery) (*Locations, error)
    Create(ctx context.Context, near DocRef, doc Document) (*DocRef, error)
    Update(ctx context.Context, ref DocRef, doc Document) (*DocRef, error)
    Search(ctx context.Context, text string) ([]DocRef, error)
    SearchTitle(ctx context.Context, text string) ([]DocRef, error)
    Export(ctx context.Context, ref DocRef) (*Exported, error)
}

type QATracker interface {
    Tracker
    ReviewTasks(ctx context.Context, scope QAScope) ([]QAIssue, error)
    AssignedTasks(ctx context.Context, scope QAScope, email string) ([]QAIssue, error)
    HasDocumentLink(ctx context.Context, issueKey, prefix string) (bool, error)
    Observations(ctx context.Context, issueKey string) (string, error)
}

type ChangeSource interface {
    ChangedFiles(ctx context.Context, workDir string, hashes []string) ([]string, error)
    RecentCommits(ctx context.Context, workDir string, n int) ([]Commit, error)
}
```

- `context.Context` en todos los puertos, el cliente HTTP y los pasos de las
  `Session`: la TUI necesita cancelar.
- El destino se construye ya acotado (un space de Confluence); el flujo no ve
  spaces ni páginas padre: `Create` recibe el documento junto al que va el
  nuevo y el adaptador resuelve dónde queda.
- Las convenciones de draco (prefijo «Documento de Despliegue») las pasa el
  flujo como parámetro (`FindByTitleKey`, `LocationQuery.TitlePrefix`); el
  adaptador no las conoce. Pasarán a la plantilla en la actividad 5.
- **`QATracker`** amplía `Tracker` con lo que necesita el consolidado QA
  (`ReviewTasks`, `AssignedTasks` por `QAScope`, `HasDocumentLink`,
  `Observations`). El esquema de Jira se le da al construirlo
  (`atlassian.NewQATracker(c, qa_jira)`). Hoy el llamador construye el
  adaptador directamente; detectar la capacidad con una aserción de tipo sobre
  un `Tracker` cualquiera llega con los perfiles (actividad 7).
- **`ChangeSource`** (`ChangedFiles`, `RecentCommits`); `git.Local` es el clon
  local.
- **`DocStore.SearchTitle`** busca por título en todo el destino (CQL en
  Confluence). Lo usa el consolidado QA para sus ubicaciones: `Locations` solo
  revisa las páginas más nuevas, y un consolidado se crea una vez por sprint.
- **`Document` y `Sections`** (actividad 4b): el flujo publica Markdown y
  declara en `Preserve` las secciones escritas a mano; `Sections` le dice antes
  cuáles tiene el documento actual (§6).
- **Markdown local** (`markdown.Store`, actividad 6): cada documento es un
  archivo `.md` cuya primera línea es `# <título>`; el ID es la ruta relativa
  a la carpeta. `Locations` devuelve carpetas (las que tienen documentos con
  el prefijo, más la raíz, para que una carpeta vacía reciba el primero) y
  `Create` escribe en la carpeta elegida o en la del documento elegido. La
  preservación corta el Markdown por títulos, sin contar los que están dentro
  de bloques de código. Nada fuera de la carpeta se lee ni se escribe, y cada
  escritura pasa por un archivo temporal.
- Tests: cada adaptador con sus propias pruebas (Confluence con `httptest`,
  Markdown con carpetas temporales) y el flujo `deploy` real de punta a punta
  sobre Markdown. Una batería común de contrato necesitaría un Confluence
  simulado completo; se decide cuando llegue un tercer destino.

## 5. Documento: plantilla → Markdown → formato del destino

1. La plantilla (`text/template` de Go) produce **Markdown**.
2. El adaptador de destino lo convierte a su formato (ADF en Confluence, bloques
   en Notion; Markdown local lo escribe tal cual). Para leer el Markdown se usa
   `goldmark` con extensiones GFM.

**Markdown mínimo garantizado:** títulos, párrafos, negrita, cursiva, código,
enlaces, listas, listas de tareas (`- [ ]`) y tablas. `draco template validate`
avisa lo que un destino no soporta.

**Conversión a ADF** (`atlassian.MarkdownToADF`, actividad 4a). Lo que Markdown
no expresa y el documento necesita, con la convención que lo recupera
(aprobadas por el autor):

| Necesidad | Convención en el Markdown | En Confluence |
|---|---|---|
| Tarjeta de enlace a un issue | Enlace cuyo texto es la URL, o `<https://…>` | inlineCard |
| Emoji ✔ / ✖ de los controles QA | `✔` / `✖` | Emoji de Confluence |
| Tabla clave/valor sin fila de encabezado | Fila de encabezado con todas las celdas vacías | Se omite la fila |
| Primera columna resaltada | Celda cuyo contenido completo está en **negrita** | Celda de encabezado, sin la marca |
| Varias líneas en una celda | `<br>` | Salto de línea |
| Ancho de tablas | — | Todas a todo el ancho, columnas automáticas (anchos por plantilla en la actividad 5) |

Se pierden, aceptado: las viñetas dentro de celdas (pasan a líneas), las
casillas dentro de una tabla (van debajo del título), el gris de los
marcadores y los anchos de columna fijos. Cualquier otro HTML queda como
texto literal, nunca se interpreta.

Por eso el criterio de «documento idéntico» aplica al pasar a puertos, cuando
el builder no cambia; desde el paso a Markdown el criterio es **contenido
equivalente**, con estas diferencias aceptadas.

## 6. Secciones escritas a mano

«A considerar» del documento de despliegue, y «Observación» y
«Consideraciones» del consolidado QA, las escribe la gente, no draco. La
preservación es **responsabilidad del adaptador de destino**, en su formato
nativo: así el estado de las tareas marcadas y el formato se copian tal cual,
sin pasar por Markdown.

Implementado en la actividad 4b:

- El flujo declara qué secciones son del usuario (`Document.Preserve`); con
  las plantillas lo hará el manifiesto (`preserve:`).
- Antes de actualizar, el flujo pregunta al destino cuáles tiene
  (`Sections`) y, si falta alguna, la UI pregunta.
- Una sección existe cuando existe su título, aunque esté vacía: una
  «Observación» recién creada no hace preguntar en cada actualización, y si QA
  la dejó vacía, se conserva vacía.
- En `Update`, el adaptador de Confluence lee la página actual, extrae esas
  secciones en ADF (hasta el siguiente título del mismo nivel o superior) y
  las pone en lugar de las de la plantilla, sobre la versión que leyó. Si no
  puede leer la página, no actualiza.
- Migración: en los consolidados publicados antes de 4b la «Observación» era
  una celda de la tabla Resumen; si no hay sección, el adaptador lee esa
  celda.

## 7. Plantillas

Implementado en la actividad 5 (`internal/document`):

- Una plantilla es una carpeta con `template.yaml` y archivos `*.md.tmpl`
  (`text/template`). El manifiesto tiene `title` (plantilla del título),
  `prefix` (inicio fijo de todo título, en el que se apoyan las búsquedas;
  un título que no lo respeta es un error) y `preserve` (secciones escritas a
  mano).
- Las plantillas del documento de despliegue (`deploy`) y del consolidado
  (`qa`) vienen incluidas en el binario (`embed.FS`) y reproducen el documento
  anterior idéntico; el único cambio fue la fecha del consolidado, ahora en
  español («21 sep 2026»).
- Resolución: la carpeta explícita (`Request.Template`, luego el flag
  `--template`), `.draco/templates/<tipo>/` del repo y
  `~/.config/draco/templates/<tipo>/`; la primera que exista personaliza la
  incluida. Sus campos de `template.yaml` reemplazan los incluidos y sus
  `*.md.tmpl` se leen encima: un `{{define "checklist"}}` reemplaza ese
  `{{block}}`, y un `main.md.tmpl` reemplaza el documento completo. Bloques
  incluidos: `encabezado`, `archivos`, `checklist` (despliegue) e
  `informacion`, `tabla`, `resumen` (consolidado).
- Seguridad: la plantilla recibe los valores **ya escapados** como Markdown
  (una plantilla del usuario no puede romper una tabla aunque no escape), y la
  única función disponible es `join`. Los secretos nunca son datos de
  plantilla.
- «Revisión de Tarea» y «Novedad::» no son del documento sino del proceso de
  QA en Jira: pasan a `qa_jira.review_prefixes` y `qa_jira.observation_marker`,
  con los valores anteriores por defecto.
- Comandos `draco template init/list/validate/preview` y plantilla por
  perfil: fase 3, actividad 6b (bitácora 023). Pendiente: plantillas remotas (Git
  con versión fija o URL con checksum SHA-256).

## 8. Configuración con perfiles

Implementado en la actividad 7 (`config.Profile`, `internal/profile`):

```yaml
default_profile: equipo

profiles:
  equipo:
    tracker: { provider: jira }
    docs:    { provider: confluence, space: DEV }
  externo:
    tracker: { provider: jira }
    docs:    { provider: markdown, path: ./docs/despliegues }
    template: ./plantillas      # deploy/ y qa/ dentro, como ~/.config/draco/templates
```

- **Migración en memoria:** una config sin `profiles` se lee como el perfil
  `atlassian` (Jira + Confluence con `confluence_space_key`); el archivo no se
  reescribe. Con un solo perfil no hace falta `default_profile`.
- `profile.Open(cfg, Options{Name, Scope, AnyScope})` arma los adaptadores
  (Jira, Confluence o Markdown local, y git local). Jira y Confluence usan
  `base_url` y las credenciales globales (variables de entorno o el archivo
  0600, como antes).
- **El space es del perfil, no del flujo:** los flujos solo resuelven lo que
  pidió el usuario (flag o proyecto) en `Target.SpaceKey`; `Open` completa con
  el del perfil y devuelve `ErrNoSpace` solo si el destino es Confluence y no
  hay ninguno (`AnyScope`, para exportar, busca en todos). Así un perfil
  Markdown no exige un space.
- **QA como capacidad:** `Bundle.QA()` devuelve el `QATracker`, el error de
  `qa_jira` si está incompleto o `ErrNoQA` si el tracker no lo soporta.
- Proyectos y perfiles se editan con funciones puras (`SetProject`,
  `RemoveProject`, `SetDefaultProject` y las de perfiles) y se guardan con
  `config.Save`: las usarán `draco init` y los formularios de la TUI.
- Pendiente: URL y credenciales por perfil (hoy son globales), y el gestor de
  secretos del sistema.

## 9. Prioridad de plataformas

| Plataforma | Rol | Prioridad |
|---|---|---|
| Jira, Confluence, git local | Tracker, destino, cambios | Actual |
| Markdown local | Destino | Hecho (actividad 6) |
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
