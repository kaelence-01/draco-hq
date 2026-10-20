# Modo visual (TUI)

`draco` sin argumentos, en una terminal interactiva, abre una interfaz dentro
de la terminal: menús, listas, formularios y vista previa. Es otra forma de
recorrer **los mismos flujos** que el modo comando; no tiene lógica propia de
negocio.

Estado: **en implementación** (roadmap, fase 4). Implementado: la base
(entrada, marco, pila de pantallas, pantalla principal). Se actualiza al
implementar cada actividad.

## 1. Alcance por fase

| Fase | Pantallas |
|---|---|
| 4 | Principal, Proyectos, Generar documento (selector de commits, vista previa, ubicación con búsqueda, confirmación) |
| 5 | Backlog (tabla filtrable), QA (progreso en vivo), Exportar |
| 6 | Barra de comandos con autocompletado, configuración (perfiles, credenciales) en formularios, temas, ayuda, aviso de versión nueva |

Mientras una pantalla no existe, su entrada del menú aparece atenuada con el
comando equivalente (`draco qa`, `draco f`, `draco backlog scan`): quien abre
la TUI sabe que la función existe y cómo usarla hoy.

## 2. Librerías

Charm v2, estable desde 2025, con rutas `charm.land/…`:

| Módulo | Para qué | Versión al planear |
|---|---|---|
| `charm.land/bubbletea/v2` | Bucle de eventos, modelo `Init`/`Update`/`View` | 2.1.0 |
| `charm.land/bubbles/v2` | `list` (con filtro), `viewport`, `spinner`, `textinput`, `help`, `key` | 2.2.1 |
| `charm.land/lipgloss/v2` | Estilos, bordes, composición de la pantalla | 2.0.6 |
| `charm.land/huh/v2` | Formularios de Proyectos (fase 4) y de configuración (fase 6) | 2.0.3 |

Las cuatro están aprobadas (CLAUDE.md). Se agregan cuando la primera actividad
las usa, no antes: Huh llega con Proyectos.

Descartado: **Bubble Tea v1** (`github.com/charmbracelet/…`). Empezar en v1
obliga a migrar después; v2 ya es la línea mantenida. **tview** y otras: no
aprobadas y con otro modelo de programación.

## 3. Entrada

```
draco                       terminal interactiva  →  TUI
draco                       sin TTY (pipe, CI)    →  ayuda, como hoy
draco <comando> …                                 →  modo comando, sin cambios
```

- `App.Run` sin argumentos abre la TUI solo si `App.tty` (entrada y salida
  son una terminal). La TUI se llama a través de una variable del paquete
  `cmd` (`runTUI`), que los tests reemplazan: un test nunca abre la TUI real.
- `Start` no cambia: si el binario no está instalado, la oferta de instalación
  va primero y, sin argumentos, termina ahí (doble clic en Windows).
- El aviso de versión nueva no se muestra en la TUI hasta la fase 6
  (`checksUpdates` ya lo omite sin argumentos).
- `NO_COLOR` se respeta (Lip Gloss v2 lo detecta).

## 4. Paquete `internal/tui`

```
internal/tui/
  tui.go        Run(ctx, Deps) error; Deps: las dependencias que pasa cmd
  app.go        modelo raíz: pila de pantallas, tamaño, cabecera, pie, teclas globales
  styles.go     estilos de Lip Gloss en un solo lugar (base para temas, fase 6)
  logo.go       DRACO en letras grandes
  home.go       pantalla principal
  projects.go   Proyectos (+ formulario)
  generate.go   Generar documento
```

`cmd` importa `tui`; `tui` nunca importa `cmd`. Lo que `cmd` reemplaza en los
tests (`loadConfig`, `openProfile`, `now`) llega a la TUI en `Deps`, con lo
que los tests de la TUI usan los mismos dobles:

```go
type Deps struct {
    LoadConfig  func() (*config.Config, error)                                 // config.LoadLocal (base)
    Version     string                                                         // base
    LoadFile    func() (*config.Config, error)                                 // para guardar (Proyectos)
    SaveConfig  func(*config.Config) error                                     // Proyectos
    OpenProfile func(*config.Config, profile.Options) (*profile.Bundle, error) // Generar
}
```

Cada campo llega con la primera pantalla que lo usa.

Igual que el modo comando, la TUI arma los adaptadores con `profile.Open` y
habla con los flujos; nunca con un adaptador.

## 5. Modelo

- **Pila de pantallas.** El modelo raíz guarda una pila; cada pantalla
  implementa una interfaz pequeña (`Update`, `View` con el tamaño disponible,
  `Keys` para el pie). Abrir una pantalla la apila; `esc` la cierra y vuelve
  a la anterior.
- **Teclas globales:** `ctrl+c` sale siempre; `q` sale desde la principal;
  `?` muestra la ayuda completa de teclas de la pantalla (`bubbles/help`).
- **Marco común:** cabecera con versión, perfil y proyecto activos; pie con
  las teclas de la pantalla. Las pantallas solo dibujan su cuerpo.
- **Pasos del flujo como comandos.** Cada paso de una `Session` corre en un
  `tea.Cmd` y vuelve como mensaje (`stepDone{step, err}`). La pantalla muestra
  un spinner mientras tanto y **no lee la `Session` hasta recibir el mensaje**:
  el paso escribe sus campos desde otra goroutine.
- **Cancelación.** Cada pantalla tiene su `context`; `esc` durante un paso lo
  cancela y vuelve atrás. Un mensaje que llega de un paso cancelado se
  descarta (cada paso lleva un número de secuencia).
