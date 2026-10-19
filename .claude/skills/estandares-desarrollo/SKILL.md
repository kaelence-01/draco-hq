---
name: estandares-desarrollo
description: Estándares obligatorios para escribir o modificar código Go en draco — comentarios mínimos, idioma, estructura de los flujos, errores, tests, dependencias y datos de organización. Úsala SIEMPRE antes de escribir o editar código en internal/, cmd/ o main.go, y al revisar un diff propio antes de cerrar la actividad.
---

# estandares-desarrollo

Las reglas de fondo están en `CLAUDE.md` («Reglas que no se negocian») y en
`docs/security/patrones-seguros.md`. Esta skill es la lista que se aplica al
escribir y al revisar.

## Comentarios: los mínimos

El código se explica con buenos nombres; la historia va a la bitácora.

- **Doc comment** en lo exportado: una línea que diga qué es o qué hace. Más
  líneas solo si hay una restricción que el lector no puede deducir (orden de
  llamada, qué significa un `nil`, un efecto sobre Atlassian).
- **Comentario en línea** solo para un *porqué* no evidente: una limitación de
  la API, un caso borde, una decisión de seguridad. Una línea.
- **Prohibido:**
  - repetir lo que el código ya dice (`// increment i`, `// returns the page`);
  - historia o cambios («antes hacía…», «as this used to», «fix de…»,
    números de issue): eso va en `docs/bitacora/`;
  - código comentado, `TODO` sin entrada en el roadmap, separadores decorativos;
  - párrafos de justificación: si una decisión necesita varias líneas, va en
    la bitácora y el código, como mucho, la referencia en una línea.
- Tests: sin comentarios salvo para explicar un dato de prueba no obvio.

## Idioma

- Código, identificadores y comentarios: **inglés**.
- Mensajes al usuario y mensajes de los tests: **español**.

## Estructura

- Flujos (`internal/<flujo>`): `Resolve` + `Session` con un método por paso;
  sin `fmt.Print`, sin stdin, sin `os.Exit`. Atlassian entra por una interfaz
  local con `var _ Atlassian = (*atlassian.Client)(nil)`.
- Lo degradado pero viable → `Warnings` o un campo; lo que la UI debe explicar
  → error centinela (`ErrXxx`). Nunca imprimir desde un flujo.
- Una consulta fallida no se convierte en un dato falso (un `false`, una lista
  vacía): se marca como desconocida o se devuelve el error.
- Funciones puras para todo lo que se pueda (JQL, títulos, ADF) y probarlas.
- Sin generalizar «por si acaso»: lo propio de una instancia va a config; lo
  demás, lo más simple que funcione.

## Seguridad y datos

- Ningún dato de una organización en código, tests, ayudas ni docs (P-001,
  P-008): URLs de instancia, space/project keys, IDs, repos, workspace,
  nombres de personas o módulos reales. Ejemplos: `OPS-7`, `pagos-api`,
  `mi-org`, `Pagos`, `Facturacion`, `example.test`.
- Todo valor en JQL/CQL pasa por `atlassian.quoteLiteral`.
- Confluence v2 filtra por `space-id`, nunca por `space-key`.
- Nunca contraseñas, tokens ni correos reales.

## Errores

- Mensajes en español, accionables (qué pasó y qué hacer).
- Envolver con `%w` cuando el llamador pueda necesitar `errors.Is`.
- Un error descartado cuyo efecto aparece después se devuelve como campo
  (`Result.LinkErr`).

## Tests

- Sin red ni Atlassian real; dobles (`fakeAPI`) definidos en el test.
- Lo que vuelve de Confluence se prueba tras un viaje por JSON (`roundTrip`).
- Cada cambio de comportamiento trae su test.

## Dependencias y formato

- Solo las aprobadas en `CLAUDE.md`; cualquier otra se justifica antes.
- `gofmt` sin diferencias, `go vet` limpio. No cambiar la versión de Go en
  `go.mod`.

## Revisión del diff propio

Antes de cerrar, recorre `git diff develop` buscando: comentarios que sobran
(aplicar la lista de arriba), datos de organización, texto en el idioma
equivocado y comportamiento sin test.
