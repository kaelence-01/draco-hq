# 035 — Exportar en la TUI

**Fecha:** 2026-10-20
**Rama:** feature/tui-exportar → develop
**Fase del roadmap:** 5

## 1. Solicitud

Seguir con la siguiente actividad de la fase 5: Exportar (actividad 2 del
plan de la bitácora 034).

## 2. Motivación

| Problema | Daño real |
|---|---|
| «Exportar página» estaba atenuada en el menú | Quien abre la TUI tenía que salir a `draco f` para bajar el documento de una tarea |
| La validación de la tarea, el formulario y las filas de documentos vivían dentro de Generar y Ubicación | Exportar los necesita iguales; copiarlos daba dos versiones que se separan con el tiempo |
| Las filas de Ubicación dejaban que una URL larga se partiera en la columna 0 | La lista calcula sus filas visibles con dos líneas por documento: con URLs partidas se pasaba del alto y el texto quedaba desalineado |

## 3. Diseño técnico

- **Tres pantallas sobre `fetch.Session`** (`internal/tui/export.go`), con
  las decisiones de `draco f`:
  1. **Formulario:** tarea (obligatoria) y space o carpeta (vacío: el del
     perfil). `fetch.Resolve` y `profile.Open` con `AnyScope`, como el
     comando: en Confluence sin space se busca en todos. Un error del perfil
     se explica en el formulario (`explainProfile`, el de Generar).
  2. **Documentos:** `Search` en un `tea.Cmd` con spinner; `esc` lo cancela.
     Con un solo documento sigue sin preguntar, como el comando; con varios,
     lista con filtro (`/`), el más reciente primero. `ErrNoPages` dice
     dónde se buscó (proveedor y space o carpeta).
  3. **Texto y guardado:** `Export` en un `tea.Cmd`; el texto en un
     `viewport` y debajo la ruta en un `textinput` con el foco, que trae la
     carpeta actual y el nombre sugerido. `↑↓`/`pgup`/`pgdn` desplazan el
     texto y lo demás va a la ruta (la pantalla captura `?`). Si el archivo
     existe, pide `s`; `esc` vuelve a la ruta sin salir. El resultado da la
     ruta y el comando equivalente (`draco f -i OPS-7 [-s X] -o <ruta>`, con
     comillas si la ruta tiene espacios); `enter`/`esc` vuelven al inicio.
- **Guardar va directo en `update`**: es un archivo de texto pequeño
  (`Export.Save`), no disco lento; `os.Stat` decide si hay que confirmar.
- **Sin diálogo nativo** (decidido en la 034): una ventana del sistema dejaría
  la TUI esperando sin señal y en un test abriría una ventana real.
- **Compartido, no copiado:** `taskInput` (tarea obligatoria), `newForm`
  (tema y ancho de los formularios; también lo usa Proyectos) y `formView`
  (error encima del formulario, ahora ajustado al ancho) salen de
  `generate.go`; `docRows` (título y URL alrededor del cursor) sale de
  `locations.go`. Ubicación y Exportar usan las mismas filas, **recortadas
  al ancho** (norma 9: las filas de una lista son de una línea); antes una
  URL larga se partía.
- **Menú:** «Exportar página» abre la pantalla; el panel ya no dice que falta.

## 4. Archivos

- `internal/tui/export.go` — nuevo: formulario, documentos y texto con
  guardado.
- `internal/tui/export_test.go` — nuevo: de punta a punta con el adaptador
  Markdown real (perfil sin space, filtro, archivo guardado y comando),
  documento único, otra ruta (con espacios y `?`), confirmación al
  reemplazar, búsqueda vacía, `esc` que cancela la búsqueda y 80 columnas.
- `internal/tui/generate.go` — `taskInput`, `newForm`, `formView`.
- `internal/tui/locations.go` — `docRows`, recortadas al ancho.
- `internal/tui/projectform.go` — usa `newForm`.
- `internal/tui/home.go` — «Exportar página» con pantalla.
- `docs/arquitectura/tui.md`, `docs/roadmap.md`, `README.md` — estado y
  detalle de Exportar.
- `docs/bitacora/fase-5/035-tui-exportar.md`, índice.

## 5. Cómo verificar

```bash
make test pkg=./internal/tui run=Export v=1
make check
make run            # Exportar página: tarea, documento, enter guarda
```

## 6. Pendientes

- Pruebas manuales del autor: Exportar en Windows Terminal y Linux/WSL con
  un perfil Markdown, y contra Confluence (con y sin space en el perfil).
