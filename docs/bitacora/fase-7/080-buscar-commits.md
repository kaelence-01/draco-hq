# 080 — Buscar commits: búsqueda, solo los míos, todas las ramas y sugeridos

**Fecha:** 2026-11-23
**Rama:** feature/buscar-commits → develop
**Fase del roadmap:** 7 (mejoras sobre la fase cerrada)

## 1. Solicitud

Al generar un documento, el autor no encontraba su commit en Commits: la
lista traía muchos de otras personas y el suyo había quedado más abajo que los
que se cargaban. Pidió un buscador dinámico y preciso (por hash, autor o
fecha), que sirviera también para agregar un hash a mano, una forma de ver
solo sus commits, y que se pudieran cargar más. Lo dividió en partes: la 079
fue pegar; esta reúne el buscador y la mejora E propuesta en la conversación,
los commits sugeridos por la clave de la tarea.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Solo se listaban los 30 últimos commits de la rama del clon | En un repo con varias personas, el commit de una tarea sale de la lista en pocos días y no había forma de pedir más |
| No había búsqueda ni filtro por autor | Había que recorrer a ojo una lista de commits ajenos |
| Un commit de otra rama solo entraba escribiendo su hash con `h` | Había que salir a la terminal, buscarlo con `git log` y copiar el hash |

## 3. Diseño técnico

- **Puerto `ChangeSource`.** `RecentCommits(n)` pasa a ser
  `Commits(CommitQuery{Limit, Skip, AllBranches, Grep})`, y se suma
  `Identity` (el `user.email` del clon; vacío si no tiene, sin error).
  `Commit` gana `Email`. Solo la TUI lista commits (el modo comando recibe los
  hashes con `-b`/`-f`), así que el cambio toca el adaptador de git y tres
  dobles de test.
- **Adaptador de git.**
  - **Todas las ramas:** `--branches --remotes` y no `--all`, que traería
    también los stash.
  - **Grep:** va con `--regexp-ignore-case --fixed-strings` y dentro de su
    propio argumento `--grep=<texto>`. El texto nunca se lee como opción ni
    como expresión regular; el test lo prueba con `--all OPS-7.*`.
  - **Identity:** `git config user.email`; el código de salida 1 (sin
    valor) es «sin identidad», no un error.
- **Funciones puras en `internal/deploy`** (`commits.go`), para que la
  pantalla no lleve lógica:
  - `CommitMatches`: cada palabra de la búsqueda tiene que estar al
    comienzo del hash o en cualquier parte del mensaje, el autor, el correo o
    la fecha `2006-01-02`, sin distinguir mayúsculas. Que el hash coincida
    solo por el comienzo evita falsos positivos (`b2c3` no encuentra
    `a1b2c3d`), y la fecha como texto permite buscar un mes (`2026-11`).
  - `Mine`: compara el correo, no el nombre, porque el nombre cambia entre
    equipos.
  - `MentionsIssue`: la clave como palabra completa, así que `OPS-7` no
    coincide con `OPS-70` ni con `XOPS-7`.
  - `LooksLikeHash`: de 7 a 40 caracteres hexadecimales.
- **Pantalla Commits.**
  - **Al abrir**, cada lado lee la rama, la identidad, una página de 100
    commits de la rama y hasta 50 **sugeridos**. Los sugeridos se buscan en
    todas las ramas y en toda la historia con `Grep` = la clave, y se filtran
    con `MentionsIssue` sobre el asunto: el puerto no trae el cuerpo del
    mensaje, y premarcar un `OPS-70` sería peor que perder una mención que
    solo está en el cuerpo.
  - **Los sugeridos van arriba con `★`.** Por decisión del autor, llegan
    marcados solo los propios (`Mine`). Si la pantalla se abre desde la barra
    con `-b`/`-f`, no se marca nada más que esos hashes (`fromBar`).
  - **`/` usa el `filter` de las listas** y filtra mientras se escribe.
    `enter` con un hash lo marca si ya se leyó (por prefijo, en cualquier
    página o rama) o lo agrega a mano. Por decisión del autor, eso reemplaza
    a la tecla `h`: hay una sola forma de buscar y de agregar. Pegar una línea
    de `git log --oneline` en la búsqueda deja solo el hash.
  - **`m`** deja los commits del `user.email` del clon. Sin identidad no
    filtra y explica cómo configurarla.
  - **`a`** alterna entre la rama y todas las ramas, y vuelve a la primera
    página; lo marcado se conserva.
  - **`↓` en la última fila** lee 100 más (`Skip`). También funciona sin
    coincidencias, así que la búsqueda sigue en lo que se va leyendo. Si
    llega una página vieja después de cambiar con `a`, se descarta: se
    compara su `all` y su `skip`.
  - **Lo marcado se guarda por hash**, no por posición en la lista: un
    commit marcado puede dejar de verse (por la búsqueda, `m` o `a`) y sigue
    marcado. Por eso el orden para el documento sale de la fecha, del más
    reciente al más antiguo (con la misma fecha, el orden en que se leyó), y
    no de la lista. El primero sigue siendo el que enlaza el documento.
  - **La línea bajo la lista** muestra siempre `/ buscar · m míos · a todas
    las ramas · N leídos` con su estado (o `N de M`). `/`, `m` y `a` van en
    la ayuda completa porque en el pie de 80 columnas no cabían junto a
    `? más`.
