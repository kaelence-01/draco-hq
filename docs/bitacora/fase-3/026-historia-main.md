# 026 — Unir la historia de `main` y `develop`

**Fecha:** 2026-10-19
**Rama:** fix/historia-main → develop
**Fase del roadmap:** 3

## 1. Solicitud

El PR #27 (`develop` → `main`, para el primer release) mostraba conflictos en
17 archivos. El autor preguntó si los *rebase and merge* que había hecho
podían haber dañado algo.

## 2. Motivación

| Problema | Daño real |
|---|---|
| `main` recibió la fase 2 con *rebase and merge*: los mismos 16 commits, copiados con otros hashes | Para Git, `main` y `develop` solo comparten el commit inicial; cada archivo que cambió después de la fase 2 sale como conflicto *add/add* |
| Resolver 17 conflictos a mano en el editor web | Riesgo real de dejar en `main` una mezcla de la fase 2 y la fase 3 justo antes del primer release |

**No se perdió nada:** el árbol de `main` (`0866b3d`) es idéntico, archivo por
archivo, al de `develop` en «Perfiles y cierre de la fase 2» (`ad1bf7d`), un
ancestro de `develop`. El daño es solo de historia.

## 3. Diseño técnico

- **Merge de `origin/main` en `develop` con la estrategia `ours`**: el commit
  de merge tiene a `main` como segundo padre y conserva el árbol de `develop`
  tal cual. Es exacto, no una elección de un lado: todo lo que `main` tiene ya
  está en `develop` (mismo árbol que `ad1bf7d`). Desde ese merge, `main` es
  ancestro de `develop` y el PR `develop` → `main` no tiene conflictos.
- **Este PR se fusiona con *merge commit***: un *squash* o un *rebase* de este
  PR descartaría justamente el segundo padre y el conflicto volvería.
- Descartado: forzar `main` a `develop` (`push --force`). Reescribe la rama
  publicada y necesita saltarse sus protecciones; el merge `ours` llega al
  mismo contenido sin reescribir nada.
- **Regla nueva** (CLAUDE.md y `docs/release.md`): `develop` → `main` siempre
  con *merge commit*. Para `feature` → `develop`, *squash* o *rebase* siguen
  valiendo (cada actividad es un commit).

## 4. Archivos

- Commit de merge de `origin/main` (estrategia `ours`), sin cambios de código.
- `CLAUDE.md`, `docs/release.md` — regla del método de merge hacia `main`.
- `docs/bitacora/fase-3/026-historia-main.md`, índice.

## 5. Cómo verificar

```bash
git merge-base --is-ancestor origin/main HEAD && echo "main ya es ancestro"
git diff 0866b3d ad1bf7d --stat        # vacío: main = develop en la fase 2
git diff develop HEAD --stat -- . ':!docs' ':!CLAUDE.md'   # vacío: el código no cambia
```

## 6. Pendientes

- Opcional, en GitHub: un *ruleset* sobre `main` que permita solo *merge
  commit*, para que no se repita.
