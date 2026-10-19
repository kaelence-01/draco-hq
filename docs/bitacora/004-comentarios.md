# 004 — Comentarios al estándar mínimo

**Fecha:** 2026-10-19
**Rama:** feature/comentarios → develop
**Fase del roadmap:** 2

## 1. Solicitud

Aplicar al código existente el estándar de comentarios mínimos fijado en la
actividad 003 (`estandares-desarrollo`), antes de seguir con el flujo de
exportar.

## 2. Motivación

| Problema | Daño real |
|---|---|
| 730 líneas de comentario en `internal/`, muchas de párrafos | El código nuevo imita al que lo rodea: exportar y lo siguiente saldrían igual y habría que recortar dos veces |
| Comentarios con historia («as this used to», «antes», «See issue #4») | Envejecen sin que nadie los mantenga; la historia ya tiene su lugar en la bitácora |
| Separadores decorativos y doc comments que repiten el código | Ruido al leer y al revisar diffs |

## 3. Diseño técnico

- **Solo comentarios.** Ninguna línea de lógica cambia. Se verificó comparando,
  archivo por archivo contra `develop`, la secuencia de tokens de Go sin
  comentarios: idéntica en los 28 archivos. Las únicas otras diferencias son
  líneas en blanco huérfanas y la realineación de `gofmt` en tres structs.
- **Criterio aplicado:** doc comment de una línea en lo exportado; una segunda
  solo para una restricción no deducible (qué significa `nil`/vacío, orden de
  pasos, efecto sobre Atlassian, reglas de seguridad). Comentario en línea solo
  para un porqué no evidente. Fuera historia, repeticiones, separadores y
  párrafos. Tests: solo lo que explica un dato de prueba.
- **Se conservaron, condensadas**, las razones que protegen contra un bug real:
  `quoteLiteral`, `space-id` en v2, estado desconocido frente a fallido,
  float64 tras JSON, no sobrescribir una página que no se pudo leer,
  `qa_email` no es control de acceso, `0600` en `Save`, referencias a P-008.
- **Correcciones de paso:** tres doc comments decían algo falso
  (`searchResponse` nombraba `SearchResult`; `FindPageByTitle` hablaba solo del
  documento de despliegue y hoy lo usa QA; `UpdatePage` no aclaraba que
  `currentVersion` es la versión leída). Cuatro comentarios en español pasaron
  a inglés.
- Resultado: **730 → 351** líneas de comentario (−52 %).

### Razones retiradas del código

Lo que explicaban los comentarios largos y conviene no perder:

**atlassian**
- `parseDevTaskKey`: gana la primera clave del summary. Partir por «-» fallaba
  con sufijos («… - OPS-7 - ajuste» daba «7 - ajuste») y con la raya larga.
- `HasDeployDocLink`: devolvía `false` ante un error de la API y marcaba sin
  documento a tareas que lo tenían; por eso hoy devuelve el error.
- `FindLastDeployDoc`: usa la API v2 de páginas, no CQL, porque el retraso del
  índice ocultaba documentos recién creados. Pide 100 porque v2 no filtra por
  prefijo de título.
- `findPageByExactTitle`: un `space-key` en v2 se ignora en silencio; con él se
  podía encontrar, y luego sobrescribir, una página homónima de otro space.
- `quoteLiteral`: los caracteres de control pasan a espacio porque JQL y CQL no
  admiten un salto de línea dentro de un literal.
- Conversor storage → texto: siete defectos heredados de gtt v1.4.0 (espacio
  entre inline, entidades HTML, tareas sin marcar, macro anidada que borraba el
  encabezado del panel, sublista pegada al ítem, `<br>` sin cerrar que se
  tragaba el resto, `start` de listas numeradas ignorado).

**deploy y qa**
- `FindExisting` (deploy): primero el título exacto, que lee la base sin
  retraso del índice y es la condición que hace fallar un create con 400; la
  búsqueda por clave del issue es un respaldo para títulos que cambiaron con el
  summary y, como depende del índice, su fallo no es error.
- `DraftUpdate`: «A considerar» lo escribe el equipo de despliegue; sobrescribir
  sin leer es la pérdida que nadie nota.
- `Result.LinkErr`: si falla el enlace remoto, semanas después QA marca la
  tarea sin documentación.
- `resolveSide`: el nombre del repo sale del directorio del clon para no
  compilar nombres de repos (P-001, P-008).
- `Locations.TeamWide`: la lista se amplía al equipo cuando el usuario no tiene
  documentos en el space o no se pudo identificar su cuenta.
- `CheckAccount`: `qa_email` vive en la config del propio usuario; evita
  publicar por accidente, la autorización real son los permisos de Atlassian.
- `EvaluateTask`: el reporte es evidencia de QA; un error de red o permisos no
  prueba que falte el documento.

**document y config**
- `ExtractSection`: «A considerar» se copia tal cual, sin mezclar, para que
  sobreviva el estado de las tareas marcadas.
- `commitFileURL`: un host de VCS desconocido devuelve "" (antes se emitía una
  URL con forma de Bitbucket que quedaba muerta). GitHub no tiene ancla por
  archivo porque es un digest del path.
- `defaultChecklist`: el default traía `php artisan migrate`; un paso de un
  stack no va en un binario que usan otros equipos.
- `taskItem`: el `localId` es posicional porque el texto viene de la config;
  los documentos existentes no se afectan porque su sección se preserva.
- `orDash`: un valor de config ausente se publica como «—», visiblemente
  vacío, no como un relleno que parezca dato.
- `GetProject`: un nombre de proyecto desconocido es error y no cae al default;
  si cayera, git correría en el repo equivocado.
- `Validate` lista todos los problemas a la vez; `LoadLocal` reporta un
  `config.yaml` inválido en vez de ignorarlo.

**backlog**
- La frecuencia de cambio es el mejor predictor de defectos; un hotspot que
  además supera `MaxLines` es el primer candidato a refactor.
- La tarea propuesta es un incremento (extraer una responsabilidad, cubrir un
  flujo), no «refactorizar el archivo»: por eso los puntos no dependen del
  tamaño.
- Un archivo grande que nadie toca queda en baja, salvo que sea enorme
  (3×`MaxLines`).
- `FixKeywords` solo suman evidencia: los equipos nombran un fix por su síntoma.
- El ID de un hallazgo es estable para que una etapa posterior recuerde cuáles
  se mostraron, tomaron o descartaron.
- `ReportVersion` sube cuando cambia la forma del JSON de `backlog scan --json`.
- Hotspot sin test es lo más riesgoso: se prioriza.

## 4. Archivos

28 archivos de `internal/` (producción y tests de `atlassian`, `backlog`,
`config`, `deploy`, `document`, `git`, `qa`): solo comentarios.
`client.go` y `confluence_fetch.go` ya cumplían.

## 5. Cómo verificar

```bash
go build ./... && go vet ./... && gofmt -l internal && go test ./...
# líneas cambiadas que no son comentario ni blanco (solo deben salir
# comentarios al final de línea y realineaciones de gofmt):
git diff -U0 develop -- '*.go' | grep -E '^[+-]' | grep -vE '^(\+\+\+|---)' \
  | grep -vE '^[+-]\s*//' | grep -vE '^[+-]\s*$'
```

## 6. Pendientes

- `internal/git/git.go` tiene un mensaje al usuario que nombra Bitbucket
  («Verifícalo en Bitbucket»): debería decir «en tu VCS» o usar `vcs_host`.
  Es código, no comentario; va en otra actividad.
