# 070 — Cierre de la fase 7

**Fecha:** 2026-10-28
**Rama:** docs/cierre-fase-7 → develop
**Fase del roadmap:** 7

## 1. Solicitud

Hacer la actividad 8 de la fase 7: el cierre de la fase y su release. Al
preguntarle la versión, el autor eligió **`v0.4.0` (*minor*)**, dijo que el
release lo publica él, y aceptó dejar fijada la regla de numeración hasta la
v1.0.

## 2. Motivación

| Problema | Daño real |
|---|---|
| El roadmap, `tui.md` y el README decían «en curso» y «por cerrar» | Con la fase terminada, el estado publicado quedaba viejo |
| La regla de numeración de los releases seguía «pendiente» en tres documentos (`CLAUDE.md`, `docs/release.md`, roadmap) | Cada release reabría la pregunta de qué `bump` usar, y los documentos se contradecían con lo que ya se hacía |
| Las pruebas manuales de las fases 4 a 7 estaban repartidas en 17 bitácoras | Sin reunirlas, se pierde de vista qué falta probar contra Atlassian real y en Windows Terminal antes de la v1.0 |

## 3. Diseño técnico

- **Sin cambios de código.** Los cambios de las actividades 2 a 7 ya están en
  `develop`; esta actividad solo cierra la fase.
- **Fase 7 cerrada** en el roadmap (estado, sección, actividad 8 y párrafo de
  cierre), en `tui.md` y en el README. El estado de la fase dice «por
  publicar» hasta que el autor haga el release.
- **Lo que entregó la fase** (bitácoras 062 a 069):
  - el diseño, el plan y el prototipo navegable que quedó en el repo (062,
    063);
  - el marco de una línea, el pie compacto y la línea de resultado (064);
  - la principal con descripciones, atajos, panel con pasos y contexto (065);
  - Generar: Tarea, Commits con detalle y archivos por commit, Ubicación con
    resumen y la línea de pasos (066);
  - la vista previa dibujada con un renderizador propio y `m` para el
    Markdown (067);
  - QA con contadores, estado por fila y detalle junto al consolidado (068);
  - Proyectos, Perfiles, Hallazgos, Exportar y Configuración con lista y
    detalle proporcionales y el filtro visible (069).
- **Lo que quedó fuera del plan y entró:** `profile.Scope`, `config.QAState`
  y `HasQAReport`, `qa.StatusOf` y `qa.Tally` (reglas que habrían vivido
  dentro de pantallas), y `moreKeys`, la parte opcional que permite un pie
  compacto sin cambiar el tipo de `keys()`.
- **Lo que el prototipo muestra y no se implementó,** por no leerlo draco, y
  que sigue pendiente si algún día se quiere (cambia puertos y adaptadores):
  estado, responsable y épica de la tarea; cuerpo y hora del commit;
  responsable de cada tarea de QA y conteo de errores o devoluciones; la
  letra A/M/D de cada archivo; el recuento de commits y archivos en la línea
  de estado de la vista previa; el texto del documento en el detalle de
  Exportar; y el progreso paso a paso al publicar (pide dividir
  `Session.Publish`).
- **Regla de numeración, fijada por el autor:** *minor* por fase cerrada y
  *patch* para correcciones y mejoras entre fases. La fase 7 sale como
  `v0.4.0` aunque no trae funciones nuevas, porque cambia lo que se ve en
  toda la TUI y varias teclas (`q`, `1`–`6`, `m`, la ayuda de `enter`). Se
  anota en `CLAUDE.md`, `docs/release.md` y la tabla de decisiones del
  roadmap, y sale de las pendientes.
- **Release `v0.4.0`** (lo publica el autor):

  ```bash
  make promote               # PR develop → main con merge commit, ramas al día
  make release bump=minor    # v0.4.0
  ```

- **Verificación del criterio de cierre,** después del release:
  1. con `v0.3.3` instalado, `draco` muestra el aviso de `v0.4.0` en la
     cabecera;
  2. `draco update` la instala y `draco version` responde `v0.4.0`;
  3. desde el binario instalado se abren las pantallas rediseñadas (la
     principal, Generar hasta la vista previa, QA, Proyectos y
     Configuración), a 80×24 y a ancho completo.

## 4. Archivos

- `docs/roadmap.md` — fase 7 cerrada, su párrafo de cierre, la regla en las decisiones y fuera de las pendientes.
- `docs/arquitectura/tui.md` — estado.
- `README.md` — estado.
- `CLAUDE.md`, `docs/release.md` — la regla de numeración.
- `docs/bitacora/fase-7/070-cierre-fase-7.md`, índice.

## 5. Cómo verificar

```bash
make check
# después del merge a develop (lo corre el autor):
make promote
make release bump=minor
draco                      # el aviso de v0.4.0 con v0.3.3 instalado
draco update && draco version
```

## 6. Pendientes

- Release `v0.4.0` y su verificación (arriba), a cargo del autor.
- Pruebas manuales, todas a cargo del autor:
  - fase 4 (029 a 032): Windows Terminal y Linux/WSL, y Generar contra Jira
    y Confluence;
  - fase 5 (035 a 038): Exportar, Backlog y QA contra Atlassian real;
  - fase 6 (042 a 047): verificación de la cuenta, asistente con `HOME`
    vacío, consolidado con la config creada en la TUI, aviso con un binario
    instalado, `/g` hasta publicar y `/qa`, temas con fondo claro y oscuro;
  - fase 7 (064 a 069): en Windows Terminal, a 80×24 y a ancho completo, el
    marco y el pie, la principal con sus cajas, Generar (Commits: marcar,
    `tab`, `h`), la **vista previa con un documento real de su plantilla**
    (067: el renderizador solo conoce el Markdown de las plantillas incluidas),
    QA con un consolidado real de varios sprints (068), y Configuración con
    el detalle de cada sección (069).
- Mejoras anotadas en las fases 6 y 7:
  - los formularios con la altura disponible (043);
  - mover el cursor dentro de la línea de la barra y escribir valores entre
    comillas (045);
  - el pie de teclas de `ansi` con colores ANSI básicos (047);
  - Huh con el ancho de su caja en las pantallas de formulario con contexto
    al lado (066 usa 64 como máximo).
- v1.0: instaladores (Homebrew tap y Scoop/winget), README completo y la
  bitácora enlazada por versión.
