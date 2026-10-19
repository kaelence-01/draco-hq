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

Funciona en Windows Terminal, PowerShell, Linux y macOS. En la consola clásica
de Windows (`cmd`) se ve más limitado.

## Estado

| Fase | Contenido | Estado |
|---|---|---|
| 1 | Nombre, identidad y repositorio | ✅ |
| 2 | Core: portar la lógica de gtt y separar los flujos | 🟡 en curso |
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

### Fase 2 — siguiente

1. **Flujo de exportar** (`internal/fetch`), a partir de `cmd/fetch.go`: buscar
   páginas que mencionen el issue (`FindPagesByText`), elegir una, convertirla
   (`GetPageWithContent` + `BuildIssueTxt`) y proponer el nombre de archivo
   (`sanitizeFilename`). El diálogo nativo para guardar es de la UI, no del flujo.
2. **Proyectos y configuración** como funciones puras: agregar, quitar, cambiar
   el default y guardar (`config.Save`), para que los usen el asistente de texto
   y los formularios de la TUI.
3. **Convenciones de texto configurables**, con sus valores actuales por
   defecto: prefijo del título del documento («Documento de Despliegue»), prefijo
   de las tareas de revisión QA («Revisión de Tarea»), marcador de observaciones
   («Novedad::»), título del consolidado QA.
4. **`internal/git`: listar commits recientes** de un repo, para el selector de
   commits de la TUI.

### Fase 3 — modo comando

- `main.go` + `cmd/` con router de mapa (sin Cobra), parser de flags cortos y
  largos (portar `parseFlagsWithShorts` de gtt y sus tests), ayuda en dos
  niveles (`draco help`, `draco help <comando>`, `-h` que nunca ejecuta).
- Comandos: `init`, `project`, `g`/`gen`/`generate`, `qa`, `f`/`fetch`,
  `backlog scan`, `update`, `version`, `help`. Mismos flags que gtt.
- `draco` sin argumentos: en una terminal interactiva abrirá la TUI (fase 4);
  sin TTY muestra la ayuda.
- Portar y adaptar de gtt: `installer` (auto-instalación al primer uso, PATH en
  Windows), `updater` (GitHub Releases de este repo, verificación SHA-256,
  aviso cacheado cada 24 h, variable `DRACO_NO_UPDATE_CHECK`), `build` (versión
  por ldflags).
- Makefile y proceso de release con `checksums.txt` (P-006). Releases solo
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

## Decisiones pendientes

- Significado del acrónimo: *Deployment Records, Audits & Code Overview*, la
  propuesta original *Deployment Records And Change Orchestrator* («orchestrator»
  sugiere que ejecuta despliegues, y no lo hace) o ninguno.
- Nombre del paquete en Homebrew/Scoop (el repo es `draco-hq`).
- Solo español o español e inglés desde el inicio.
- Leer el config de gtt para migrar.
- Portar o no la skill `backlog-tareas`.
- Logo: confirmar origen y licencia de la imagen, quitar el ©, vectorizar,
  versión para fondo claro y versión para 16–32 px.
