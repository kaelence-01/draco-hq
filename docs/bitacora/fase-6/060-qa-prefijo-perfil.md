# 060 — QA reconoce el documento con cualquier plantilla

**Fecha:** 2026-10-22
**Rama:** fix/qa-prefijo-perfil → develop
**Fase del roadmap:** 6

## 1. Solicitud

Pendiente de la 056, que el autor pidió corregir antes del release `v0.3.2`.

## 2. Motivación

| Problema | Daño real |
|---|---|
| QA buscaba el enlace de cada tarea a su documento de despliegue con un solo prefijo, sacado de `.draco/templates/deploy` de la carpeta actual y de las plantillas del usuario; nunca de la carpeta de un perfil ni del repo de cada proyecto | Si una plantilla cambia `prefix`, el consolidado marca esas tareas como **sin documentación** aunque lo tengan: un control fallido falso, que es lo que la arquitectura prohíbe |
| Con la 055 cada proyecto puede tener su perfil y su plantilla | Un mismo consolidado cubre tareas documentadas con plantillas distintas |
| Cambiar el prefijo dejaba fuera los documentos publicados antes | Las tareas viejas pasaban a «sin documento» |

## 3. Diseño técnico

- **El puerto recibe varios prefijos:** `QATracker.HasDocumentLink(ctx,
  issueKey, prefixes)`. El adaptador de Atlassian sigue leyendo los enlaces
  remotos de la tarea una sola vez y acepta cualquier prefijo; uno vacío
  nunca cuenta (aceptaría cualquier enlace).
- **`qa.deployPrefixes`:** los prefijos de cada plantilla de despliegue que
  `draco g` puede usar con la config, resueltos igual que al generar:
  - la de cada proyecto: carpeta de su perfil (`ProfileFor`), luego
    `.draco/templates/deploy` de su `backend_path`, luego la del usuario;
  - la de una ejecución sin proyecto (perfil por defecto, carpeta actual);
  - siempre la incluida, para los documentos publicados antes de un cambio
    de prefijo.

  Sin repetidos y en ese orden. `Target.DeployPrefix` pasa a
  `DeployPrefixes`.
- **Una plantilla que no carga** (perfil inexistente, YAML inválido) se avisa
  en `Warnings` y se omite: `draco g` también fallaría con ella, y el
  consolidado sigue con los demás prefijos.
- **Descartado:** consultar Jira una vez por prefijo. Multiplicaría las
  llamadas por tarea sin necesidad.

## 4. Archivos

- `internal/provider/provider.go` — firma del puerto.
- `internal/provider/atlassian/jira_qa.go`, `ports.go`, `ports_test.go` —
  `HasRemoteLink` con varios prefijos y su test.
- `internal/qa/qa.go`, `qa_test.go` — `deployPrefixes`, `DeployPrefixes` y
  su test.
- `cmd/qa_test.go`, `internal/tui/qa_test.go` — dobles con la firma nueva.
- `docs/arquitectura/proveedores.md`, `docs/guia-usuario.md`.
- `docs/bitacora/fase-6/060-qa-prefijo-perfil.md`, índice.

## 5. Cómo verificar

```bash
make check
make test pkg=./internal/qa run='DeployPrefixes|EvaluateTask'
make test pkg=./internal/provider/atlassian run=HasRemoteLink
```

A mano (pendiente del autor): un proyecto cuya plantilla cambia `prefix`,
un documento publicado con ella enlazado a una tarea en QA, y `draco qa
--dry-run`: la tarea sale con documentación.
