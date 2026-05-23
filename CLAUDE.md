# Loopi v2 — Backend (Go)

La fuente de verdad del proyecto es la constitución en `loopi-specs-v2/.specify/memory/constitution.md`.
Este archivo resume las reglas operativas que Claude Code necesita para escribir código correcto en este repo.

## Idioma

Todo en **español**: nombres de tablas, columnas, endpoints, campos JSON, mensajes de error.
Excepción: identificadores Go en camelCase/PascalCase según convención del lenguaje.

## Gates — ejecutar antes de cada commit, push o PR (en orden)

```bash
go build ./...              # compila
golangci-lint run           # linter (govet + errcheck + staticcheck + unused + gosec)
govulncheck ./...           # CVEs en dependencias Go
gitleaks detect --no-git    # secrets en archivos del commit
go test ./...               # tests unitarios
```

## Convenciones de API REST

- Prefijo obligatorio: `/api/v1/`
- Recursos en snake_case plural: `/api/v1/pedidos`, `/api/v1/lineas_pedido`
- Acciones no-CRUD como sub-recursos: `/api/v1/pedidos/{id}/confirmar`
- Jobs programados en `/internal/jobs/` — nunca en `/api/v1/`
- Formato de error (mismo esquema para todos):

```json
{ "error": "codigo_snake_case", "mensaje": "Texto para el usuario", "campo": "opcional", "detalles": [] }
```

- HTTP: 200 OK · 201 Created · 204 sin contenido · 400 input inválido · 401 sin auth
  · 403 sin permiso · 404 no existe · 409 conflicto · 422 regla de negocio · 500 error interno

## Convenciones de base de datos

- PKs: `BIGINT UNSIGNED` auto-incremental. Sin UUIDs.
- Toda tabla tiene `creado_en DATETIME NOT NULL` y `actualizado_en DATETIME NOT NULL`.
- Zona horaria: **America/Bogota** (UTC-5). Configurar `loc=America%2FBogota` en el DSN de MySQL.
- Campos de dominio en español: `iniciado_en`, `completado_en`, `enviado_en`, `expira_en`.
- Tablas: snake_case, español, plural — `pedidos`, `lineas_pedido`, `despachos`.
- Claves foráneas: `{tabla_singular}_id` — ej. `tienda_id`, `item_id`.
- Índices únicos: `uq_{tabla}_{campos}` — ej. `uq_pedidos_tienda_semana`.
- **Nunca `DELETE` físico**. Catálogo: `activo TINYINT(1)`. Ciclo de vida: `estado ENUM(...)`.
- Moneda: `INT` COP sin decimales. Cantidades: `DECIMAL(12,4)`.

## Roles y JWT

Cuatro roles: `admin`, `lider_compras`, `lider_tienda`, `barista`.

- `admin` y `lider_compras`: JWT sin `tienda_id` fijo.
- `lider_tienda` y `barista`: JWT con `tienda_id`.
- Validar rol en **cada** endpoint — la validación backend es vinculante.
- Un usuario inactivo nunca puede autenticarse.

## Caché

**Ristretto** solo para catálogo de baja volatilidad: items, unidades de medida, parámetros del
algoritmo. Datos operacionales (stock, pedidos, inventarios): siempre desde la BD. TTL explícito
obligatorio en cada entrada de caché.

## Jobs programados

- Endpoints en `/internal/jobs/`, nunca en `/api/v1/`.
- Protegidos con header `X-CloudScheduler: true`. Sin ese header → `403`.
- Log obligatorio: tipo de job, `iniciado_en`, `completado_en`, resultado, registros procesados.

## Observabilidad

- Instrumentar con **OpenTelemetry** (trazas y métricas). Backend de observabilidad: **Datadog**.
- Logs estructurados JSON a stdout con: `tienda_id`, `user_id`, `rol`, timestamp, operación.

---

# Git Workflow

## Gitflow — regla obligatoria

Todo cambio en este repositorio debe seguir el flujo **Gitflow**. Nunca hagas cambios directamente en `main` o `develop`.

### Ramas principales

| Rama | Propósito |
|------|-----------|
| `main` | Código en producción. Solo recibe merges desde `release/*` o `hotfix/*`. |
| `develop` | Base de integración. Todo trabajo nuevo parte desde aquí. |

### Crear un branch nuevo

**Siempre parte desde `develop`** (excepto `hotfix/*`, que parte desde `main`):

```bash
git checkout develop
git pull origin develop
git checkout -b <tipo>/<nombre-descriptivo>
```

### Convención de nombres

| Tipo | Prefijo | Cuándo usarlo | Ejemplo |
|------|---------|---------------|---------|
| Nueva funcionalidad | `feature/` | Cualquier nueva feature o mejora | `feature/auth-google-login` |
| Corrección urgente en prod | `hotfix/` | Bug crítico que requiere parche inmediato en `main` | `hotfix/fix-payment-crash` |
| Corrección no urgente | `bugfix/` | Bug detectado en `develop` o QA | `bugfix/fix-empty-cart-error` |
| Preparación de versión | `release/` | Estabilización antes de merge a `main` | `release/v1.2.0` |
| Tareas técnicas / refactor | `chore/` | Dependencias, CI, configuración, refactor | `chore/upgrade-node-20` |

### Reglas

- El nombre del branch debe ser en **minúsculas**, palabras separadas por `-`.
- Los `hotfix/*` parten desde `main` y se mergean a `main` **y** `develop`.
- Los `feature/*`, `bugfix/*` y `chore/*` parten desde `develop` y se mergean solo a `develop`.
- Los `release/*` parten desde `develop` y se mergean a `main` **y** `develop`.
- Nunca hagas `git push --force` en `main` o `develop`.
