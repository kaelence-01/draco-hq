# Bitácora — draco

Una entrada por actividad: cada rama `feature/<tema>` (o `fix/`, `docs/`) que
llega a `develop` por PR. La entrada viaja en el mismo commit que el cambio que
describe y vive en la carpeta de su fase del roadmap, `fase-N/`. Cuando haya
releases, cada versión enlazará las actividades que incluye.

La historia y el porqué de cada decisión van aquí, no en comentarios del código.

## Índice

La numeración es global y correlativa: no se reinicia en cada fase, porque las
entradas se citan por número.

### Fase 1 — Nombre, identidad y repositorio

Sin entradas: se hizo antes de que existiera la bitácora.

### Fase 2 — Core: lógica de gtt, flujos sin terminal y proveedores

| # | Actividad | Rama | Fecha | Resumen |
|---|---|---|---|---|
| [001](./fase-2/001-core.md) | Base del core | `feature/core` | 2026-10-19 | Migración del `internal/` de gtt, QA configurable, flujo del documento de despliegue |
| [002](./fase-2/002-qa.md) | Flujo de QA | `feature/qa` | 2026-10-19 | `internal/qa` sin E/S, preservación de lo escrito a mano en el consolidado |
| [003](./fase-2/003-bitacora.md) | Bitácora y reglas de trabajo | `feature/bitacora` | 2026-10-19 | Bitácora, flujo de ramas y skills obligatorias |
| [004](./fase-2/004-comentarios.md) | Comentarios al estándar mínimo | `feature/comentarios` | 2026-10-19 | 730 → 351 líneas de comentario, sin cambios de lógica; razones retiradas conservadas aquí |
| [005](./fase-2/005-fetch.md) | Flujo de exportar | `feature/fetch` | 2026-10-19 | `internal/fetch` sin E/S; nombres de archivo con tildes transliteradas |
| [006](./fase-2/006-arquitectura.md) | Diseño de proveedores y plantillas | `docs/arquitectura` | 2026-10-19 | Puertos y adaptadores, plantillas → Markdown, fase 2 reordenada en 7 actividades |
| [007](./fase-2/007-golden.md) | Goldens del documento actual | `feature/golden` | 2026-10-19 | 7 goldens (builder y flujos) como criterio de «documento idéntico»; fecha del consolidado fijable |
| [008](./fase-2/008-puertos.md) | Puertos y Atlassian como adaptador (3a) | `feature/puertos` | 2026-10-19 | `internal/provider`, `context` en todo, `deploy` y `fetch` sobre puertos; goldens sin regenerar |
| [009](./fase-2/009-puertos-qa.md) | QA y cambios sobre puertos (3b) | `feature/puertos-qa` | 2026-10-19 | `QATracker`, `ChangeSource` con commits recientes, convenciones al flujo; ningún flujo importa un adaptador |
| [010](./fase-2/010-markdown-adf.md) | Conversor Markdown → ADF (4a) | `feature/markdown-adf` | 2026-10-19 | `goldmark` con las convenciones aprobadas; tests, golden y fuzzing; nada publicado cambia |
| [011](./fase-2/011-markdown.md) | Documentos en Markdown y preservación en el adaptador (4b) | `feature/markdown` | 2026-10-19 | `document` genera Markdown escapado; `Document.Preserve` y `Sections`; goldens en `.md` |
| [012](./fase-2/012-makefile.md) | Makefile para el día a día | `feature/makefile` | 2026-10-19 | `make start/sync/push/pr/finish`, tests y goldens con opciones por variables |
| [013](./fase-2/013-bitacora-fases.md) | Bitácora por fases | `docs/bitacora-fases` | 2026-10-19 | Entradas en `fase-N/`; `check.sh` exige que carpeta y fase coincidan |
| [014](./fase-2/014-salida-markdown.md) | Destino Markdown local | `feature/salida-markdown` | 2026-10-19 | `markdown.Store` como `DocStore`; el flujo `deploy` publica sin Confluence y conserva lo editado a mano |
| [015](./fase-2/015-plantillas.md) | Plantillas del documento | `feature/plantillas` | 2026-10-19 | Plantillas incluidas idénticas al documento anterior, personalizables por bloques; convenciones de QA a la config |
| [016](./fase-2/016-perfiles.md) | Perfiles y cierre de la fase 2 | `feature/perfiles` | 2026-10-19 | `profiles` con migración en memoria, `profile.Open` arma los adaptadores; el space pasa del flujo al perfil |

### Fase 3 — Modo comando

