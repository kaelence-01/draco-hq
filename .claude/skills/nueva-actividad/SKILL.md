---
name: nueva-actividad
description: Prepara la rama de una actividad nueva en draco (feature, fix, docs) desde develop actualizada. Úsala SIEMPRE antes de empezar a cambiar archivos para una tarea nueva, cuando el usuario diga "sigamos con…", "empecemos…", o al pasar al siguiente punto del roadmap.
---

# nueva-actividad

Toda actividad nace de `develop` actualizada, en su propia rama, y termina en
un solo commit con PR hacia `develop`. Esta skill deja la rama lista.

```
main ── develop ── feature/<tema>   (PR → develop)
```

## Pasos

1. **Leer el contexto:** `docs/roadmap.md` (fase y punto que toca) y la última
   entrada de `docs/bitacora/`.

2. **Árbol limpio.** `git status --porcelain` debe salir vacío. Si hay cambios,
   no los muevas ni los descartes: pregunta al usuario qué hacer con ellos.

3. **`develop` al día y rama creada:** `make start name=<tema> [type=feature|fix|docs]`.
   Valida el nombre y el tipo, exige el árbol limpio, actualiza `develop` solo
   con *fast-forward* (`make sync`), crea `<type>/<tema>` desde `develop` y le
   quita el upstream. Si el `fetch` falla por la llave SSH (la sesión no tiene
   su passphrase), pide al usuario `! make sync` y luego crea la rama tú con
   `git switch -c <type>/<tema> develop` y `git branch --unset-upstream`.
   Si `make sync` falla porque `develop` local se desvió: detenerse y preguntar.

4. **Nombre de la rama:** `feature/<tema>`, tema corto en minúsculas y con
   guiones, igual al que llevará la entrada de bitácora (`feature/fetch` →
   `NNN-fetch.md`). Para correcciones, `fix/<tema>`; para solo documentación,
   `docs/<tema>`. Si el tema no es obvio, propón uno y confirma.
   Git no admite una rama `feature` junto a `feature/<tema>`: la integración es
   `develop`.

5. **Nunca dejes la rama siguiendo a `develop` o `main`:** un `git push` iría
   ahí. El upstream lo fija el usuario con su primer `make push`.

6. **Identidad de git:** `git config user.name` y `user.email` deben estar
   configurados con la identidad que el autor usa en este repo público. Si
   faltan, pregunta; no la deduzcas del correo de la sesión.

7. Si el cambio es grande, **propón el plan** antes de editar (lo pide el
   autor). Aplica `estandares-desarrollo` mientras trabajas y
   `cerrar-actividad` al terminar.

## Lo que esta skill no hace

- No hace push, no abre PR, no fusiona: eso lo hace el usuario.
- No trabaja directo en `develop` ni en `main`.
