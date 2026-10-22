# 059 — Importar la configuración de gtt

**Fecha:** 2026-10-22
**Rama:** feature/importar-gtt → develop
**Fase del roadmap:** 6

## 1. Solicitud

El autor pidió que quien ya usa gtt no tenga que volver a configurar draco:
al correr `draco init` después de instalarlo, detectar el `config.yaml` de
gtt y copiarlo; y en el modo visual, algo similar. Pidió también marcar en
el roadmap que este punto de la v1.0 se adelanta.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Quien viene de gtt tenía que repetir credenciales, space y todos sus proyectos | El primer contacto con draco es un trámite que ya hizo; con varios proyectos, horas de copiar rutas y workspaces a mano |
| Copiar el archivo a mano no era evidente | Nadie sabe que el formato es compatible ni dónde va el de draco |

## 3. Diseño técnico

- **El archivo de gtt ya es una config de draco.** Sus campos (credenciales,
  `confluence_space_key`, `default_project`, `projects` con checklist y
  backlog, `qa_email`, `qa_report`, `deploy_checklist`) son un subconjunto
  de los de draco, con el mismo nombre y significado, así que se lee con el
  mismo `config.Config`, sin transformar nada.
- **`config.GttConfig`** (pura salvo leer el archivo): busca
  `~/.config/gtt/config.yaml` y, como respaldo, `~/.config/deploy-doc/`,
  donde lo guardaban las versiones de gtt anteriores a su renombre. Normaliza
  `base_url` (sin `/` final). Un archivo que no se puede leer o con YAML
  inválido es un error: no se importa a medias.
- **Solo si draco no tiene configuración** (`Config.IsEmpty`, que también
  cubre un archivo vacío): importar nunca pisa una config de draco.
- **Se copia, no se enlaza:** el archivo de gtt no se toca (gtt queda
  congelado y se pueden usar los dos). Se guarda con `config.Save`
  (permisos 0600, porque lleva el token).
- **`config.ImportNotes`:** el resumen que muestran los dos modos (cuenta y
  sitio, space, cuántos proyectos y el por defecto), nunca el token. Avisa
  si falta `qa_jira`: gtt tenía en su código el proyecto, los estados y los
  campos de Jira del consolidado QA; draco los lee de la config (el repo es
  público) y no se pueden traer.
- **`draco init`:** antes de las preguntas, «Se encontró la configuración de
  gtt en … ¿Importarla? La de gtt no se modifica [S/n]». Si dice que sí:
  guarda, muestra el resumen, verifica la cuenta (corrige el email si hace
  falta, como siempre) y, si gtt no tenía proyectos, ofrece uno. Si dice
  que no, `init` sigue como antes. La ayuda de `draco init` lo explica.
- **Modo visual:** si el asistente de Configuración arranca con el archivo
  vacío y `Deps.LoadGtt` encuentra la config, abre antes **Importar de gtt**
  (`gttimport.go`): el resumen, `enter` importa con `state.save` y sigue con
  la verificación de la cuenta; `n` sigue con Conexión. Si la config de gtt
  no se puede leer, el asistente sigue con Conexión y Configuración dice
  por qué no lo ofreció.
- **Descartado por ahora:** buscar la config de gtt de Windows desde WSL (o
  al revés). Son carpetas personales distintas; la guía explica copiar el
  archivo. Queda anotado en el roadmap.
- **Roadmap:** el punto «Migración desde gtt» de la v1.0 pasa a hecho y
  adelantado; sale de las decisiones pendientes y entra en las tomadas.

## 4. Archivos

- `internal/config/gtt.go`, `gtt_test.go` — nuevos: `GttConfig`, `IsEmpty`,
  `ImportNotes`.
- `cmd/init.go`, `init_test.go` — la oferta de importar y la ayuda.
- `cmd/visual.go` — `LoadGtt` en `tui.Deps`.
- `internal/tui/tui.go` — `Deps.LoadGtt`.
- `internal/tui/settings.go`, `gttimport.go` (nuevo), `settings_test.go` —
  la pantalla del asistente; el helper de los tests acepta opciones de
  `Deps`.
- `docs/guia-usuario.md` — «¿Vienes de gtt?» y Configuración.
- `docs/arquitectura/tui.md` — el asistente con gtt.
- `docs/roadmap.md` — migración adelantada y decisión tomada.
- `docs/bitacora/fase-6/059-importar-gtt.md`, índice.

## 5. Cómo verificar

```bash
make check
make test pkg=./internal/config run='Gtt|ImportNotes'
make test pkg=./cmd run=Init
make test pkg=./internal/tui run='Wizard|Gtt'
```

A mano (pendiente del autor), con un `HOME` temporal que tenga solo
`.config/gtt/config.yaml`: `draco init` lo ofrece e importa sin pedir el
token; el asistente del modo visual hace lo mismo; `~/.config/gtt` queda
igual.
