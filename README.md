# be-extension-erp

Go API for the Extension ERP frontend.

## Stack

- **Gin** — HTTP router/middleware
- **GORM** — ORM (PostgreSQL driver)
- **Redis** — caching layer (`pkg/cache` with JSON + generic `Remember`)
- **JWT** — `golang-jwt/jwt/v5` access + refresh tokens
- **bcrypt** — password hashing
- **godotenv** — `.env` loading

## Layout

```
cmd/api/main.go              # entry point + graceful shutdown
internal/
  config/                    # env-driven config
  database/                  # postgres + redis connectors
  middleware/                # auth, recovery
  server/                    # gin engine, routes, composition root
  auth/                      # handler / service / repository / dto
  user/                      # gorm model
pkg/
  cache/                     # redis JSON wrapper (reusable)
  hash/                      # bcrypt helpers
  jwt/                       # token manager
  response/                  # standard JSON envelope helpers
```

## Run

```powershell
copy .env.example .env       # edit values
go run ./cmd/api
```

Health check: `GET http://localhost:8080/health`

## Auth endpoints

| Method | Path                | Auth |
|--------|---------------------|------|
| POST   | /api/auth/register  | no   |
| POST   | /api/auth/login     | no   |
| POST   | /api/auth/refresh   | no   |
| GET    | /api/auth/me        | yes  |
| POST   | /api/auth/logout    | yes  |

## Adding a new module

1. Create `internal/<module>/` with `model.go`, `dto.go`, `repository.go`, `service.go`, `handler.go`.
2. Wire it in `internal/server/server.go` (composition root).
3. Register routes in `internal/server/routes.go`.
4. Add migrations as SQL files under `migrations/` and apply them using `golang-migrate` instead of `db.AutoMigrate(...)`.

## Deploy ke Railway (sementara)

Repo `api-extension-erp` adalah salinan dari `be-extension-erp` yang dipakai khusus untuk deploy sementara di Railway.

### 1. Siapkan service di Railway

1. Login ke [railway.app](https://railway.app) → **New Project** → **Deploy from GitHub repo** → pilih repo `api-extension-erp`.
2. Tambahkan plugin:
   - **Postgres** (`+ New` → `Database` → `Add PostgreSQL`)
   - **Redis** (`+ New` → `Database` → `Add Redis`)
3. Buka service API → tab **Settings** → pastikan:
   - **Builder**: Dockerfile (Railway akan otomatis pakai `Dockerfile` + `railway.toml`).
   - **Healthcheck Path**: `/health` (sudah di-set lewat `railway.toml`).

### 2. Set environment variables

Salin isi [.env.railway.example](./.env.railway.example) ke tab **Variables** pada service API. Field penting:

| Var                     | Catatan                                                                 |
|-------------------------|-------------------------------------------------------------------------|
| `APP_PORT`              | **Jangan di-set.** API otomatis listen di `$PORT` yang diberikan Railway. |
| `DB_*`                  | Pakai reference variable `${{Postgres.PG*}}`.                            |
| `DB_SSLMODE`            | Set ke `require` (Railway Postgres mewajibkan TLS).                      |
| `REDIS_ADDR`            | `${{Redis.REDISHOST}}:${{Redis.REDISPORT}}`                              |
| `REDIS_PASSWORD`        | `${{Redis.REDISPASSWORD}}`                                               |
| `JWT_SECRET`            | Generate baru, contoh: `openssl rand -hex 64`.                           |
| `CORS_ALLOWED_ORIGINS`  | Domain frontend production, dipisah koma.                                |

### 3. Deploy

- Push ke branch yang di-track Railway → build & deploy berjalan otomatis.
- Migrations dijalankan saat startup oleh `cmd/api/main.go` lewat `golang-migrate`, jadi tidak perlu langkah tambahan.
- Health check `GET /health` harus mengembalikan `200` setelah service hidup.

### 4. CLI alternatif

```powershell
npm i -g @railway/cli
railway login
railway link            # pilih project
railway up              # build & deploy dari folder ini
railway logs            # tail logs
```

### Catatan

- File [railway.toml](./railway.toml) berisi konfigurasi builder, start command, dan healthcheck.
- Hanya satu perubahan kode dibanding `be-extension-erp`: `internal/config/config.go` sekarang memakai `PORT` sebagai fallback `APP_PORT` agar kompatibel dengan PaaS.
