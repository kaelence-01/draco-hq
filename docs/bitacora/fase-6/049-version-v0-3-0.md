# 049 — La fase 6 salió como `v0.3.0`

**Fecha:** 2026-10-21
**Rama:** docs/version-v0-3-0 → develop
**Fase del roadmap:** 6

## 1. Solicitud

El autor publicó el cierre de la fase 6 como *minor* (`v0.3.0`) en vez del
*patch* planeado (`v0.2.2`), porque le pareció lo mejor. Pidió seguir con lo
que viene; antes hacía falta que la documentación dijera la versión real.

## 2. Motivación

| Problema | Daño real |
|---|---|
| El roadmap, `CLAUDE.md`, `docs/release.md` y la bitácora 048 decían `v0.2.2` y que la fase 6 seguía en `v0.2.x` | Quien lea el estado buscaría un release que no existe, y la regla de `CLAUDE.md` llevaría a proponer un *patch* donde el autor ya eligió *minor* |

## 3. Diseño técnico

- **Decisión del autor:** la fase 6 es `v0.3.0`. Semver pide *minor* para
  funciones nuevas sin romper nada, y la fase 6 suma Configuración, la
  barra, la Ayuda y los temas.
- **Lo que cambia en la documentación:**
  - el roadmap: estado, actividad 7 y párrafo de cierre con `v0.3.0`;
  - la decisión de la 039 se reescribe en «Decisiones tomadas» con lo que
    pasó: la 5 como *patch* y la 6 como *minor*;
  - «Decisiones pendientes» suma cómo numerar hasta la v1.0. *Minor* por
    fase y *patch* para correcciones es lo natural, pero lo decide el
    autor;
  - `CLAUDE.md` y `docs/release.md` dejan de fijar `bump=patch`, y el
    ejemplo de la guía vuelve a `bump=minor`;
  - el ejemplo de la cabecera en `tui.md` pasa a `v0.3.0 · v0.3.1`.
- **Bitácoras 039 y 048:** ya estaban fusionadas, así que solo reciben una
  nota aclaratoria al final, con fecha (`check.sh` lo marca como
  advertencia, no como error).
- **El release** salió por el camino previsto: `make promote` (su primer uso
  real, PR #54 con *merge commit*) y `make release`.

## 4. Archivos

- `docs/roadmap.md` — `v0.3.0` en el estado, la fase 6 y su cierre;
  decisión tomada y decisión pendiente sobre la numeración.
- `CLAUDE.md`, `docs/release.md` — el `bump` lo decide el autor.
- `docs/arquitectura/tui.md` — ejemplo de la cabecera.
- `docs/bitacora/fase-5/039-cierre-fase-5.md`,
  `docs/bitacora/fase-6/048-cierre-fase-6.md` — notas aclaratorias.
- `docs/bitacora/fase-6/049-version-v0-3-0.md`, índice.

## 5. Cómo verificar

```bash
git tag --sort=-v:refname | head -1   # v0.3.0
make check
```

## 6. Pendientes

- Verificación de `v0.3.0` con `draco update` desde `v0.2.1`, a cargo del
  autor.
- Decidir la numeración hasta la v1.0 (roadmap, «Decisiones pendientes»).
