# 031 — Generar en la TUI (1): tarea, commits y progreso

**Fecha:** 2026-10-20
**Rama:** feature/tui-generar → develop
**Fase del roadmap:** 4

## 1. Solicitud

Seguir con la actividad 4a de la fase 4, propuesta al cerrar la 030: de la
tarea a los archivos leídos de los commits, con la función pura de los repos
del proyecto en `deploy`, el selector de commits, los pasos con progreso y
`esc` para cancelar el paso en curso. El autor la aprobó («procede»).

## 2. Motivación

| Problema | Daño real |
|---|---|
| Generar, la función principal de draco, no estaba en el modo visual | Para documentar un despliegue había que copiar hashes del `git log` a mano en `draco g` |
| `deploy.Resolve` exige los hashes, pero el selector de commits necesita antes las carpetas de los repos | Sin una función en el flujo, la TUI habría repetido cómo se arman los lados de un proyecto |
| `ChangeSource` no decía en qué rama está el clon | Un commit de otra rama simplemente no aparecía en la lista, sin explicación |
| El modelo de cancelación existía solo en el diseño | Un paso lento contra Jira habría dejado la pantalla bloqueada o un resultado viejo pisando una pantalla nueva |

## 3. Diseño técnico

**Flujo (`internal/deploy`):** `ProjectSides(cfg, proyecto)` devuelve los
lados del proyecto (repo y carpeta de backend y frontend) y su space, sin
hashes. `Sides.Deployable` dice qué lados tienen un clon. Sin ruta de
frontend, el backend siempre la tiene: su clon o el directorio actual, como
`draco g -b` sin proyecto. `Resolve` arma su `Target` con la misma función,
y sus tests no cambiaron.

**Puerto:** `ChangeSource.CurrentBranch` (`git rev-parse --abbrev-ref HEAD`)
es la rama que lee `RecentCommits`. Lo implementan el adaptador git, con un
test en un repo temporal, y los tres dobles que ya existían.

**Pantallas** (una por paso que el usuario revisa, norma 3 de `tui.md` §10):

1. **Generar documento** (`generate.go`): formulario de Huh con la tarea y el
   proyecto (un `Select`; por defecto, el activo). Al enviarlo se abre el
   perfil con el space del proyecto. Así, `ErrNoSpace` o `ErrNoCredentials`
   se explican en el mismo formulario antes de elegir commits, y el selector
   usa el `ChangeSource` del perfil. Es un cambio frente al diseño, que abría
   el perfil en el paso 3.
2. **Commits de OPS-7** (`commits.go`): una pestaña por lado (`tab`) para
   caber en 80 columnas, con el repo, la carpeta y la rama, y los 30 commits
   más recientes con hash, asunto, autor y fecha. `espacio` marca, `h`
   agrega un hash a mano (otra rama) y `enter` sigue, con al menos un
   commit. Los marcados van en el orden de la lista y los manuales al final.
   Las pestañas cuentan lo marcado. La lista se desplaza con el cursor.
3. **Preparar documento** (`progress.go`): `Resolve` (puro), luego
   `FetchIssue`, `FindExisting` y `ReadCommits`, cada uno en un `tea.Cmd`
   con spinner. Si ya existe un documento, se elige `1` actualizar, `2`
   crear otro o `3` cancelar, igual que en `draco g`; la elección queda
   guardada para la 4b. Un lado que falla se muestra y se sigue con el otro.
   Hasta la 4b, la pantalla termina con los archivos por carpeta y el
   comando equivalente con `--dry-run`.

**Cancelación:** las pantallas con pasos crean su `context` y lo cancelan en
`close()`. Cada mensaje de un paso lleva su pantalla de origen (`owner`), y
la pantalla lo descarta si no es suyo o si su `context` ya terminó. Así, un
resultado tardío no pisa la pantalla que se abrió después. El diseño hablaba
de un número de secuencia; el puntero a la pantalla cumple lo mismo sin
contador.

**Marco:** `init()` corre también al volver a quedar arriba. El formulario
de Generar, ya enviado, se rehace con los mismos valores, y el selector no
vuelve a leer los commits porque los pide una sola vez.

**Tests** (`generate_test.go`): dobles del tracker y de git, y el adaptador
Markdown real en una carpeta temporal, con `HOME` y el directorio de trabajo
también temporales, para que no entre una plantilla del desarrollador.
Cubren el formulario y su error, el space con que se abre el perfil, las
pestañas y la rama, el orden de los hashes que llegan a `ChangedFiles`, un
lado que falla, el error de la tarea, el documento existente con sus tres
opciones, `esc` en un paso en curso (el tracker espera al `context` y el test
comprueba que se cancela), el descarte de un resultado viejo y la vuelta al
formulario.

## 4. Archivos

- `internal/deploy/deploy.go`: `Sides`, `ProjectSides`, `Deployable`;
  `Resolve` los usa.
- `internal/deploy/deploy_test.go`: `TestProjectSides`; `CurrentBranch` en
  el doble.
- `internal/provider/provider.go`: `ChangeSource.CurrentBranch`.
- `internal/provider/git/git.go`, `git_test.go`: `CurrentBranch` y su test.
- `internal/provider/markdown/deploy_test.go`, `cmd/generate_test.go`:
  `CurrentBranch` en los dobles.
- `internal/tui/generate.go`, `commits.go`, `progress.go`: nuevas, las tres
  pantallas.
- `internal/tui/generate_test.go`: nuevo.
- `internal/tui/app.go`: `init()` al volver a una pantalla.
- `internal/tui/tui.go`: `Deps.OpenProfile`. `cmd/visual.go`: la pasa.
- `internal/tui/home.go`: Generar encendido.
- `docs/arquitectura/tui.md`: estado, paquete, `init`, pasos de Generar,
  tests y decisiones.
- `docs/arquitectura/proveedores.md`: `CurrentBranch` en `ChangeSource`.
- `docs/roadmap.md`: 4a ✅. `README.md`: el estado.
- `docs/bitacora/fase-4/031-tui-generar.md`, índice.

## 5. Cómo verificar

```bash
make check
go test -race -count=3 ./internal/tui ./internal/deploy ./internal/provider/... ./cmd
make run        # en un repo con un proyecto configurado: Generar, tarea, commits
```

Las pantallas se revisaron además como texto (`View` sin color a 110×34) en
un test temporal que no quedó en el repo: el formulario, el selector con el
hash a mano y el resultado con los archivos por carpeta.

## 6. Pendientes

- Prueba manual del autor contra su Jira y sus repos, en Windows Terminal y
  en Linux/WSL: tiempos del spinner, la rama y el desplazamiento de la lista
  con muchos commits. Sin probar todavía contra una instancia real.
- La 4b sigue desde la pantalla de progreso: vista previa, `NotPreserved`,
  ubicación con búsqueda, confirmación y resultado con `LinkErr`.
- Los tests de la TUI tardan unos 6 s con `-race`: cada tecla en un campo de
  Huh espera los 20 ms del parpadeo descartado.
