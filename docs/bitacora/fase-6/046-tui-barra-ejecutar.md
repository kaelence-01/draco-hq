# 046 — Barra de comandos (2): cada comando abre su pantalla

**Fecha:** 2026-10-21
**Rama:** feature/tui-barra-ejecutar → develop
**Fase del roadmap:** 6

## 1. Solicitud

Seguir con la actividad 5b del plan de la fase 6 (bitácora 041): que la
barra abra la pantalla de cada comando con los datos de la línea.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Desde la 045, una línea válida solo decía «elígelo en el menú» | Quien conoce los comandos tenía que volver a escribir en un formulario lo que ya había escrito en la barra |
| Los ejemplos del panel mostraban un comando que la barra no ejecutaba | El modo visual enseñaba el modo comando, pero no lo aceptaba |

## 3. Diseño técnico

- **`openFromBar`** (`barrun.go`) abre la pantalla del comando en el primer
  paso que el usuario revisa:
  - un formulario con lo necesario se envía, y la pila queda como si el
    usuario lo hubiera llenado, así que `esc` vuelve al formulario con los
    valores;
  - un error del paso queda sobre el formulario;
  - sin un dato obligatorio (`g` o `f` sin `-i`), se abre el formulario
    con lo que se dio.

  Usa los pasos de cada pantalla (`next()`); no repite lógica.
- **Generar:** en Commits, `preset` guarda los hashes de `-b` y `-f`, y cada
  lado los marca al cargar (`side.mark`). Un hash de la lista se marca por
  prefijo, en los dos sentidos (`e4f5` marca `e4f5a6b`), y uno que no está
  se agrega como escrito a mano. Si el proyecto no tiene ese lado, se
  ignora con un aviso.
- **QA:** la validación del sprint pasa a `validSprint`, que usan el
  formulario y la barra. Sin `-s`, como el comando: Kanban.
- **Backlog:** `backlog scan` analiza con las opciones de la línea, y
  `backlog` solo abre el formulario, porque el comando sin `scan` muestra
  su ayuda. **Cambio frente al diseño**, que decía «con el formulario
  lleno»: se envía como los demás, porque así se comporta el comando.
- **Proyectos y Configuración:**
  - `project` y `project add`;
  - `init` (Conexión);
  - `profile` y `profile add`.

  Configuración se abre con el asistente ya iniciado (`started`), para no
  abrir Conexión dos veces.
- **Flags sin pantalla** (`barFlags` y `barUnsupported`): se explica el
  primero en orden alfabético (el mapa de Go no tiene orden), sin abrir
  nada. `--dry-run` tiene su propio mensaje: la vista previa no publica.
  `update` y `template` se corren fuera del modo visual.
- **Bug encontrado por los tests:** con un `-p` que no existe, la lista
  de proyectos de Huh lo reemplazaba en silencio por el primero. Generar
  seguía con otro proyecto, y Backlog habría analizado la carpeta actual.
  - La barra rechaza un `-p` desconocido con `cfg.GetProject`, como el
    modo comando.
  - Generar y Backlog leen los valores de la línea antes de reconstruir su
    formulario. Así `backlog.Targets` rechaza un `-r web` en vez de
    recibir el `""` de la lista.
- **Norma 8** del modo visual y `CLAUDE.md`: una función con comando se
  abre desde la barra, y su ejemplo funciona ahí.

## 4. Archivos

- `internal/tui/barrun.go` — nuevo: `openFromBar`, `barUnsupported`,
  `barFlags` y una función por pantalla.
- `internal/tui/barrun_test.go` — nuevo. Cubre:
  - Generar hasta Commits con los hashes marcados y agregados, y `esc` al
    formulario;
  - Generar sin tarea, con un lado que falta y con un proyecto que no
    existe;
  - QA con sprint y con un sprint inválido;
  - Exportar;
  - Backlog con opciones, con un repo inválido, con un proyecto que no
    existe y sin `scan`;
  - Proyectos y Configuración;
  - los flags y comandos sin pantalla;
  - los ejemplos del panel.
- `internal/tui/cmdbar.go` — `enter` llama a `openFromBar`.
- `internal/tui/cmdbar_test.go` — sin el caso «todavía no abre».
- `internal/tui/commits.go` — `preset`, `side.given` y `side.mark`.
- `internal/tui/qa.go` — `validSprint`.
- `CLAUDE.md`, `docs/arquitectura/tui.md` (§6 y norma 8),
  `docs/roadmap.md`, `README.md` — estado y diseño ajustado.
- `docs/bitacora/fase-6/046-tui-barra-ejecutar.md`, índice.

## 5. Cómo verificar

```bash
make test pkg=./internal/tui run='BarOpens|BarGenerate|BarExplains|PanelExamples' v=1
make check
make run            # /g -i OPS-7 -b <hash>, /qa, /f -i OPS-7, /backlog scan, /init
```

## 6. Pendientes

- Prueba manual del autor: `/g` con un hash real de su repo hasta
  publicar, y `/qa` contra su Jira.
- Actividad 6: temas.
