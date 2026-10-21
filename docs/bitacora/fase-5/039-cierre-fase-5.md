# 039 — Cierre de la fase 5

**Fecha:** 2026-10-21
**Rama:** docs/cierre-fase-5 → develop
**Fase del roadmap:** 5

## 1. Solicitud

Hacer la actividad 5 de la fase 5: el cierre de la fase y su release,
saltando las pruebas manuales. El autor decidió además que las fases 5 y 6
se publiquen como *patch* de `v0.2.x` (solo sube el último número), y pidió
dejarlo registrado como contexto del proyecto.

## 2. Motivación

| Problema | Daño real |
|---|---|
| El roadmap, el README y `docs/arquitectura/tui.md` decían «en curso» y «por cerrar» | Con la fase terminada, el estado publicado quedaba viejo |
| `tui.md` decía que QA, Exportar y Backlog seguían atenuadas en el menú | Quien lee el diseño esperaría entradas sin pantalla que ya funcionan |
| El plan fijaba `v0.3.0` y la guía de release mostraba `bump=minor` | El release habría subido el *minor* contra la decisión del autor |
| La regla de versiones solo estaba en la conversación | En la fase 6 se volvería a planear un `v0.3.0` |

## 3. Diseño técnico

- **Versiones (decisión del autor):** las fases 5 y 6 se publican con
  `make release bump=patch`. La fase 5 sale como `v0.2.1` y la fase 6 sigue
  con `v0.2.2` en adelante. El *minor* no sube hasta que el autor lo decida.
  Queda en «Decisiones tomadas» del roadmap, en `CLAUDE.md` (junto a
  `make release`) y en `docs/release.md`, cuyo ejemplo pasa a `bump=patch`.
  La fila de la actividad en el roadmap conserva el `v0.3.0` planeado (034)
  para que se entienda el cambio.
- **Pruebas manuales saltadas (decisión del autor):** las de las bitácoras
  035 a 038 no se hicieron. Las filas del roadmap siguen diciendo «faltan las
  pruebas manuales del autor» y el párrafo de cierre lo repite: el release
  sale sin haberse probado contra Jira y Confluence reales.
- **Sin cambios de código:** el menú de la principal ya abre Generar,
  Proyectos, QA, Exportar y Backlog. Solo Configuración queda atenuada, y es
  de la fase 6. La ayuda general ya menciona el modo visual desde la 033.
- **Lo que entregó la fase** (bitácoras 034 a 038):
  - el diseño de las pantallas y el plan;
  - Exportar: documentos con filtro, texto y ruta editable;
  - el núcleo del backlog en `internal/backlog`, `Scan` con `context`, y la
    pantalla con progreso por repo, hallazgos, detalle y guardado;
  - QA con evaluación en vivo, una fila por tarea;
  - vista previa, ubicación y publicar compartidos por Generar y QA
    (`docRun`).
- **Release `v0.2.1`** (lo publica el autor, que tiene la llave SSH y `gh`):
  `develop` → `main` por PR con *merge commit*, y `make release bump=patch`
  desde `main` al día (`docs/release.md`).
- **Verificación del criterio de cierre**, después del release:
  1. con `v0.2.0` instalado, `draco update --check` anuncia `v0.2.1`;
  2. `draco update` la instala y `draco version` responde `v0.2.1`;
  3. en una terminal nueva, `draco` abre la TUI, y Consolidado QA, Exportar
     y Backlog se abren desde el menú.

## 4. Archivos

- `docs/roadmap.md` — fase 5 cerrada, `v0.2.0` publicado, párrafo de cierre
  y la decisión de versiones.
- `docs/arquitectura/tui.md` — estado; solo Configuración atenuada.
- `README.md` — estado: fases 4 y 5, Configuración en la fase 6.
- `docs/release.md` — ejemplo con `bump=patch` y la regla de `v0.2.x`.
- `CLAUDE.md` — la regla de versiones junto a `make release`.
- `docs/bitacora/fase-5/039-cierre-fase-5.md`, índice.

## 5. Cómo verificar

```bash
make check
# después del merge a main (lo corre el autor):
git switch main && git pull --ff-only
make release bump=patch             # v0.2.1
draco update --check && draco update && draco version
draco                               # QA, Exportar y Backlog desde el menú
```

## 6. Pendientes

- Release `v0.2.1` y su verificación (arriba), a cargo del autor.
- Pruebas manuales de las bitácoras 035 a 038, saltadas en este cierre:
  Exportar, Backlog y QA contra Jira y Confluence reales, y Generar de punta
  a punta después de la 038. Un error ahí se corrige con un `fix/` y un
  *patch*.
- Fase 6: barra de comandos con autocompletado, Configuración en
  formularios, temas, ayuda y aviso de versión nueva, con plan antes de
  escribir código, como en las fases 4 y 5. Sus releases siguen en `v0.2.x`.
