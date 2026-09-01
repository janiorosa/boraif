# Como rodar o BoraIF com Docker

Este é o guia completo e definitivo para colocar o BoraIF no ar usando
Docker — o jeito recomendado de rodar o projeto. Se você não pode ou não
quer usar Docker, veja `withoutdocker.md`.

## 1. Pré-requisitos

- **Docker** e **Docker Compose** instalados (Docker Desktop no
  Mac/Windows já inclui os dois; no Linux, instale `docker` e o plugin
  `docker-compose-plugin`).
- **Nada mais.** Go, Node.js, PostgreSQL e Chromium rodam *dentro* dos
  containers — você não precisa instalar nenhum deles na sua máquina.
- Acesso à internet na primeira subida (para baixar as imagens base do
  Docker e as dependências de Go/npm) e, depois, sempre que for gerar um
  PDF (o KaTeX, usado para renderizar fórmulas matemáticas, é carregado de
  uma CDN no momento da geração — veja a nota no fim deste documento).
- Portas livres na sua máquina: `5173` (frontend), `8080` (backend) e
  `5433` (Postgres) — ou ajuste as portas no `.env` se algo já as estiver
  usando.

## 2. Clonar e entrar na pasta do projeto

```bash
git clone <url-do-repositorio> boraif
cd boraif
```

## 3. Criar o arquivo `.env`

```bash
cp .env.example .env
```

Abra o `.env` num editor de texto e preencha **obrigatoriamente** estes
dois valores (o resto já vem com um padrão razoável para desenvolvimento):

### 3.1 `POSTGRES_PASSWORD`

Qualquer senha — só não deixe em branco nem use a que vem de exemplo em
produção.

### 3.2 `API_KEY_ENCRYPTION_SECRET` (obrigatória, sem valor padrão)

Essa chave cifra (AES-256-GCM) a API Key da OpenAI que cada professor
cadastra depois. **O backend se recusa a iniciar sem ela** — nem para
subir o servidor, nem para rodar o comando de criar o primeiro
administrador.

Gere uma com:

```bash
openssl rand -base64 32
```

Copie o resultado (uma linha só, algo como
`k3f9s02JFls9adKJasdf0912+asdKLJlkj39asdfLKJ==`) e cole no `.env`:

```
API_KEY_ENCRYPTION_SECRET=k3f9s02JFls9adKJasdf0912+asdKLJlkj39asdfLKJ==
```

**Guarde essa chave em lugar seguro** (um cofre de senhas, um secret
manager, etc.), fora do controle de versão. Se você trocá-la depois de já
ter professores com chaves da OpenAI cadastradas, aquelas chaves ficam
permanentemente impossíveis de descriptografar — os professores
precisariam cadastrar a chave de novo.

Se você não tem `openssl` disponível, qualquer forma de gerar 32 bytes
aleatórios e codificá-los em base64 serve, por exemplo:

```bash
python3 -c "import secrets, base64; print(base64.b64encode(secrets.token_bytes(32)).decode())"
```

### 3.3 Demais variáveis (opcional alterar)

| Variável | Padrão | Para que serve |
|---|---|---|
| `POSTGRES_DB` | `boraif` | nome do banco |
| `POSTGRES_USER` | `boraif` | usuário do Postgres |
| `POSTGRES_PORT` | `5433` | porta do Postgres exposta no seu host |
| `BACKEND_PORT` | `8080` | porta do backend exposta no seu host |
| `FRONTEND_PORT` | `5173` | porta do frontend exposta no seu host |
| `SESSION_COOKIE_NAME` | `boraif_session` | nome do cookie de sessão |
| `COOKIE_SECURE` | `false` | mude para `true` em produção com HTTPS |
| `MAX_UPLOAD_SIZE_MB` | `5` | tamanho máximo de cada imagem enviada |
| `OPENAI_MODEL` | `gpt-4o-mini` | modelo usado no assistente de revisão por IA |
| `UPLOADS_DIR` / `GENERATED_DIR` / `CHROME_PATH` | já corretos | só mude se você souber o que está fazendo |

## 4. Subir os containers

```bash
docker compose up --build
```

Na primeira vez isso demora alguns minutos: o Docker baixa as imagens
base (Go, Node, Postgres, Debian+Chromium), o backend baixa suas
dependências Go (`go mod tidy` roda dentro do build), e o frontend baixa
suas dependências npm (incluindo TipTap e KaTeX).

O que acontece ao subir:

- O container `postgres` sobe e cria o volume de dados (se ainda não
  existir).
- O container `backend` espera o Postgres ficar saudável, **executa as
  migrations automaticamente** (cria todas as tabelas e os dados de
  catálogo — disciplinas, anos, dificuldades, status) e começa a escutar
  na porta 8080.
- O container `frontend` sobe um Nginx servindo o React já compilado, e
  encaminha `/api/*` para o backend.

Quando parar de rolar log e aparecer algo como
`boraif backend listening on :8080`, está no ar.

