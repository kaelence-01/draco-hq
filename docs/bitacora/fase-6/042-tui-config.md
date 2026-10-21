# 042 — Configuración en la TUI (1): conexión, perfiles y asistente

**Fecha:** 2026-10-21
**Rama:** feature/tui-config → develop
**Fase del roadmap:** 6

## 1. Solicitud

Seguir con la actividad 2 del plan de la fase 6 (bitácora 041):
Configuración (1).

## 2. Motivación

| Problema | Daño real |
|---|---|
| Configuración era la única entrada atenuada del menú | Quien instala draco y abre la TUI tiene que salir a `draco init` para poder usarla |
| Tres reglas de configuración vivían en `cmd` (perfil implícito, URL, email de la cuenta) | La TUI no importa `cmd`: copiarlas daría dos formas de configurar que guardan distinto |
| Un token de `ATLASSIAN_TOKEN` no se ve en la config cargada | Sin un aviso, quien cambia el token en la TUI no sabe por qué draco sigue usando otro |

## 3. Diseño técnico

- **Núcleo fuera de `cmd`:** tres funciones puras, con sus tests.
  `draco init` y `draco profile` las usan, y sus tests pasaron sin cambios:
  - `config.KeepImplicitProfile` (antes en `runProfileAdd`);
  - `config.NormalizeBaseURL` (antes en `runInit`);
  - `profile.SameAccount` (antes en `verifyAccount`).
- **Configuración** (`settings.go`): Conexión y Perfiles, cada una con su
  estado. QA y Tema se suman en las actividades 3 y 6.
- **Conexión** (`connection.go`):
  - formulario que parte de `LoadFile`;
  - token con `EchoModeNone`. El diseño decía `EchoModePassword`, pero ese
    muestra un asterisco por carácter, es decir, el largo del token. Con
    `EchoModeNone` es igual a `draco init` (`term.ReadPassword`);
  - la URL se valida con `NormalizeBaseURL` en el mismo campo;
  - guarda con `state.save` y abre la verificación.
- **Verificación** (`accountCheck`): un `tea.Cmd` con `OpenProfile` sobre la
  config del archivo (`AnyScope`) y `Tracker.Me`.
  - Usa lo guardado, como `draco init`, y no la config del estado, donde
    ganaría el entorno. El diseño decía lo contrario y se corrigió: lo útil
    es saber si sirve lo que se acaba de escribir.
  - Un error no deshace lo guardado.
  - Si el email no es el de la cuenta, `s` lo corrige.
  - El paso se cancela con `esc` (`close()`).
- **Entorno:** el aviso aparece cuando la config del estado trae un valor
  que no está en el archivo. Nombra la variable, nunca su valor.
- **Perfiles** (`profiles.go`, `profileform.go`): lista con filtro y
  detalle, `d`, `n`, `e` y `x` como en Proyectos.
  - El formulario es de páginas: el grupo del space o el de la carpeta se
    oculta según el destino (`WithHideFunc`).
  - Tiene tres modos, uno por comando:
    - `profileNew` (`draco profile add`): conserva el implícito;
    - `profileEdit`: conserva lo que el formulario no muestra;
    - `profileFirst` (`draco init` sin perfiles): sin implícito, queda por
      defecto y no pregunta plantillas.
- **Asistente:** si el archivo no tiene credenciales ni perfiles,
  Configuración abre Conexión sola. Tras la verificación, si no hay
  perfiles, `enter` sigue con el primero. Al volver, si no hay proyectos,
  `p` abre el formulario de Proyectos (pantalla de Proyectos y formulario
  apilados con `tea.Sequence`).
  - El paso «primer perfil» depende de que el archivo no tenga perfiles,
    no de haber entrado por el asistente: así se comporta igual que
    `draco init`.
- **`backTo(s)`** en el marco cierra las pantallas que están sobre `s`. Lo
  usa el asistente para volver a Configuración desde el formulario del
  perfil.
- **Principal:** Configuración abre su pantalla, y el aviso sin credenciales
  dice «abre Configuración (o corre draco init)».
  - Ya no queda ninguna entrada sin pantalla, pero el dibujo de las
    atenuadas se conserva para las funciones futuras.
  - Sus tests usan una entrada de prueba (`withFutureItem`).
- **Sin cambios de dependencias.**

## 4. Archivos

- `internal/config/config.go`, `profile.go`, `profile_test.go` —
  `NormalizeBaseURL` y `KeepImplicitProfile`, con tests.
- `internal/profile/profile.go`, `profile_test.go` — `SameAccount`, con
  test.
- `cmd/init.go`, `cmd/profile.go` — usan las funciones extraídas; misma
  salida.
- `internal/tui/settings.go`, `connection.go`, `profiles.go`,
  `profileform.go` — nuevos.
- `internal/tui/settings_test.go` — nuevo. Cubre:
  - las secciones;
  - el asistente de punta a punta: token sin eco que no llega a la vista,
    el token del entorno que no se guarda, el email corregido, el primer
    perfil sin implícito y la oferta de un proyecto;
  - token vacío que conserva el actual, URL sin https y verificación
    fallida;
  - perfiles: por defecto, eliminar, nuevo con el implícito, editar y
    nombre repetido.
- `internal/tui/app.go` — `backTo`.
- `internal/tui/home.go`, `tui.go`, `app_test.go` — Configuración en el
  menú y el aviso; tests de las entradas atenuadas con una entrada de
  prueba.
- `docs/arquitectura/tui.md`, `docs/roadmap.md`, `README.md` — estado y
  diseño ajustado a lo implementado.
- `docs/bitacora/fase-6/042-tui-config.md`, índice.

## 5. Cómo verificar

```bash
make test pkg=./internal/tui run='Settings|Wizard|Connection|Check|Profile' v=1
make test pkg=./cmd
make check
make run            # Configuración: conexión, perfiles; con config vacía, el asistente
```

## 6. Pendientes

- Prueba manual del autor: la verificación contra su Atlassian (cuenta
  correcta, email distinto, token inválido) y el asistente con
  `HOME` apuntando a una carpeta vacía.
- Actividad 3: Configuración (2), QA.
