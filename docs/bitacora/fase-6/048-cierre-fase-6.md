# 048 — Cierre de la fase 6

**Fecha:** 2026-10-21
**Rama:** docs/cierre-fase-6 → develop
**Fase del roadmap:** 6

## 1. Solicitud

Hacer la actividad 7 de la fase 6: el cierre de la fase y su release
`v0.2.2`.

## 2. Motivación

| Problema | Daño real |
|---|---|
| El roadmap, el README y `docs/arquitectura/tui.md` decían «en curso» y «por cerrar» | Con la fase terminada, el estado publicado quedaba viejo |
| Las pruebas manuales pendientes estaban repartidas en las bitácoras 042 a 047 | Sin juntarlas en el cierre, se pierde de vista qué falta probar contra Atlassian real antes de la v1.0 |

## 3. Diseño técnico

- **Sin cambios de código.** La ayuda general ya presenta el modo visual
  (bitácora 033), y la barra y Configuración se descubren desde la
  principal: la tecla `/` en el pie y Configuración en el menú.
- **Fase 6 cerrada** en el roadmap (estado, sección, actividad 7 y párrafo
  de cierre), en el README (lo siguiente es la v1.0) y en
  `docs/arquitectura/tui.md`.
- **Lo que entregó la fase** (bitácoras 041 a 047, más la 040):
  - el diseño y el plan;
  - Configuración: conexión con el token sin eco y la verificación de la
    cuenta, perfiles, QA y tema, con un asistente cuando no hay
    configuración;
  - el aviso de versión nueva en el marco;
  - la barra de comandos con sugerencias, que abre cada pantalla con los
    datos de la línea;
  - la Ayuda con el texto de `draco help`;
  - tres temas con colores ANSI básicos;
  - lo extraído de `cmd`: las reglas de `init` y `profile`, y la
    descripción de los comandos con su parser (`internal/cli`);
  - fuera del plan, `make promote` para llevar `develop` a `main`
    (bitácora 040).
- **Release `v0.2.2`** (lo publica el autor), como *patch*, según la
  decisión de la bitácora 039:

  ```bash
  make promote               # PR develop → main con merge commit, ramas al día
  make release bump=patch    # v0.2.2
  ```

  Es el primer uso real de `make promote`.
- **Verificación del criterio de cierre**, después del release:
  1. con `v0.2.1` instalado, `draco` muestra el aviso de `v0.2.2` en la
     cabecera (lo primero que prueba la actividad 4);
  2. `draco update` la instala y `draco version` responde `v0.2.2`;
  3. desde el binario instalado se abren Configuración y la barra (`/`).

## 4. Archivos

- `docs/roadmap.md` — fase 6 cerrada y su párrafo de cierre.
- `docs/arquitectura/tui.md` — estado.
- `README.md` — estado.
- `docs/bitacora/fase-6/048-cierre-fase-6.md`, índice.

## 5. Cómo verificar

```bash
make check
# después del merge a develop (lo corre el autor):
make promote
make release bump=patch
draco                      # el aviso de v0.2.2 con v0.2.1 instalado
draco update && draco version
```

## 6. Pendientes

- Release `v0.2.2` y su verificación (arriba), a cargo del autor.
- Pruebas manuales, todas a cargo del autor:
  - fase 4 (029 a 032): Windows Terminal y Linux/WSL, y Generar contra Jira
    y Confluence;
  - fase 5 (035 a 038): Exportar, Backlog y QA contra Atlassian real;
  - fase 6 (042 a 047):
    - la verificación de la cuenta (cuenta correcta, email distinto, token
      inválido) y el asistente con `HOME` vacío;
    - un consolidado con la config de QA creada en la TUI;
    - el aviso con un binario instalado;
    - `/g` hasta publicar y `/qa` contra Jira;
    - los temas con fondo claro y oscuro.
- Mejoras anotadas en la fase:
  - los formularios con la altura disponible (043);
  - mover el cursor dentro de la línea de la barra y escribir valores
    entre comillas (045);
  - el pie de teclas de `ansi` con colores ANSI básicos (047).
- v1.0: instaladores, migración desde gtt, guía de usuario.

**Nota (2026-10-21):** el release salió como `v0.3.0`
(`make release bump=minor`), no como `v0.2.2`. Ver la bitácora 049.