- **`paste` devuelve un `tea.Cmd`** (la interfaz es de la 079). Pegar en
  la búsqueda cambia el commit que queda bajo el cursor, y sus archivos tienen
  que leerse en un paso; sin esto, el detalle quedaba en «Leyendo…» hasta la
  siguiente tecla.
- **Descartado:**
  - Buscar en git por autor o fecha (`--author`, `--since`). La búsqueda
    local con una sola función cubre todos los campos a la vez, y con
    páginas de 100 es inmediata.
  - Sugerir por el nombre de la rama (`feature/OPS-7-…`): `--branches=*OPS-7*`
    trae también toda la historia de la que salió la rama.
  - Una fila «cargar más» con su propia tecla: `↓` al final hace lo mismo
    sin tecla nueva.
- **Fuera de draco, en esta rama:** el autor cambió la pregunta de
  `scripts/merge.sh` y `scripts/promote.sh` de `[s/N]` a `[y/N]`. La
  respuesta sigue aceptando `s` o `y`.

## 4. Archivos

- `internal/provider/provider.go` — `CommitQuery`, `Commit.Email`; `ChangeSource` con `Commits` e `Identity` en lugar de `RecentCommits`.
- `internal/provider/git/git.go` — `Commits` con `logArgs` (páginas, todas las ramas, `--grep` literal); `Identity`; el log trae `%ae`.
- `internal/provider/git/git_test.go` — `logArgs`; repo temporal con dos ramas: páginas, todas las ramas, `Grep` sin mayúsculas y literal, identidad con y sin `user.email` (config global aislada).
- `internal/deploy/commits.go`, `commits_test.go` — nuevos: `CommitMatches`, `Mine`, `MentionsIssue`, `LooksLikeHash`.
- `internal/tui/commits.go` — búsqueda, `m`, `a`, páginas, sugeridos con `★` y premarcado, lo marcado por hash y ordenado por fecha; sin `h`.
- `internal/tui/app.go` y las pantallas con filtro (`locations.go`, `profiles.go`, `projects.go`, `findings.go`, `export.go`, `home.go`) — `paste` devuelve `tea.Cmd`.
- `internal/tui/commits_test.go` — nuevo: búsqueda por autor, palabras y fecha; pegar una línea de `git log`; `enter` con un hash; `m` con y sin identidad; sugeridos; la barra no premarca; leer más al final y desde una búsqueda sin coincidencias; todas las ramas.
- `internal/tui/generate_test.go` — `fakeGit` con `Commits`, `Identity`, commits de otras ramas y commits viejos; el hash que no existe pasa a `badc0de` (hexadecimal, para agregarlo con `/`).
- `internal/tui/generate_layout_test.go`, `paste_test.go` — `/` en lugar de `h`; la línea bajo la lista; la ayuda completa con `/`, `m` y `a`.
- `cmd/generate_test.go`, `internal/deploy/deploy_test.go`, `internal/provider/markdown/deploy_test.go` — los dobles con el puerto nuevo.
- `docs/arquitectura/proveedores.md` — el puerto nuevo y el porqué de `--branches --remotes` y `--fixed-strings`.
- `docs/arquitectura/tui.md` — Commits: pasos, orden de lo marcado, páginas y ramas, sugeridos, búsqueda, teclas; `paste` con `tea.Cmd`.
- `docs/guia-usuario.md` — Commits: sugeridos, búsqueda, hash en la búsqueda, `m`, `a`, leer más; pegar en la búsqueda.
- `scripts/merge.sh`, `scripts/promote.sh` — `[y/N]` (cambio del autor).
- `docs/roadmap.md` — 080 ✅.
- `docs/bitacora/fase-7/080-buscar-commits.md`, índice, nota en la 079.

## 5. Cómo verificar

```bash
gofmt -l ./internal ./cmd                 # corrido: limpio
go vet ./...                              # corrido: limpio
go test ./internal/...                    # corrido: ok
make check                                # lo corre el autor (incluye ./cmd)
```

En una terminal real (lo corre el autor), `go run .`, Generar documento con
una tarea que tenga commits:

- Los commits que mencionan la tarea aparecen arriba con `★`, y los propios
  ya marcados.
- `/` y un nombre, una palabra o una fecha (`2026-11`); `esc` quita el filtro.
- `/`, pegar un hash de otra rama y `enter`: queda marcado o «a mano», con
  sus archivos en el detalle.
- `m` (con y sin `user.email` en el repo), `a`, y `↓` hasta el final de la
  lista para leer más.

## 6. Pendientes

- **Probar contra repos reales grandes.** Los tests usan un repo temporal
  pequeño y un doble.
- **Menciones solo en el cuerpo del mensaje.** No se sugieren; haría falta
  que el puerto trajera el cuerpo.
- **Sugerir por el nombre de la rama** (`feature/OPS-7-…`): descartado por
  ahora (§3).
