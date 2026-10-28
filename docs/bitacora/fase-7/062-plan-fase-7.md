# 062 — Diseño y plan de la fase 7: maestro-detalle elástico

**Fecha:** 2026-10-27
**Rama:** docs/plan-fase-7 → develop
**Fase del roadmap:** 7

## 1. Solicitud

Antes de seguir hacia la v1.0, el autor pidió mejorar el modo visual porque
se pierde mucho espacio. Pidió ver tres propuestas en algo visual para
iterarlas y elegir; si ninguna convencía, otras tres que no repitieran las
descartadas. Eligió la segunda, maestro-detalle elástico. Después pidió
probarla en un prototipo interactivo y, con el prototipo visto, implementar
todo lo que mostraba.

## 2. Motivación

| Problema | Daño real |
|---|---|
| A 160×40, la principal centra todo y deja unas 60 columnas vacías por lado; el panel tiene ancho fijo | En una terminal ancha, la mayor parte de la pantalla no dice nada |
| Commits, Proyectos y QA ocupan un tercio del ancho y ponen el detalle debajo | Para ver el detalle hay que bajar la vista, y una lista larga lo empuja fuera de la pantalla |
| El marco gasta tres líneas arriba y el pie es una línea larga | En 80×24 cada línea cuenta |
| La vista previa muestra el Markdown crudo con los comentarios `draco:` | Se revisa un documento que no se parece al que se publicará |

## 3. Diseño técnico

- **Cómo se midió:** se renderizaron las pantallas reales con un test
  temporal a 160×40 (Inicio, Proyectos, Generar, Commits, Vista previa, QA,
  Backlog), que no quedó en el repo.
- **Propuestas comparadas en maquetas, con los mismos datos de ejemplo:**
  - A, riel lateral fijo (menú, proyecto y pasos siempre a la vista);
  - B, maestro-detalle elástico;
  - C, mosaico de paneles al estilo lazygit.

  **Decisión del autor: B.** Mantiene la pila de pantallas y los flujos.
  C reemplazaba la pila del diseño, y A sumaba un riel a todas las
  pantallas.
- **Prototipo navegable de B,** con teclado, tres tamaños de terminal y los
  tres temas. Sus mejoras se podían encender y apagar; las que no venían en
  la maqueta eran la vista previa renderizada, el filtro visible y la línea
  del último resultado. El autor pidió implementarlas todas.
- **Lo que cambió del prototipo al plan:**
  - Los atajos de la principal son `1`–`6` y no letras: `q` ya es salir.
  - No se muestran datos que draco no lee: estado, asignado y épica de la
    tarea; cuerpo y hora del commit; responsable y número de errores de
    cada tarea de QA. Mostrarlos pide cambiar los puertos y los adaptadores,
    y eso queda fuera de una fase de interfaz.
- **Decisiones técnicas** (detalle en `tui.md` §11):
  - Los archivos de cada commit se leen con `ChangedFiles` de un solo hash,
    en un `tea.Cmd`, y se guardan por hash. El puerto no cambia.
  - La vista previa se renderiza con un renderizador propio del Markdown
    que genera draco. `goldmark` sigue solo en los adaptadores; se descartó
    sumar una dependencia de Charm para esto.
  - Los contadores de QA salen de una función pura del flujo (`qa.Tally`),
    no de la pantalla. Una consulta fallida cuenta como sin verificar.
  - Las ayudas de layout van en `layout.go`, compartidas.
- **Orden de las actividades:**
  1. El marco, que usan todas las pantallas.
  2. La principal.
  3. Generar y su vista previa, lo más usado.
  4. QA, que reusa la vista previa.
  5. Las listas restantes.
- **Fase nueva antes de la v1.0:** cambia lo que el usuario ve en todas las
  pantallas. Por eso no entra como una mejora suelta de la fase 6.

## 4. Archivos

- `docs/arquitectura/tui.md` — estado, fila de la fase 7 en el §1 y §11
  nuevo (reglas de layout, marco, pantallas, decisiones, tests).
- `docs/roadmap.md` — fase 7 en el estado, su sección con 8 actividades y la
  decisión tomada.
- `docs/bitacora/fase-7/062-plan-fase-7.md`, índice con la sección de la
  fase 7.

## 5. Cómo verificar

```bash
make check
```

## 6. Pendientes

- Las actividades 2 a 8 de la fase 7.
- Mostrar estado, asignado y épica de la tarea o el responsable en QA, si
  se quieren: pide ampliar `provider.Issue` y `provider.QAIssue` y sus
  adaptadores.
