# 066 — Generar de la fase 7: tarea con lo que sigue, commits con detalle y ubicación con resumen

**Fecha:** 2026-10-28
**Rama:** feature/tui-generar-b → develop
**Fase del roadmap:** 7

## 1. Solicitud

El autor pidió seguir con la actividad 4 de la fase 7: Generar según el
prototipo aprobado. Al presentarle el plan eligió **un solo commit** (no
partirla en 4a y 4b) y **sin progreso falso al publicar**.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Commits ocupaba un tercio del ancho y mostraba cada commit en una línea sin sus archivos | Para saber qué archivos entraban al documento había que marcar, pasar al paso siguiente y mirar el resultado; un commit equivocado se descubría tarde |
| Nada decía cuántos archivos sumaban los commits marcados de los dos repos | No se podía comprobar que lo marcado era lo esperado antes de leer el documento |
| Tarea y Ubicación eran una columna angosta con el resto de la pantalla vacía | A 160×40 casi todo el ancho quedaba sin usar, y Ubicación no decía qué haría elegir un lugar |
| Los pasos del flujo no se veían | Quien llega a Ubicación no sabe cuánto falta |

## 3. Diseño técnico

- **`layout.go` suma sus consumidores:** `columns` (reparto proporcional),
  `stretch` (caja que llena el alto), `stepper` y `titleLine` (la línea de
  pasos, desde 96 columnas). Cada flujo dice sus pasos en `docAbout.steps`
  (`deploySteps`, `qaSteps`): Ubicación y Publicar son compartidos con QA.
- **`profile.Scope`:** la regla del destino (space de la petición, del
  proyecto, del perfil, o la carpeta Markdown) salía de `profile.Open` y
  estaba enterrada ahí. Ahora es una función pura con su tabla de pruebas, y
  `Open` la usa sin cambiar de comportamiento. La caja «Lo que sigue» muestra el
  destino con ella (norma 1: la pantalla no repite la regla).
- **Generar › Tarea:** desde 96 columnas, el formulario de Huh en una caja y,
  a su derecha, «Lo que sigue»: pasos, repos del proyecto elegido
  (`deploy.ProjectSides`) con su clon, destino y una frase. Sigue al proyecto
  del formulario.
- **Commits** (`commits.go`, reescrito):
  - Tabla (hash, mensaje, autor, fecha, archivos) al 60 % y detalle al 40 %,
    con la franja **Seleccionados** debajo, de los dos repos y con el total de
    archivos por lado, como el documento.
  - **Archivos por commit, un hash a la vez** (`need`, en un `tea.Cmd`): al llegar
    el cursor, al agregar un hash a mano y, para los hashes de la barra
    (`/g -i OPS-7 -b a1b2c3d`), al cargar el lado. Se guardan por hash; un
    commit no se lee dos veces. El puerto `ChangeSource` no cambia.
  - La tabla suelta autor y fecha si el mensaje se quedaría con menos de 24
    columnas (a 110 columnas el mensaje ganaba 16 y no se leía): están en el
    detalle.
  - Debajo de 96 columnas el detalle baja: una línea de archivos y la cuenta de
    marcados.
  - Teclas: principales `↑↓ elegir`, `espacio`, `tab` y `enter`; `h`, `↑/k` y
    `↓/j` en la ayuda completa (`moreKeys`).
- **Ubicación:** lista con la línea `/ filtrar` siempre visible (`filter.line`; las
  demás listas la reciben en la actividad 7) y el **Resumen** al lado: título,
  destino, página nueva junto a la elegida y enlace posterior. Debajo de 96
  columnas el resumen baja. Es compartido con QA.
- **Publicar** y **Preparar documento** llevan la línea de pasos; el resultado
  de Publicar va en una caja.
- **Decisión del autor: sin progreso falso al publicar.** El prototipo muestra
  «Creando la página… / Enlazando…», pero `Session.Publish` es una sola
  llamada sin avisos; partirla tocaba `internal/deploy` e `internal/qa` en una
  fase de interfaz. El resumen de Ubicación dice qué pasará y Publicar sigue con
  su spinner único.
- **Lo que no se implementa, por no leerlo draco:** estado y responsable de la
  tarea, cuerpo y hora del commit (ya acordado en el §11); la letra A/M/D de
  cada archivo (`ChangedFiles` devuelve rutas); y el título de la tarea en la
  línea de Commits: se lee en el paso siguiente, y leerlo antes sería una
  llamada más al tracker solo para esa línea.
- **Descubierto al probar a 80×24:** con la ayuda de `tab` como «backend/frontend»
  el pie de Commits se cortaba y **desaparecía `? más`**. Ahora dice «repo» y
  hay un test que lo vigila.
- **Tests existentes:** `fakeGit` cuenta las llamadas a `ChangedFiles` (la
  pantalla de Commits también lee, de un hash), y los tests que comprobaban
  «aún no se leyó nada» comparan contra la cuenta anterior. Cambió el texto
  de unas aserciones (`pagos-api · develop`, el error de perfil partido en dos
  líneas por el ancho de la caja); ningún flujo cambió.

## 4. Archivos

- `internal/profile/profile.go`, `scope_test.go` — `Scope`, usada por `Open`.
- `internal/tui/layout.go`, `layout_test.go` — `columns`, `stretch`, `stepper`, `titleLine`.
- `internal/tui/filter.go` — `line`.
- `internal/tui/docrun.go` — `docAbout.steps`.
- `internal/tui/generate.go` — la caja «Lo que sigue».
- `internal/tui/commits.go` — tabla, detalle, franja, lectura de archivos por hash, versión angosta.
- `internal/tui/locations.go` — dos columnas, filtro visible, resumen.
- `internal/tui/publish.go`, `progress.go` — línea de pasos; resultado en una caja.
- `internal/tui/generate_layout_test.go` — nuevo: 160×40, 110×34 y 80×24 de Tarea, Commits, Ubicación, Publicar y Preparar; lectura de archivos una vez por hash; franja deduplicada; barra.
- `internal/tui/generate_test.go` — `fakeGit` con llamadas contadas; textos de la tabla.
- `docs/arquitectura/tui.md`, `docs/guia-usuario.md` — Generar, Ubicación y los pasos.
- `docs/roadmap.md` — actividad 4 con ✅.
- `docs/bitacora/fase-7/066-tui-generar-b.md`, índice.

## 5. Cómo verificar

```bash
go vet ./internal/tui ./internal/config ./internal/profile && go test ./internal/tui ./internal/config ./internal/profile   # corridos: ok
make check      # lo corre el autor
make run        # Generar a 80x24 y a más de 120 columnas: Tarea, Commits (marcar, tab, h), Ubicación
```

## 6. Pendientes

- `make check` completo y la mirada en Windows Terminal (80×24 y ancho), a
  cargo del autor; Generar de punta a punta contra Jira y Confluence reales
  sigue sin probarse (heredado de las bitácoras 029 a 032).
- Vista previa renderizada (actividad 5) y QA (actividad 6): comparten Ubicación
  y Publicar, que ya llevan su línea de pasos.
- `filter.line` en Proyectos, Hallazgos, Exportar y Configuración (actividad 7).
