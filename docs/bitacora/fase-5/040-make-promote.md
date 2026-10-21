# 040 — `make promote`: develop → main

**Fecha:** 2026-10-21
**Rama:** feature/make-promote → develop
**Fase del roadmap:** 5

## 1. Solicitud

Mientras corría el release de la fase 5, el autor pidió un objetivo del
Makefile que haga para `develop` → `main` lo que `make pr` y `make finish`
hacen para una actividad: crear el PR y, después de fusionarlo, dejar las
ramas locales al día.

## 2. Motivación

| Problema | Daño real |
|---|---|
| El PR de `develop` a `main` se creaba y se fusionaba a mano en GitHub | Elegir *squash* o *rebase and merge* por error rompe la historia común de las ramas; ya pasó (bitácora 026) |
| Después de fusionar, `main` y `develop` locales se actualizaban a mano | `make release` exige `main` igual a `origin/main`: un paso olvidado lo hace fallar |
| La guía de release creaba el PR con `--fill` y un título con la versión | Con varios commits, `--fill` no lista las actividades, y la versión todavía no existe al crear el PR |

## 3. Diseño técnico

- **`scripts/promote.sh`**, en el estilo de `release.sh` (`fail`/`ok`,
  `set -euo pipefail`), que el Makefile llama como `make promote`.
- **Antes de tocar GitHub** se detiene en tres casos:
  - `gh` no está autenticado;
  - hay cambios sin commitear;
  - `develop` local tiene commits que no están en `origin/develop`. Lo que
    llega a `develop` entra por PR, y si no, esos commits quedarían fuera
    del PR sin aviso.

  Si `main` ya tiene todo `develop`, termina sin hacer nada.
- **El PR:** reutiliza el que ya está abierto de `develop` a `main`, si
  existe, o crea uno con el título «Integrar develop en main». Su
  descripción lista las actividades (`git log --no-merges` entre
  `origin/main` y `origin/develop`, en orden). El título no lleva la
  versión: el tag lo pone `make release` después.
- **Pregunta antes de fusionar** (`[s/N]`, como `release.sh`): fusionar es
  publicar en `main`. Con `N`, el PR queda abierto para revisarlo en GitHub,
  y correr el objetivo otra vez lo reutiliza.
- **Fusión:** solo con `gh pr merge --merge`, nunca *squash* ni *rebase*.
  No usa `--delete-branch`, porque la rama del PR es `develop`.
- **El «finish»:** `main` y `develop` locales se actualizan solo con
  *fast-forward*: `merge --ff-only` para la rama actual y
  `fetch origin rama:rama` para la otra. El script termina en `main`, porque
  el paso siguiente es `make release`.
- **Regla de `CLAUDE.md` matizada:** decía «ningún objetivo escribe commits
  en `main`». `make promote` crea en GitHub el mismo *merge commit* que el
  botón del PR, así que la regla pasa a decir «commits locales». La fusión
  la sigue confirmando el autor.
- Descartado encadenar el release (`make promote bump=patch`): el autor
  pidió PR y finish, y el release ya tiene su propio objetivo y su propia
  pregunta.

## 4. Archivos

- `scripts/promote.sh` — nuevo.
- `Makefile` — objetivo `promote` en «Release».
- `docs/release.md` — el paso 1 usa `make promote` y explica qué hace.
- `CLAUDE.md` — `make promote` en la lista y la regla de `main`.
- `docs/bitacora/fase-5/040-make-promote.md`, índice.

## 5. Cómo verificar

```bash
make check
make help | grep promote
bash -n scripts/promote.sh
```

El script se probó sin GitHub, con un remoto *bare* local y un `gh` simulado
que fusiona con `--no-ff` (una prueba de la sesión, fuera del repo). Pasaron
cinco escenarios:

1. Con `N`: crea el PR, su descripción lista las actividades y no fusiona.
2. Con `s`, desde `develop`: reutiliza el PR, fusiona con `--merge` y sin
   `--delete-branch`, deja un *merge commit* en `main`, `main` y `develop`
   quedan iguales a `origin`, y termina en `main`.
3. Sin nada pendiente: no hace nada.
4. Con `develop` local adelantado: falla.
5. Con cambios sin commitear: falla.

## 6. Pendientes

- Primer uso real en el próximo paso de `develop` a `main`, a cargo del autor.