- **Sin emojis en el marco.** Su ancho varía entre terminales y rompe bordes y
  columnas (sobre todo en Windows). Se usan símbolos de un ancho: `▸ ✓ ⚠ ✗ ●`.

## 6. Pantallas de la fase 4

### Principal

- DRACO en letras grandes con caracteres de bloque (ninguna terminal común
  muestra el logo como imagen). Si el ancho no alcanza, solo `draco`.
- Menú: Generar documento, Proyectos; atenuadas: Consolidado QA, Exportar
  página, Backlog técnico, Configuración (fases 5 y 6), cada una con su
  comando.
- Sin configuración: la principal lo dice y propone `draco init` (los
  formularios de configuración son de la fase 6).

### Proyectos

- Lista de proyectos (`bubbles/list`, con filtro) con backend, frontend y
  space; marca el proyecto por defecto.
- **Proyecto activo**: el que usa Generar en esta sesión. Se elige aquí, vive
  en memoria y no se guarda. **Por defecto** (`default_project`) sí se guarda.
- Crear, editar y borrar con un formulario de Huh (nombre, rutas y repos de
  backend y frontend, `vcs_host`, `vcs_org`, space) y confirmación al borrar.
- Guardar repite la regla del modo comando: se lee con `LoadFile` (solo el
  archivo), se modifica con las funciones puras de `config` (`SetProject`,
  `RemoveProject`, `SetDefaultProject`) y se guarda con `config.Save`. Nunca
  `LoadLocal`: un token de `ATLASSIAN_*` terminaría escrito en el disco.
  Editar conserva lo que el formulario no muestra (checklist, backlog).

### Generar documento

Los pasos de `deploy.Session`, en el orden en que la TUI los necesita, con las
mismas decisiones que pregunta `draco g`:

| # | Pantalla | Paso del flujo | Decisión del usuario |
|---|---|---|---|
| 1 | Tarea y proyecto (formulario) | — | Clave de la tarea, proyecto (por defecto, el activo) |
| 2 | Commits | `ChangeSource.RecentCommits` por lado | Marcar commits de backend y de frontend; `h` agrega un hash a mano |
| 3 | Progreso | `Resolve`, `profile.Open`, `FetchIssue`, `FindExisting` | Si ya existe: actualizar, crear otro o cancelar |
| 4 | Progreso | `ReadCommits` | Si falla un lado, aviso y seguir con el otro |
| 5 | Vista previa | `DraftCreate` / `DraftUpdate` | Pestañas: archivos por carpeta y documento (Markdown en `viewport`). Si `NotPreserved`, confirmar |
| 6 | Ubicación (solo al crear) | `Locations` | Elegir de la lista con búsqueda; aviso si `TeamWide` u `OwnerUnknown` |
| 7 | Confirmación y resultado | `Publish` | Confirmar; resultado con la URL y `LinkErr` si falló el enlace |

- **Los repos antes del `Resolve`.** `deploy.Resolve` exige los hashes, pero
  el selector de commits necesita antes las carpetas de los repos del
  proyecto. Se extrae del flujo una función pura que devuelve los repos de un
  proyecto, y `Resolve` la usa también; la TUI no repite esa lógica.
- **Orden de los commits.** El documento enlaza el primer hash de cada lado.
  Los marcados se pasan en el orden de la lista (el más reciente primero); el
  hash manual va al final.
- **Rama actual.** `RecentCommits` lee la rama en la que está el clon. Un
  commit de otra rama se agrega con `h`; la pantalla dice en qué rama está.
- La vista previa reemplaza a `--dry-run`: salir ahí no publica nada.
- Los centinelas se explican con texto de la pantalla: `ErrNoSpace` y
  `ErrNoCredentials` (al abrir el perfil), `ErrNoLocations` (propone un perfil
  Markdown).

## 7. Tests

- Sin terminal ni red: los tests llaman a `Update` con mensajes (teclas,
  tamaño, resultados de pasos) y revisan el estado y el texto de `View` con
  un tamaño fijo y sin color.
- Los pasos del flujo usan los dobles del modo comando: config en memoria,
  tracker y git simulados, adaptador Markdown real en una carpeta temporal.
- Un test recorre Generar de punta a punta y comprueba el archivo publicado.
- **Nunca ejecutar un binario recién compilado** (instalador); `make run`
  usa `go run`.

## 8. Terminales

Windows Terminal, PowerShell, Linux y macOS. En la consola clásica de Windows
(`cmd`) se ve más limitado: sin colores de 24 bits y con los caracteres de
bloque según la fuente. La prueba manual de cada actividad incluye Windows
Terminal y una terminal de Linux o WSL.

## 9. Decisiones

| Decisión | Motivo |
|---|---|
| Charm v2 (`charm.land/…`) | Línea estable y mantenida; empezar en v1 obligaría a migrar |
| `cmd` llama a la TUI por una variable; `tui` recibe `Deps` | Sin ciclo de imports; los tests reemplazan la TUI y la TUI usa los mismos dobles |
| Pantallas en una pila, `esc` vuelve | Modelo simple de navegación; cada pantalla se prueba sola |
| La `Session` no se lee mientras un paso corre | El paso escribe desde otra goroutine |
| Proyecto activo en memoria; «por defecto» en la config | Cambiar de proyecto para una tarea no debe cambiar la config |
| Entradas futuras del menú atenuadas con su comando | Muestra lo que existe y cómo usarlo hoy |
| Función pura de repos del proyecto en `deploy` | El selector de commits la necesita antes de `Resolve`; la TUI no reimplementa el flujo |
| Sin emojis en el marco | Ancho variable entre terminales |
