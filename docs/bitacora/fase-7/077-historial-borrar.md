# 077 — Borrar un documento desde el historial

**Fecha:** 2026-11-09
**Rama:** feature/historial-borrar → develop
**Fase del roadmap:** 7 (mejoras sobre la fase cerrada)

## 1. Solicitud

Es la segunda parte del pedido de la 076. Si un documento quedó mal, el
autor quiere ir directo a él desde el historial y borrarlo «porque no estaba
bien creado». En la 076 el autor decidió dejar el borrado para una actividad
aparte.

Antes de empezar, el autor aprobó dos recomendaciones:

- **sin `--yes`** en el comando: borrar siempre pide escribir la tarea;
- **solo se borra lo que está en el historial**, es decir, lo que el usuario
  publicó con draco, nunca cualquier página de una tarea.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Un documento mal creado había que borrarlo a mano en Confluence | Había que buscar la página, y el enlace desde la tarea quedaba en Jira |
| Un enlace que apunta a una página borrada sigue contando para QA | `HasDocumentLink` lo ve y el consolidado da la tarea por documentada sin documento: una evidencia falsa |
| El historial de la 076 conservaría un documento que ya no existe | Abrirlo o actualizarlo fallaría, y la lista mentiría |

## 3. Diseño técnico

- **Puertos:**
  - `DocStore.Delete(ref)` y `Tracker.UnlinkDocument(issue, ref)`.
  - `provider.ErrNotFound` marca un documento que ya no existe. Los 404 de
    Atlassian (`apiError`) ahora lo envuelven, con el mismo texto de antes,
    así que ningún mensaje cambia.
- **Confluence:** `DELETE /wiki/api/v2/pages/{id}`, sin `purge`. La página
  va a la papelera del space y se puede restaurar. Es lo menos destructivo
  que ofrece la API.
- **Jira:** `DELETE /rest/api/3/issue/{key}/remotelink?globalId=…`.
  - Usa el mismo `globalId` con el que `LinkDocument` crea el enlace
    (`linkGlobalID`). Así solo se quita el enlace de draco; otros enlaces
    de la tarea quedan.
  - Jira responde 404 si no hay enlace: no es un error, no hay nada que
    quitar.
  - La clave va con `url.PathEscape` y el `globalId` con `QueryEscape`.
- **Markdown:** `os.Remove`, solo de un `.md` y dentro de la carpeta (el
  `path()` que ya existía). Una carpeta no se borra nunca; tampoco algo
  fuera de ella aunque el historial, que es texto editable, lo pida.
- **Flujo `deploy.Remove`:**
  - **Primero el documento, después el enlace.** Si el documento no se pudo
    borrar, el enlace queda: una página que sigue existiendo no puede quedar
    sin enlace (QA la daría por indocumentada).
  - Si el enlace no se pudo quitar, viaja en `Removal.LinkErr` y la UI dice
    que hay que quitarlo a mano. Es un error descartado cuyo efecto
    aparecería semanas después, en QA.
  - Un documento que ya no existía es `Removal.Missing`: se quita el enlace
    igual.
- **Historial:**
  - `history.Remove` saca la entrada.
  - `Entry.Ref` arma el `DocRef`.
  - `OfIssue` busca los documentos de una tarea, sin importar mayúsculas.
  - **`Entry.CheckDest`:** si el perfil del documento ahora publica en otro
    destino, no se borra nada. Sin esto, el id de un archivo Markdown se le
    pediría a Confluence: un 404 se tomaría por «ya no existía» y la entrada
    saldría del historial con el archivo todavía en disco.
- **Perfil:**
  - Se abre el del documento (`Entry.Profile`) con `AnyScope`: para borrar
    por id no hace falta el space, y un proyecto ya borrado de la config no
    lo impide.
  - Una salvedad: en Markdown la carpeta es la del perfil de hoy. Si cambió,
    el archivo no se encuentra y se toma como «ya no existía».
- **Comando `draco history rm -i OPS-7`** (también `remove`):
  - Si la tarea tiene varios documentos en el historial (por ejemplo, el
    usuario eligió «Crear nuevo»), los lista y pregunta cuál.
  - Dice qué va a pasar y pide escribir la tarea. No hay flag para
    saltárselo (decisión del autor).
  - Una tarea sin documentos en el historial es un error que explica que
    solo se borra lo publicado con draco.
  - `draco history` sin subcomando sigue listando. El subcomando se resuelve
    antes de leer los flags, para que `-i` no choque con los de la lista.
