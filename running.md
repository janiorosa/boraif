# Como colocar o BoraIF para rodar

O BoraIF tem três partes que precisam rodar juntas: **backend** (Go),
**frontend** (React) e **banco de dados** (PostgreSQL). Este documento
explica o que cada parte precisa para rodar, e os dois caminhos possíveis
para subir o projeto inteiro. Este arquivo se sobrepõe em boa parte a
`RUNNINGDOCKER.MD` — mantemos os dois porque foi pedido explicitamente;
`RUNNINGDOCKER.MD` é o guia exaustivo, passo a passo, só de Docker.

## Os dois caminhos

| Caminho | Quando usar | Guia completo |
|---|---|---|
| **Com Docker** (recomendado) | Quase sempre — não precisa instalar Go, Node nem Postgres na sua máquina | `RUNNINGDOCKER.MD` |
| **Sem Docker** | Depuração local, desenvolvimento sem Docker Desktop, ambientes que não suportam containers | `withoutdocker.md` |

## O que cada parte do projeto precisa para rodar

### Banco de dados (PostgreSQL)

- **Versão**: PostgreSQL 16 (é a que a imagem Docker usa; versões próximas
  também devem funcionar).
- **Extensões necessárias**: `citext` e `pg_trgm` — o backend as cria
  sozinho na primeira migration (`CREATE EXTENSION IF NOT EXISTS`), então
  você só precisa que o usuário do Postgres tenha permissão para isso
  (o superusuário padrão tem).
- **Não precisa rodar nenhum script manualmente**: o backend aplica as
  migrations (schema + dados de catálogo) sozinho, toda vez que inicia.
- Precisa de um volume/diretório persistente para os dados — via Docker,
  isso já é um volume nomeado (`postgres_data`); sem Docker, é a pasta de
  dados padrão do PostgreSQL instalado na sua máquina.

### Backend (Go)

- **Linguagem**: Go 1.23+.
- **Dependências**: resolvidas automaticamente via `go mod tidy`/`go
  build` (não precisa instalar nada manualmente além do próprio Go).
- **Chromium**: necessário só para gerar PDF (Fase 10) — via Docker já
  vem instalado na imagem; sem Docker, você precisa ter um Chromium/Chrome
  instalado e apontar `CHROME_PATH` para o executável.
- **Variáveis de ambiente obrigatórias**: `DATABASE_URL` e
  `API_KEY_ENCRYPTION_SECRET` (gere com `openssl rand -base64 32`) — o
  processo recusa iniciar sem elas. Veja a lista completa em
  `.env.example`.
- **Como roda**: um único binário (`cmd/server`) que, ao iniciar, aplica
  as migrations e sobe um servidor HTTP na porta configurada
  (`BACKEND_PORT`, padrão `8080`). O mesmo binário também tem um
  subcomando, `create-admin`, para criar o primeiro usuário administrador.

### Frontend (React + TypeScript)

- **Runtime**: Node.js 20+ só é necessário para *build* (`npm run build`)
  ou para rodar o servidor de desenvolvimento (`npm run dev`) — o
  resultado final em produção é HTML/CSS/JS estático, servido por Nginx
  (via Docker) ou por qualquer servidor de arquivos estáticos.
- **Dependências**: instaladas via `npm install` (React, React Router,
  TipTap e suas extensões, KaTeX).
- **Configuração**: nenhuma variável de ambiente própria — o frontend
  sempre fala com o backend através do caminho relativo `/api/*`, que
  precisa estar disponível na mesma origem (via proxy do Vite em
  desenvolvimento, ou via proxy do Nginx em produção).

## Resumo rápido — com Docker

```bash
git clone <url-do-repositorio> boraif && cd boraif
cp .env.example .env
# edite o .env: defina POSTGRES_PASSWORD e API_KEY_ENCRYPTION_SECRET
#   (gere a segunda com: openssl rand -base64 32)
docker compose up --build
# em outro terminal, depois que os containers subirem:
docker compose exec backend /app/server create-admin \
  --name="Seu Nome" --email="admin@escola.example" --password="senha-forte"
```

Acesse http://localhost:5173 — ou, de outra máquina na mesma rede local,
`http://<IP-do-servidor-docker>:5173` (o `docker-compose.yml` já publica
frontend e backend em todas as interfaces de rede, não só localhost; veja
a seção 5.1 de `RUNNINGDOCKER.MD` se não conectar). Detalhes completos,
backup/restore, troubleshooting: `RUNNINGDOCKER.MD`.

### Ver os logs do backend e do frontend

```bash
docker compose logs -f backend    # logs do backend (Go), em tempo real
docker compose logs -f frontend   # logs do frontend (Nginx), em tempo real
docker compose logs -f            # os dois (e o postgres) juntos, misturados
```

`Ctrl+C` para parar de acompanhar (não para os containers). Tire o `-f`
para ver só o que já foi logado até agora, sem ficar seguindo.

## Resumo rápido — sem Docker

```bash
# 1) Banco: crie um banco Postgres local (ex.: "boraif")

# 2) Backend
cd backend
export DATABASE_URL="postgres://usuario:senha@localhost:5432/boraif?sslmode=disable"
export API_KEY_ENCRYPTION_SECRET="$(openssl rand -base64 32)"
go run ./cmd/server create-admin --name="Seu Nome" --email="admin@escola.example" --password="senha-forte"
go run ./cmd/server   # sobe o backend em :8080 (aplica as migrations sozinho)

# 3) Frontend (em outro terminal)
cd frontend
npm install
npm run dev   # sobe em :5173, com proxy de /api para localhost:8080
```

Detalhes completos (instalar Go/Node/Postgres/Chromium, variáveis de
ambiente inteiras, como rodar em modo produção sem Docker):
`withoutdocker.md`.

## Rodando os testes automatizados do backend

```bash
cd backend
go test ./...
```

Não precisa de banco de dados para os testes atuais (cobrem regras de
negócio puras — veja `DEVELOPMENT.md`, seção "Testes").

## Documentos relacionados

- `RUNNINGDOCKER.MD` — guia exaustivo de Docker (recomendado).
- `withoutdocker.md` — guia exaustivo sem Docker.
- `architecture.md` — como o sistema é organizado por dentro.
- `database.md` — todas as tabelas do banco, explicadas.
- `project.md` — o que foi construído, fase a fase.
- `README.md` — visão geral do projeto para qualquer pessoa.
- `DEVELOPMENT.md` — histórico técnico detalhado de cada fase e decisão.
