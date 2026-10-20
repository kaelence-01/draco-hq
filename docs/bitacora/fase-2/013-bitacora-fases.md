# 013 — Bitácora por fases

**Fecha:** 2026-10-19
**Rama:** docs/bitacora-fases → develop
**Fase del roadmap:** 2

## 1. Solicitud

Ajustar la bitácora por fases: cada entrada dentro de la carpeta de la fase que
le corresponde, y lo mismo con las siguientes.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Todas las entradas en una sola carpeta | Con varias fases por delante, ubicar qué se hizo en cada una obliga a abrir entradas una por una |
| La fase solo figuraba como columna del índice | Nada impedía archivar una entrada con la fase equivocada |

## 3. Diseño técnico

- **`docs/bitacora/fase-N/NNN-<tema>.md`.** Las 12 entradas existentes son de
  la fase 2 y se movieron con `git mv` (conservan el historial). La fase 1 se
  hizo antes de que existiera la bitácora: no tiene entradas, y el índice lo
  dice.
- **Numeración global**, no por fase: las entradas se citan por número («ver
  004») y reiniciar la cuenta lo volvería ambiguo. La próxima es la 014.
- **Un solo índice** (`docs/bitacora/README.md`) con una sección por fase; la
  columna «Fase» desaparece porque la da la sección. La primera entrada de una
  fase crea su carpeta y su sección.
- **`check.sh` lo verifica:** la entrada nueva debe estar en `fase-N/`, el `N`
  de la carpeta debe coincidir con el «Fase del roadmap» del encabezado, y
  debe estar enlazada en el índice. Cuenta como entrada solo un archivo
  agregado (un archivo movido no), y avisa si se modificó una entrada que ya
  estaba en `develop`.
- Las entradas ya fusionadas no se reescriben: la 003 menciona rutas como
  `docs/bitacora/001-core.md`, ciertas cuando se escribió.

## 4. Archivos

- `docs/bitacora/fase-2/001…012` — movidos desde `docs/bitacora/`.
- `docs/bitacora/fase-2/013-bitacora-fases.md` — esta entrada.
- `docs/bitacora/README.md` — índice por fase, formato y reglas.
- `.claude/skills/cerrar-actividad/check.sh` — verificaciones de la bitácora.
- `.claude/skills/bitacora/SKILL.md` — cómo elegir fase, número y carpeta.
- `CLAUDE.md`, `docs/roadmap.md` — ruta de la bitácora.

## 5. Cómo verificar

```bash
make check
ls docs/bitacora/fase-2/
```

Probado además el caso inválido: una entrada nueva en `fase-3/` con
encabezado de fase 2 hace fallar `make check`.
