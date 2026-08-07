# Processos — Completar a Etapa 0 (app Go mínima)

Você está na **Etapa 0** do lab: a aplicação Go precisa rodar inteira
(encurtar → redirecionar → ver clique no stats) **antes** de qualquer Docker.
As Etapas 1–10 são só Docker e vêm depois.

## Estado atual

| Peça | Estado |
|---|---|
| `internal/config` | ✅ pronto (falta só senha via secret — Etapa 4) |
| `internal/store` (Postgres) | 🟡 quase — 1 bug no `GetLink` |
| `cmd/main.go` | 🔴 hoje é um script de teste, não um servidor |
| Redis (contador de cliques) | 🔴 não existe |
| Gerador de código curto | 🔴 não existe |
| Handlers HTTP (as 4 rotas) | 🔴 não existem |
| Worker (`./app worker`) | 🔴 não existe |

`/metrics` (Etapa 7) e senha via secret (Etapa 4) ficam pra depois — não são da Etapa 0.

---

## Passo 0 — Subir o Redis (pré-requisito de infra)

```bash
docker run -d --name redis7 -p 127.0.0.1:6379:6379 redis:7-alpine
docker exec redis7 redis-cli ping        # deve responder: PONG
```

## Passo 1 — Corrigir o `GetLink`

`internal/store/store.go` — troque o `SELECT *` por colunas explícitas:

```go
sql := "SELECT url, clicks FROM links WHERE code = $1;"
```

**Verifica:** `go run ./cmd` roda até o fim sem o erro `4 and 2`.

## Passo 2 — Store do Redis (`internal/store/redis.go`)

```bash
go get github.com/redis/go-redis/v9
```

Métodos mínimos:

```go
type RedisStore struct { client *redis.Client }

func NewRedisStore(cfg config.Config) *RedisStore         // redis.NewClient({Addr: cfg.RedisAddr})
func (r *RedisStore) IncrementClick(ctx, code) error      // INCR  "clicks:"+code
func (r *RedisStore) PendingClicks(ctx, code) (int64,err) // GET   "clicks:"+code  (pro /stats)
func (r *RedisStore) DrainClicks(ctx) (map[string]int64, error) // SCAN "clicks:*" + GETDEL de cada
func (r *RedisStore) Ping(ctx) error                      // pro /healthz
func (r *RedisStore) Close() error
```

> No drain use **`GETDEL`** (lê e apaga atômico). Isso blinda contra o worker
> escalado da Etapa 5 contar o mesmo clique duas vezes.

## Passo 3 — Gerador de código curto (`internal/shortcode/shortcode.go`)

```go
func Generate(n int) (string, error)   // n bytes de crypto/rand -> base62 (ou base64.RawURLEncoding)
```

Mantenha simples — 6–8 caracteres bastam.

## Passo 4 — Handlers HTTP (`internal/api/`)

```go
type Handler struct {
    pg      *store.PostgresStore
    rc      *store.RedisStore
    baseURL string   // cfg.PublicBaseURL, pra montar short_url
}
```

| Rota | Lógica |
|---|---|
| `POST /api/shorten` | decodifica `{"url"}` → `shortcode.Generate` → `pg.SaveLink` → responde `{"code","short_url"}` (`baseURL+"/"+code`) |
| `GET /{code}` | `pg.GetLink`; se `ErrNotFound` → 404; senão `rc.IncrementClick` e `http.Redirect(w,r,url,302)` |
| `GET /api/stats/{code}` | `pg.GetLink` (cliques persistidos) **+** `rc.PendingClicks` (ainda no Redis) → soma e devolve JSON |
| `GET /healthz` | `pg.Ping` **e** `rc.Ping`; ambos OK → 200, senão 503 |

Go 1.25: `mux.HandleFunc("GET /{code}", ...)` + `r.PathValue("code")` — sem router externo.

> O `+ PendingClicks` no `/stats` faz o clique aparecer **na hora**, sem esperar os 30s do worker.

## Passo 5 — `main` com dispatch de subcomando

```go
// ./app          -> API (default)
// ./app worker   -> worker
if len(os.Args) > 1 && os.Args[1] == "worker" {
    return runWorker(ctx, cfg, pg, rc)
}
return runAPI(ctx, cfg, pg, rc)   // monta o mux e http.ListenAndServe(":"+cfg.AppPort, mux)
```

Adicione `defer pg.Close()` / `defer rc.Close()`.

## Passo 6 — Worker (`internal/worker/`)

```go
t := time.NewTicker(30 * time.Second)
for {
    select {
    case <-ctx.Done(): return ctx.Err()
    case <-t.C:
        counts, _ := rc.DrainClicks(ctx)
        for code, delta := range counts {
            pg.IncrementClicks(ctx, code, delta)   // reusa o método atual do Postgres
        }
    }
}
```

Seu `IncrementClicks` (Postgres) que já existe é a peça de persistência do worker.

## Passo 7 — Verificação (critério de aceite da Etapa 0)

```bash
# terminal 1
go run ./cmd
# terminal 2
go run ./cmd worker
# terminal 3
curl -s -XPOST localhost:8080/api/shorten -d '{"url":"https://google.com"}'
#   -> {"code":"abc123","short_url":"http://localhost:8080/abc123"}
curl -sI localhost:8080/abc123           # -> HTTP/1.1 302 ... Location: https://google.com
curl -s  localhost:8080/api/stats/abc123 # -> mostra url + clicks: 1
curl -sI localhost:8080/healthz          # -> 200
```

Se fecha, a Etapa 0 acabou — commita e parte pro Dockerfile (Etapa 1).

---

## Ordem de ataque recomendada

1 → 0 → 2 → 3 → 4 → 5 → 6 → 7
(destrava com 1 e 0, depois constrói de baixo pra cima: stores → shortcode → api → main → worker)
