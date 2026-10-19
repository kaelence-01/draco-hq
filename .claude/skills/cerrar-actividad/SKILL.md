---
name: cerrar-actividad
description: Cierra una actividad de draco — verificaciones (build, vet, gofmt, tests, datos de organización, rama, bitácora), un solo commit y las instrucciones de push y PR hacia develop. Úsala SIEMPRE al terminar cualquier tarea que modifique archivos del repo, antes de dar el trabajo por cerrado o de commitear.
---

# cerrar-actividad

## Pasos

1. **Estándares.** Revisa el diff con `estandares-desarrollo` (sobre todo
   comentarios sobrantes y datos de organización) y corrige.

2. **Bitácora.** Aplica la skill `bitacora`: entrada de la actividad, índice y
   roadmap al día.

3. **Verificaciones.** Desde la raíz del repo:

   ```bash
   bash .claude/skills/cerrar-actividad/check.sh
   ```

   Es de solo lectura. Interpreta cada línea:

   | Marca | Significado | Qué hacer |
   |---|---|---|
   | ✓ | En orden | Nada |
   | ⚠ | Advertencia | Revisar; corregir o explicarla al usuario |
   | ✗ | Bloqueante | Corregir antes del commit |

   El escaneo de datos de organización también lee, si existe,
   `.claude/terminos-privados.txt` (ignorado por git): un término por línea con
   nombres reales que nunca deben llegar al repo.

4. **Un solo commit por actividad.**
   - Sin commits en la rama: `git add` de los archivos de la actividad (nunca
     `-A` a ciegas: revisa `git status`) y commit.
   - Ya hay un commit y **no** se ha subido: `git commit --amend`.
   - Ya se subió: pregunta al usuario antes de reescribirlo (requiere
     `push --force-with-lease`).
   - Mensaje en español: título corto en imperativo o sustantivo
     («Flujo de exportar (internal/fetch)»), cuerpo con viñetas de lo hecho, y
     la línea de coautoría que indique la sesión.

5. **Entregar al usuario:**
   - Resumen de lo hecho y de las verificaciones, con fallos si los hubo.
   - Push (lo hace el usuario, la sesión no tiene su llave SSH):
     `! git push -u origin <rama>`
   - PR: `<rama>` → **`develop`** (nunca a `main`).
   - Después de fusionar, la siguiente actividad empieza con `nueva-actividad`.

## Lo que esta skill no hace

- No hace push, no abre ni fusiona PR, no toca `main` ni `develop`.