| # | Actividad | Rama | Fecha | Resumen |
|---|---|---|---|---|
| [017](./fase-3/017-readme.md) | README y plan de la fase 3 | `docs/readme` | 2026-10-19 | README del repo; la fase 3 en 7 actividades con criterio de cierre |
| [018](./fase-3/018-cli-base.md) | Base del modo comando | `feature/cli-base` | 2026-10-19 | `main.go`, router de mapa, flags declarados por comando, ayuda en dos niveles, `version`, códigos de salida |
| [019](./fase-3/019-cli-generate.md) | `draco g` desde el comando | `feature/cli-generate` | 2026-10-19 | Despliegue de punta a punta sobre el flujo y el perfil, `--dry-run` en Markdown, preguntas como gtt |
| [020](./fase-3/020-cli-qa-fetch.md) | `draco qa` y `draco f` | `feature/cli-qa-fetch` | 2026-10-19 | Consolidado y exportar desde el comando, diálogo nativo de guardar; el primer consolidado ya se crea en una carpeta Markdown |
| [021](./fase-3/021-cli-backlog.md) | `draco backlog scan` | `feature/cli-backlog` | 2026-10-19 | Mismo informe y flags que gtt, sin perfil ni credenciales; flags con valor opcional (`-o`) |
| [022](./fase-3/022-cli-init.md) | `draco init`, `project` y `profile` (6a) | `feature/cli-init` | 2026-10-19 | La config se crea sin editar YAML; token oculto con `x/term`; lo guardado nunca trae variables de entorno |
| [023](./fase-3/023-cli-template.md) | `draco template` (6b) | `feature/cli-template` | 2026-10-19 | Copiar, listar, validar y previsualizar plantillas; la carpeta del perfil pasa a tener `deploy/` y `qa/` |
| [024](./fase-3/024-updater.md) | Instalador, actualizador y `draco update` (7a) | `feature/updater` | 2026-10-19 | Instalación en el primer uso, actualización verificada con `checksums.txt`, aviso con caché de 24 h; probado contra un GitHub simulado |
| [025](./fase-3/025-release.md) | Proceso de release (7b) | `feature/release` | 2026-10-19 | `make release-build` y `make release` desde `main` sin commits automáticos; 5 binarios con `checksums.txt`; guía e instalación en el README |
| [026](./fase-3/026-historia-main.md) | Unir la historia de `main` y `develop` | `fix/historia-main` | 2026-10-19 | `main` recibió la fase 2 por *rebase and merge*; merge `ours` sin cambiar código y regla de *merge commit* hacia `main` |
| [027](./fase-3/027-cierre-fase-3.md) | Cierre de la fase 3 | `docs/cierre-fase-3` | 2026-10-19 | `v0.1.0` instalado desde el release; pie de la ayuda al repo; actualización a verificar con `v0.1.1` |

### Fase 4 — Modo visual mínimo

| # | Actividad | Rama | Fecha | Resumen |
|---|---|---|---|---|
| [028](./fase-4/028-plan-fase-4.md) | Diseño de la TUI y plan de la fase 4 | `docs/plan-fase-4` | 2026-10-20 | `docs/arquitectura/tui.md` (Charm v2, pila de pantallas, pasos como comandos, tests sin terminal); la fase 4 en 5 actividades |
| [029](./fase-4/029-tui-base.md) | Base de la TUI | `feature/tui-base` | 2026-10-20 | `draco` sin argumentos abre la TUI en una terminal; pila de pantallas, marco, principal con DRACO en letras grandes; tests sin terminal |
| [030](./fase-4/030-tui-proyectos.md) | Proyectos en la TUI y normas del modo visual | `feature/tui-proyectos` | 2026-10-20 | Proyectos (lista con filtro, activo, por defecto, formulario de Huh); panel y ruta en el marco; `captures`/`close`; normas obligatorias en `tui.md` §10, `CLAUDE.md` y `estandares-desarrollo` |
| [031](./fase-4/031-tui-generar.md) | Generar en la TUI (1): tarea, commits y progreso | `feature/tui-generar` | 2026-10-20 | `deploy.ProjectSides`; `ChangeSource.CurrentBranch`; formulario, selector de commits por lado con hash a mano y pasos con progreso y cancelación con `esc` |
| [032](./fase-4/032-tui-publicar.md) | Generar en la TUI (2): vista previa, ubicación y publicación | `feature/tui-publicar` | 2026-10-20 | Vista previa (Markdown en `viewport`, `NotPreserved` con `s`), ubicación con filtro, publicar sin salida a medias y `LinkErr`; filtro compartido; texto largo ajustado, no cortado |
| [033](./fase-4/033-cierre-fase-4.md) | Cierre de la fase 4 | `docs/cierre-fase-4` | 2026-10-20 | `draco help` menciona el modo visual; fase 4 cerrada; release `v0.2.0` y su verificación a cargo del autor |

