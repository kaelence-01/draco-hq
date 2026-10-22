# 056 — Consistencia de la guía de usuario

**Fecha:** 2026-10-22
**Rama:** docs/consistencia-guia → develop
**Fase del roadmap:** 6

## 1. Solicitud

El autor vio inconsistencias en la documentación (un artifact que no se pudo
leer desde la sesión) y pidió seguir con una revisión propia de la guía
contra el código.

## 2. Motivación

| Problema | Daño real |
|---|---|
| Las actividades 051 a 055 cambiaron la configuración, los perfiles y las rutas en un día, y cada una actualizó solo su parte de la guía | Frases que quedaron a medias (el space «con perfiles», el índice de la sección 3) y otras que nunca fueron exactas contradicen lo que hace draco |
| La guía es pública y es lo que lee quien viene de gtt | Una indicación falsa en «Solución de problemas» o en las plantillas deja a la persona sin salida |

## 3. Diseño técnico

- **Método:** dos revisiones en paralelo, solo de lectura (secciones 1 a 3 y
  el README; secciones 4 a 8). Cada afirmación se contrastó con el código;
  solo se aplicó lo confirmado con su `archivo:línea`.
- **Correcciones (15):**
  1. Índice: el enlace a la sección 3 quedó roto al renombrarla en la 051.
  2. README: `draco init` configura el space y un proyecto, no un perfil.
  3. Perfiles: «con perfiles, el space sale de cada perfil» omitía que
     `--space` y el space del proyecto siguen ganando (la 053 lo dejó así).
  4. Proyecto: su `confluence_space_key` reemplaza al de Conexión o al del
     perfil, y solo aplica con destino Confluence.
  5. `CONFLUENCE_SPACE_KEY` solo reemplaza el space del nivel principal: sin
     efecto con perfiles, y el del proyecto y `--space` le ganan.
  6. El «ejemplo completo» pasa a «campos principales», suma el space del
     proyecto y remite el resto del backlog a `draco help backlog`.
  7. `draco template` también usa el proyecto por defecto.
  8. Cabecera de la TUI: muestra el perfil del proyecto activo, que usa
     Generar; QA y Exportar usan el por defecto.
  9. Barra: `--profile` se sugiere pero el modo visual no lo aplica; va en
     la lista de flags que se explican sin abrir nada.
  10. Backlog del formulario: saltar la página es dejar en blanco los dos
      campos.
  11. `draco qa` sin `-s` pregunta el sprint; Enter elige Kanban.
  12. Plantillas: `draco template init` escribe en la carpeta actual; al
      generar, el repo es el backend del proyecto, y en QA la carpeta
      actual.
  13. «Confluence space key no configurado»: sin perfiles se arregla en
      Conexión, y en `draco qa` no sirve el space del proyecto.
  14. «falta la sección 'qa_jira'» se suma a la fila de QA incompleta.
  15. Solución de problemas: `~/.zshrc` con zsh; el texto del SHA-256 al
      actualizar; el tema `mono` arregla colores, no caracteres.
- **No se cambió:** afirmaciones correctas en la práctica aunque el código
  no las compruebe de forma explícita (permisos del archivo en Windows,
  `NO_COLOR` en el modo visual a través de Lip Gloss, «nunca en CI» por la
  falta de terminal).

## 4. Archivos

- `docs/guia-usuario.md` — las correcciones.
- `README.md` — qué configura `draco init`.
- `docs/bitacora/fase-6/056-consistencia-guia.md`, índice.

## 5. Cómo verificar

```bash
make check
grep -n "3-conceptos" docs/guia-usuario.md   # el ancla coincide con el encabezado
```

## 6. Pendientes

- **QA y el prefijo de la plantilla de un perfil** (código, no guía):
  `qa.loadTemplates` toma el prefijo de los documentos de despliegue solo
  de `.draco/templates/deploy` de la carpeta actual y de la carpeta del
  usuario (`internal/qa/qa.go:145`), nunca de la carpeta de plantillas de
  un perfil. Si un perfil cambia `prefix` en su `template.yaml`, QA no
  encuentra esos documentos y marca las tareas sin documentación. Con la
  055 (un perfil por proyecto) el caso es más probable: el consolidado
  cubre tareas de proyectos con plantillas distintas.
