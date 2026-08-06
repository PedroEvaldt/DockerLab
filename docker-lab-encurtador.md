# Docker Lab — Encurtador de URLs com Analytics

Projeto-laboratório para exercitar Docker e Docker Compose de ponta a ponta. A aplicação é propositalmente simples: o objetivo não é escrever Go complexo, é ter serviços de verdade com dependências, estado, rede e ciclo de build para justificar cada recurso do Docker.

## Arquitetura alvo

```
                     host:8080
                         │
                   ┌─────▼─────┐
                   │   nginx   │   (única porta publicada)
                   └─────┬─────┘
              rede: frontend
                   ┌─────▼─────┐
                   │  api (Go) │
                   └──┬─────┬──┘
              rede: backend (internal)
              ┌───────▼─┐  ┌▼────────┐
              │ postgres│  │  redis  │
              └─────────┘  └────┬────┘
                                │
                          ┌─────▼─────┐
                          │  worker   │  (mesma imagem da api)
                          └───────────┘

perfis opcionais: observability (prometheus, grafana) | tools (adminer)
```

## Regras do jogo

- Você escreve todo `Dockerfile`, `compose.yaml` e arquivos de override. Aqui só tem requisito e critério de aceite.
- Toda etapa termina com um comando de verificação. Se o comando não passar, a etapa não acabou.
- As perguntas ao final de cada etapa são pra você responder em voz alta (ou num `NOTES.md`). Se travar em alguma, me chama que a gente destrincha.
- Faça commit ao final de cada etapa. Você vai querer voltar.

---

## Etapa 0 — Aplicação mínima em Go

Sem Docker ainda. Rodando na sua máquina.

**Requisitos da API** (`net/http` puro basta):

| Rota | Método | Comportamento |
|---|---|---|
| `/api/shorten` | POST | Recebe `{"url": "..."}`, gera um código curto, grava no Postgres, devolve `{"code": "...", "short_url": "..."}` |
| `/{code}` | GET | Busca o código, incrementa um contador de cliques no Redis, responde `302` para a URL original |
| `/api/stats/{code}` | GET | Devolve a URL original e o total de cliques |
| `/healthz` | GET | `200` se Postgres e Redis respondem; `503` caso contrário |
| `/metrics` | GET | Formato Prometheus (`prometheus/client_golang`) — usado só na Etapa 7 |

**Worker**: mesmo binário, subcomando diferente (ex.: `./app worker`). A cada 30s, drena os contadores do Redis e persiste os totais no Postgres.

**Configuração**: tudo por variável de ambiente. Nada de `localhost` hardcoded — isso é o ponto inteiro do exercício.

- `DATABASE_URL` ou `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_NAME`
- `REDIS_ADDR`
- `APP_PORT`
- A senha do banco **não** vem de env var. Ver Etapa 4.

**Critério de aceite**: com Postgres e Redis rodando localmente, você encurta uma URL, acessa o código curto, é redirecionado, e o `/api/stats` mostra o clique.

---

## Etapa 1 — Dockerfile multi-stage

**Requisitos**:

- Estágio de build a partir de uma imagem `golang`, estágio final a partir de `scratch` ou `gcr.io/distroless/static`.
- Binário estático (pense: `CGO_ENABLED`, `-ldflags`).
- Rodar como usuário não-root.
- Copiar os certificados CA para o estágio final (senão qualquer chamada HTTPS quebra — e você vai descobrir isso do jeito difícil se pular).
- `.dockerignore` cobrindo `.git`, binários locais, `README`, etc.

**Verificação**:

```bash
docker build -t shortener:v1 .
docker images shortener:v1 --format '{{.Size}}'
docker run --rm shortener:v1 --version
```

**Critérios de aceite**:
- Imagem final abaixo de 20 MB.
- `docker run --rm --entrypoint sh shortener:v1` falha (não existe shell — é uma imagem mínima de verdade).
- Anote esse tamanho. Você vai comparar na Etapa 8.

**Perguntas**:
1. Por que o binário precisa ser estático se a imagem final é `scratch`?
2. Se você fizer `COPY . .` antes do `go mod download`, o que acontece com o cache de camadas quando você muda uma linha de código?
3. Qual a diferença entre `ENTRYPOINT` e `CMD` aqui, e por que a combinação escolhida importa pro worker rodar subcomando?

---

## Etapa 2 — Compose básico, volumes e healthcheck

**Requisitos**:

- Serviços `api`, `postgres`, `redis`.
- Volume **nomeado** para o Postgres. Nada de bind mount pra dados de banco.
- Script de inicialização do schema montado em `/docker-entrypoint-initdb.d` (bind mount read-only — aqui bind mount é o certo).
- `healthcheck` nos três serviços. O da API usa o `/healthz`; o do Postgres, `pg_isready`; o do Redis, `redis-cli ping`.
- A `api` só inicia depois que Postgres e Redis estiverem **healthy** — não apenas iniciados.
- `restart: unless-stopped` na api.

**Verificação**:

```bash
docker compose up -d
docker compose ps            # todos com status (healthy)
docker compose down          # dados sobrevivem?
docker compose up -d
curl localhost:8080/api/stats/<codigo-criado-antes>
```

