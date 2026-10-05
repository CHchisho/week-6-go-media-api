# week-6-go-media-api

Movie catalog. Go + MariaDB.

## Endpoints

- `POST /api/register`, `POST /api/login` — public, return a JWT (24h)
- `GET /api/movies`, `GET /api/movies/{id}` — public
- `POST /api/movies` — auth
- `PUT /api/movies/{id}`, `DELETE /api/movies/{id}` — auth, owner only

List query: `genre`, `year`, `director`, `sort` (`title|year|rating|created`), `order` (`asc|desc`), `page`, `limit` (1–50).

Genres: Action, Comedy, Drama, Horror, Sci-Fi, Documentary.

Auth header: `Authorization: Bearer <token>`.

Docs: `openapi.yaml` (also `GET /openapi.yaml`).

## Run

Local MariaDB (user `media`, password `media`, db `media_api`, port 3306). Load `schema.sql`, then:

```bash
go run .
```

Env: `PORT`, `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `JWT_SECRET`.

Docker (API + MariaDB):

```bash
docker compose up --build
```

API: http://localhost:8080. Reset DB: `docker compose down -v`.

## Example

```bash
curl.exe -s -X POST http://localhost:8080/api/register -H "Content-Type: application/json" -d "{\"username\":\"alice\",\"password\":\"secret123\"}"

curl.exe -s "http://localhost:8080/api/movies?genre=Sci-Fi&sort=year&order=asc&page=1&limit=10"
```

Tests (no database): `go test ./...`