Para rodar em segundo plano (sem prender o terminal):

```bash
docker compose up --build -d
```

E para ver os logs depois:

```bash
docker compose logs -f backend
docker compose logs -f frontend
docker compose logs -f postgres
```

## 5. Acessar a aplicação

- **Frontend (interface)**: http://localhost:5173
- **Backend (API)**: http://localhost:8080/api/health — deve responder
  `{"status":"ok"}`
- **Postgres**: `localhost:5433` — para inspecionar com um cliente SQL
  (DBeaver, TablePlus, `psql`, etc.), usando o usuário/senha/banco do
  `.env`.

### 5.1 Acessando de outra máquina na mesma rede local (LAN)

Se o Docker roda num servidor e você acessa esse servidor remotamente (por
SSH, por exemplo), mas quer abrir o BoraIF no navegador de outra máquina
na mesma rede — usando o IP do servidor em vez de `localhost` — não é
preciso instalar nada a mais: o `docker-compose.yml` já publica o
frontend e o backend em `0.0.0.0` (todas as interfaces de rede do
servidor), não só em `localhost`/`127.0.0.1`.

1. Descubra o IP do servidor Docker na rede local (rode isto **no
   servidor**, não na sua máquina):

   ```bash
   ip addr show | grep "inet " | grep -v 127.0.0.1
   # ou, em muitas distros:
   hostname -I
   ```

   Vai aparecer algo como `192.168.1.200`.

2. Na sua máquina (a que está longe do servidor, mas na mesma rede),
   acesse:

   ```
   http://192.168.1.200:5173
   ```

   (troque `192.168.1.200` pelo IP real do servidor, e `5173` pela porta
   que você configurou em `FRONTEND_PORT`, se tiver mudado o padrão).

3. **Mantenha `COOKIE_SECURE=false`** no `.env` do servidor — você está
   acessando por HTTP puro (sem HTTPS), então um cookie de sessão marcado
   como "Secure" seria recusado pelo navegador e o login pareceria não
   funcionar (o formulário aceita a senha, mas a sessão nunca gruda). Isso
   já é o padrão do `.env.example`; só não mude para `true` a menos que
   você tenha colocado um HTTPS de verdade na frente (ex.: um proxy
   reverso com certificado).

4. Se mesmo assim não conectar, o problema mais comum é o **firewall do
   próprio servidor** (não do Docker) bloqueando a porta. O Docker
   configura suas próprias regras de rede para publicar a porta, mas se o
   servidor tiver `ufw`, `firewalld` ou um security group de nuvem
   (AWS/GCP/Azure) ativo, você também precisa liberar a porta ali. Exemplo
   com `ufw` no Linux:

   ```bash
   sudo ufw allow 5173/tcp
   sudo ufw allow 8080/tcp
   ```

   Depois, confirme que a porta está realmente escutando em todas as
   interfaces (não só localhost) rodando isto **no servidor**:

   ```bash
   ss -tlnp | grep -E ':5173|:8080'
   ```

   Você deve ver `0.0.0.0:5173` e `0.0.0.0:8080` na saída — se aparecer
   `127.0.0.1:5173` em vez disso, o `docker-compose.yml` não foi
   atualizado com as portas explícitas `0.0.0.0:...` (confira se você está
   na versão mais recente do arquivo).

O Postgres continua publicado só em `127.0.0.1` do servidor (não na LAN),
de propósito — o backend fala com ele pela rede interna do Docker Compose,
e não há motivo para expor um banco de dados para a rede inteira. Se você
realmente precisar conectar um cliente SQL direto da sua máquina remota
(não do servidor), edite o `docker-compose.yml` e troque
`"127.0.0.1:${POSTGRES_PORT:-5433}:5432"` por
`"0.0.0.0:${POSTGRES_PORT:-5433}:5432"` — e considere colocar uma senha
forte e/ou uma regra de firewall restringindo quem pode alcançar essa
porta, já que ficaria exposta à rede local inteira.

## 6. Criar o primeiro administrador

O banco **não vem com nenhum usuário pré-cadastrado** (isso seria um
risco de segurança). Crie o seu com o próprio binário do backend, dentro
do container já em execução:

```bash
docker compose exec backend /app/server create-admin \
  --name="Seu Nome" \
  --email="admin@escola.example" \
  --password="escolha-uma-senha-forte"
```

Se der certo, aparece algo como
`administrador criado com id=1, email=admin@escola.example`.

Agora entre em http://localhost:5173 com esse e-mail e senha.

## 7. Primeiros passos depois de logado (como ADMIN)

1. **Cadastre professores** em **Usuários** (`/admin/usuarios`): cada
   ELABORADOR precisa de uma disciplina; ADMIN e GESTOR não têm disciplina.
2. As **13 disciplinas** do Ensino Médio já vêm prontas (seed automático).
3. **Assuntos** são criados pelos próprios professores (ou por você) em
   **Assuntos**, dentro da disciplina de cada um.
