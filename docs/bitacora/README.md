# Bitácora — draco

Una entrada por actividad: cada rama `feature/<tema>` que llega a `develop` por
PR. La entrada viaja en el mismo commit que el cambio que describe. Cuando haya
releases, cada versión enlazará las actividades que incluye.

La historia y el porqué de cada decisión van aquí, no en comentarios del código.

## Índice

| # | Actividad | Rama | Fecha | Fase | Resumen |
|---|---|---|---|---|---|
| [001](./001-core.md) | Base del core | `feature/core` | 2026-10-19 | 2 | Migración del `internal/` de gtt, QA configurable, flujo del documento de despliegue |
| [002](./002-qa.md) | Flujo de QA | `feature/qa` | 2026-10-19 | 2 | `internal/qa` sin E/S, preservación de lo escrito a mano en el consolidado |
| [003](./003-bitacora.md) | Bitácora y reglas de trabajo | `feature/bitacora` | 2026-10-19 | 2 | Bitácora, flujo de ramas y skills obligatorias |
| [004](./004-comentarios.md) | Comentarios al estándar mínimo | `feature/comentarios` | 2026-10-19 | 2 | 730 → 351 líneas de comentario, sin cambios de lógica; razones retiradas conservadas aquí |
| [005](./005-fetch.md) | Flujo de exportar | `feature/fetch` | 2026-10-19 | 2 | `internal/fetch` sin E/S; nombres de archivo con tildes transliteradas |
| [006](./006-arquitectura.md) | Diseño de proveedores y plantillas | `docs/arquitectura` | 2026-10-19 | 2 | Puertos y adaptadores, plantillas → Markdown, fase 2 reordenada en 7 actividades |
| [007](./007-golden.md) | Goldens del documento actual | `feature/golden` | 2026-10-19 | 2 | 7 goldens (builder y flujos) como criterio de «documento idéntico»; fecha del consolidado fijable |
| [008](./008-puertos.md) | Puertos y Atlassian como adaptador (3a) | `feature/puertos` | 2026-10-19 | 2 | `internal/provider`, `context` en todo, `deploy` y `fetch` sobre puertos; goldens sin regenerar |

## Formato de cada entrada

Archivo `NNN-<tema>.md` (número correlativo de tres dígitos, tema igual al de
la rama). Encabezado y secciones obligatorias:

```markdown
# NNN — Título corto

**Fecha:** AAAA-MM-DD
**Rama:** feature/<tema> → develop
**Fase del roadmap:** N

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

- Una entrada de una actividad ya fusionada en `develop` **no se reescribe**.
  Solo admite notas aclaratorias al final, con fecha. Lo nuevo va en la entrada
  de la actividad nueva.
- Sin datos de una organización: mismos criterios que el código
  (`docs/security/patrones-seguros.md`). Ejemplos genéricos (`OPS-7`, `Pagos`).
- Sin contraseñas, tokens, URLs de instancia ni correos.
