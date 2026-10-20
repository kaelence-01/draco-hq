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