4. Cada ELABORADOR cadastra sua própria **API Key da OpenAI** em
   **Minha Conta**, se quiser usar o assistente de revisão por IA.
5. Um GESTOR (ou você mesmo como ADMIN) cria as **Aplicações** e
   **Cadernos** quando chegar a hora de montar uma prova.

Veja `DEVELOPMENT.md` para o passo a passo detalhado de cada tela.

## 8. Rebuild, atualização e reinício — **sem perder o banco de dados**

Esta é a regra mais importante de operação do projeto: **o PostgreSQL vive
num volume Docker nomeado (`postgres_data`), independente do ciclo de vida
dos containers `frontend` e `backend`.**

### Reiniciar frontend ou backend (sem rebuildar)

```bash
docker compose restart backend
docker compose restart frontend
```

### Atualizar o código do backend (depois de um `git pull`, por exemplo)

```bash
git pull
docker compose build backend
docker compose up -d backend
```

O volume do Postgres não é tocado. Se a atualização trouxe migrations
novas, elas rodam automaticamente na inicialização do backend.

### Atualizar o código do frontend

```bash
git pull
docker compose build frontend
docker compose up -d frontend
```

### Recriar backend e frontend do zero (sem tocar no banco)

```bash
docker compose up -d --build --force-recreate backend frontend
```

Repare que `postgres` **não** está nessa lista — o comando não recria o
container do banco, e mesmo que recriasse, o volume nomeado permaneceria
intacto.

### ⚠️ O que NUNCA fazer numa atualização normal

```bash
docker compose down -v      # -v REMOVE o volume do Postgres — só rode isso
                             # de propósito, para apagar tudo do zero.
```

Comandos como `docker compose down` (sem `-v`), `docker compose up
--build` e `docker compose restart` **não apagam** os dados do Postgres.
Só o `-v` apaga.

## 9. Backup do PostgreSQL

```bash
docker compose exec -T postgres pg_dump -U boraif boraif > backup_$(date +%Y%m%d).sql
```

(troque `boraif boraif` por `${POSTGRES_USER} ${POSTGRES_DB}` se você
alterou esses valores no `.env`.)

Guarde o arquivo `.sql` gerado em lugar seguro (fora do repositório git).

## 10. Restaurar o PostgreSQL

Para restaurar em um banco **vazio** (recomendado, evita conflitos de
chave):

```bash
docker compose down
docker volume rm boraif_postgres_data   # ATENÇÃO: apaga o banco atual
docker compose up -d postgres
# espere alguns segundos o Postgres ficar pronto, depois:
cat backup_20260101.sql | docker compose exec -T postgres psql -U boraif -d boraif
docker compose up -d
```

O nome exato do volume pode variar conforme o nome da pasta do projeto —
confira com `docker volume ls` se o comando acima não encontrar
`boraif_postgres_data`.

## 11. Parar tudo

```bash
docker compose down
```

Isso para e remove os containers, mas **mantém o volume do Postgres** e
os arquivos em `./uploads` e `./generated` no seu disco (eles são bind
mounts do host, não voláteis).

## 12. Onde ficam os arquivos gerados pela aplicação

- **Imagens enviadas pelos professores**: pasta `./uploads/` no seu
  computador (fora dos containers) — sobrevive a qualquer rebuild.
- **PDFs de prova gerados**: pasta `./generated/` no seu computador —
  também sobrevive a qualquer rebuild.

## 13. Solução de problemas comuns

- **`docker compose up` recusa subir e reclama de
  `API_KEY_ENCRYPTION_SECRET`**: você esqueceu de preencher essa variável
  no `.env` (veja o passo 3.2).
- **Backend reinicia em loop / logs mostram "config error"**: confira se
  todas as variáveis obrigatórias (`POSTGRES_PASSWORD`,
  `API_KEY_ENCRYPTION_SECRET`) estão no `.env` e se o arquivo `.env` está
  na raiz do projeto (mesmo nível do `docker-compose.yml`).
- **Login não funciona / cookie não gruda**: se você estiver acessando por
  HTTPS com um domínio de verdade, mude `COOKIE_SECURE=true`; em
  `localhost` sem HTTPS, mantenha `false`.
- **Geração de PDF fica com fórmulas em branco**: o container do backend
  provavelmente não tem acesso de saída à internet no momento da geração
  (o KaTeX é carregado de uma CDN nesse momento — veja `DEVELOPMENT.md`,
  seção "Fase 10").
- **Porta já em uso** (`bind: address already in use`): outra aplicação já
  está usando `5173`, `8080` ou `5433` no seu computador — mude
  `FRONTEND_PORT`, `BACKEND_PORT` ou `POSTGRES_PORT` no `.env`.
- **Quero ver o que está acontecendo com mais detalhe**:
  `docker compose logs -f backend` mostra cada requisição e erro do
  backend, incluindo o progresso da geração de PDF.