### Fase 5 — Resto de pantallas

| # | Actividad | Rama | Fecha | Resumen |
|---|---|---|---|---|
| [034](./fase-5/034-plan-fase-5.md) | Diseño de las pantallas y plan de la fase 5 | `docs/plan-fase-5` | 2026-10-20 | Exportar, Backlog y QA en `tui.md` §6; núcleo del backlog fuera de `cmd` y `Scan` con `context`; vista previa, ubicación y publicar compartidos con QA; la fase en 6 actividades |
| [035](./fase-5/035-tui-exportar.md) | Exportar en la TUI | `feature/tui-exportar` | 2026-10-20 | Formulario, documentos con filtro (uno solo sigue sin preguntar) y texto con ruta editable y confirmación al reemplazar; formulario y filas de documentos compartidos con Generar, filas recortadas al ancho |
| [036](./fase-5/036-tui-backlog.md) | Backlog en la TUI | `feature/tui-backlog` | 2026-10-20 | Núcleo del backlog de `cmd` a `internal/backlog` y `Scan` con `context`; formulario en dos páginas, progreso por repo y hallazgos con filtro, detalle y guardado |
| [037](./fase-5/037-tui-qa.md) | QA en la TUI (1): cuenta, tareas y evaluación en vivo | `feature/tui-qa` | 2026-10-20 | Formulario de sprint o Kanban; cuenta, tareas y una fila por tarea al evaluarse, con copia propia de las filas; `ErrNotQAAccount` y `ErrNoQA` explicados; `Deps.Now` |
| [038](./fase-5/038-tui-qa-publicar.md) | QA en la TUI (2): vista previa, ubicación y publicar compartidos | `feature/tui-qa-publicar` | 2026-10-20 | `docRun` con adaptadores de Generar y QA; el consolidado se publica desde la TUI; tests de Generar sin cambios; pie de teclas dentro del ancho |
| [039](./fase-5/039-cierre-fase-5.md) | Cierre de la fase 5 | `docs/cierre-fase-5` | 2026-10-21 | Fase 5 cerrada sin las pruebas manuales (decisión del autor); fases 5 y 6 como *patch* de `v0.2.x`: release `v0.2.1` a cargo del autor |
| [040](./fase-5/040-make-promote.md) | `make promote`: develop → main | `feature/make-promote` | 2026-10-21 | PR de `develop` a `main` creado o reutilizado, fusión confirmada solo con *merge commit* y ramas locales al día con *fast-forward*; probado con un remoto local y un `gh` simulado |

### Fase 6 — Configuración, barra de comandos y temas

