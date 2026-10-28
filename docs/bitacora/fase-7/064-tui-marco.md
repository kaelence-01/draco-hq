# 064 — Marco de la fase 7: cabecera de una línea, pie compacto y línea de resultado

**Fecha:** 2026-10-28
**Rama:** feature/tui-marco → develop
**Fase del roadmap:** 7

## 1. Solicitud

El autor pidió seguir con la fase 7. La primera actividad sin ✅ del roadmap es
la 2: el marco (cabecera, pie, línea de resultado y `layout.go`), según el
§11 de `tui.md` y el prototipo aprobado.

## 2. Motivación

| Problema | Daño real |
|---|---|
| El marco gastaba tres líneas arriba en toda pantalla apilada (cabecera, raya y ruta) | En 80×24 cada línea cuenta: el cuerpo de las pantallas de varios pasos quedaba corto |
| El pie listaba todas las teclas más `ctrl+c`; Proyectos llevaba ocho | Una línea larga que se corta en terminales angostas y esconde `?` |
| Tras publicar, el resultado solo estaba en la pantalla que se cierra con `esc` | Al volver al inicio no queda rastro de qué se publicó ni de que faltó el enlace |

## 3. Diseño técnico

- **Cabecera de una línea** (`app.go`, `layout.go`): `draco v0.3.3 │ Inicio ›
  Proyectos` a la izquierda y perfil y proyecto a la derecha, más la raya.
  La principal queda igual de alta (ya no tenía ruta); cada pantalla apilada
  gana una línea. Cuando no cabe todo, cede primero el perfil (queda el
  proyecto), después el proyecto y, al final, la ruta pierde sus primeras
  pantallas (`… › Ubicación › Publicar`): la pantalla actual siempre se ve.
  `edges` y `trail` son funciones puras de `layout.go`.
- **Pie compacto:** teclas principales de la pantalla, `esc volver` y `? más`.
  Con `?` se muestran también las secundarias, `ctrl+c salir` y `? menos`; el
  cuerpo se encoge lo que crece el pie.
- **Teclas secundarias con `moreKeys()` opcional**, en vez de marcarlas dentro
  de `keys()` como decía el §11. Alternativas descartadas:
  - cambiar el tipo que devuelve `keys()`: `key.Binding` no puede llevar la
    marca, y el cambio tocaba 28 funciones en 23 archivos a la vez;
  - un registro global de teclas secundarias: la misma tecla puede ser
    principal en una pantalla y secundaria en otra.

  Sin `moreKeys`, todas las teclas de `keys()` van en el pie. Se clasificó
  solo Proyectos (ocho teclas: `d` y `x` pasan a la ayuda completa). Cada
  pantalla se clasifica cuando se rediseña (actividades 3 a 7).
- **Línea de resultado:** la pantalla Publicar, al terminar bien, pide al
  marco (`report`, mensaje `resultMsg`) mostrar `✓ OPS-7 publicado en «…»`
  (o `actualizado`) sobre el pie. Es estado en memoria de `state`, como el
  proyecto activo; dura hasta salir de la TUI. Si el documento no se pudo
  enlazar desde la tarea (`LinkErr`), la línea lleva `⚠` y
  «; sin enlace desde OPS-7»: no da por bueno lo que QA marcará sin
  documentación. En QA el sujeto es «Consolidado».
- **`layout.go` solo con lo que el marco usa.** Las columnas proporcionales, la
  caja que se estira y la línea de pasos llegan con la primera pantalla que
  las usa (actividades 3 y 4); sin consumidor serían código muerto.
- **La ruta larga ya no cabe completa a 110 columnas.** El test de Publicar
  que comprobaba la ruta de siete niveles ahora se redimensiona a 160×40;
  la elisión tiene su propio test.
- **Pie en español:** `? ayuda` pasa a `? más` (y `? menos` con la ayuda
  completa), como el prototipo.

## 4. Archivos

- `internal/tui/layout.go` — nuevo: `edges` (extremos de una línea) y `trail` (ruta que cede por la izquierda).
- `internal/tui/app.go` — cabecera de una línea, pie compacto y completo, `moreKeyer`, `resultMsg` y `report`; se quita `crumbs`.
- `internal/tui/tui.go` — `state.result` y `state.resultWarn`.
- `internal/tui/publish.go` — `summary` y el aviso al marco al publicar.
- `internal/tui/projects.go` — `d` y `x` pasan a `moreKeys`.
- `internal/tui/app_test.go` — marco a 80×24 y 160×40, cabecera que cede, pie compacto y completo, línea de resultado; `? ayuda` → `? más`.
- `internal/tui/publish_test.go` — el resultado queda al volver al inicio, con `⚠` si falla el enlace y `actualizado`; ruta completa a 160×40.
- `internal/tui/projects_test.go` — el pie lleva las principales y la ayuda completa, todas.
- `docs/arquitectura/tui.md` — árbol, interfaz `screen` y `moreKeys`, marco común, §11 (marco y decisiones), norma 4.
- `docs/guia-usuario.md` — cabecera, pie con `? más`, línea de resultado, `?` y teclas de Proyectos.
- `docs/roadmap.md` — actividad 2 de la fase 7 con ✅.
- `docs/bitacora/fase-7/064-tui-marco.md`, índice.

## 5. Cómo verificar

```bash
go vet ./internal/tui && go test ./internal/tui   # corridos: ok
make check                                        # lo corre el autor
make run                                          # abrir la TUI y mirar la cabecera, el pie y ?
```

## 6. Pendientes

- `make check` completo y la mirada en una terminal real (Windows Terminal,
  80×24 y ancha) quedan a cargo del autor.
- El resto de las pantallas clasifica sus teclas al rediseñarse (actividades 3 a 7).
- La línea de pasos del flujo (`✓ Tarea › Commits › …`) llega con la actividad 4.
