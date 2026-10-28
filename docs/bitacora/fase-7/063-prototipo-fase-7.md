# 063 — El prototipo de la fase 7 en el repo

**Fecha:** 2026-10-27
**Rama:** docs/prototipo-fase-7 → develop
**Fase del roadmap:** 7

## 1. Solicitud

El autor preguntó si una sesión en otra computadora sabría seguir la fase 7
tal como se pensó. El plan estaba en el repo, pero el prototipo aprobado era
un artifact privado. El autor pidió guardarlo en el repo, con todo lo que
haga falta para retomarlo.

## 2. Motivación

| Problema | Daño real |
|---|---|
| El §11 describe las pantallas en texto; cómo se ven solo estaba en un artifact privado | Otra sesión implementaría el plan con otra disposición, y el autor no reconocería lo que aprobó |
| El link del artifact depende de la cuenta y no está en el repo | Quien no tenga el link no sabe que existe |
| El prototipo muestra datos que draco no lee | Copiándolo tal cual, alguien cambiaría puertos y adaptadores en una fase de interfaz |

## 3. Diseño técnico

- **`docs/arquitectura/prototipo-fase-7.html`:** el mismo prototipo, con
  `doctype`, `charset` y `viewport` para abrirlo como archivo local. Se
  quitó la etiqueta propia del visor de artifacts. Solo tiene datos
  genéricos (`OPS-7`, `pagos`, `mi-org`, `example.test`), así que cumple
  `patrones-seguros.md`. Usa fuentes de Google Fonts y, sin conexión, cae a
  la monoespaciada del sistema.
- **`tui.md` §11, «Referencia visual»:**
  - cómo abrirlo y para qué sirve cada control;
  - la tabla de pantalla del prototipo → archivo → actividad;
  - lo que muestra y no se implementa, y lo que simplifica;
  - que el texto del §11 manda si no coincide con el prototipo.
- **Roadmap:** el enlace al prototipo y cómo retomar la fase en otra sesión
  (leer el §11, abrir la pantalla de la actividad, seguir las normas del
  §10, tomar la primera actividad sin ✅).
- **`CLAUDE.md`:** el archivo en el árbol de `docs/`, para que una sesión
  nueva lo encuentre.
- **Descartado:** dejar solo el link del artifact. Es privado, puede
  cambiar y no viaja con el repo.

## 4. Archivos

- `docs/arquitectura/prototipo-fase-7.html` — nuevo.
- `docs/arquitectura/tui.md` — §11, «Referencia visual».
- `docs/roadmap.md` — fase 7: enlace al prototipo y cómo retomarla.
- `CLAUDE.md` — el prototipo en el árbol de `docs/`.
- `docs/bitacora/fase-7/063-prototipo-fase-7.md`, índice.

## 5. Cómo verificar

```bash
make check
xdg-open docs/arquitectura/prototipo-fase-7.html   # o explorer.exe desde WSL
```
