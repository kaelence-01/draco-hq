# 075 — Formularios con paginador y línea bajo cada campo

**Fecha:** 2026-11-05
**Rama:** feature/tui-formularios → develop
**Fase del roadmap:** 7 (mejoras sobre la fase cerrada)

## 1. Solicitud

El autor vio el «Simulador QA de draco», un artifact de otra sesión, y pidió
traer a draco dos detalles:

- el paginador del formulario de QA, «Página 1 de 4» con los puntos que
  indican en cuál estás;
- la línea que se extiende bajo cada input.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Los formularios de varias páginas no decían cuántas tenían ni en cuál estabas | En Configuración › QA (cinco páginas) no se sabía cuánto faltaba ni qué agrupaba cada página |
| Entre campo y campo había solo una línea en blanco | Con descripciones largas, costaba ver dónde terminaba un campo y empezaba el siguiente |
| Descubierto al dibujar: el formulario no recibía el alto disponible | A 80×24, en Nuevo proyecto (ocho campos) y en la página Jira de QA, el campo con foco quedaba bajo el pie y no se veía lo que se escribía |

## 3. Diseño técnico

- **`form.go`**, nuevo, reúne lo de los formularios que estaba en
  `generate.go`: `newForm`, `formView` y `formBody`, más lo nuevo.
- **Páginas con nombre:**
  - `newPagedForm(newPage("Jira", campos…), …)` arma los grupos de huh y
    devuelve el `pager`, que la pantalla guarda junto al formulario.
  - `hiddenWhen` reemplaza a `WithHideFunc`: una página oculta (por ejemplo,
    el backlog sin ruta) no cuenta en el paginador.
  - huh no expone en qué grupo está el foco. El `pager` lo deduce del campo
    con foco (`GetFocusedField`), buscándolo entre los campos de cada
    página.
- **Paginador:** `●○○○○  Página 2 de 5 · Jira`, sobre el formulario y solo
  con dos páginas visibles o más. `●` va en el color de acento y `○` tenue.
  Son símbolos de un ancho, como pide la norma 9.
- **Nombres de las páginas:**

  | Formulario | Páginas |
  |---|---|
  | QA | Cuenta y encabezado · Jira · Otros proyectos · Campos personalizados · Revisión y observaciones |
  | Proyecto | Repositorios y VCS · Perfil · Backlog del backend · Backlog del frontend |
  | Perfil | Destino · Confluence o Carpeta Markdown · Plantilla |
  | Backlog | Qué analizar · Opciones |
  | Detectar desde Jira | Estados · Campos |

- **Línea bajo cada campo:**
  - El tema de huh (`withFieldRule`, en los tres temas) cambia la línea en
    blanco que separa los campos por una marca. `formBody`, que sí conoce
    el ancho, la reemplaza por una línea `┄` tenue del ancho del formulario.
  - Así la altura no cambia: la línea ocupa el lugar del blanco.
  - Un ancho fijo en el tema no servía: el formulario de Generar es más
    angosto (va en una caja al lado de «Lo que sigue»).
  - Queda bajo cada campo menos el último. Sobre el foco no hace falta
    marcarla: huh ya pone la barra `┃` a la izquierda del campo activo.
- **Alto disponible:** `formView` y `pager.formView` reciben el alto de la
  vista y se lo dan a huh (`WithHeight`), descontando el paginador y el
  error.
  - huh solo recalcula el desplazamiento en `Update`, así que tras fijar el
    alto se le pasa un mensaje vacío (`relayoutMsg`). Sin eso, el primer
    dibujo después de mover el foco mostraba la parte de arriba.
  - Generar dibuja su formulario en una caja con su propio alto, y sigue sin
    límite (`formBody(..., 0)`).
- **Descartado del simulador:** los botones «‹ Anterior / Siguiente ›». En la
  terminal eso lo hacen `enter` y `shift+tab`, y el pie ya lo dice.
- `docs/arquitectura/tui.md` suma la norma 12 (formularios) y el archivo al
  mapa. También corrige la línea de Proyectos que todavía decía «`esc`
  cierra el formulario sin guardar», que quedó desactualizada con la 074.

## 4. Archivos

- `internal/tui/form.go` — nuevo: `newForm`, `newPagedForm`, `page`, `pager` (`line`, `formView`), `formView` y `formBody` con alto, línea entre campos y `relayoutMsg`.
- `internal/tui/form_test.go` — nuevo: paginador con página oculta y con una sola página; QA con paginador y línea; a 80×24 el último campo con foco queda a la vista y ninguna línea pasa de 80 columnas.
- `internal/tui/styles.go` — `withFieldRule` en los tres temas.
- `internal/tui/generate.go` — sin los ayudantes de formulario; usa `formMaxWidth`.
- `internal/tui/qaconfig.go`, `projectform.go`, `profileform.go`, `backlog.go`, `qasetup.go` — páginas con nombre y su `pager`.
- `internal/tui/connection.go`, `export.go`, `qa.go` — `formView` con alto.
- `docs/arquitectura/tui.md` — norma 12, mapa de archivos y Proyectos con borradores.
- `docs/guia-usuario.md` — el paginador y el desplazamiento, en las teclas de formulario.
- `docs/roadmap.md` — 075 ✅.
- `docs/bitacora/fase-7/075-tui-formularios.md`, índice.

## 5. Cómo verificar

```bash
go vet ./... && gofmt -l ./cmd ./internal                  # corridos: limpio
go test ./...                                              # corridos: ok
make check                                                 # lo corre el autor
draco    # Configuración › QA › e, y Proyectos › n, a 80x24 y a ancho completo: lo corre el autor
```

QA (páginas 1 y 2) y Nuevo proyecto (desplazado hasta su último campo)
revisados a 80×24 con un test temporal.

## 6. Pendientes

- Verlo en Windows Terminal y en los tres temas (`┄` y `●○` dependen de la
  fuente; con la de Windows Terminal se ven bien).
- **076 (opcional):** módulo y sprint de QA elegidos de listas.
