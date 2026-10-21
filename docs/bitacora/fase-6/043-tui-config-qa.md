# 043 — Configuración en la TUI (2): QA

**Fecha:** 2026-10-21
**Rama:** feature/tui-config-qa → develop
**Fase del roadmap:** 6

## 1. Solicitud

Seguir con la actividad 3 del plan de la fase 6 (bitácora 041):
Configuración (2), la sección QA.

## 2. Motivación

| Problema | Daño real |
|---|---|
| `qa_email`, `qa_report` y `qa_jira` solo se podían escribir a mano en el YAML | Quien publica el consolidado tenía que conocer las claves, los `customfield_*` y la forma de cada lista, y no había comando que lo hiciera |
| Un `qa_jira` incompleto aparecía recién al abrir el consolidado | El error decía qué faltaba, pero no dónde arreglarlo sin salir de la TUI |

## 3. Diseño técnico

- **Dos funciones puras en `config`**, con tests:
  - `QAJiraConfig.Empty`: nada de `qa_jira` escrito. QA es opcional, así
    que esa sección vacía no se guarda;
  - `SplitList`: una lista separada por comas, sin espacios ni elementos
    vacíos (estados y prefijos de revisión).
- **Formulario** (`qaconfig.go`), de cuatro páginas para caber en 24
  líneas. Parte del archivo (`LoadFile`). Muestra las listas unidas por
  comas, y los valores por defecto en la descripción de su campo
  (`DefaultTestingStatus`, `DefaultSprintName`, `DefaultReviewPrefixes`,
  `DefaultObservationMarker`), sin copiarlos.
- **Guardar:**
  - si `qa_jira` tiene algo escrito, `Validate` muestra todos los
    problemas a la vez sobre el formulario y no guarda. El formulario se
    rehace con lo escrito, como el de Generar;
  - si no, `state.save` escribe `qa_email` y los nombres del encabezado
    (`qa_report`, `nil` sin ninguno), y `qa_jira` solo si no está vacío.
- **Estado de la sección:** sin configurar (con la aclaración de que solo
  hace falta para el consolidado), incompleta, sin `qa_email` (nadie podría
  publicar) o completa, con el proyecto y la cuenta.
- **Consolidado:** un `qa_jira` inválido al abrir el perfil, o un
  `qa_email` que no es la cuenta, ahora remite a Configuración › QA. El
  segundo mensaje conserva la ruta del archivo.
- **Criterio de cierre:** un test llena el formulario desde cero y pasa lo
  guardado a `qa.Resolve`. Sale sin error, con el sprint
  `Pagos_Sprint 12`, los nombres del encabezado y sin avisos.

## 4. Archivos

- `internal/config/config.go`, `config_test.go` — `Empty` y `SplitList`,
  con tests.
- `internal/tui/qaconfig.go` — nuevo.
- `internal/tui/qaconfig_test.go` — nuevo. Cubre:
  - la config desde cero, aceptada por `qa.Resolve`;
  - todos los problemas a la vez, sin guardar y con lo escrito conservado;
  - sin Jira ni nombres, solo `qa_email`;
  - el formulario que parte del archivo y lo conserva.
- `internal/tui/settings.go`, `settings_test.go` — la sección QA.
- `internal/tui/qa.go` — los errores remiten a Configuración › QA.
- `docs/arquitectura/tui.md`, `docs/roadmap.md`, `README.md` — estado y
  diseño ajustado a lo implementado.
- `docs/bitacora/fase-6/043-tui-config-qa.md`, índice.

## 5. Cómo verificar

```bash
make test pkg=./internal/tui run='QAConfig|Settings' v=1
make test pkg=./internal/config
make check
make run            # Configuración › QA; luego Consolidado QA con lo guardado
```

## 6. Pendientes

- Prueba manual del autor: un consolidado contra su Jira con la config de
  QA creada desde la TUI.
- En una terminal de 24 líneas, el error de validación (unas 7 líneas) más
  una página del formulario no caben juntos: los últimos campos de la
  página quedan fuera de la vista, aunque conservan su valor. Se resolvería
  dándole a los formularios la altura disponible, para todas las pantallas
  con formulario; queda como mejora.
- Actividad 4: aviso de versión nueva.
