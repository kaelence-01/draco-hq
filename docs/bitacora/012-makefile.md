# 012 — Makefile para el día a día

**Fecha:** 2026-10-19
**Rama:** feature/makefile → develop
**Fase del roadmap:** 2

## 1. Solicitud

Un Makefile que simplifique los comandos que se repiten, con opciones por
variables cuando hagan falta. Ejemplos del autor: `git fetch origin develop &&
git pull origin develop` y `git push -u origin feature/<rama>` para la rama que
corresponda.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Cada actividad repite fetch, pull, crear rama, quitar upstream, push y PR a mano | Tiempo perdido y errores: una rama siguiendo a `develop`, un push a la rama equivocada |
| `git pull origin develop` hace merge si `develop` local se desvió | Commits de merge accidentales en `develop` local |
| Las ramas fusionadas se acumulan en local | Ruido al elegir rama; riesgo de trabajar sobre una vieja |

## 3. Diseño técnico

- **Opciones como variables** en minúscula (`name=`, `type=`, `pkg=`, `run=`,
  `v=1`, `force=1`, `web=1`, `time=`); `make` sin argumentos muestra la ayuda,
  generada desde los comentarios `##` de cada objetivo y agrupada con `##@`.
- **Git:**
  - `sync`: `fetch` y luego `develop` solo con *fast-forward*. En `develop`
    usa `pull --ff-only`; en otra rama actualiza `develop` sin cambiarse
    (`fetch origin develop:develop`). Si `develop` se desvió, falla en vez de
    mezclar.
  - `start name=<tema> [type=feature|fix|docs]`: valida nombre y tipo, exige
    el árbol limpio, hace `sync`, crea la rama desde `develop` y le quita el
    upstream (lo que pide la skill `nueva-actividad`).
  - `push [force=1]`: la rama actual con `-u`; nunca `main` ni `develop`;
    `force=1` usa `--force-with-lease`.
  - `pr [web=1]`: con `gh` con sesión crea el PR hacia `develop` (`--fill`);
    sin `gh`, imprime el enlace de comparación, derivado del remoto (nada
    escrito a mano).
  - `finish`: vuelve a `develop`, la actualiza y corre `prune`.
  - `prune`: borra con `git branch -d` las ramas fusionadas en `develop`,
    salvo `main`, `develop`, la actual y las recién creadas (apuntan al mismo
    commit que `develop` y git las cuenta como fusionadas; sin este filtro,
    `finish` borraría una actividad recién empezada).
- **Go:** `build`, `test`, `vet`, `fmt`, `lint` (gofmt sin diferencias + vet),
  `golden` (regenera y muestra qué goldens cambiaron), `fuzz`; `ci` = `lint` +
  `test`. Los archivos Go salen de `git ls-files`, así que `cmd/` y `main.go`
  entrarán solos.
- **Actividad:** `check` corre el script de `cerrar-actividad`.
- Las skills `nueva-actividad` y `cerrar-actividad` y `CLAUDE.md` pasan a usar
  los objetivos. La sesión de Claude no tiene la llave SSH: `sync`, `push` y
  `pr` los corre el autor (`! make push`).
- Los objetivos de release (fase 3) se sumarán a este Makefile.

## 4. Archivos

- `Makefile` — nuevo.
- `CLAUDE.md` — sección «Comandos».
- `.claude/skills/nueva-actividad/SKILL.md`, `.claude/skills/cerrar-actividad/SKILL.md`.
- `docs/roadmap.md` — fase 3: el Makefile ya existe.

## 5. Cómo verificar

```bash
make                      # ayuda
make lint && make test
make test pkg=./internal/qa run=Kanban v=1
make start name=Mal_Nombre        # rechaza el nombre
make -n push force=1              # muestra el push sin ejecutarlo
make -n prune | sed 's/git branch -d/echo borraría/' | bash
```

Verificado en esta sesión: `help`, `lint`, `build`, `test` (con y sin
opciones), `check`, las validaciones de `start` (nombre, tipo, árbol sucio) y
la simulación de `push` y `prune`. No se ejecutaron `sync`, `push` ni `pr`
reales (necesitan la llave SSH o crean un PR).

## 6. Pendientes

- Objetivos de release y de instalación local en la fase 3.
