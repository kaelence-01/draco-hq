# 058 — `make merge` y atajos combinados

**Fecha:** 2026-10-22
**Rama:** feature/make-merge → develop (sobre `fix/documento-como-gtt`)
**Fase del roadmap:** 6

## 1. Solicitud

El autor preguntó cómo fusionar el PR con `gh` sin ir a GitHub, y pidió un
`make merge` con variantes: push y PR a la vez; push, PR y merge; merge y
finish; y todo junto (`make push && make pr && gh pr merge --merge && make
finish`).

## 2. Motivación

| Problema | Daño real |
|---|---|
| Fusionar el PR de una actividad exigía abrir GitHub o recordar `gh pr merge --merge` | Con `--squash` o `--rebase` (lo que GitHub ofrece primero según el repo), `develop` y `main` dejan de compartir historia (bitácora 026) |
| Cerrar una actividad eran cuatro comandos seguidos | Un `finish` corrido antes de fusionar, o un merge con commits sin subir, deja el trabajo a medias sin aviso |
| `make pr` fallaba si el PR ya existía | No se podía repetir después de un `make push force=1` ni encadenar |

## 3. Diseño técnico

- **`scripts/merge.sh`** (`make merge`), con la forma de `promote.sh`:
  - exige `gh` autenticado y una rama de actividad (ni `main`, ni `develop`);
  - la rama debe estar en `origin` y sin commits sin subir; si no, pide
    `make push`;
  - busca el PR abierto de la rama y exige que vaya hacia `develop`: un PR
    hacia `main` se rechaza y remite a `make promote`;
  - pregunta antes de fusionar, con el número y el título del PR; `yes=1`
    no pregunta;
  - fusiona siempre con `gh pr merge --merge`. No usa `--delete-branch`:
    `gh` dejaría al autor en la rama por defecto (`main`), y `make finish`
    ya vuelve a `develop` y borra la rama fusionada.
  - Responder que no es un error (sale con 1), para que una cadena no siga
    con `finish` sobre un PR sin fusionar.
- **`make pr` repetible:** si la rama ya tiene un PR abierto, muestra su
  enlace y termina bien.
- **Combinados:** `push-pr`, `push-pr-merge`, `merge-finish` y `ship` (los
  cuatro pasos). Cada uno llama a los objetivos en orden con `$(MAKE)` y se
  detiene en el primero que falla. Las opciones (`force=1`, `web=1`,
  `yes=1`) pasan a todos.
- **Nombres:** en inglés, como los demás objetivos (`push`, `pr`,
  `finish`); `ship` es «despachar»: todo el cierre de la actividad.
- Sigue siendo el autor quien los corre: la sesión no tiene su llave SSH ni
  su `gh`.

## 4. Archivos

- `scripts/merge.sh` — nuevo.
- `Makefile` — `merge`, combinados, `pr` repetible, `yes=1` en la ayuda.
- `CLAUDE.md`, `README.md` — los atajos nuevos.
- `.claude/skills/cerrar-actividad/SKILL.md` — `make merge` y `make ship`
  en la entrega.
- `docs/bitacora/fase-6/058-make-merge.md`, índice.

## 5. Cómo verificar

```bash
make check
make help | grep -E 'merge|ship|push-pr'
```

Probado con un remoto local y un `gh` simulado, con 8 casos, todos con el
resultado esperado:
1. `merge` sin push se detiene;
2. `push-pr` crea el PR;
3. `pr` otra vez muestra el abierto;
4. con commits sin subir, `merge` se detiene;
5. `merge-finish` respondiendo «n» no llega a `finish`;
6. `ship yes=1` fusiona con merge commit, deja `develop` al día y borra la
   rama;
7. un PR hacia `main` se rechaza;
8. en `develop`, `merge` se niega.

Falta el primer uso real contra GitHub, a cargo del autor.
