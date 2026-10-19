# 003 — Bitácora, flujo de ramas y skills obligatorias

**Fecha:** 2026-10-19
**Rama:** feature/bitacora → develop
**Fase del roadmap:** 2

## 1. Solicitud

Llevar una bitácora de todo lo que se hace o cambia, empezando por el core y
QA, que se actualice con cada desarrollo; que sea una regla. Agregar skills que
se deban cumplir: bitácora, inicio y cierre de actividad, y estándares de
desarrollo, entre ellos evitar dejar tantos comentarios en el código.

Junto con eso, el autor fijó el flujo de ramas: `main` → `develop` →
`feature/<tema>`, pocos commits por push, PR hacia `develop`.

## 2. Motivación

| Problema | Daño real |
|---|---|
| El porqué de cada decisión quedaba en la conversación o en comentarios largos | Se pierde al cerrar la sesión; el código se llena de historia que nadie mantiene |
| Sin un flujo de ramas escrito | Commits en `main` local, ramas que siguen a `develop` y un `git push` que publica donde no debe (pasó en esta misma jornada) |
| Las verificaciones dependían de recordarlas | Un dato de la organización o un `gofmt` pendiente puede llegar a un repo público |

## 3. Diseño técnico

- **Una entrada por actividad**, no por versión como en gtt: draco aún no tiene
  releases. Archivo `NNN-<tema>.md`, mismo tema que la rama; índice en
  `docs/bitacora/README.md`. El formato es el de gtt (Solicitud → Motivación →
  Diseño técnico → Archivos → Cómo verificar) más «Pendientes».
- **La entrada viaja en el mismo commit** que el cambio; las de actividades ya
  fusionadas no se reescriben. 001 y 002 se escribieron después de su fusión y
  lo dicen.
- **Integración en `develop`**, no en `feature`: git no admite una rama
  `feature` y a la vez `feature/<tema>` (una sería archivo y la otra carpeta en
  `refs/heads/`).
- **Skills** en `.claude/skills/`: `nueva-actividad`, `estandares-desarrollo`,
  `bitacora`, `cerrar-actividad`. La regla queda en `CLAUDE.md` («Flujo de
  trabajo») para que aplique aunque no se invoque una skill.
- **`cerrar-actividad/check.sh`**, de solo lectura, como el `version-check` de
  gtt: rama y upstream, cantidad de commits sobre `develop`, build, vet, gofmt,
  tests, búsqueda de URLs de instancia, workspaces y correos, y la entrada de
  bitácora con sus secciones. Lee además `.claude/terminos-privados.txt`
  (ignorado por git) para que cada quien liste nombres reales que nunca deben
  llegar al repo sin publicarlos.
- **Comentarios mínimos:** doc comment de una línea en lo exportado, comentario
  en línea solo para un porqué no evidente, sin historia ni código comentado.
  El código existente todavía tiene comentarios largos; recortarlos es una
  actividad aparte (ver Pendientes).
- `.claude/settings.local.json` pasa al `.gitignore` del repo: hasta ahora solo
  lo excluía la configuración global de git de un colaborador.

## 4. Archivos

- `docs/bitacora/README.md` — índice, formato y reglas.
- `docs/bitacora/001-core.md`, `002-qa.md`, `003-bitacora.md` — entradas.
- `.claude/skills/nueva-actividad/SKILL.md` — preparar la rama.
- `.claude/skills/estandares-desarrollo/SKILL.md` — estándares de código.
- `.claude/skills/bitacora/SKILL.md` — escribir la entrada.
- `.claude/skills/cerrar-actividad/SKILL.md`, `check.sh` — verificar y cerrar.
- `CLAUDE.md` — sección «Flujo de trabajo (regla)».
- `docs/roadmap.md` — dos decisiones nuevas.
- `.gitignore` — `.claude/settings.local.json`, `.claude/terminos-privados.txt`.

## 5. Cómo verificar

```bash
bash .claude/skills/cerrar-actividad/check.sh
```

## 6. Pendientes

- Recortar los comentarios del código existente según
  `estandares-desarrollo` (actividad propia, `feature/comentarios`).
