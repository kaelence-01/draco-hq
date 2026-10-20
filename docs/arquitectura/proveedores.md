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

Implementado en `internal/provider/provider.go` (actividad 3a): `Issue`,
`User`, `DocRef` (con la versión leída), `Doc` (referencia + cuerpo),
`LocationQuery` / `Locations` y `Exported` (texto + metadatos, con `Source`, el
nombre de la plataforma).

**Transición:** `Doc.Body` y el cuerpo que reciben `Create` y `Update` son el
formato nativo del destino (ADF en Confluence) hasta que el documento pase a
Markdown (actividad 4); la preservación de secciones sigue en los flujos hasta
entonces. Así los goldens se mantienen idénticos en el cambio a puertos.

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
    Read(ctx context.Context, ref DocRef) (*Doc, error)
    Locations(ctx context.Context, q LocationQuery) (*Locations, error)
    Create(ctx context.Context, near DocRef, title string, body map[string]any) (*DocRef, error)
    Update(ctx context.Context, ref DocRef, title string, body map[string]any) (*DocRef, error)
    Search(ctx context.Context, text string) ([]DocRef, error)
    Export(ctx context.Context, ref DocRef) (*Exported, error)
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
- **Pendiente (3b):** capacidad opcional `QATracker` (consultas del
  consolidado, observaciones, enlace al documento), detectada con una aserción
  de tipo, y `ChangeSource` (archivos de un commit, commits recientes).
- **Pendiente (4):** `Sections` para preguntar qué secciones escritas a mano
  encuentra el destino, y cuerpo en Markdown.
- Tests: el adaptador se prueba con `httptest` (sin red). La batería común de
  contrato llega con el segundo adaptador (actividad 6).

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
