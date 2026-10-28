# 061 — Roadmap: después de la v1.0, trackers y modo equipo

**Fecha:** 2026-10-27
**Rama:** docs/roadmap-equipos → develop
**Fase del roadmap:** 6

## 1. Solicitud

Con draco en `v0.3.3` y usuarios trabajando sobre todo con Jira, el autor
pidió ponerse al día con el estado del proyecto y con las plataformas que
draco soporta hoy. Propuso que un líder de equipo prepare las plantillas y
que los desarrolladores las consuman desde un lugar compartido, para no
rehacer cada cambio a mano. Pidió anotar en el roadmap esa idea, para
después de la v1.0, junto con el tema de los trackers.

## 2. Motivación

| Problema | Daño real |
|---|---|
| El roadmap terminaba en `v0.3.0`, sin `v0.3.1` a `v0.3.3` | Quien lea el estado no sabe qué trae cada release publicado |
| La idea de pasar de gtt a draco mezclaba plantilla y tracker | Se esperaría que una plantilla sumara otro tracker, y eso pide un adaptador en Go |
| «Segundo tracker» estaba fechado «después de la fase 3» y sin requisitos | Ese plazo ya pasó, y faltan requisitos que bloquean: credenciales por perfil y QA atado a Jira |
| El modo equipo no estaba anotado | La idea se pierde, o se diseña como un segundo producto |

## 3. Diseño técnico

- **Sin cambios de código.** Solo cambian el roadmap y la bitácora.
- **`v0.3.1`–`v0.3.3`:** una fila en «Estado» y una tabla por release con
  sus bitácoras, armada desde los tags (`git log vA..vB`).
- **«Después de la v1.0»**, con dos líneas sin orden fijo; se elige según la
  demanda.
  - **Otros trackers y plataformas:** primero la aclaración de plantilla
    frente a tracker, luego lo soportado hoy y los candidatos con la
    prioridad de `proveedores.md` §9. Los trackers de la fila «Sin evaluar»
    (Asana, monday, Zoho Projects, ClickUp, Trello, YouTrack) son aporte de
    esta conversación y no están en el diseño. Los requisitos previos son
    credenciales y URL por perfil, y QA solo con Jira hasta decidir el
    mapeo de estados. El adaptador declarativo queda como idea a evaluar.
  - **Modo equipo:** un solo draco con una fuente compartida; un paquete del
    equipo en un repo Git versionado con tags, que absorbe las plantillas
    remotas pendientes del diseño. Las credenciales nunca viajan en el
    paquete.
- **Decisión del autor:** el modo equipo va después de la v1.0 y dentro del
  mismo draco. Descartado: una edición individual y otra para grupos, porque
  duplicaría el producto. Queda en «Decisiones tomadas».
- **Decisiones pendientes nuevas:** cuál es el segundo tracker y si se
  intenta el adaptador declarativo; dónde publica el líder (Git, o
  Confluence o una carpeta compartida si no es técnico) y qué entra en el
  paquete.
- **Carpeta `fase-6`:** no hay fase nueva en curso; las mejoras sobre la
  fase 6 siguen en su carpeta, como 050 a 060.

## 4. Archivos

- `docs/roadmap.md` — releases `v0.3.1`–`v0.3.3`, sección «Después de la
  v1.0», decisión tomada y decisiones pendientes; se quitó el párrafo
  «después de la fase 3».
- `docs/bitacora/fase-6/061-roadmap-equipos.md`, índice.

## 5. Cómo verificar

```bash
make check
git log --no-merges --format='%h %s' v0.3.2..v0.3.3   # coincide con la tabla
```

## 6. Pendientes

- Diseño y plan de cada línea de «Después de la v1.0», cuando se retome.
- Las pruebas manuales contra Jira y Confluence de las fases 4 a 6 siguen
  pendientes (roadmap, cierre de la fase 6).
