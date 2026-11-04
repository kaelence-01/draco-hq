# 073 — Detectar desde Jira sin buscarlo: enter, sin ejemplos engañosos y con resumen

**Fecha:** 2026-11-04
**Rama:** fix/qa-setup-directo → develop
**Fase del roadmap:** 7 (mejoras sobre la fase cerrada)

## 1. Solicitud

Con la `v0.4.2` instalada, el autor dijo que en la TUI «no carga nada». La
pantalla que pegó era el **formulario manual** de Configuración › QA (se
abre con `enter`), no la detección (`d`). Ahí, el campo de estados mostraba
`Testing, Aprobado, Finalizado`, que era el texto de ejemplo y no un valor.
Además, en la detección esperaba que los estados vinieran ya cargados, sin
pasar por las listas. Aprobó las tres correcciones propuestas.

## 2. Motivación

| Problema | Daño real |
|---|---|
| `enter` en QA abría el formulario manual; la detección estaba en `d` | Quien entra a configurar QA por primera vez cae donde hay que escribir ids, y la detección no se descubre |
| El formulario tenía ejemplos (`Testing, Aprobado, Finalizado`, `OPS`, `customfield_10001`) que en la terminal se ven casi igual que un valor | El autor creyó que los estados ya estaban cargados; el campo estaba vacío |
| La detección siempre abría las dos páginas de listas | Aunque la propuesta fuera correcta, había que recorrer las listas: «me sigue saliendo para elegir» |

## 3. Diseño técnico

- **`enter` en QA según el estado:**
  - Mientras `qa_jira` no valida (vacío o incompleto), `enter` abre «Detectar
    desde Jira» y `e` abre el formulario. Ahí siguen el email y los nombres
    del encabezado, y también se puede escribir a mano.
  - Ya completo, `enter` abre el formulario y `d` vuelve a detectar.
  - El pie y el texto de ayuda de la sección cambian con el estado.
    `e` siempre abre el formulario, aunque no se muestre.
- **Formulario sin ejemplos que parezcan valores:**
  - Se quitaron los de proyecto, estados, otros proyectos y campos.
  - La descripción de estados y campos dice cómo elegirlos sin escribir
    («esc y d en QA»).
  - Se dejó el ejemplo del email (`qa@example.test`), que no se confunde con
    un dato real.
- **Resumen antes de las listas.** Tras leer de Jira, la pantalla muestra lo
  que se guardaría, por nombre: con qué estado entra a QA, los estados
  elegidos y **los demás**, y los tres campos. Lo que falta aparece como
  «falta elegirlo».
  - **El origen se dice** (`Choice.Suggested`): «Propuesto desde tu
    configuración», o «Sugerido por draco: Testing y los finalizados. Revisa
    si falta alguno…».
  - **Por qué no se guarda solo:** draco no puede saber si un estado como
    «Aprobado» cuenta como «ya pasó por QA». Jira lo clasifica «en curso»,
    igual que «En desarrollo». Listar «los demás» al lado deja verlo de un
    vistazo, sin abrir listas.
  - **Completo** (lo normal al volver a configurar): `enter` guarda.
  - **Incompleto** (la primera vez faltan los contadores): `enter` acepta los
    estados y abre solo la página de campos.
  - **`e`** abre las listas completas, como en la 072.
- **Flujo:** `Choice.Complete`, `Choice.Suggested`, `Setup.StatusNames`
  (elegidos y demás, en el orden de Jira) y `Setup.FieldName`. El resumen
  solo nombra; no decide nada.
- **Modo comando, igual:** `draco qa setup` imprime el resumen.
  - Completo: «¿Guardar así?», y Enter guarda.
  - Incompleto: «¿Usar estos estados?», y Enter pide solo los contadores.
  - `n` lleva a las listas numeradas de la 072.
  - Se separaron `pickStatuses`, `pickFieldsAndSave`, `setupSummary` y
    `saveSetup`.
- Al guardar sin `qa_email`, la nota de Configuración avisa que falta y que
  `enter` abre el formulario.

## 4. Archivos

- `internal/qa/setup.go`, `setup_test.go` — `Choice.Complete`, `Choice.Suggested`, `StatusNames`, `FieldName`.
- `internal/tui/qasetup.go`, `qasetup_test.go` — resumen, `enter` guarda o pide solo los campos, `e` cambia; tests de los tres caminos, de la config completa y de la barra.
- `internal/tui/settings.go` — `enter` detecta con `qa_jira` incompleto, `e` formulario, pie y ayuda según el estado.
- `internal/tui/qaconfig.go`, `qaconfig_test.go` — sin ejemplos que parezcan valores; los tests abren el formulario con `e`.
- `cmd/qasetup.go`, `qasetup_test.go` — resumen y las dos preguntas; test de aceptar lo sugerido.
- `docs/guia-usuario.md` — Configuración › QA, el resumen y `draco qa setup`.
- `docs/roadmap.md` — 073 (el opcional de listas pasa a 074), 072 en `v0.4.2`.
- `docs/bitacora/fase-7/073-qa-setup-directo.md`, índice.

## 5. Cómo verificar

```bash
go vet ./... && gofmt -l ./cmd ./internal                  # corridos: limpio
go test ./...                                              # corridos: ok
make check                                                 # lo corre el autor
draco    # Configuración › QA › enter, con qa_jira vacío; contra Jira real: lo corre el autor
```

Configuración y el resumen revisados a 80×24 con un test temporal.

## 6. Pendientes

- **Probar contra Jira real**, sigue pendiente desde la 072: que se lean los
  estados y los campos del proyecto, y comparar lo guardado con los ids que
  gtt tenía fijos.
- **074 (opcional):** módulo y sprint elegidos de listas.
