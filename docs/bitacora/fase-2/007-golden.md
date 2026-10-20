# 007 — Goldens del documento actual

**Fecha:** 2026-10-19
**Rama:** feature/golden → develop
**Fase del roadmap:** 2 (proveedores, actividad 2)

## 1. Solicitud

Actividad 2 de la secuencia de proveedores: fijar el documento que draco genera
hoy, antes de refactorizar, para que la actividad de puertos pueda demostrar que
el resultado no cambió.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Los tests del builder revisan piezas sueltas (título, URL de commit, preservación), no el documento completo | Un refactor puede cambiar una tabla, un ancho o un orden sin que ningún test falle |
| El consolidado QA imprime `time.Now()` | No se puede fijar byte a byte |
| La actividad de puertos cambia cómo los flujos pasan los datos al documento | Sin una foto previa no hay forma de probar «documento idéntico», el criterio de cierre de esa actividad |

## 3. Diseño técnico

- **`internal/golden`**: `Assert(t, name, v)` serializa `v` como JSON indentado
  y lo compara con `testdata/<name>.golden.json`. Con `DRACO_UPDATE_GOLDEN=1`
  lo reescribe. Variable de entorno y no un flag: un flag solo existe en los
  binarios de test que importan el paquete y rompe `go test ./... -update`.
  El error muestra la primera línea distinta.
- **Fecha del consolidado:** `document.QADoc.Date`; vacía usa la fecha de hoy,
  así que nada cambia para quien no la pasa. `qa.Target.Date` guarda el `now`
  que ya recibe `Resolve` y el flujo lo pasa al builder.
- **Siete goldens, con título y ADF:**
  - builder de deploy: completo (Bitbucket, backend y frontend, directorio `.`,
    checklist de config con un paso en blanco), GitHub con checklist por
    defecto, sin VCS con «A considerar» preservado;
  - builder de QA: sprint (controles, estado desconocido, observaciones, tarea
    de revisión, nombres de cabecera) y Kanban con observación y
    consideraciones preservadas;
  - flujos: `deploy.DraftCreate` y `qa.DraftCreate` en modo Sprint después de
    `Collect` y `EvaluateTask`. Fijan cómo cada flujo arma los datos.
- Verificado: deterministas en 5 corridas seguidas, y un cambio de un título
  del builder hace fallar los goldens señalando la línea.

## 4. Archivos

- `internal/golden/golden.go` — helper (nuevo).
- `internal/document/golden_test.go`, `testdata/*.golden.json` — 5 goldens.
- `internal/deploy/golden_test.go`, `testdata/draft-create.golden.json`.
- `internal/qa/golden_test.go`, `testdata/draft-create-sprint.golden.json`.
- `internal/document/qa_builder.go` — `QADoc.Date`.
- `internal/qa/qa.go` — `Target.Date`, pasado al builder.
- `CLAUDE.md` — sección de tests: goldens y cómo regenerarlos.
- `docs/roadmap.md` — actividades 1 y 2 marcadas.

## 5. Cómo verificar

```bash
go test -count=5 ./internal/document ./internal/deploy ./internal/qa -run Golden
bash .claude/skills/cerrar-actividad/check.sh
```

## 6. Pendientes

- La fecha del consolidado usa el formato `02 Jan 2006`, con el mes en inglés.
  Se fija tal cual; pasará a la plantilla en la actividad 5.
