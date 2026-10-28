# 069 — Listas de la fase 7: Proyectos, Hallazgos, Exportar y Configuración en columnas proporcionales

**Fecha:** 2026-10-28
**Rama:** feature/tui-listas-b → develop
**Fase del roadmap:** 7

## 1. Solicitud

El autor pidió seguir con la actividad 7 de la fase 7: las listas restantes
(Proyectos, Hallazgos, Exportar y Configuración) en columnas proporcionales y
con el filtro visible, según el §11.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Proyectos y Perfiles tenían una lista de 36 columnas y un detalle del ancho de su contenido, ambos de la altura de lo que tuvieran | A 160×40 casi todo el ancho quedaba vacío, y con muchos proyectos la lista no cabía |
| Configuración era una lista sola: para saber qué tenía cada sección había que abrirla | Se abría el formulario de QA para descubrir que faltaba `qa_report` |
| Exportar mostraba la lista de documentos sin decir nada más de cada uno | Dos documentos con títulos parecidos se distinguían abriéndolos |
| El filtro no se veía hasta pulsar `/` | Nadie descubría que las listas se filtran |

## 3. Diseño técnico

- **`panels` en `layout.go`:** lista y detalle como dos cajas en la
  proporción pedida y tan altas como el cuerpo. Cada lado recibe el sitio que
  tiene dentro de su caja (ancho y líneas) y devuelve su contenido; la caja
  recorta con «…» lo que no cabe. Una sola ayuda, usada por las cuatro
  pantallas y por Perfiles.
- **Proyectos y Perfiles (4/6), Hallazgos y Exportar (6/4),
  Configuración (4/6)**, desde 96 columnas. Debajo de 96 cada una conserva su
  disposición de antes. Perfiles no estaba en la lista de la actividad, pero
  es el mismo patrón que Proyectos y quedaba como la única lista con
  detalle sin rediseñar.
- **Filtro visible:** `filter.line()` (de la actividad 4) en Proyectos,
  Perfiles, Hallazgos y Exportar, también debajo de 96 columnas.
- **Configuración: detalle real, no el texto de ejemplo del prototipo.**
  Conexión (email, URL, space y *si* hay token: nunca su valor, con test),
  Proyectos, Perfiles con su destino, QA (`qa_email`, `qa_jira` con su
  cantidad de estados y `qa_report` con los nombres que hay, y el aviso de la
  actividad 3 si faltan) y Tema (los tres, con el activo marcado).
  El estado de cada sección en la lista usa ahora sangría francesa: se ajusta
  bajo su marca, no bajo el margen.
- **Exportar: el detalle no lleva el texto del documento** (el prototipo sí).
  Leerlo pide una llamada por documento al destino, y la pantalla siguiente
  ya lo hace. Muestra título, enlace, versión y qué hará `enter`.
- **Teclas:** `↑↓` y la acción principal en el pie; el resto, en `?`. La
  ayuda de `enter` en Proyectos pasa de «usar en esta sesión» a «usar»:
  con la larga, el pie de 80 columnas perdía `? más` (como pasó en Commits).
  Perfiles tenía siete teclas y también lo perdía.
- **Descubierto al dibujar:** a 80×24 Proyectos se desarmaba. Mi detalle ya
  no medía su contenido sino 56 columnas, y lista (36) + detalle (60) pasaban
  de 80. El detalle debajo de 96 columnas recibe ahora el ancho que queda
  (`width-44` al lado de la lista, `width-6` debajo), y un test lo cubre a 60,
  80 y 90 columnas.
- **Tests existentes que cambian por diseño:** el estado de cada sección y el
  texto explicativo de Perfiles se ajustan dentro de una caja más angosta, y
  las aserciones que dependían de una frase entera pasan a fragmentos.

## 4. Archivos

- `internal/tui/layout.go` — `panels`.
- `internal/tui/projects.go`, `profiles.go` — lista y detalle como contenido (`listText`, `detailText`), 4/6, filtro visible, teclas.
- `internal/tui/findings.go` — 6/4, filtro visible, teclas.
- `internal/tui/export.go` — lista y detalle (6/4), filtro visible, teclas.
- `internal/tui/settings.go` — secciones y detalle real (4/6), sangría francesa, teclas.
- `internal/tui/lists_layout_test.go` — nuevo: `panels`, y cada pantalla a 160×40 y 80×24 (detalle a la derecha, cajas hasta el pie, filtro, teclas, el token nunca visible).
- `internal/tui/projects_test.go`, `settings_test.go`, `qaconfig_test.go` — aserciones del diseño nuevo.
- `docs/arquitectura/tui.md`, `docs/guia-usuario.md` — las cuatro pantallas y sus teclas.
- `docs/roadmap.md` — actividad 7 con ✅.
- `docs/bitacora/fase-7/069-tui-listas-b.md`, índice.

## 5. Cómo verificar

```bash
go vet ./internal/tui && go test ./internal/tui   # corridos: ok
make check                                        # lo corre el autor
make run                                          # Proyectos, Configuración (↑↓ y el detalle), Backlog técnico y Exportar, a 80x24 y ancho
```

## 6. Pendientes

- `make check` completo y la mirada en Windows Terminal; Exportar con
  varios documentos de Confluence reales (heredado de la bitácora 035).
- Hallazgos con cientos de filas y la ventana de la lista: se probó con dos.
- Siguiente: la actividad 8, el cierre de la fase 7 y su release.
