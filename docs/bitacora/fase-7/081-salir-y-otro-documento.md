# 081 — Salir con q desde cualquier pantalla y otro documento al publicar

**Fecha:** 2026-11-23
**Rama:** feature/salir-y-otro-documento → develop
**Fase del roadmap:** 7 (mejoras sobre la fase cerrada)

## 1. Solicitud

Tras probar la 080, el autor pidió dos cosas: una opción para salir que no
fuera solo `ctrl+c`, y que al terminar de publicar un documento hubiera una
tecla (propuso `n` o `ctrl+n`) para ir directo a crear otro, escribiendo solo
la tarea, sin volver al inicio y entrar de nuevo a Generar.

## 2. Motivación

| Problema | Daño real |
|---|---|
| `q` solo salía desde la principal; en las demás pantallas había que usar `ctrl+c` o volver con `esc` hasta el inicio | Salir no era obvio, y `ctrl+c` se siente como interrumpir |
| Tras publicar, `enter` volvía al inicio | Documentar varias tareas seguidas pedía volver al menú, elegir Generar y el proyecto cada vez |

## 3. Diseño técnico

- **`q` pasa al marco** (`exitKey` en `app.go`), al lado de `?` y `esc`:
  sale desde cualquier pantalla que no capture la tecla, y cierra antes las
  pantallas (`app.quit`, lo mismo que `ctrl+c`), así que los pasos en curso
  se cancelan. La principal ya no la maneja aparte.
- **Quién se queda con `q`:** lo que ya capturaba las teclas al escribir
  (formularios de Huh, búsquedas `/`, la barra de comandos, la ruta de
  Exportar), así que una `q` escrita nunca cierra draco. Además, **Publicar
  y Borrar del historial** la capturan mientras trabajan: salir a medias
  podría dejar un documento sin su enlace en la tarea, o borrado con el
  enlace todavía puesto. `ctrl+c` sigue saliendo siempre.
- **Ayuda:** el marco agrega `q salir` a la ayuda completa (`?`) de todas las
  pantallas, salvo cuando la pantalla la captura, y `ctrl+c salir` siempre.
  En el pie compacto no hay sitio a 80 columnas.
- **`n` al terminar de publicar** (`againKey`). Publicar lo comparten Generar
  y QA y no sabe de qué flujo es, así que lo ofrece `docAbout.again`: una
  función que da la primera pantalla de otra corrida, o `nil` si no hay. Solo
  la tiene Generar (`genRun.again`): el formulario con la tarea vacía y el
  proyecto recién usado (`Target.ProjectName`), no el activo, para seguir con
  la siguiente tarea del mismo proyecto.
- **Navegación:** `tea.Sequence(toHome, open(...))`. Se cierran las pantallas
  del documento publicado y Generar queda sobre la principal
  (`Inicio › Generar documento`), así que `esc` vuelve al inicio y no a un
  documento ya publicado. La línea del resultado (`✓ OPS-7 publicado…`) sigue
  sobre el pie.
- **`n` y no `ctrl+n`:** una sola tecla, y en esa pantalla no hay nada que
  escribir. El resultado lo dice («n genera otro documento · enter vuelve al
  inicio») y el pie lo muestra.

## 4. Archivos

- `internal/tui/app.go` — `exitKey` en el marco, `app.quit`; la ayuda completa agrega `q salir` cuando la pantalla no la captura.
- `internal/tui/home.go` — sin su propia `q`.
- `internal/tui/publish.go` — `againKey` (`n`) y la línea que lo explica; `q` capturada mientras publica.
- `internal/tui/docrun.go` — `docAbout.again`; `genRun.again` (Generar vacío con el proyecto usado).
- `internal/tui/historyremove.go` — `q` capturada mientras borra.
- `internal/tui/app_test.go` — `q` sale desde cualquier pantalla y la cierra; una pantalla que la captura la recibe, y la ayuda solo ofrece `ctrl+c`.
- `internal/tui/publish_test.go` — `q` no sale mientras publica; `n` abre Generar vacío, con el proyecto usado, sobre la principal, y sigue a los commits.
- `docs/guia-usuario.md` — `q` en las teclas de todas las pantallas; `n` tras publicar.
- `docs/arquitectura/tui.md` — teclas globales y decisiones (`q` en el marco, `n` con `docAbout.again`).
- `scripts/release.sh` — `[y/N]` (cambio del autor, como `merge.sh` y `promote.sh` en la 080).
- `docs/roadmap.md` — 081 ✅; `qa-listas` pasa a 082.
- `docs/bitacora/fase-7/081-salir-y-otro-documento.md`, índice.

## 5. Cómo verificar

```bash
gofmt -l ./internal ./cmd     # corrido: limpio
go vet ./...                  # corrido: limpio
go test ./internal/tui        # corrido: ok
make check                    # lo corre el autor
```

En una terminal real (lo corre el autor), `go run .`:

- `q` sale desde Proyectos, Commits (sin búsqueda abierta), la vista previa y
  el Historial; en el formulario de Generar o con `/` abierto, se escribe.
- Generar un documento y, al terminar, `n`: Generar con la tarea vacía y el
  mismo proyecto; `esc` desde ahí vuelve al inicio.

## 6. Pendientes

- Un «otro consolidado» para QA con `n`: no se pidió, y basta con darle
  `again` a `qaRun`.
