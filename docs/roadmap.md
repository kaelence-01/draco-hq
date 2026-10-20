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
| 2 | Core: portar la lógica de gtt, separar los flujos y desacoplar los proveedores | 🟡 en curso |
| 3 | Modo comando (`cmd/`, `main.go`), instalador, updater, Makefile, releases | ⬜ |
| 4 | Modo visual mínimo: pantalla principal, proyectos, generar documento | ⬜ |
| 5 | Resto de pantallas: backlog, QA, exportar | ⬜ |
| 6 | Barra de comandos con autocompletado, temas, ayuda | ⬜ |
| v1.0 | Instaladores, checksums, migración desde gtt, README | ⬜ |

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

### Fase 2 — siguiente: proveedores y plantillas

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
| 5 | Plantillas (`text/template`, manifiesto, resolución por fuentes, plantilla incluida) | `feature/plantillas` | La plantilla incluida reproduce el documento; otra plantilla lo cambia sin recompilar. Absorbe las convenciones de texto («Documento de Despliegue», «Revisión de Tarea», «Novedad::», título del consolidado) |
| 6 | Destino Markdown local (`internal/provider/markdown`) | `feature/salida-markdown` | Un documento de despliegue completo publicado sin Confluence |
| 7 | Config con perfiles y migración del `config.yaml` actual | `feature/perfiles` | Un `config.yaml` de hoy se migra solo; absorbe «proyectos y configuración» como funciones puras |

Listar commits recientes para el selector de la TUI entra en el puerto de
cambios (`ChangeSource.RecentCommits`) en la actividad 3b.

Después de la fase 3 (modo comando): segundo tracker (GitHub Issues o Linear),
segundo destino remoto (Notion o SharePoint), PRs de Bitbucket, notificaciones
y plantillas remotas, en ese orden según demanda (diseño, §9).

### Fase 3 — modo comando

- `main.go` + `cmd/` con router de mapa (sin Cobra), parser de flags cortos y
  largos (portar `parseFlagsWithShorts` de gtt y sus tests), ayuda en dos
  niveles (`draco help`, `draco help <comando>`, `-h` que nunca ejecuta).
- Comandos: `init`, `project`, `g`/`gen`/`generate`, `qa`, `f`/`fetch`,
  `backlog scan`, `update`, `version`, `help`. Mismos flags que gtt.
- `draco` sin argumentos: en una terminal interactiva abrirá la TUI (fase 4);
  sin TTY muestra la ayuda.
- `fetch`: portar de gtt los diálogos nativos para guardar (zenity,
  PowerShell también desde WSL, osascript), con la carpeta actual como respaldo.
- Portar y adaptar de gtt: `installer` (auto-instalación al primer uso, PATH en
  Windows), `updater` (GitHub Releases de este repo, verificación SHA-256,
  aviso cacheado cada 24 h, variable `DRACO_NO_UPDATE_CHECK`), `build` (versión
  por ldflags).
- Objetivos de release en el Makefile (ya existe para el día a día) y proceso
  de release con `checksums.txt` (P-006). Releases solo
  desde `main` actualizado y con el árbol limpio.
- Pendiente de decidir: si se porta la skill `backlog-tareas` para Claude Code
  que gtt trae embebida.

### Fases 4 a 6 — modo visual

Bubble Tea + Bubbles + Lip Gloss + Huh. Cada pantalla usa el mismo flujo que el
comando equivalente; los pasos de la `Session` se ejecutan como comandos de
Bubble Tea para no bloquear la interfaz.

- **4:** pantalla principal, proyectos, generar documento con selector de
  commits, vista previa, selector de ubicación y confirmación.
- **5:** backlog (tabla filtrable), QA (progreso en vivo), exportar.
- **6:** barra de comandos con los mismos comandos del modo comando y
  autocompletado, configuración en formularios, temas, ayuda.

La pantalla de inicio no puede mostrar el logo como imagen (la mayoría de las
terminales no lo soporta): necesita el nombre en letras grandes con caracteres.

### v1.0

- Instaladores (script, Homebrew tap, Scoop/winget). El paquete `draco` de
  Homebrew, Debian y MacPorts es Google Draco (compresión 3D): publicar con otro
  nombre de paquete; el comando sigue siendo `draco`.
- Migración desde gtt: leer `~/.config/gtt/config.yaml` para no repetir el
  asistente (pendiente de confirmar).
- README, guía de usuario y una bitácora por versión
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
| Bitácora por actividad (`docs/bitacora/`) y comentarios mínimos en el código | La historia de cada decisión en un solo lugar, el código limpio |

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