**Critérios de aceite**:
- Os dados persistem entre `down` e `up`.
- `docker compose down -v` apaga tudo — confirme que você entende a diferença.
- Matar o Postgres (`docker compose kill postgres`) faz o `/healthz` da api virar `503` em poucos segundos.

**Perguntas**:
1. `depends_on` simples versus `depends_on` com `condition: service_healthy` — o que exatamente o primeiro garante?
2. Onde vive fisicamente um volume nomeado? Ache com `docker volume inspect`.
3. Por que `start_period` existe no healthcheck e o que quebra se você não usar?

---

## Etapa 3 — Redes e isolamento

**Requisitos**:

- Duas redes: `frontend` e `backend`. A `backend` marcada como `internal: true`.
- `nginx` como reverse proxy, na `frontend`, **único serviço com porta publicada** (8080).
- `api` nas duas redes. `postgres`, `redis` e `worker` **só** na `backend`.
- Remova todo `ports:` de postgres e redis. Se você precisar acessar o banco, use `docker compose exec`.

**Verificação**:

```bash
docker compose exec api ping -c1 postgres        # resolve pelo nome do serviço
docker compose exec nginx ping -c1 postgres      # deve falhar
docker compose exec postgres ping -c1 8.8.8.8    # deve falhar (internal)
ss -tlnp | grep 5432                             # nada escutando no host
```

**Perguntas**:
1. Quem resolve o nome `postgres` dentro do container da api? Em que endereço?
2. Se dois Compose projects diferentes usam um serviço chamado `redis`, eles colidem? Por quê?
3. `expose` versus `ports` — depois desta etapa, qual dos dois você realmente precisou?

---

## Etapa 4 — Variáveis de ambiente e secrets

**Requisitos**:

- `.env` com configuração não-sensível: portas, nome do banco, usuário, nível de log, tag da imagem.
- `.env.example` versionado; `.env` no `.gitignore`.
- Senha do Postgres via **Docker secret** montado como arquivo (`/run/secrets/db_password`), tanto no serviço `postgres` (`POSTGRES_PASSWORD_FILE`) quanto na sua aplicação Go — que deve ler o arquivo, não a env var.
- Use interpolação no Compose (`${VAR}`, `${VAR:-default}`, `${VAR:?erro se ausente}`) em pelo menos três lugares.

**Verificação**:

```bash
docker compose config              # confira os valores já interpolados
docker compose exec api env | grep -i pass    # não pode aparecer nada
docker inspect <container-api> | grep -i password
```

**Critério de aceite**: a senha não aparece em `docker inspect`, em `docker compose config`, nem em `env` dentro do container.

**Perguntas**:
1. Por que env var é considerada um lugar ruim pra segredo? Cite dois vazamentos concretos.
2. Qual a ordem de precedência entre `.env`, `environment:` no Compose, e uma variável exportada no seu shell?
3. Qual a diferença entre secrets no Compose standalone e no Swarm?

---

## Etapa 5 — Worker e reuso de imagem

**Requisitos**:

- Serviço `worker` usando **exatamente a mesma imagem** da api, mudando só o `command`.
- Apenas um serviço declara `build:`; o outro referencia a imagem por `image:`.
- O worker escala: `docker compose up -d --scale worker=3` deve funcionar sem conflito de porta ou de nome.
- Garanta que 3 workers não contabilizem o mesmo clique duas vezes (dica: pense em operação atômica no Redis, não em lock na aplicação).

**Verificação**:

```bash
docker compose up -d --scale worker=3
docker compose ps
# gere 100 cliques, espere o flush, confira o total no Postgres — tem que dar 100 exato
```

**Perguntas**:
1. Se ambos os serviços declaram `build:` com o mesmo contexto, quantas vezes o Docker constrói? Teste.
2. Por que `container_name:` impede o `--scale`?

---

## Etapa 6 — Override files e ambiente de dev

**Requisitos**:

- `compose.yaml` como base neutra (produção-like).
- `compose.override.yaml` — carregado automaticamente — com o modo dev: bind mount do código, hot reload (`air`), log em debug, e um estágio `dev` do Dockerfile com toolchain completo.
- `compose.prod.yaml` — explícito — com imagem de registry em vez de build, `restart: always`, limites de recursos (`deploy.resources.limits`), sem bind mounts.
- O Dockerfile ganha um `target: dev` usado só pelo override.

**Verificação**:

```bash
docker compose up -d                                       # dev, hot reload ativo
# edite um handler, salve, veja recarregar sem rebuild manual

docker compose -f compose.yaml -f compose.prod.yaml config  # confira o merge
docker compose -f compose.yaml -f compose.prod.yaml up -d
```

**Perguntas**:
1. Como o Compose mescla listas (`ports`, `command`, `environment`) entre arquivos? Não é igual pra todas — descubra qual substitui e qual concatena.
2. `COMPOSE_FILE` como variável de ambiente: como isso simplifica sua vida?
3. Por que bind mount do código-fonte é ótimo em dev e inaceitável em produção?

