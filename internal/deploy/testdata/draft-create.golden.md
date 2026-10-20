# Documento de Despliegue - OPS-7 - Filtro de pagos

|  |  |
|---|---|
| **Épica** | _Enlace a épica o función de Jira relacionada_ |
| **Tarea(s)** | <https://example.test/browse/OPS-7> |

## Arquitecturas e interfaces

### Proyectos y formularios - Backend:

| Servidor | Aplicación web | Ubicación | Nombre del archivo | Observación |
|---|---|---|---|---|
| [api](https://bitbucket.org/mi-org/api) | api | app | Pago.php<br>Cobro.php | [link](https://bitbucket.org/mi-org/api/commits/a1b2c3d#chg-app/Pago.php)<br>[link](https://bitbucket.org/mi-org/api/commits/a1b2c3d#chg-app/Cobro.php) |
| [api](https://bitbucket.org/mi-org/api) | api | routes | api.php | [link](https://bitbucket.org/mi-org/api/commits/a1b2c3d#chg-routes/api.php) |

## A considerar:

- [ ] Respaldar base de datos
- [ ] Desplegar backend