| # | Actividad | Rama | Fecha | Resumen |
|---|---|---|---|---|
| [041](./fase-6/041-plan-fase-6.md) | Diseño de las pantallas y plan de la fase 6 | `docs/plan-fase-6` | 2026-10-21 | Configuración, aviso de versión, barra, Ayuda y temas en `tui.md` §6; reglas de `init`/`profile` y descripción de los comandos fuera de `cmd`; la barra abre pantallas con datos; la fase en 8 actividades, release `v0.2.2` |
| [042](./fase-6/042-tui-config.md) | Configuración en la TUI (1): conexión, perfiles y asistente | `feature/tui-config` | 2026-10-21 | Reglas de `init`/`profile` a funciones puras; conexión con token sin eco y verificación de la cuenta; perfiles con filtro y formulario; asistente sin configuración; aviso del entorno sin leerlo |
| [043](./fase-6/043-tui-config-qa.md) | Configuración en la TUI (2): QA | `feature/tui-config-qa` | 2026-10-21 | `qa_email`, `qa_report` y `qa_jira` en un formulario de cuatro páginas; `Validate` muestra todos los problemas sin guardar; `qa_jira` vacío no se guarda; lo guardado lo acepta `qa.Resolve` |
| [044](./fase-6/044-tui-aviso.md) | Aviso de versión nueva en la TUI | `feature/tui-aviso` | 2026-10-21 | `Deps.CheckUpdate` con la caché y las reglas del aviso de texto (`updatesOff` compartido); la cabecera y la principal lo muestran; actualizar sigue fuera de la TUI |
| [045](./fase-6/045-tui-barra.md) | Barra de comandos (1): descripción de los comandos, sugerencias y Ayuda | `feature/tui-barra` | 2026-10-21 | `internal/cli` con el parser y la descripción de cada comando (flags con `Desc`); barra en la principal con sugerencias de comandos, flags y valores; Ayuda general y de cada comando con el texto de `draco help` |
| [046](./fase-6/046-tui-barra-ejecutar.md) | Barra de comandos (2): cada comando abre su pantalla | `feature/tui-barra-ejecutar` | 2026-10-21 | `openFromBar`: formularios enviados con los datos de la línea, hashes marcados en Commits, flags sin pantalla explicados; un `-p` desconocido es un error, como en el modo comando |
| [047](./fase-6/047-tui-temas.md) | Temas del modo visual | `feature/tui-temas` | 2026-10-21 | `ansi`, `alto-contraste` y `mono` con `applyTheme` (estilos, pie y formularios); `theme` en la config; Configuración › Tema con vista previa en vivo; `mono` sin códigos de color |
| [048](./fase-6/048-cierre-fase-6.md) | Cierre de la fase 6 | `docs/cierre-fase-6` | 2026-10-21 | Fase 6 cerrada; pruebas manuales pendientes reunidas; release `v0.2.2` (primer uso de `make promote`) a cargo del autor |
| [049](./fase-6/049-version-v0-3-0.md) | La fase 6 salió como `v0.3.0` | `docs/version-v0-3-0` | 2026-10-21 | El autor publicó *minor* en vez del *patch* planeado; roadmap, `CLAUDE.md` y la guía de release al día; numeración hasta la v1.0, pendiente |
| [050](./fase-6/050-install-y-guia.md) | Instalación con un comando y guía de usuario | `feature/install-y-guia` | 2026-10-21 | `install.sh` e `install.ps1` verifican `checksums.txt` antes de instalar; `docs/guia-usuario.md` pública y completa; `make check` avisa si cambia lo que el usuario ve sin tocar la guía |
| [051](./fase-6/051-config-gtt-primero.md) | La configuración de gtt primero | `feature/config-gtt-primero` | 2026-10-21 | Configuración separa lo necesario (Conexión con el space, Proyectos) de lo opcional (Perfiles, QA, Tema); el asistente y `draco init` ya no crean un perfil: siguen con el primer proyecto |
| [052](./fase-6/052-proyecto-backlog.md) | Backlog técnico en el formulario de proyecto | `feature/proyecto-backlog` | 2026-10-21 | Página opcional por repo con `class_globs` y `modules`, validados antes de guardar (`backlog.CheckPatterns`); el resto del backlog se conserva |
| [053](./fase-6/053-rutas-config.md) | `~` en las rutas y space del proyecto con perfil Markdown | `fix/rutas-config` | 2026-10-21 | `config.ExpandHome` donde se usan las rutas de la config; el space del proyecto viaja aparte (`profile.Options.ProjectSpace`) y un destino Markdown lo ignora |
| [054](./fase-6/054-regla-proyecto.md) | La regla del proyecto en `config` | `feature/regla-proyecto` | 2026-10-21 | `ProjectConfig.Complete` (rescatada de un borrador de la 030) reemplaza la regla duplicada en `draco project add` y en el formulario de la TUI |
| [055](./fase-6/055-perfil-del-proyecto.md) | Cada proyecto con su perfil | `feature/perfil-del-proyecto` | 2026-10-21 | `projects.<n>.profile`; al generar, `--profile` > el del proyecto > el por defecto (`Config.ProfileFor`); la TUI cambia de perfil al elegir proyecto; un perfil en uso no se elimina |
| [056](./fase-6/056-consistencia-guia.md) | Consistencia de la guía de usuario | `docs/consistencia-guia` | 2026-10-22 | 15 correcciones de la guía y el README contra el código (precedencia del space, plantillas del repo, cabecera, barra, solución de problemas); QA y el prefijo de un perfil, pendiente |
| [057](./fase-6/057-documento-como-gtt.md) | El documento se ve como el de gtt | `fix/documento-como-gtt` | 2026-10-22 | Anchos de columna, viñetas en celdas y «A considerar» en un recuadro, con convenciones del Markdown (`draco:tabla`, `draco:recuadro`, líneas con `- `); también los anchos del consolidado QA |
| [058](./fase-6/058-make-merge.md) | `make merge` y atajos combinados | `feature/make-merge` | 2026-10-22 | `make merge` fusiona en `develop` con merge commit tras confirmar; `push-pr`, `push-pr-merge`, `merge-finish` y `ship`; `make pr` muestra el PR abierto en vez de fallar |
| [059](./fase-6/059-importar-gtt.md) | Importar la configuración de gtt | `feature/importar-gtt` | 2026-10-22 | `draco init` y el asistente de Configuración ofrecen importar `~/.config/gtt/config.yaml` cuando draco no tiene configuración; resumen sin el token; la de gtt no se toca |
| [060](./fase-6/060-qa-prefijo-perfil.md) | QA reconoce el documento con cualquier plantilla | `fix/qa-prefijo-perfil` | 2026-10-22 | `HasDocumentLink` recibe varios prefijos: los de la plantilla de cada proyecto (perfil, repo, usuario), sin proyecto y el incluido; una plantilla que no carga se avisa |
| [061](./fase-6/061-roadmap-equipos.md) | Roadmap: después de la v1.0, trackers y modo equipo | `docs/roadmap-equipos` | 2026-10-27 | `v0.3.1`–`v0.3.3` en el roadmap; otros trackers con sus requisitos (credenciales por perfil, QA solo Jira) y modo equipo (paquete compartido en Git) después de la v1.0, en el mismo draco |