---

## Etapa 7 — Profiles

**Requisitos**:

- Profile `observability`: `prometheus` (raspando `/metrics` da api e o `postgres-exporter`) e `grafana` com datasource provisionado por arquivo.
- Profile `tools`: `adminer` ou `pgadmin`, mais um `redis-commander`.
- Sem profile ativo, `docker compose up` sobe **apenas** o núcleo: nginx, api, worker, postgres, redis.

**Verificação**:

```bash
docker compose up -d                                 # 5 containers
docker compose --profile observability up -d         # 7+
COMPOSE_PROFILES=observability,tools docker compose up -d
docker compose --profile observability down
```

**Perguntas**:
1. O que acontece se um serviço sem profile faz `depends_on` de um serviço com profile?
2. Um serviço pode pertencer a mais de um profile?

---

## Etapa 8 — Cache de build e otimização

**Requisitos**:

- BuildKit com cache mounts para o módulo e o cache de compilação (`--mount=type=cache,target=/go/pkg/mod` e `/root/.cache/go-build`).
- Ordem de camadas otimizada: dependências antes do código-fonte.
- Build multi-arquitetura para `linux/amd64` e `linux/arm64` com `docker buildx`.
- Um `Dockerfile.naive` de propósito ruim (single-stage, `FROM golang`, `COPY . .` primeiro) pra comparação.

**Verificação**:

```bash
time docker build -f Dockerfile.naive -t shortener:naive .
time docker build -t shortener:opt .
# mude uma linha de código e refaça os dois:
time docker build -f Dockerfile.naive -t shortener:naive .
time docker build -t shortener:opt .

docker images | grep shortener
docker history shortener:opt
```

**Critérios de aceite**: monte uma tabela com tamanho da imagem, tempo de build frio e tempo de rebuild após mudança de uma linha, para as duas versões. A diferença deve ser gritante nas três colunas.

**Perguntas**:
1. Por que cache mount não vira camada da imagem?
2. O que invalida o cache de uma camada `COPY`? E de uma `RUN`?
3. `docker buildx build --platform linux/amd64,linux/arm64` precisa de quê pra funcionar na sua máquina?

---

## Etapa 9 — Registry

**Requisitos**:

- Suba um registry local (`registry:2`) com volume persistente. Tagueie, faça push e pull da sua imagem.
- Depois, publique no GHCR (`ghcr.io/pedroevaldt/shortener`) com tags semânticas: `v1.0.0`, `v1.0`, `latest`, e uma tag com o SHA do commit.
- `compose.prod.yaml` passa a puxar do registry em vez de buildar.
- Extra: `docker manifest inspect` numa imagem multi-arch, pra ver a lista de plataformas.

**Verificação**:

```bash
docker tag shortener:opt localhost:5000/shortener:v1.0.0
docker push localhost:5000/shortener:v1.0.0
docker rmi localhost:5000/shortener:v1.0.0
docker pull localhost:5000/shortener:v1.0.0

curl localhost:5000/v2/_catalog
curl localhost:5000/v2/shortener/tags/list
```

**Perguntas**:
1. Onde ficam suas credenciais depois do `docker login`? Isso te incomoda?
2. Por que `latest` é uma péssima ideia em produção?
3. O que exatamente é o *digest* de uma imagem, e por que `image: repo@sha256:...` é mais forte que uma tag?

---

## Etapa 10 — Quebra tudo

Sem receita. Diagnostique cada cenário:

1. Suba tudo, rode `docker compose down` **sem** `-v`, apague o diretório do projeto, clone de novo e suba. Os dados voltaram? Por quê?
2. Preencha o disco do container da api (`fallocate` num tmpfs ou não). O que o Docker faz? E se você aplicar `deploy.resources.limits.memory: 64M` e a api estourar?
3. Faça o healthcheck da api falhar de propósito com `restart: unless-stopped`. Descreva o comportamento observado — e por que ele *não* é o que muita gente espera.
4. Rode dois Compose projects do mesmo arquivo com `-p lab1` e `-p lab2`. O que colide e o que não colide?
5. `docker compose logs -f --tail=50 api worker` — configure driver de log com rotação (`max-size`, `max-file`) e prove que funciona.
6. Rode `docker system df` e depois `docker system prune -a`. Meça o antes e o depois. Entenda o que você perdeu.

---

## Entregável final

Ao terminar, o repositório deve ter:

```
.
├── cmd/                    # api + worker
├── internal/
├── Dockerfile              # multi-stage: deps → build → dev → runtime
├── Dockerfile.naive        # só pra comparação
├── .dockerignore
├── compose.yaml
├── compose.override.yaml
├── compose.prod.yaml
├── .env.example
├── secrets/db_password.txt # gitignored
├── nginx/nginx.conf
├── prometheus/prometheus.yml
├── grafana/provisioning/
├── db/init.sql
└── NOTES.md                # suas respostas + a tabela da Etapa 8
```

O `NOTES.md` é o que faz esse projeto valer no portfólio. Um `compose.yaml` qualquer um copia; a explicação de por que cada decisão foi tomada, não.
