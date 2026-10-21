# 047 — Temas del modo visual

**Fecha:** 2026-10-21
**Rama:** feature/tui-temas → develop
**Fase del roadmap:** 6

## 1. Solicitud

Seguir con la actividad 6 del plan de la fase 6 (bitácora 041): los temas
`ansi`, `alto-contraste` y `mono`, elegidos en Configuración.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Un solo juego de colores | En terminales con poco contraste o paletas raras, los avisos y el texto tenue se leen mal, y quien no quiere color solo tenía `NO_COLOR` para todo el sistema |
| Los estilos estaban fijos en variables de `styles.go`, en el pie de teclas y en el tema de los formularios | Un tema habría tenido que tocar cada pantalla |

## 3. Diseño técnico

- **`applyTheme(name)`** (`styles.go`) reasigna los estilos del paquete, los
  del pie de teclas (`helpStyles`) y el tema de los formularios de Huh
  (`formTheme`). Ninguna pantalla cambió: leen los estilos al dibujar.
  - `ansi`: el de siempre.
  - `alto-contraste`: colores brillantes, negrita en lo importante, y blanco
    en vez de tenue.
  - `mono`: sin color. Usa negrita, tenue y subrayado, y en los formularios
    la base de Huh sin sus colores (botones en vídeo inverso).

  Todo con colores ANSI básicos (norma 9).
- **Config:** campo `theme` (`config.Config.Theme`), que `mergeMissing`
  copia del archivo. `ansi` se guarda vacío, porque es el valor por
  defecto. `state.load` aplica el tema, así que rige desde que se abre la
  TUI y después de cada guardado.
- **Configuración › Tema** (`themes.go`):
  - `↑↓` prueba cada tema en toda la pantalla, con una muestra de cada
    estilo;
  - `enter` guarda;
  - salir sin guardar vuelve al tema guardado (`close()`).

  La sección muestra el tema actual, y avisa si la config trae uno que no
  existe (se usa `ansi`).
- **`NO_COLOR`** sigue ganando sin código propio: Bubble Tea quita el color
  al escribir en la terminal.
- **Sin cambio en `ansi`:** el pie de teclas conserva los estilos por
  defecto de Bubbles (grises en hexadecimal, desde la fase 4).

## 4. Archivos

- `internal/config/config.go`, `load_local_test.go` — `Theme` y su lectura
  desde el archivo.
- `internal/tui/styles.go` — los temas, `applyTheme` y `monoForm`.
- `internal/tui/themes.go`, `themes_test.go` — nuevos. Cubren:
  - probar, guardar y aplicar;
  - salir sin guardar;
  - `ansi` guardado vacío;
  - los tres temas en 80 columnas, con color salvo `mono`;
  - `mono` sin ningún código de color en la salida;
  - un tema desconocido.
- `internal/tui/settings.go` — la sección Tema.
- `internal/tui/tui.go` — `state.load` aplica el tema.
- `internal/tui/app.go` — el pie usa `helpStyles`.
- `internal/tui/generate.go` — `newPagedForm` usa `formTheme`.
- `docs/arquitectura/tui.md`, `docs/roadmap.md`, `README.md` — estado y
  diseño ajustado a lo implementado.
- `docs/bitacora/fase-6/047-tui-temas.md`, índice.

## 5. Cómo verificar

```bash
make test pkg=./internal/tui run=Theme v=1
make test pkg=./internal/config run=Theme
make check
make run            # Configuración › Tema: probar con ↑↓, enter guarda, esc deshace
```

## 6. Pendientes

- Prueba manual del autor: los tres temas en Windows Terminal y en una
  terminal de Linux o WSL, con fondo claro y oscuro.
- Mejora posible: el pie de teclas de `ansi` con colores ANSI básicos en
  vez de los grises de Bubbles.
- Actividad 7: cierre de la fase 6 y release `v0.2.2`.