- **TUI:**
  - **`x`** en Recientes y en Historial abre `removeDoc`, una pantalla de la
    pila. Está en la ayuda completa y en el detalle («x lo borra»), para no
    alargar el pie de 80 columnas.
  - La confirmación es un formulario de huh con validación: escribir otra
    cosa muestra «no coincide» y `esc` vuelve sin borrar.
  - El borrado corre en un `tea.Cmd` (perfil, `CheckDest`, `deploy.Remove`,
    `state.forget`). `forget`, como `record`, solo lee `deps` desde la
    goroutine.
  - Mientras borra, `esc` no sale, como al publicar.
  - El resultado separa documento, enlace e historial. La línea sobre el pie
    va con `⚠` si el enlace quedó.
  - **Barra:** `/history rm -i OPS-7` abre Historial en ese documento y su
    confirmación. Con varios documentos de la tarea, abre solo Historial,
    para elegir.
- **Descartado:** borrar sin pasar por el historial (`draco rm -i` sobre
  cualquier página de la tarea). Podría borrar el documento de un compañero
  (decisión del autor).

## 4. Archivos

- `internal/provider/provider.go` — `Tracker.UnlinkDocument`, `DocStore.Delete`, `ErrNotFound`.
- `internal/provider/atlassian/client.go` — `Client.Delete`; el 404 envuelve `ErrNotFound`.
- `internal/provider/atlassian/confluence.go` — `DeletePage` (a la papelera).
- `internal/provider/atlassian/jira.go` — `linkGlobalID`, `DeleteJiraRemoteLink`.
- `internal/provider/atlassian/ports.go` — `Tracker.UnlinkDocument` (sin enlace no es error), `DocStore.Delete`.
- `internal/provider/atlassian/delete_test.go` — nuevo: papelera sin `purge`, 404 como `ErrNotFound`, 403 como error; enlace por su `globalId`, sin enlace, sin permiso.
- `internal/provider/markdown/markdown.go` — `Delete`: solo un `.md` dentro de la carpeta.
- `internal/provider/markdown/delete_test.go` — nuevo: borra el archivo y no la carpeta; ya borrado; carpeta, ruta sin `.md` y ruta fuera de la carpeta rechazadas.
- `internal/deploy/deploy.go` — `Removal` y `Remove`.
- `internal/deploy/deploy_test.go` — `Remove`: borra y quita el enlace, ya no existía, el enlace queda si el documento queda, `LinkErr`.
- `internal/history/history.go` — `Ref`, `CheckDest`, `Remove`, `OfIssue`; `save` compartido con `Record`.
- `internal/history/history_test.go` — `Remove`, `Ref` y `CheckDest`.
- `cmd/history.go` — `draco history remove/rm -i`, su ayuda.
- `cmd/history_test.go` — borrar, confirmar escribiendo la tarea, solo lo del historial, elegir entre varios, avisos (ya no existía, enlace que queda), otro destino.
- `internal/tui/historyremove.go` — nuevo: pantalla Borrar.
- `internal/tui/historyremove_test.go` — nuevo: borrar desde la principal y desde Historial (80×24), la tarea mal escrita, enlace que queda, documento que no se pudo borrar, otro destino, barra.
- `internal/tui/history.go` — `x` y «x lo borra».
- `internal/tui/tui.go` — `state.forget`.
- `internal/tui/barrun.go` — `/history rm -i` (`openRemove`).
- `internal/tui/history_test.go` — `withProfile`, `fakeDocs`.
- `cmd/generate_test.go`, `internal/tui/generate_test.go`, `internal/deploy/deploy_test.go`, `internal/provider/markdown/deploy_test.go`, `internal/profile/profile_test.go`, `internal/tui/barrun_test.go` — los dobles implementan los puertos nuevos.
- `docs/arquitectura/proveedores.md` — puertos y la nota de borrar.
- `docs/arquitectura/tui.md` — mapa de archivos y Borrar en Historial.
- `docs/guia-usuario.md` — `x`, qué pasa al borrar y `draco history rm`.
- `docs/roadmap.md` — 077 ✅.
- `docs/bitacora/fase-7/077-historial-borrar.md`, índice.

## 5. Cómo verificar

```bash
gofmt -l ./cmd ./internal && go vet ./...     # corridos: limpio
go test ./cmd ./internal/...                  # corridos: ok
make check                                    # lo corre el autor
```

Contra una instancia propia (lo corre el autor):

- **Confluence:** publicar con `draco g`, luego `draco history rm -i <TAREA>`.
  La página debe quedar en la papelera del space, el enlace debe desaparecer
  de la tarea en Jira y `draco history` ya no debe listarla.
- **Markdown:** lo mismo con un perfil Markdown. El archivo desaparece.
- **TUI:** `x` en Recientes, escribir la tarea, `enter`.

## 6. Pendientes

- Probar contra Atlassian real. La papelera de la v2 sin `purge` y el 404 de
  Jira sin enlace salen de la documentación de sus APIs, no de una prueba.
- En Markdown, si la carpeta del perfil cambió desde que se publicó, el
  archivo no se encuentra y se toma como «ya no existía».
- **078 (opcional):** módulo y sprint de QA elegidos de listas.
