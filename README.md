# loopi-api-v2

Backend en Go para Loopi v2.

## Requisitos locales

| Herramienta | Versión mínima | Instalación |
|---|---|---|
| Go | 1.25 | https://go.dev/dl/ |
| golangci-lint | latest | `brew install golangci-lint` |
| govulncheck | latest | `go install golang.org/x/vuln/cmd/govulncheck@latest` |
| gitleaks | latest | `brew install gitleaks` |
| Google Cloud SDK (`gcloud`) | latest | https://cloud.google.com/sdk/docs/install |
| Cloud SQL Proxy | v2 | `brew install cloud-sql-proxy` |

## Variables de entorno

Crea un archivo `.env` en la raíz del proyecto (no se versiona):

```env
DB_DSN=<usuario>:<password>@tcp(127.0.0.1:3306)/<base_de_datos>?parseTime=true&loc=America%2FBogota
JWT_SECRET=<cadena aleatoria — genera con: openssl rand -base64 32>
ENV=dev
PORT=8080
APP_VERSION=1.0.0
OTEL_SERVICE_NAME=loopi-api
OTEL_EXPORTER_OTLP_ENDPOINT=
OTEL_EXPORTER_OTLP_HEADERS=
```

> `DB_DSN` apunta siempre a `127.0.0.1:3306` porque en local la conexión va a través del Cloud SQL Proxy.

## Levantar en local

### 1. Autenticarse en GCP

```bash
gcloud auth application-default login
```

### 2. Levantar el Cloud SQL Proxy

Déjalo corriendo en una terminal aparte:

```bash
cloud-sql-proxy loopi-dev-497600:us-central1:loopi-db-dev
```

### 3. Levantar el servidor

Desde VS Code: panel **Run and Debug** → seleccionar **Run API** o **Debug API**.

Desde terminal:

```bash
make run
```

### Confirmación de arranque correcto

Cuando el servidor esté listo verás este mensaje en la consola:

```
servidor iniciado en :8080 (CORS origin: http://localhost:4200)
```

## Comandos de desarrollo

```bash
make run    # levanta el servidor
make test   # ejecuta tests unitarios
make build  # compila el binario
make lint   # ejecuta go vet
```

## Gates — ejecutar antes de cada PR

```bash
go build ./...
golangci-lint run
govulncheck ./...
gitleaks detect --no-git
go test ./...
```
