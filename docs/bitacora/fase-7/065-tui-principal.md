# 065 — Principal de la fase 7: menú con descripciones, panel con pasos y contexto

**Fecha:** 2026-10-28
**Rama:** feature/tui-principal → develop
**Fase del roadmap:** 7

## 1. Solicitud

El autor pidió seguir con la actividad 3 de la fase 7, la pantalla principal,
tal como la aprobó en el prototipo: descripciones, atajos `1`–`6`, panel con
pasos, **Proyecto activo** y **Tu configuración**.

## 2. Motivación

| Problema | Daño real |
|---|---|
| La principal centraba todo y dejaba unas 60 columnas vacías por lado a 160×40; el panel tenía ancho fijo | En una terminal ancha casi nada de la pantalla decía algo |
| El menú era solo nombres: para saber qué hacía cada función había que moverse una por una | Quien llega por primera vez no sabe qué elegir |
| Para saber si faltaba algo de la config (conexión, `qa_report`) había que entrar a Configuración | Un `qa_report` vacío produce un consolidado con el encabezado en blanco y nadie lo ve hasta publicarlo |

## 3. Diseño técnico

- **Disposición** (`home.go`): logo, eslogan y menú arriba a la izquierda; a la
  derecha, el panel de la opción elegida (qué hace, sus pasos, el comando).
  Desde 96 columnas, una línea tenue de descripción bajo cada opción y el
  número de la opción al borde derecho de su columna (como el prototipo).
  Debajo de 96 columnas: sin panel ni descripciones, y bajo el menú el texto y
  el comando de la opción elegida.
- **Atajos `1`–`6`:** abren la función directo (`numberKey` se arma con las
  opciones que haya, hasta 9). No son letras porque `q` sale. Con la barra de
  comandos abierta el número se escribe, no abre nada (`captures`).
- **Cede lo accesorio:** el menú prueba tres formas (con espacios entre
  opciones, con descripciones, solo opciones) y toma la primera que cabe en el
  alto. La versión angosta retira el eslogan y luego las letras grandes antes
  que el texto o el comando de la opción. Los avisos (config, versión nueva) y
  el mensaje de una función sin pantalla ocupan todo el ancho: ajustados a la
  línea siguiente, no recortados.
- **Proyecto activo y Tu configuración** (`homecontext.go`): dos cajas bajo el
  panel desde 120 columnas, solo si caben en el alto (no se dibujan cortadas).
  Las reglas salen de `config`, no de la pantalla (norma 1):
  - `Config.QAState` (sin configurar, incompleta, sin cuenta, lista) y
    `HasQAReport`, nuevas y probadas. **`settings.sections()` pasó a usar
    `QAState`**, así que Configuración y la principal comparten la regla; los
    textos de Configuración no cambian.
  - El resto ya eran funciones de `config` (`HasAtlassianCredentials`,
    `Profile`, `len(Projects)`).
  - Una opcional sin configurar (QA) se marca `·`, no `⚠`: no es un problema.
- **`hang` en `layout.go`:** el texto largo de una caja (una ruta, un detalle)
  se ajusta bajo sí mismo y no bajo su etiqueta.
- **Pie de la principal:** compacto con `↑↓ elegir`, `enter`, `1-6` y `/`; `q
  salir`, `↑/k` y `↓/j` pasan a `moreKeys` (en el prototipo `q` también es
  secundaria). Con `↑/k subir • ↓/j bajar` sueltos, `? más` no cabía en 80
  columnas.
- **Decisión que merece mirada del autor:** que falte `qa_report` se marca `⚠`
  en la principal (como el prototipo), pero Configuración sigue dando QA por
  lista con `qa_jira` y `qa_email`: cambiarlo allí cambiaba una pantalla y
  tests fuera de esta actividad.
- **Descartado:** columnas proporcionales para la principal. Las descripciones
  son de una línea (~55 columnas) y un menú más angosto las partiría; las
  columnas proporcionales son de las pantallas de lista (§11).
- **Prototipo, lo que no se copió:** las etiquetas `api` y `web` de la caja del
  proyecto pasaron a `backend` y `frontend`, como el resto del modo visual.

## 4. Archivos

- `internal/config/status.go` — nuevo: `QAState` y `HasQAReport`.
- `internal/config/status_test.go` — nuevo: una tabla por cada función.
- `internal/tui/home.go` — nueva disposición, atajos numéricos, descripciones, panel con pasos, versión angosta que cede.
- `internal/tui/homecontext.go` — nuevo: Proyecto activo, Tu configuración y `configChecks`.
- `internal/tui/layout.go` — `hang`.
- `internal/tui/settings.go` — la sección QA usa `QAState`.
- `internal/tui/home_test.go` — nuevo: 160×40, 80×24, avisos que quitan sitio, poca altura, cajas que no caben, sin proyecto, atajos, números en la barra y `configChecks`.
- `internal/tui/app_test.go`, `export_test.go`, `qa_test.go` — la principal ya no se reconoce por la ausencia de `›` en toda la pantalla (el panel lleva pasos con `›`) sino en la cabecera; el pie ya no lleva `q salir`; el test del menú angosto sigue el diseño nuevo.
- `docs/arquitectura/tui.md`, `docs/guia-usuario.md` — la principal, los atajos y las cajas.
- `docs/roadmap.md` — actividad 3 con ✅.
- `docs/bitacora/fase-7/065-tui-principal.md`, índice.

## 5. Cómo verificar

```bash
go vet ./internal/tui ./internal/config && go test ./internal/tui ./internal/config   # corridos: ok
make check                                                                            # lo corre el autor
make run                                                                              # 80×24 y una terminal ancha: menú, panel, cajas, teclas 1 a 6
```

## 6. Pendientes

- `make check` completo y la mirada en una terminal real (Windows Terminal) a
  80×24 y a más de 120 columnas, a cargo del autor.
- ¿Configuración debe marcar también la falta de `qa_report`? (ver Diseño técnico).
- El resto de las pantallas clasifica sus teclas al rediseñarse (actividades 4 a 7).
