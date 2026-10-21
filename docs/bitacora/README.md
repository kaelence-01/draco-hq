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
