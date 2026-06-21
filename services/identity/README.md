# Identity

Identity service for the Omni Platform. Handles authentication, users, sessions, and roles.

## Prerequisites

- Go 1.26+
- PostgreSQL (user: `identity`, password: `identity`, db: `identity`, port: `5432`)

## Setup

```bash
go mod download
```

Start PostgreSQL (via Docker Compose):

```bash
docker compose up -d identity-postgres
```

Generate an Ed25519 JWK key pair:

```bash
go run ./cmd/server gen-jwk
```

Or with Python:

```bash
python3 -c "
import base64, json
from cryptography.hazmat.primitives.asymmetric import ed25519
from cryptography.hazmat.primitives import serialization
key = ed25519.Ed25519PrivateKey.generate()
pub = key.public_key()
jwk = {
    'kty': 'OKP', 'crv': 'Ed25519',
    'd': base64.urlsafe_b64encode(key.private_bytes(encoding=serialization.Encoding.Raw, format=serialization.PrivateFormat.Raw, encryption_algorithm=serialization.NoEncryption())).decode().rstrip('='),
    'x': base64.urlsafe_b64encode(pub.public_bytes(encoding=serialization.Encoding.Raw, format=serialization.PublicFormat.Raw)).decode().rstrip('='),
}
print(json.dumps(jwk))
"
```

## Run

```bash
export IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK='{"kty":"OKP","crv":"Ed25519","d":"<seed>","x":"<pub>"}'
go run ./cmd/server
```

Or build and run:

```bash
go build -trimpath -o server ./cmd/server
export IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK='{"kty":"OKP","crv":"Ed25519","d":"<seed>","x":"<pub>"}'
./server
```

The server starts on `http://localhost:8080`. Health check at `/healthz`.

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `IDENTITY_DB_HOST` | `localhost` | PostgreSQL host |
| `IDENTITY_DB_PORT` | `5432` | PostgreSQL port |
| `IDENTITY_DB_USER` | `identity` | PostgreSQL user |
| `IDENTITY_DB_PASSWORD` | `identity` | PostgreSQL password |
| `IDENTITY_DB_NAME` | `identity` | PostgreSQL database |
| `IDENTITY_DB_SSLMODE` | `disable` | PostgreSQL SSL mode |
| `IDENTITY_SERVER_PORT` | `8080` | HTTP server port |
| `IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK` | *required* | Ed25519 JWK key pair JSON |
| `IDENTITY_AUTH_ACCESS_TOKEN_TTL` | `15m` | Access token TTL |
| `IDENTITY_AUTH_REFRESH_TOKEN_TTL` | `672h` | Refresh token TTL |
| `IDENTITY_AUTH_BCRYPT_COST` | `12` | Bcrypt cost factor |
| `IDENTITY_EMAIL_PROVIDER` | `log` | Email provider (`log` or `smtp`) |
| `IDENTITY_OTEL_ENDPOINT` | `localhost:4317` | OpenTelemetry collector endpoint |

## Run with Docker Compose

```bash
export IDENTITY_AUTH_JWT_PRIVATE_KEY_JWK='{"kty":"OKP","crv":"Ed25519","d":"<seed>","x":"<pub>"}'
docker compose up -d
```

This starts PostgreSQL, runs migrations, and starts the server on `:8080`.