### Fase 7 — Rediseño del modo visual: maestro-detalle elástico

| # | Actividad | Rama | Fecha | Resumen |
|---|---|---|---|---|
| [062](./fase-7/062-plan-fase-7.md) | Diseño y plan de la fase 7 | `docs/plan-fase-7` | 2026-10-27 | Tres propuestas en maquetas; el autor eligió maestro-detalle elástico y la probó en un prototipo; `tui.md` §11 y la fase en 8 actividades |
| [063](./fase-7/063-prototipo-fase-7.md) | El prototipo de la fase 7 en el repo | `docs/prototipo-fase-7` | 2026-10-27 | `prototipo-fase-7.html` como referencia visual; `tui.md` §11 con la tabla pantalla → archivo → actividad y lo que no se implementa; cómo retomar la fase en otra sesión |
| [064](./fase-7/064-tui-marco.md) | Marco: cabecera de una línea, pie compacto y línea de resultado | `feature/tui-marco` | 2026-10-28 | Cabecera con la ruta que cede por la izquierda; pie con las teclas principales y `? más` (`moreKeys` para las secundarias); tras publicar queda `✓ OPS-7 publicado en «…»` sobre el pie, con `⚠` si falta el enlace; `layout.go` |
| [065](./fase-7/065-tui-principal.md) | Principal: descripciones, atajos, panel con pasos y contexto | `feature/tui-principal` | 2026-10-28 | Menú con descripción y atajos `1`–`6`; panel con los pasos; desde 120 columnas, Proyecto activo y Tu configuración; la regla de QA pasa a `config.QAState`; en 80 columnas el texto de la opción va bajo el menú |

## Formato de cada entrada

Archivo `fase-N/NNN-<tema>.md`: `N` es la fase del roadmap, `NNN` el número
correlativo global de tres dígitos y el tema, el de la rama. La primera entrada
de una fase crea su carpeta y su sección en el índice. Encabezado y secciones
obligatorias:

```markdown
# NNN — Título corto

**Fecha:** AAAA-MM-DD
**Rama:** feature/<tema> → develop
**Fase del roadmap:** N   (la misma N de la carpeta)

## 1. Solicitud
Qué se pidió, en lenguaje natural.

## 2. Motivación
Por qué hacía falta. Problema → daño real si no se hace.

## 3. Diseño técnico
Decisiones tomadas y sus razones, incluidas las alternativas descartadas.

## 4. Archivos
Cada archivo tocado con una línea sobre su cambio.

## 5. Cómo verificar
Comandos concretos.

## 6. Pendientes
Lo que quedó fuera o se descubrió en el camino. Omitir si no hay.
```

## Reglas

- La fase del encabezado y la carpeta coinciden; `check.sh` lo verifica.
- Una entrada de una actividad ya fusionada en `develop` **no se reescribe**.
  Solo admite notas aclaratorias al final, con fecha. Lo nuevo va en la entrada
  de la actividad nueva.
- Sin datos de una organización: mismos criterios que el código
  (`docs/security/patrones-seguros.md`). Ejemplos genéricos (`OPS-7`, `Pagos`).
- Sin contraseñas, tokens, URLs de instancia ni correos.
