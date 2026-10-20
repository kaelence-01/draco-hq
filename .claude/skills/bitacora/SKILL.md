---
name: bitacora
description: Escribe o actualiza la entrada de bitácora (docs/bitacora/fase-N/) de la actividad en curso y su fila en el índice. Úsala SIEMPRE en toda actividad que cambie código, config o docs de draco, antes del commit (cerrar-actividad la invoca), y cuando el usuario pregunte qué se hizo en una actividad.
---

# bitacora

Cada actividad (una rama `feature/<tema>` → `develop`) tiene **una** entrada en
`docs/bitacora/fase-N/NNN-<tema>.md`, en la carpeta de su fase del roadmap, que
viaja en el mismo commit que el cambio.

## Pasos

1. **Fase, número y nombre.** Si la rama ya tiene su entrada (`git diff
   --name-only develop -- docs/bitacora/`), actualízala. Si no:
   - **fase:** la que está en curso en `docs/roadmap.md` (tabla «Estado»);
     la carpeta es `docs/bitacora/fase-N/` y se crea con la primera entrada
     de esa fase, junto con su sección en el índice;
   - **número:** el siguiente al mayor del índice; es global, no se reinicia
     por fase;
   - **tema:** el de la rama: `feature/fetch` → `fase-2/005-fetch.md`.

2. **Contenido.** Formato y reglas en `docs/bitacora/README.md`; la última
   entrada sirve de referencia. Fuentes:
   - **Solicitud:** lo que pidió el usuario en esta conversación, en sus
     términos.
   - **Motivación:** tabla problema → daño real.
   - **Diseño técnico:** decisiones con su porqué, alternativas descartadas y
     las decisiones que tomó el autor (indicarlo). Aquí va la historia que no
     debe quedar en comentarios del código.
   - **Archivos:** `git diff --stat develop` + una línea por archivo.
   - **Cómo verificar:** comandos que de verdad se corrieron.
   - **Pendientes:** lo que quedó fuera; si toca el roadmap, que coincida.

3. **Índice.** Agrega o ajusta la fila en la sección de su fase en
   `docs/bitacora/README.md`, con el enlace `./fase-N/NNN-<tema>.md`.

4. **Roadmap.** Si la actividad cierra un punto de `docs/roadmap.md`, muévelo a
   «hecho» y renumera lo pendiente.

## Reglas

- **No reescribir entradas de actividades ya fusionadas** en `develop`. Solo
  una nota aclaratoria al final, con fecha. Lo nuevo va en la entrada actual.
- Hechos, no intenciones: no anotar como hecho lo que no se verificó; si algo
  no se probó (por ejemplo, contra Atlassian real), decirlo en Pendientes.
- Sin datos de una organización, contraseñas, tokens, URLs de instancia ni
  correos.
- Español, directo; sin relleno.
