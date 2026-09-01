# Como rodar o BoraIF sem Docker

Este guia é para quem quer (ou precisa) rodar o BoraIF diretamente na
máquina, sem containers — útil para depurar, desenvolver sem Docker
Desktop, ou em ambientes que não suportam containers. O caminho
recomendado para produção continua sendo Docker (`RUNNINGDOCKER.MD`).

## 1. O que instalar

| Componente | Versão | Para que serve |
|---|---|---|
| **Go** | 1.23 ou mais recente | compilar/rodar o backend |
| **Node.js** | 20 ou mais recente (inclui `npm`) | rodar/compilar o frontend |
| **PostgreSQL** | 16 (versões próximas devem funcionar) | banco de dados |
| **Chromium ou Google Chrome** | qualquer versão recente | só necessário para gerar PDF (Fase 10) |

### macOS (via [Homebrew](https://brew.sh))

```bash
brew install go node postgresql@16 chromium
brew services start postgresql@16
```

Se `brew install chromium` falhar (a fórmula às vezes é removida do
Homebrew), instale o Google Chrome normalmente
(https://www.google.com/chrome/) e aponte `CHROME_PATH` para
`/Applications/Google Chrome.app/Contents/MacOS/Google Chrome` mais
adiante.

### Ubuntu/Debian

```bash
# Go: baixe em https://go.dev/dl/ (o pacote apt costuma estar desatualizado)
sudo apt update
sudo apt install -y postgresql postgresql-contrib chromium nodejs npm
sudo systemctl start postgresql
```

Confira a versão do Node instalada (`node -v`); se for menor que 20, use o
[nvm](https://github.com/nvm-sh/nvm) para instalar uma versão mais nova.

### Windows

Recomendamos WSL2 (Ubuntu) e seguir as instruções acima dentro dele. Rodar
nativamente no Windows é possível, mas os caminhos de arquivo e os
comandos abaixo (`export`, `openssl`) são de shell Unix — adapte para
PowerShell (`$env:VAR = "valor"`) se for esse o caso.

## 2. Criar o banco de dados

Com o PostgreSQL rodando localmente:

```bash
psql -U postgres -c "CREATE USER boraif WITH PASSWORD 'troque-esta-senha';"
psql -U postgres -c "CREATE DATABASE boraif OWNER boraif;"
```

Você **não** precisa criar tabelas manualmente — o backend faz isso
sozinho na primeira vez que inicia (migrations automáticas).

## 3. Configurar e rodar o backend

```bash
cd backend
```

### 3.1 Variáveis de ambiente

O backend lê configuração só de variáveis de ambiente (não lê arquivo
`.env` sozinho fora do Docker Compose — exporte as variáveis no seu
shell, ou use um gerenciador como [direnv](https://direnv.net/)).

Variáveis **obrigatórias**:

```bash
export DATABASE_URL="postgres://boraif:troque-esta-senha@localhost:5432/boraif?sslmode=disable"
export API_KEY_ENCRYPTION_SECRET="$(openssl rand -base64 32)"
```

⚠️ Se você rodar o backend de novo mais tarde numa sessão de terminal
diferente sem exportar a **mesma** `API_KEY_ENCRYPTION_SECRET` de antes,
qualquer API Key da OpenAI já cadastrada por um professor fica
impossível de descriptografar. Anote o valor gerado (ex.: num arquivo
`.env.local` que você mesmo carrega com `export $(cat .env.local | xargs)`
ou similar) em vez de gerar um novo a cada vez.

Variáveis **opcionais** (mostrando os valores padrão):

```bash
export BACKEND_PORT=8080
export SESSION_COOKIE_NAME=boraif_session
export COOKIE_SECURE=false          # true só em produção com HTTPS
export UPLOADS_DIR=uploads          # criado automaticamente, relativo ao diretório de trabalho
export MAX_UPLOAD_SIZE_MB=5
export OPENAI_MODEL=gpt-4o-mini
export GENERATED_DIR=generated      # criado automaticamente
export CHROME_PATH=/usr/bin/chromium   # ajuste para o caminho real do seu Chromium/Chrome
```

No macOS com Chrome instalado via `.app`, por exemplo:

```bash
export CHROME_PATH="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
```

### 3.2 Criar o primeiro administrador

```bash
go run ./cmd/server create-admin \
  --name="Seu Nome" \
  --email="admin@escola.example" \
  --password="escolha-uma-senha-forte"
```

Isso já aplica as migrations automaticamente antes de criar o usuário (não
precisa rodar nada antes).

### 3.3 Subir o backend

```bash
go run ./cmd/server
```

Ou, para compilar um binário e rodá-lo depois (mais rápido em execuções
subsequentes):

```bash
go build -o server ./cmd/server
./server
```

O backend fica escutando em `http://localhost:8080` (ou na porta que você
definiu em `BACKEND_PORT`). Teste com:

```bash
curl http://localhost:8080/api/health
# deve responder {"status":"ok"}
```

### 3.4 Rodar os testes (opcional, mas recomendado)

```bash
go test ./...
```

## 4. Configurar e rodar o frontend

Em outro terminal:

```bash
cd frontend
npm install
```

### 4.1 Modo desenvolvimento

```bash
npm run dev
```

Isso sobe o Vite em `http://localhost:5173`, com um proxy embutido que
encaminha `/api/*` para `http://localhost:8080` (configurado em
`vite.config.ts`) — **o backend precisa estar rodando na porta 8080**
para o frontend funcionar (ou ajuste o `target` do proxy em
`vite.config.ts` se o backend estiver noutra porta).

### 4.2 Modo produção (build estático)

```bash
npm run build
```

Isso gera a pasta `frontend/dist/` com HTML/CSS/JS estático. Sirva essa
pasta com qualquer servidor de arquivos estáticos, por exemplo:

```bash
npx serve dist -p 5173
```

Nesse caso, como não há mais o proxy automático do Vite, você precisa que
as requisições para `/api/*` cheguem até o backend de alguma forma — as
opções mais simples são:

- rodar o servidor estático atrás de um Nginx configurado com o mesmo
  proxy usado em `frontend/nginx.conf` (o mesmo arquivo que o Dockerfile
  do frontend usa), ou
- rodar o backend na mesma origem/porta usando um proxy reverso de sua
  escolha (Caddy, Nginx, Traefik).

## 5. Onde ficam os arquivos gerados

Relativos ao diretório de onde você rodou o backend (`backend/`, se você
seguiu os comandos acima ao pé da letra):

- `backend/uploads/` — imagens enviadas pelos professores.
- `backend/generated/` — PDFs de prova gerados.

Ajuste `UPLOADS_DIR`/`GENERATED_DIR` se quiser esses diretórios em outro
lugar.

## 6. Backup e restauração do banco (sem Docker)

```bash
# backup
pg_dump -U boraif boraif > backup_$(date +%Y%m%d).sql

# restaurar (em um banco vazio, para evitar conflito de chaves)
psql -U boraif -d boraif < backup_20260101.sql
```

## 7. Rodando tudo de novo depois de reiniciar o computador

1. Garanta que o PostgreSQL está rodando (`brew services start
   postgresql@16` ou `sudo systemctl start postgresql`).
2. Exporte de novo as variáveis de ambiente do passo 3.1 (elas não
   persistem entre sessões de terminal, a não ser que você as tenha posto
   no seu `.bashrc`/`.zshrc` ou em um arquivo carregado por `direnv`).
3. Rode `go run ./cmd/server` (backend) e `npm run dev` (frontend).

## 8. Diferenças em relação a rodar com Docker

- Você é responsável por manter o PostgreSQL, o Chromium e as versões
  corretas de Go/Node instaladas e atualizadas.
- As variáveis de ambiente precisam ser exportadas manualmente a cada
  sessão (ou automatizadas com `direnv`/scripts próprios) — não existe um
  `.env` lido automaticamente fora do Docker Compose.
- Não há isolamento entre o BoraIF e o resto do que roda na sua máquina.
- A porta e o caminho do Chromium variam por sistema operacional — ajuste
  `CHROME_PATH` conforme necessário.

Para o restante (criar assuntos, cadastrar API Key da OpenAI, gerar
provas, etc.), o comportamento é idêntico ao descrito em
`RUNNINGDOCKER.MD` e `DEVELOPMENT.md` — só a forma de subir os processos
muda.
