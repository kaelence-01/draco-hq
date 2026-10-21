# 051 — La configuración de gtt primero

**Fecha:** 2026-10-21
**Rama:** feature/config-gtt-primero → develop
**Fase del roadmap:** 6

## 1. Solicitud

El autor llegó en la TUI hasta «Carpeta de plantillas del perfil», dio
enter y el formulario terminó sin pedir las rutas de los repos. Pidió que lo
que ya tenía gtt (credenciales, `confluence_space_key`, `default_project` y
`projects`) sea lo principal y tenga su propia sección, y que lo nuevo de
draco (perfiles, QA, tema) quede aparte, como opcional.

## 2. Motivación

| Problema | Daño real |
|---|---|
| El asistente y `draco init` exigían crear un perfil | Quien viene de gtt contesta preguntas que no necesita, y su config queda distinta a la de gtt (`profiles.equipo.docs.space` en vez de `confluence_space_key`) |
| `confluence_space_key` no se pedía en ningún lado | El space quedaba dentro de un perfil, aunque una config sin perfiles ya funciona como gtt (perfil implícito) |
| Proyectos no estaba en Configuración; el asistente solo lo ofrecía con `p` al final | Lo único indispensable para generar el documento se pasaba por alto: el formulario de perfil terminaba y no decía qué seguía |
| «Vacío: las de siempre» en la carpeta de plantillas | No dice nada a quien no conoce la historia de draco |

## 3. Diseño técnico

- **Configuración en dos bloques** (`settings.go`): Conexión con Atlassian y
  Proyectos arriba; bajo «Opcional», Perfiles, QA y Tema. `section` suma
  `optional`, y la vista escribe «Opcional» antes de la primera.
  - Proyectos abre la misma pantalla de la principal (`newProjects`); la
    principal la conserva, porque ahí se elige el proyecto de la sesión.
  - Estado de Proyectos: `⚠ ninguno: hace falta uno para generar el
    documento`, o cuántos hay y cuál es el por defecto.
  - Perfiles sin perfiles dice `ninguno: se usa jira → confluence (space
    DEV)`, en vez del nombre del implícito con «(implícito)».
- **Conexión pide el space solo sin perfiles.** `Config.Profile` devuelve
  el implícito con `confluence_space_key` solo cuando no hay perfiles; con
  perfiles, cada uno trae su space y el del nivel principal no se usa.
  Pedirlo ahí sería un campo sin efecto. `CONFLUENCE_SPACE_KEY` se suma a
  las variables cuyo valor del entorno se avisa (gana sobre el archivo, como
  las `ATLASSIAN_*`).
- **El asistente sigue con el primer proyecto** (`connection.go`): tras
  verificar la cuenta, si el archivo no tiene proyectos, `enter` vuelve a
  Configuración y abre Proyectos con el formulario de uno nuevo. Al
  guardarlo queda la lista con la nota; `esc` vuelve a Configuración. Vale
  también fuera del asistente: quien edita la conexión sin proyectos sigue
  con uno.
  - Se quitan el modo `profileFirst` del formulario de perfil, la oferta
    `p` de Configuración (`offerProject`, `projectKey`) y
    `settings.firstProfile`: Proyectos ya es una sección visible.
- **`draco init`, igual que el asistente** (decisión del autor: cambiar los
  dos modos juntos, para no tener dos formas de configurar que guardan
  distinto):
  - sin perfiles, pregunta el space (`confluence_space_key`, opcional);
    con perfiles, lo muestra como antes;
  - ya no crea perfiles; sin proyectos, explica que generar un documento
    necesita uno antes de ofrecerlo;
  - resumen y ayuda dicen «credenciales, el space y un proyecto», y que los
    perfiles se agregan con `draco profile add`.
- **Plantillas del perfil:** «Opcional, con deploy/ y qa/ dentro. Vacío: la
  plantilla incluida».
- **Guía de usuario:** primeros pasos con el space y el proyecto; Proyecto
  antes que Perfil (opcional), con para qué sirve; el ejemplo de config
  separa lo necesario de lo opcional; tabla de Configuración al día.
  - Se corrigen dos errores de la guía encontrados al revisarla: los
    ejemplos usaban `~/` en rutas del YAML, y draco no expande `~` (la
    carpeta quedaba literal); y `CONFLUENCE_SPACE_KEY` gana sobre el
    archivo, no «si la config no trae uno».
- **Descartado:** ofrecer `p crear un proyecto` también en la lista de
  Perfiles tras guardar uno (propuesto antes en la conversación). Con
  Proyectos como sección de Configuración, el siguiente paso ya se ve.

## 4. Archivos

- `internal/tui/settings.go` — secciones necesarias y opcionales; Proyectos;
  sin la oferta `p`.
- `internal/tui/connection.go` — space sin perfiles, aviso de
  `CONFLUENCE_SPACE_KEY`, el asistente sigue con el primer proyecto.
- `internal/tui/profileform.go` — sin `profileFirst`; descripción de la
  carpeta de plantillas.
- `internal/tui/home.go` — descripción de Configuración en el menú.
- `internal/tui/settings_test.go` — asistente con space y proyecto; Proyectos
  como sección; la conexión con perfiles no pide space.
- `internal/tui/qaconfig_test.go`, `themes_test.go` — navegación a la nueva
  posición de QA y Tema.
- `cmd/init.go`, `cmd/init_test.go` — space en vez de perfil; aviso del
  proyecto; ayuda.
- `docs/guia-usuario.md` — primeros pasos, conceptos, ejemplo de config,
  variables de entorno y Configuración.
- `docs/arquitectura/tui.md` — Configuración y asistente.
- `docs/bitacora/fase-6/051-config-gtt-primero.md`, índice.

## 5. Cómo verificar

```bash
make check
make test pkg=./internal/tui run='Settings|Wizard|Connection|Projects|Profile|QAConfig|Theme'
make test pkg=./cmd run=Init
```

A mano (pendiente del autor): con una config vacía en un `HOME` temporal,
`make run` → Configuración: conexión con space, verificación, primer
proyecto; el `config.yaml` resultante no tiene `profiles:` y `draco g
--dry-run` usa el space de la conexión.

## 6. Pendientes

- Actividad B (`feature/proyecto-backlog`): paso opcional «Backlog técnico»
  en el formulario de proyecto (`class_globs` y `modules` de backend y
  frontend).
- Encontrado al revisar los perfiles, fuera de esta actividad:
  - draco no expande `~` en las rutas de `config.yaml` (`docs.path`,
    `template`, rutas de proyectos); la ayuda de `draco template` sugiere
    `template: ~/plantillas/equipo`;
  - con un perfil Markdown, el `confluence_space_key` de un proyecto se usa
    como carpeta de destino en `draco g`: el space resuelto llega como
    `profile.Options.Scope`, que en Markdown reemplaza a `docs.path`
    (`cmd/generate.go`, `internal/tui/generate.go`).
