<!-- synced: constitution v2.0.0 | backend-standards v1.0.0 | environments-ci v1.0.0 -->

# Loopi v2 — Backend (Go)

La fuente de verdad del proyecto es `loopi-specs-v2/.specify/memory/`:
- `constitution.md` — principios de producto/arquitectura (rara vez cambian).
- `standards/backend.md` — normativo para este repo, cómo se implementa (cambia con frecuencia).
- `standards/environments-ci.md` — ambientes y CI comunes a los tres repos.

Este archivo es un **excerpt sincronizado** de esos documentos (ver cabecera de versión arriba).
Si trabajas en una tarea que no está cubierta aquí con suficiente detalle, lee el `standards/backend.md`
completo en `loopi-specs-v2` antes de implementar — no asumas que este resumen es exhaustivo para
casos que no haya visto todavía.

## Idioma

Todo en **español**: nombres de tablas, columnas, endpoints, campos JSON, mensajes de error.
Excepción: identificadores Go en camelCase/PascalCase según convención del lenguaje.

## Gates — ejecutar antes de cada commit, push o PR (en orden) [BE-CI-01]

```bash
go build ./...              # compila
golangci-lint run           # linter (govet + errcheck + staticcheck + unused + gosec)
govulncheck ./...           # CVEs en dependencias Go
gitleaks detect --no-git    # secrets en archivos del commit
go test ./...               # tests unitarios
```

Gate adicional en CI (GitHub Actions): Trivy fs scan (`HIGH,CRITICAL`, `ignore-unfixed: true`).

## Arquitectura del backend — separación de capas [BE-ARCH-01]

Cada módulo (`internal/<dominio>/`) tiene tres capas con responsabilidad exclusiva:

| Capa | Archivo | Responsabilidad única |
|------|---------|----------------------|
| Handler | `handler.go` | HTTP: parsear request, llamar al service, escribir response. Sin lógica de negocio ni SQL. |
| Service | `service.go` | Lógica de negocio. **Sin ninguna sentencia SQL ni dependencia directa a `*sql.DB`.** |
| Repository | `repository.go` | **Todo** acceso a BD. Única capa que importa `database/sql`. |

- El handler NUNCA llama al repositorio directamente (salvo handlers de jobs sin lógica de negocio).
- Si el service necesita un dato de BD que no está en el repositorio: **agregar el método al
  repositorio**, nunca un `dbQuerier` al service.
- Métodos del repositorio con nombres de dominio (`BuscarUsuarioPorNombre`), no de SQL (`QueryUsuarios`).
- **Test corolario**: si un test de `service_test.go` necesita BD real o mock de `*sql.DB`, hay SQL
  que no pertenece al service.
- Paginación: SIEMPRE server-side. Prohibida la paginación en memoria para colecciones no acotadas.

## Caché — patrón decorador Ristretto [BE-CACHE-01]

Solo catálogo de baja volatilidad (tiendas, empleados, unidades de medida, categorías, proveedores,
ítems, menú/recetas). Datos operacionales (stock, pedidos, inventarios, tokens de sesión) **nunca**
se cachean.

- **`internal/cache/`** — paquete compartido, único en el repo: `EntityCache[T any]` (wrapper sobre
  una instancia propia de Ristretto por entidad) + `ReadThrough[T any](cache, key, fetch)`.
- **`internal/<dominio>/cached_repository.go`** — decorador por módulo. Implementa la misma interfaz
  `Repository`, envuelve `repository.go` (que NO se modifica). Constructor:
  `NewCachedRepository(inner Repository, ttl time.Duration) Repository`.
- **`internal/<dominio>/cached_repository_test.go`** — obligatorio, cobertura ≥ 90%. Cubre: hit
  (inner no se invoca), miss (inner se invoca + se cachea), invalidación en escritura, error del
  inner no cachea nada.
- **TTL**: 24 h, sin excepción salvo justificación explícita en el plan.
- **Claves**: `"list"` (listar todos), `"id:<id>"` (por ID), `"<campo>:<valor>"` (filtro específico).
- **Invalidación**: Crear → `cache.Clear()`. Actualizar/Inactivar/Reactivar →
  `cache.Delete("id:<id>")` + `cache.Clear()`. Solo afecta la `EntityCache` de la entidad modificada.
  Si el repositorio SQL retorna error, no se invalida caché.
- **Multi-instancia**: Ristretto es in-process; la invalidación no se propaga entre instancias de
  App Engine (tradeoff aceptado para catálogo de baja volatilidad, máximo 24 h de desfase).
- Wiring en `main.go`: `cachedRepo := tiendas.NewCachedRepository(rawRepo, 24*time.Hour)`. Sin esa
  llamada explícita, no hay caché activa — nunca implícita.

## Testing — técnica por capa y cobertura [BE-TEST-01]

| Capa | Archivo | Técnica |
|------|---------|---------|
| Handler | `handler_test.go` | `httptest.NewRecorder()` + mock `Service` |
| Service | `service_test.go` | Mock de interfaz `Repository` (sin BD) |
| Middleware | `middleware_test.go` | `httptest` + JWT de test + mock `Repository` |
| Repository | `repository_test.go` | `go-sqlmock` |
| Cached repository | `cached_repository_test.go` | Mock de `Repository` (inner) |
| Config | `config_test.go` | `t.Setenv()` |

