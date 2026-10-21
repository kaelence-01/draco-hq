# 050 — Instalación con un comando y guía de usuario

**Fecha:** 2026-10-21
**Rama:** feature/install-y-guia → develop
**Fase del roadmap:** 6

## 1. Solicitud

El autor pidió instalar draco con un solo comando, como hacía gtt
(`iwr … install.ps1 | iex`), y una guía de usuario detallada, pública y que
se mantenga actualizada. Preguntó si podía ser un artifact o un Markdown en
el repo.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Instalar exigía descargar dos archivos, comparar un hash a mano y ejecutar el binario | Un paso de más y mucha gente no verifica el hash, o no instala |
| No había guía de usuario: solo el README y `draco help` de cada comando | El modo visual (teclas, barra, Configuración) y la config completa no estaban explicados en ningún lugar |
| Nada obligaba a mantener la documentación del usuario al día | La guía se desactualizaría con el primer cambio de una pantalla o un flag |

## 3. Diseño técnico

- **`scripts/install.sh`** (POSIX `sh`, para `curl | sh`):
  1. detecta el sistema y la arquitectura (Linux o macOS, amd64 o arm64);
  2. descarga el binario y `checksums.txt` del último release
     (`DRACO_VERSION` fija una versión);
  3. verifica el SHA-256 con `sha256sum` o `shasum`, y **si no coincide no
     instala nada**;
  4. lo mueve a `~/.local/bin` (`DRACO_INSTALL_DIR` cambia la carpeta);
  5. si la carpeta no está en el PATH, agrega la misma línea que el
     instalador integrado a `~/.bashrc` o `~/.zshrc`, una sola vez.
- **`scripts/install.ps1`** (para `iwr | iex`), con las mismas reglas que el
  instalador integrado y `draco update`:
  - va en un bloque `& { }` para no dejar variables ni
    `$ErrorActionPreference` en la sesión del usuario;
  - instala en `%LOCALAPPDATA%\Programs\draco`;
  - un `draco.exe` en uso se aparta como `.old`, que draco borra en su
    siguiente ejecución;
  - el PATH del usuario se edita en el registro conservando su tipo
    (`REG_EXPAND_SZ`) y sin expandir las variables de otros programas,
    como `path_windows.go`;
  - también suma la carpeta al PATH de la sesión actual;
  - los mensajes no usan `✓`: la consola clásica de PowerShell 5.1 puede no
    mostrarlo.
- **Se sirven desde `main`** (`raw.githubusercontent.com`), así que funcionan
  apenas el cambio llega a `main`, sin otro release.
- **Guía en el repo** (`docs/guia-usuario.md`), y no un artifact (decisión
  tomada con el autor):
  - el repo es público y GitHub muestra el Markdown sin cuenta;
  - cada versión tiene la guía de su código.

  Cubre la instalación, los primeros pasos, los perfiles, los proyectos, la
  config completa con datos ficticios, las variables de entorno, cada
  pantalla del modo visual con sus teclas, la barra, el modo comando, las
  plantillas, la solución de problemas y cómo desinstalar.
  - Los textos de ayuda se tomaron de `draco help` y los mensajes de error,
    del código.
- **Mantenerla al día:**
  - regla en `CLAUDE.md`: la guía se actualiza en el mismo commit que cambia
    lo que el usuario ve;
  - `check.sh` suma la sección «Guía de usuario», que avisa (no bloquea) si
    cambian `cmd/`, `internal/tui/`, `internal/config/` o
    `scripts/install.*` sin tocar la guía. Los tests no cuentan;
  - `cerrar-actividad` explica el aviso.
- **README:** la instalación con un comando, el enlace a la guía y los
  scripts en la estructura.
- **Fase:** la actividad adelanta dos puntos de la v1.0 (instaladores y
  guía). Va en la carpeta de la fase 6, la última, como la 040 y la 049
  después de su cierre, porque la v1.0 todavía no tiene plan ni número de
  fase.

## 4. Archivos

- `scripts/install.sh`, `scripts/install.ps1` — nuevos.
- `docs/guia-usuario.md` — nuevo.
- `README.md` — instalación, enlace a la guía y estructura.
- `CLAUDE.md` — scripts y guía en la arquitectura; regla de la guía.
- `.claude/skills/cerrar-actividad/check.sh`, `SKILL.md` — aviso de la guía.
- `docs/roadmap.md` — v1.0: script de instalación y guía hechos.
- `docs/bitacora/fase-6/050-install-y-guia.md`, índice.

## 5. Cómo verificar

```bash
make check
sh -n scripts/install.sh
```

Lo que se probó:

- **`install.sh`, sin red,** con un release simulado (un `curl` falso que
  sirve archivos locales). Pasaron cinco escenarios:
  1. instala, verifica, ejecuta `draco version` y agrega el PATH;
  2. una segunda vez no duplica la línea;
  3. con la carpeta en el PATH no toca `.bashrc`;
  4. `DRACO_VERSION` usa `releases/download/<versión>`;
  5. un SHA-256 que no coincide falla y no instala.
- **`install.sh` contra el release real `v0.3.0`**, con un `HOME` temporal:
  descargó, verificó, instaló y respondió `draco v0.3.0`.
- **`install.ps1`:** solo el análisis sintáctico de PowerShell y la lectura de
  `checksums.txt`. **No se ejecutó de punta a punta**: habría modificado el
  PATH del usuario en el registro de Windows.

## 6. Pendientes

- Primera instalación real con `install.ps1` en Windows, a cargo del autor,
  después de que el cambio llegue a `main`.
- v1.0: Homebrew tap y Scoop/winget (con otro nombre de paquete).
