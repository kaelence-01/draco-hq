# 028 — Diseño de la TUI y plan de la fase 4

**Fecha:** 2026-10-20
**Rama:** docs/plan-fase-4 → develop
**Fase del roadmap:** 4

## 1. Solicitud

Avanzar con la fase 4 de draco (modo visual). La bitácora 027 dejó como
pendiente hacer el plan antes de escribir código.

## 2. Motivación

| Problema | Daño real |
|---|---|
| La fase 4 era una lista de pantallas, sin actividades ni criterio de cierre | No había forma de saber qué cierra cada actividad ni la fase, como sí la hubo en las fases 2 y 3 |
| Sin diseño de la TUI | Decisiones de fondo (versión de las librerías, cómo se cancelan los pasos, cómo se prueba sin terminal, cómo se guarda la config) se habrían tomado a mitad de una pantalla, una distinta en cada una |
| `deploy.Resolve` exige los hashes, y el selector de commits necesita antes las carpetas de los repos | Sin decidirlo, la TUI habría repetido la lógica del flujo, contra la regla de un solo núcleo |

## 3. Diseño técnico

Diseño completo en `docs/arquitectura/tui.md`. Decisiones principales:

- **Charm v2** (`charm.land/bubbletea/v2` 2.1.0, `bubbles/v2` 2.2.1,
  `lipgloss/v2` 2.0.6, `huh/v2` 2.0.3, versiones consultadas en el proxy de Go
  al planear). Descartada la v1 (`github.com/charmbracelet/…`): empezar ahí
  obliga a migrar. Cada librería se agrega en la actividad que la usa: Huh
  llega con Proyectos.
- **Entrada:** `App.Run` sin argumentos abre la TUI solo con `App.tty`; sin
  TTY sigue la ayuda. `cmd` la llama por una variable (`runTUI`) que los
  tests reemplazan, y le pasa sus dependencias en `tui.Deps`: sin ciclo de
  imports y con los mismos dobles de los tests de `cmd`. `Start` (oferta de
  instalación) no cambia; el aviso de versión nueva en la TUI queda para la
  fase 6.
- **Modelo:** pila de pantallas, `esc` vuelve; marco común (cabecera con
  perfil y proyecto, pie de teclas). Cada paso de la `Session` corre como
  `tea.Cmd`; la pantalla no lee la `Session` hasta recibir el resultado
  (el paso escribe desde otra goroutine), y un resultado de un paso cancelado
  se descarta por número de secuencia.
- **Proyecto activo en memoria, «por defecto» en la config:** cambiar de
  proyecto para una tarea no debe cambiar la config. Guardar sigue la regla
  del modo comando: `LoadFile` + funciones puras + `config.Save`, nunca
  `LoadLocal`.
- **Generar** recorre los pasos de `deploy.Session` con las mismas decisiones
  que `draco g`. Para el selector de commits se extraerá del flujo una función
  pura con los repos del proyecto, que `Resolve` también usará (actividad
  4a). Los commits marcados van en el orden de la lista (el más reciente
  primero, que es el que enlaza el documento); el hash a mano cubre los
  commits de otra rama.
- **Menú:** lo que aún no tiene pantalla (QA, Exportar, Backlog,
  Configuración) aparece atenuado con su comando, en vez de ocultarse.
- **Sin emojis en el marco** (la maqueta del roadmap los tiene): su ancho
  varía entre terminales y rompe bordes, sobre todo en Windows.
- **Plan:** 5 actividades (base, proyectos, generar en dos partes y cierre con
  `v0.2.0`), cada una con criterio de cierre, en el roadmap. El autor aprobó
  el plan y la división de Generar.

## 4. Archivos

- `docs/arquitectura/tui.md` — nuevo: diseño del modo visual.
- `docs/roadmap.md` — fase 4 en curso, sus actividades y la referencia al
  diseño.
- `CLAUDE.md` — el diseño en el árbol de `docs/`.
- `docs/bitacora/fase-4/028-plan-fase-4.md`, `docs/bitacora/README.md` —
  primera entrada de la fase 4 y su sección en el índice.

## 5. Cómo verificar

```bash
make check
```
