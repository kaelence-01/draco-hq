# 041 — Diseño de las pantallas y plan de la fase 6

**Fecha:** 2026-10-21
**Rama:** docs/plan-fase-6 → develop
**Fase del roadmap:** 6

## 1. Solicitud

Con la fase 5 cerrada y `v0.2.1` publicado, seguir con la fase 6, empezando
por el plan antes de escribir código, como en las fases 4 y 5.

## 2. Motivación

| Problema | Daño real |
|---|---|
| La fase 6 era una línea del roadmap («barra de comandos, temas, ayuda») | Sin actividades ni criterio de cierre, no habría forma de saber qué cierra cada una ni la fase |
| La configuración en formularios y el aviso de versión nueva estaban en el diseño, pero no en la fila de la fase | Configuración, la única entrada atenuada del menú, habría quedado fuera del plan |
| Tres reglas de configuración viven en `cmd` (perfil implícito al agregar el primero, URL con `https://`, email de la cuenta) | La TUI no importa `cmd`: la pantalla las habría copiado, y los dos modos guardarían distinto |
| Nombres, flags y parser de los comandos viven en `cmd`, y los flags no tienen descripción | La barra no podría sugerir ni leer los mismos comandos sin duplicar la tabla de flags |
| `/` ya filtra las listas de Proyectos y Ubicación, y Huh usa `ctrl+p`/`ctrl+n` | Una tecla global para la barra habría chocado con pantallas que ya existen |

## 3. Diseño técnico

Diseño en `docs/arquitectura/tui.md` §6 (secciones nuevas al final, para no
renumerar el §10 que citan `CLAUDE.md` y `estandares-desarrollo`).
Decisiones:

- **Configuración:**
  - Primero, las reglas de `init` y `profile` salen de `cmd` a funciones
    puras de `config` y `profile`; los comandos no cambian su salida.
  - La pantalla tiene secciones con su estado: Conexión, Perfiles, QA y
    Tema.
  - Conexión: el token con eco oculto, que vacío conserva el actual; la
    cuenta se verifica en un `tea.Cmd` y se ofrece corregir el email.
  - Una variable `ATLASSIAN_*` se detecta comparando la config del estado
    con la del archivo, sin que la TUI lea el entorno.
  - Sin configuración, la principal abre Configuración como asistente, en
    el orden de `draco init`.
- **Aviso de versión nueva:** `Deps.CheckUpdate` lo arma `cmd` con las
  reglas de `updateNotice`. La TUI lo pide al abrir y lo muestra en la
  cabecera y en la principal. Actualizar sigue siendo fuera del modo
  visual: `draco update` reemplaza el binario en uso.
- **Barra de comandos:**
  - La descripción de los comandos (nombres, subcomandos, flags con una
    descripción nueva) y el parser pasan a un paquete neutro
    (`internal/cli`, tentativo). La TUI la recibe en `Deps.Commands`.
  - Se abre con `/` solo desde la principal, por los choques de la tabla de
    arriba.
  - Sugiere comandos, flags y valores de la config.
  - Ejecutar **abre la pantalla del comando con los datos cargados**, en el
    primer paso a revisar. Descartado ejecutar el comando de texto dentro
    de la TUI: lo que el comando pregunta por texto, la TUI ya lo pregunta
    con pantallas.
  - Los flags sin pantalla (`--dry-run`, `--template`, `-o`…) se explican
    y no abren nada.
- **Ayuda:** `/help` muestra las teclas, la barra y los comandos. `/help g`
  muestra el mismo texto de `draco help g`, que `cmd` entrega por
  `Deps.Help`: sin texto duplicado.
- **Temas:** tres incluidos (`ansi`, `alto-contraste`, `mono`), solo con
  colores ANSI básicos (norma 9).
  - Se elige en Configuración y se guarda en `config.yaml` (`theme`).
  - `applyTheme` reasigna los estilos de `styles.go`, y ninguna pantalla
    cambia.
  - Descartados los colores del usuario: romperían el contraste en
    terminales claras u oscuras.
- **Plan:** 8 actividades en el roadmap, cada una con criterio de cierre:
  1. este plan;
  2. Configuración en dos partes (conexión y perfiles; QA);
  3. el aviso de versión nueva;
  4. la barra en dos partes (escribir y sugerir; ejecutar);
  5. los temas;
  6. el cierre con `v0.2.2`.

  Las dos partes de Configuración y de la barra cuentan como actividades
  separadas. Configuración va primero por ser la entrada atenuada, y los
  temas al final por ser lo de menor valor.
- **Versión:** `v0.2.2` al cierre, según la decisión de la bitácora 039
  (fases 5 y 6 como *patch* de `v0.2.x`).

## 4. Archivos

- `docs/arquitectura/tui.md` — estado, alcance de la fase 6, aviso en la
  entrada, `Deps` de la fase 6, §6 con Configuración, Aviso de versión
  nueva, Barra de comandos, Ayuda y Temas, tests y decisiones.
- `docs/roadmap.md` — fase 6 en curso, sus actividades y criterios; fase 5
  con `v0.2.1` publicado.
- `README.md` — la fase 6, en curso.
- `docs/bitacora/fase-6/041-plan-fase-6.md`, `docs/bitacora/README.md` —
  primera entrada de la fase 6 y su sección en el índice.

## 5. Cómo verificar

```bash
make check
```

## 6. Pendientes

- Actividad 2: Configuración (1).
- Siguen abiertas las pruebas manuales de las bitácoras 029 a 032 y 035 a
  038.
- Mejoras posibles, fuera del plan: historial de la barra entre sesiones y
  elegir otro perfil desde la barra (`--profile`).
