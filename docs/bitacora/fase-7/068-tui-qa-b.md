# 068 — QA de la fase 7: contadores, tabla a todo el ancho y detalle junto al consolidado

**Fecha:** 2026-10-28
**Rama:** feature/tui-qa-b → develop
**Fase del roadmap:** 7

## 1. Solicitud

El autor pidió seguir con la actividad 6 de la fase 7: la pantalla Evaluar del
consolidado QA según el prototipo aprobado, con la regla de los contadores en
el flujo (`qa.Tally`).

## 2. Motivación

| Problema | Daño real |
|---|---|
| Evaluar mostraba una fila por tarea y nada más: para saber cuántas estaban bien había que contarlas a ojo | Con 20 tareas, un consolidado con fallas se publicaba sin haber visto el resumen |
| El detalle de la tarea elegida eran dos líneas bajo la tabla | Qué control falló y por qué no se veía hasta la vista previa |
| La pantalla usaba un tercio del ancho a 160×40 | El resto no decía nada |
| «Completa», «con fallas» y «sin verificar» no existían como concepto en el flujo | Cualquier pantalla que contara tendría su propia regla, y una consulta fallida podría contarse como falla |

## 3. Diseño técnico

- **La regla va en el flujo** (`internal/qa/tally.go`, norma 1): `StatusOf` y
  `Tally`. Una tarea está **completa** si pasan los cuatro controles, **con
  fallas** si falla alguno, y **sin verificar** si ninguno falla pero la
  consulta del documento falló (`DeployDocUnknown`). Dos reglas con test:
  - una consulta fallida **nunca** es una falla (regla del proyecto: es
    evidencia que falta);
  - una falla real **nunca** se esconde detrás de una consulta fallida (una
    tarea con errores de código y consulta fallida es «con fallas»).

  `Tally` cuenta solo las tareas ya evaluadas: una sin evaluar no tiene el
  documento comprobado y contarla daría una falla falsa.
- **Disposición desde 96 columnas** (`qa.go`): línea de pasos con el avance
  (`Evaluar 2/8`), los avisos y la cuenta; una caja a todo el ancho con los
  contadores y la tabla (tarea, `ERR DEV DOC PR`, resumen y estado); y,
  debajo y hasta el pie, el detalle (6/10) al lado del consolidado (4/10).
  Mientras se evalúa, la fila en curso dice `evaluando…` y las demás `por
  evaluar`; el avance del consolidado es la línea de estado.
- **Cede lo accesorio:** si la altura no deja al menos 6 líneas para las dos
  cajas (reservando 11 como máximo para ellas, el resto es de la tabla), la
  pantalla vuelve a la lista de una columna en vez de dibujarlas cortadas.
  Debajo de 96 columnas, la estructura de antes con los contadores.
- **El detalle no inventa datos.** El tracker responde sí o no (`QAIssue`
  trae booleanos): «con errores registrados» / «ninguno», sin conteo, y sin
  la columna de responsable del prototipo. El motivo de un «no se pudo
  verificar» es el aviso real de cada tarea: la pantalla guarda los
  `warnings` por tarea (`notes`) y los muestra con ella, en lugar de solo en
  una lista general.
- **Pie:** compacto con `↑↓ elegir` y `enter vista previa`; `↑/k` y `↓/j`
  en la ayuda completa.
- **`panel` en `layout.go`:** la caja con contenido ajustado y recortado con
  «…», que ya hacía a mano el detalle de Commits, ahora es una ayuda y la
  usan los dos.
- **Descartado:** contar en la pantalla. Era lo más corto, pero repetía una
  regla de QA en la interfaz, y la TUI y un futuro comando de resumen
  tendrían dos.
- **Tests existentes que cambian por diseño:** cabeceras `ERR DEV DOC PR`, las
  filas con su estado, frases que ahora se ajustan dentro de una caja, y el
  detalle (`Revisión QA OPS-20` en vez de `revisión OPS-20`).

## 4. Archivos

- `internal/qa/tally.go`, `tally_test.go` — nuevos: `StatusOf`, `Tally`, `Counts`; casos de cada estado, la consulta fallida y la falla real, y la tarea sin evaluar.
- `internal/tui/qa.go` — diseño ancho (`wide`, `wideTable`, `detailText`, `consolidated`), contadores, estado por fila, avisos por tarea, teclas.
- `internal/tui/layout.go` — `panel`.
- `internal/tui/commits.go` — el detalle usa `panel`.
- `internal/tui/qa_layout_test.go` — nuevo: 160×40, cursor y detalle, consulta fallida, evaluación en curso, tarea pendiente, 80×24, poca altura y teclas.
- `internal/tui/qa_test.go` — aserciones del diseño nuevo.
- `docs/arquitectura/tui.md`, `docs/guia-usuario.md` — QA › Evaluar.
- `docs/roadmap.md` — actividad 6 con ✅.
- `docs/bitacora/fase-7/068-tui-qa-b.md`, índice.

## 5. Cómo verificar

```bash
go vet ./internal/qa ./internal/tui && go test ./internal/qa ./internal/tui   # corridos: ok
make check                                                                    # lo corre el autor
make run                                                                      # Consolidado QA: contadores, tabla, ↑↓ y el detalle
```

## 6. Pendientes

- `make check` completo y la mirada en Windows Terminal a 80×24 y ancho,
  con un consolidado real de varios sprints: QA contra Jira real sigue sin
  probarse (heredado de las bitácoras 037 y 038).
- Siguiente: Proyectos, Hallazgos, Exportar y Configuración en columnas
  proporcionales, con el filtro visible (actividad 7).