Cobertura mínima: lógica (`service.go`, `middleware.go`) ≥ 95% · infraestructura
(`repository.go`, `config/`) ≥ 90% · wiring OTel (`metrics.go`, `otel.go`) ≥ 70% · excluidos:
`cmd/`, `main.go`.

Middleware DEBE cubrir: token ausente/vacío/firma inválida/expirado/algoritmo incorrecto, token
revocado, BD no disponible en blacklist (fail-closed → 503), token válido.

Integraciones con BD/servicios externos (POS, GCP): siempre con **mocks** en test — nunca
integración directa (CI no tiene acceso a infraestructura real).

## Convenciones de API REST [BE-API-01]

- Prefijo obligatorio: `/api/v1/`. Recursos en snake_case plural: `/api/v1/pedidos`,
  `/api/v1/lineas_pedido`. Acciones no-CRUD como sub-recursos: `/api/v1/pedidos/{id}/confirmar`.
- Jobs programados en `/internal/jobs/` — nunca en `/api/v1/`.
- **Filtro de estado — `?estado`** (normativo, TODOS los módulos con campo `activo`): valores
  exactos `estado=activo|inactivo|todos`. Inválido → `400 { "error": "estado_invalido" }`.
  Default server-side sin el parámetro: `todos`. **Nunca** `?activo=true|false` ni `?active=` ni
  `?status=` — el nombre del parámetro es siempre `estado`.
- Formato de error (mismo esquema para todos):

```json
{ "error": "codigo_snake_case", "mensaje": "Texto para el usuario", "campo": "opcional", "detalles": [] }
```

- HTTP: 200 OK · 201 Created · 204 sin contenido · 400 input inválido · 401 sin auth
  · 403 sin permiso · 404 no existe · 409 conflicto · 422 regla de negocio ·
  **423 cuenta bloqueada** (ej. intentos fallidos de login) · 500 error interno.

## Convenciones de base de datos [BE-DATA-01]

- PKs: `BIGINT UNSIGNED` auto-incremental. Sin UUIDs. Se exponen como entero en la API.
- Toda tabla tiene `creado_en DATETIME NOT NULL` y `actualizado_en DATETIME NOT NULL`.
- Zona horaria: **America/Bogota** (UTC-5, sin DST). DSN de MySQL con `loc=America%2FBogota`.
- Campos de dominio en español: `iniciado_en`, `completado_en`, `enviado_en`, `expira_en`.
- Tablas: snake_case, español, plural — `pedidos`, `lineas_pedido`, `despachos`.
- Claves foráneas: `{tabla_singular}_id` — ej. `tienda_id`, `item_id`.
- Índices únicos: `uq_{tabla}_{campos}` — ej. `uq_pedidos_tienda_semana`.
- **Nunca `DELETE` físico**. Catálogo: `activo TINYINT(1)`. Ciclo de vida: `estado ENUM(...)`.
- Moneda: `INT` COP sin decimales. Cantidades: `DECIMAL(12,4)`. Semanas: ISO 8601 `YYYY-WNN`.
- Migraciones versionadas, reversibles, vía `golang-migrate/migrate`.

## Roles y JWT (principio P-III, ver constitution.md)

Cuatro roles: `admin`, `lider_compras`, `lider_tienda`, `barista`.

- `admin` y `lider_compras`: JWT sin `tienda_id` fijo.
- `lider_tienda` y `barista`: JWT con `tienda_id`.
- Validar rol en **cada** endpoint — la validación backend es vinculante.
- Un usuario inactivo nunca puede autenticarse.

## Jobs programados [BE-JOBS-01]

- Endpoints en `/internal/jobs/`, nunca en `/api/v1/`.
- Protegidos con header `X-CloudScheduler: true`. Sin ese header → `403`.
- Log obligatorio por ejecución: tipo de job, `iniciado_en`, `completado_en`, resultado
  (`ok`/`error`), registros procesados, mensaje de error si falla.
- Un job que falla no reintenta en la misma ejecución (lo gestiona Cloud Scheduler).

## Observabilidad [BE-OBS-01]

- Instrumentar con **OpenTelemetry** (trazas y métricas). Backend de observabilidad: **Datadog**
  (APM y métricas, NO logs). Logs estructurados JSON a stdout → **GCP Cloud Logging** exclusivamente.
- Campos obligatorios en logs: `tienda_id`, `user_id`, `rol`, timestamp, operación, nivel.
- **Nomenclatura de métricas**: `[dominio].[entidad].[operacion].[tipo]` (tipo: `duration` ms,
  `total` contador, `size` gauge). Etiqueta `resultado` siempre en operaciones que pueden fallar.
  Etiqueta `tienda_id` en operaciones de negocio (cardinalidad ≤ 20 tiendas).
- **`user_id` NUNCA como etiqueta de métrica** (alta cardinalidad → costo Datadog). Va en el
  atributo del span, no en la métrica. Tampoco IPs, UUIDs de request ni tokens como etiquetas.
- Todo feature con endpoints críticos DEBE declarar spans y métricas en la sección
  `## Observabilidad` de su `spec.md` (fuente de verdad de qué instrumentar en esta feature).

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
