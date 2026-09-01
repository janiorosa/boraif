# BoraIF — Diário de desenvolvimento

> Este arquivo era o `README.md` original do projeto (fase a fase, desde a
> Fase 1). Ele foi renomeado para `DEVELOPMENT.md` para abrir espaço para um
> `README.md` novo, curto e de alto nível — comece por ele. Este documento
> aqui é o histórico técnico detalhado: toda decisão de arquitetura, todo
> endpoint, todo bug encontrado e corrigido nas revisões, fase por fase.
> Para as outras referências (rodar com/sem Docker, arquitetura resumida,
> banco de dados, visão geral do produto), veja `README.md`,
> `RUNNINGDOCKER.MD`, `running.md`, `withoutdocker.md`, `architecture.md`,
> `database.md` e `project.md` na raiz do projeto.

Sistema de elaboração, organização, revisão e reutilização de questões para
montagem de cadernos de prova do Ensino Médio brasileiro.

Este documento é atualizado a cada fase de implementação. Estado atual:
**Fase 11 — Refinamento**, sobre a base de todas as fases anteriores:
Fundação, Usuários e permissões, Assuntos, CRUD estrutural de questões,
Editor TipTap, Autosave, Biblioteca de imagens, Assistente de IA,
Aplicações e cadernos, e Geração de PDF. **Todas as 11 fases da
especificação foram implementadas.**

## Arquitetura em uma frase

Monólito modular: **3 containers** (`frontend`, `backend`, `postgres`),
sem microservices, sem fila de mensagens, sem Redis. Ver
`especificacoes.md` para o documento completo de requisitos.

## Pré-requisitos

- Docker e Docker Compose
- Nada além disso é necessário na sua máquina: Go, Node e Chromium rodam
  dentro dos containers.

## 1. Clonar e configurar

```bash
git clone <url-do-repositorio> boraif
cd boraif
cp .env.example .env
```

Abra o `.env` e ajuste:

- `POSTGRES_PASSWORD` — obrigatório.
- `API_KEY_ENCRYPTION_SECRET` — **obrigatório, sem valor padrão**; cifra a
  API Key da OpenAI de cada professor (seção 17). Sem ele, o backend nem
  inicia (nenhum comando, nem `create-admin`). Gere o seu com:

  ```bash
  openssl rand -base64 32
  ```

  Guarde essa chave com cuidado: trocá-la depois torna as API Keys já
  salvas no banco indescriptografáveis.

Os demais valores padrão servem para desenvolvimento local.

## 2. Subir o sistema

```bash
docker compose up --build
```

Isso sobe os três containers. O backend, ao iniciar, **executa as
migrations automaticamente** antes de aceitar requisições — não é preciso
rodar nenhum comando manual de migration/seed em uma instalação nova.

- Frontend: http://localhost:5173
- Backend (API): http://localhost:8080/api/health
- PostgreSQL: localhost:5432 (útil para inspecionar com um cliente SQL)

## 3. Criar o primeiro administrador

O banco nunca vem com um usuário/senha padrão pré-cadastrado (risco de
segurança). Crie o admin com o próprio binário do backend, dentro do
container:

```bash
docker compose exec backend /app/server create-admin \
  --name="Seu Nome" \
  --email="admin@escola.example" \
  --password="escolha-uma-senha-forte"
```

Depois disso, faça login em http://localhost:5173 com esse e-mail/senha.

## 4. Cadastrar disciplinas, usuários e assuntos

- **Disciplinas**: as 13 disciplinas do Ensino Médio já vêm pré-cadastradas
  via seed (`backend/migrations/0001_init.sql`). Não há tela de cadastro de
  disciplinas — a lista é fixa.
- **Usuários (elaboradores/gestores)**: logado como ADMIN, acesse
  **Usuários** no menu (`/admin/usuarios`) para criar/editar. Um
  ELABORADOR precisa de uma disciplina associada; ADMIN e GESTOR não têm
  disciplina.
- **Assuntos**: acesse **Assuntos** no menu (`/assuntos`), disponível para
  ADMIN e ELABORADOR (não para GESTOR). ADMIN vê/filtra por qualquer
  disciplina e pode editar/excluir; ELABORADOR só cria e vê assuntos da
  própria disciplina. Ao criar, se já existir um assunto de nome igual ou
  parecido na disciplina, o sistema pede confirmação antes de criar mesmo
  assim (seção 14).

## 4.1 Criar e editar questões

Acesse **Questões** no menu (`/questoes`), disponível para ADMIN e
ELABORADOR. ELABORADOR só vê/edita questões da própria disciplina (mas de
qualquer autor dela — seção 15); ADMIN vê todas e filtra por disciplina.

Em **Nova questão**, escolha assunto/ano/dificuldade e a questão já é
criada como RASCUNHO — sem precisar preencher tudo antes de começar a
trabalhar nela (seção 37). Você cai direto na tela de edição, com:

- metadados editáveis (assunto, ano, dificuldade, status);
- enunciado e comando como editores TipTap independentes (seção 6/9);
- as cinco alternativas A–E, cada uma com seu próprio editor TipTap, e um
  seletor (rádio) indicando qual é a correta — o sistema sempre exige
  exatamente uma marcada (seção 7);
- um botão **Salvar** manual, que convive com o autosave (seção 18): a
  questão inteira (metadados + enunciado + comando + 5 alternativas) é
  salva automaticamente 2 segundos depois da última edição em qualquer
  campo, nunca a cada tecla, com indicação de estado (`Salvando...` /
  `Salvo` / `Erro ao salvar`). Duas requisições nunca correm em paralelo:
  se uma edição nova chega enquanto um save anterior ainda está em voo, ela
  espera terminar e salva de novo em seguida, sempre com o conteúdo mais
  recente.

Cada um dos sete editores tem a mesma barra de ferramentas (seção 10):
negrito, itálico, sublinhado, tachado, subscrito, sobrescrito, listas,
alinhamento, link, imagem, tabela e fórmula matemática — sem virar um
processador de texto completo.

### Imagens no editor

O botão de imagem abre um seletor com duas abas: **Enviar nova**
(PNG/JPEG/GIF/WEBP, até `MAX_UPLOAD_SIZE_MB`) e **Escolher da biblioteca**
— uma grade com busca por nome do arquivo, mostrando as imagens que
qualquer professor da disciplina já enviou (seção 13). As duas inserem
direto na questão. A mesma biblioteca também tem uma tela própria — seção
5.1 "Biblioteca de imagens" abaixo.

### Fórmulas matemáticas

O botão **∑** abre um painel com os dois caminhos da seção 11:

- **visual**: botões para fração, raiz, expoente, índice, letras gregas,
  soma, integral etc., que montam o LaTeX por trás sem o professor
  precisar conhecê-lo;
- **LaTeX direto**: uma caixa de texto para quem já sabe LaTeX, com preview
  ao vivo renderizado pelo KaTeX antes de inserir.

A fórmula é renderizada no editor pelo KaTeX (seção 12). O mesmo HTML
gerado a partir do LaTeX será reaproveitado na geração do PDF na Fase 10 —
nenhuma fórmula é gerada como PDF individualmente.

## 5. Cadastrar a API Key da OpenAI

Cada professor cadastra a própria chave (seção 17 — nunca uma chave
global). Logado como ELABORADOR, acesse **Minha Conta** no menu
(`/minha-conta`) e cole a chave (formato `sk-...`). Ela é cifrada
(AES-256-GCM) antes de ir para o banco e **nunca é exibida de novo** depois
de salva — a tela só mostra "configurada" ou "não configurada".

Sem uma chave cadastrada, os botões de revisão por IA no editor de questões
respondem com um erro claro pedindo para configurá-la primeiro.

## 5.1 Assistente de revisão por IA

Na tela de edição de questão (`/questoes/{id}`), abaixo das alternativas,
há 4 botões: **Revisar enunciado**, **Revisar comando**, **Revisar
alternativas** e **Revisar questão inteira** (seção 16). Cada um chama
`POST /api/questions/{id}/ai/review` com o texto simples do campo (extraído
do editor via `editor.getText()`) e devolve um resumo + listas de
problemas/sugestões. **A IA nunca altera a questão** — o professor lê e
decide o que aplicar manualmente.

## 5.2 Biblioteca de imagens

Acesse **Imagens** no menu (`/imagens`), disponível para ADMIN e
ELABORADOR. ELABORADOR vê a biblioteca da própria disciplina; ADMIN
escolhe a disciplina num seletor. A tela tem busca por nome do arquivo, um
botão para enviar imagem direto por ali (sem precisar estar dentro de uma
questão) e a grade de imagens — a mesma fonte de dados usada pelo seletor
"Escolher da biblioteca" dentro do editor de questões (seção 4.1).

## 5.3 Aplicações e cadernos

Acesse **Aplicações** no menu (`/aplicacoes`), disponível para ADMIN e
GESTOR (seção 20 — ELABORADOR não participa da geração de provas).

1. Crie uma aplicação (ex.: "2026/1").
2. Na tela da aplicação, adicione um ou mais **cadernos** (seção 21.1 —
   o mais comum são 2, mas pode ser 1, 3 ou mais). Cada caderno já nasce
   com a configuração padrão copiada (seção 22 — ver abaixo).
3. Abra um caderno para configurar: total de questões, anos, e cotas por
   disciplina (opcionalmente refinadas por assunto e/ou dificuldade —
   seção 23). A soma das cotas precisa bater com o total.
4. Clique em **Verificar disponibilidade** (seção 24) antes de pensar em
   gerar a prova — mostra, cota por cota, quantas questões elegíveis
   existem de fato hoje.

A **configuração padrão** (`/configuracao-padrao`, só ADMIN) é o modelo
copiado para cada caderno novo (seção 22); alterá-la não muda cadernos já
criados.

## 5.4 Geração de PDF

Na mesma tela do caderno (`/cadernos/{id}`), abaixo da disponibilidade,
clique em **Gerar PDF**:

1. Se as questões do caderno ainda não foram selecionadas, o backend
   seleciona aleatoriamente as questões elegíveis de cada cota (seção 25),
   congela essa seleção (seção 26) e trava a configuração (seção 27, campo
   `is_frozen`) — tudo isso antes de montar o PDF.
2. A geração roda em background (seção 30): a tela mostra o status
   evoluindo de "Na fila" → "Gerando..." → "Concluído" (ou "Falhou", com o
   motivo), consultando o servidor a cada poucos segundos só enquanto
   houver algo em andamento.
3. Quando concluído, aparece um link **Baixar PDF**.

Gerar de novo um caderno já congelado não sorteia outras questões — reusa
o mesmo snapshot e só refaz o PDF (útil se a primeira tentativa falhar por
algum motivo transitório). Cada geração fica registrada no histórico da
tela, então dá para ver tentativas anteriores e seus erros.

**Nota**: a renderização de fórmulas no PDF carrega o KaTeX de uma CDN no
momento da geração — o container do backend precisa de acesso de saída à
internet nesse momento. Sem rede, o PDF ainda é gerado, só que com as
fórmulas em branco.

## Rebuild, atualização e reinício — **sem perder o banco de dados**

Esta é a regra mais importante do projeto: **o PostgreSQL vive em um volume
Docker nomeado (`postgres_data`), independente do ciclo de vida dos
containers `frontend` e `backend`.**

### Reiniciar frontend/backend

```bash
docker compose restart backend
docker compose restart frontend
```

### Atualizar o backend (nova versão do código)

```bash
git pull
docker compose build backend
docker compose up -d backend
```

O volume do Postgres não é tocado. As migrations novas (se houver) rodam
automaticamente na inicialização do backend.

### Atualizar o frontend

```bash
git pull
docker compose build frontend
docker compose up -d frontend
```

### Recriar todos os containers da aplicação

```bash
docker compose up -d --build --force-recreate backend frontend
```

Note que **`postgres` não está na lista** — o comando acima não recria o
container do banco, e mesmo que recriasse, o volume nomeado permanece.

### O que NUNCA fazer em uma atualização normal

```bash
docker compose down -v      # -v REMOVE o volume do Postgres — só use isso
                             # intencionalmente para apagar tudo do zero.
```

Comandos como `docker compose down` (sem `-v`), `docker compose up --build`
e `docker compose restart` **não apagam** os dados do Postgres.

## Backup do PostgreSQL

```bash
docker compose exec -T postgres pg_dump -U ${POSTGRES_USER} ${POSTGRES_DB} > backup_$(date +%Y%m%d).sql
```

(ajuste `${POSTGRES_USER}`/`${POSTGRES_DB}` pelos valores do seu `.env`, ou
exporte as variáveis antes de rodar o comando).

## Restaurar o PostgreSQL

```bash
cat backup_20260101.sql | docker compose exec -T postgres psql -U ${POSTGRES_USER} -d ${POSTGRES_DB}
```

Restaurar sobre um banco com dados existentes pode gerar conflitos de chave
— restaure preferencialmente em um banco recém-criado (`docker compose down`
+ `docker volume rm boraif_postgres_data` + `docker compose up -d postgres`
+ restauração + subir o restante).

## Criar o banco do zero em outro ambiente

Não é necessário nenhum passo manual: suba os containers normalmente
(seção "Subir o sistema"). O backend cria o schema completo e os dados de
catálogo (disciplinas, anos, dificuldades, status) na primeira execução,
via `backend/migrations/0001_init.sql` (formato
[goose](https://github.com/pressly/goose)).

## Onde ficam os arquivos

- **Imagens enviadas pelos professores**: `./uploads/images/{código da
  disciplina}/{nome aleatório}.{ext}` no host (bind mount para
  `/app/uploads` no container `backend`), servidas publicamente em
  `/uploads/images/...`. Upload, busca e reutilização por disciplina
  (tela `/imagens` e seletor dentro do editor) — seções 4.1 e 5.1.
- **PDFs gerados**: `./generated/applications/` no host (bind mount para
  `/app/generated`). Recurso da Fase 10.

Ambos os diretórios sobrevivem a rebuild/restart dos containers, pois são
bind mounts para o host, não armazenamento interno da imagem.

## Estrutura do projeto

```
boraif/
├── backend/           # Go — API REST, migrations, auth, (futuro: PDF/IA)
│   ├── cmd/server/    # binário principal (serve | create-admin)
│   ├── internal/      # código por domínio (auth, users, config, db, httpserver, ...)
│   └── migrations/    # SQL versionado (goose), embutido no binário
├── frontend/          # React + TypeScript (Vite), servido por Nginx em produção
├── uploads/           # imagens enviadas (bind mount)
├── generated/         # PDFs gerados (bind mount)
├── docker-compose.yml
├── .env.example
└── especificacoes.md  # documento de requisitos completo
```

## Documentação da API (estado atual)

| Método | Rota | Descrição | Autenticação |
|---|---|---|---|
| GET | `/api/health` | health check | não |
| POST | `/api/auth/login` | login (`{email, password}`), define cookie de sessão | não |
| POST | `/api/auth/logout` | encerra a sessão atual | sim |
| GET | `/api/auth/me` | usuário autenticado atual | sim |
| GET | `/api/users` | lista usuários | sim (ADMIN) |
| POST | `/api/users` | cria usuário | sim (ADMIN) |
| GET | `/api/users/{id}` | detalhe de um usuário | sim (ADMIN) |
| PUT | `/api/users/{id}` | atualiza dados e, opcionalmente, a senha | sim (ADMIN) |
| GET | `/api/disciplines` | lista as disciplinas (fixas, via seed) | sim (qualquer papel) |
| GET | `/api/subjects?disciplineId=` | lista assuntos, opcionalmente filtrados por disciplina | sim (qualquer papel) |
| POST | `/api/subjects` | cria assunto (ADMIN: qualquer disciplina; ELABORADOR: só a própria); `confirmDuplicate: true` força a criação mesmo havendo nome parecido | sim |
| PUT | `/api/subjects/{id}` | renomeia um assunto | sim (ADMIN) |
| DELETE | `/api/subjects/{id}` | exclui um assunto (falha com 409 se estiver em uso por questões) | sim (ADMIN) |
| GET | `/api/grade-years` | lista os anos (1º/2º/3º, fixos via seed) | sim (qualquer papel) |
| GET | `/api/difficulties` | lista as dificuldades (fixas via seed) | sim (qualquer papel) |
| GET | `/api/question-statuses` | lista os status de questão (fixos via seed) | sim (qualquer papel) |
| GET | `/api/questions?search=&subjectId=&gradeYearId=&difficultyId=&statusId=&authorId=&disciplineId=&page=&pageSize=&sortBy=&sortDir=` | lista paginada de questões (ELABORADOR: só a própria disciplina) | sim (ADMIN, ELABORADOR) |
| POST | `/api/questions` | cria questão com assunto/ano/dificuldade + enunciado/comando/5 alternativas; status inicial sempre RASCUNHO | sim (ADMIN, ELABORADOR) |
| GET | `/api/questions/{id}` | detalhe completo (metadados + enunciado + comando + alternativas) | sim (ADMIN, ELABORADOR da disciplina) |
| PUT | `/api/questions/{id}` | atualiza metadados editáveis + conteúdo + alternativas de uma vez (é o contrato usado tanto pelo botão Salvar quanto pelo autosave) | sim (ADMIN, ELABORADOR da disciplina) |
| DELETE | `/api/questions/{id}` | exclui a questão (alternativas em cascata; snapshots de cadernos que já a usaram são preservados) | sim (ADMIN, ELABORADOR da disciplina) |
| POST | `/api/images` | upload de imagem (`multipart/form-data`: `file` + `disciplineId`); devolve `{id, url}` | sim (ADMIN, ELABORADOR da disciplina) |
| GET | `/api/images?disciplineId=&search=&page=&pageSize=` | biblioteca de imagens da disciplina, mais recentes primeiro (ELABORADOR: só a própria disciplina) | sim (ADMIN, ELABORADOR) |
| GET | `/uploads/{caminho}` | serve o arquivo estático da imagem | não (nome imprevisível — seção 13) |
| PUT | `/api/me/openai-key` | cadastra/substitui a própria API Key da OpenAI (cifrada antes de gravar) | sim (ELABORADOR) |
| GET | `/api/me/openai-key/status` | informa só se há uma chave configurada (nunca a chave em si) | sim |
| POST | `/api/questions/{id}/ai/review` | análise por IA (`target`: statement/command/alternatives/full) usando a própria API Key do professor; devolve `{summary, issues, suggestions}` | sim (ADMIN, ELABORADOR da disciplina) |
| GET | `/api/applications` | lista aplicações | sim (ADMIN, GESTOR) |
| POST | `/api/applications` | cria aplicação; status inicial sempre RASCUNHO | sim (ADMIN, GESTOR) |
| GET | `/api/applications/{id}` | detalhe de uma aplicação | sim (ADMIN, GESTOR) |
| PUT | `/api/applications/{id}` | atualiza nome/descrição/status | sim (ADMIN, GESTOR) |
| GET | `/api/applications/{id}/booklets` | lista os cadernos da aplicação | sim (ADMIN, GESTOR) |
| POST | `/api/applications/{id}/booklets` | cria um caderno; configuração nasce copiada do padrão | sim (ADMIN, GESTOR) |
| GET | `/api/booklets/{id}` | detalhe de um caderno | sim (ADMIN, GESTOR) |
| GET | `/api/booklets/{id}/configuration` | configuração do caderno (total, anos, cotas) | sim (ADMIN, GESTOR) |
| PUT | `/api/booklets/{id}/configuration` | atualiza a configuração; recusa se já congelada, valida soma das cotas == total | sim (ADMIN, GESTOR) |
| GET | `/api/booklets/{id}/availability` | disponibilidade de questões elegíveis por cota (seção 24) | sim (ADMIN, GESTOR) |
| GET | `/api/default-configuration` | lê a configuração padrão (seção 22) | sim (ADMIN, GESTOR) |
| PUT | `/api/default-configuration` | grava a configuração padrão (upsert) | sim (ADMIN) |
| POST | `/api/booklets/{id}/generate` | dispara a geração do PDF em background (seleciona/congela na primeira vez); devolve `{id, status}` na hora | sim (ADMIN, GESTOR) |
| GET | `/api/booklets/{id}/generated-documents` | histórico de gerações do caderno, mais recentes primeiro | sim (ADMIN, GESTOR) |
| GET | `/api/generated-documents/{id}/file` | baixa o PDF já gerado (só quando `status = COMPLETED`) | sim (ADMIN, GESTOR) |

Todos os endpoints previstos na especificação foram implementados. Ajustes
futuros (se houver) serão registrados aqui.

## Decisões arquiteturais desta fase

- **Autenticação por sessão server-side** (cookie `httpOnly` com token
  aleatório; hash do token guardado em `sessions`), em vez de JWT — permite
  revogar uma sessão instantaneamente e evita gerenciar refresh tokens, sem
  precisar de Redis (Postgres já é obrigatório no projeto).
- **Roteamento HTTP com `net/http` puro** (Go 1.22+), sem framework externo.
- **Migrations com `goose`**, embutidas no binário via `embed.FS`, aplicadas
  automaticamente na subida do backend.
- **Catálogos como tabelas** (`disciplines`, `grade_years`, `difficulties`,
  `question_statuses`) em vez de strings soltas no código.
- Nenhum usuário/senha padrão é criado automaticamente — o primeiro admin é
  criado explicitamente via `create-admin` (evita credencial padrão exposta
  em produção).
- **Hash de senha isolado em `internal/security`**, separado de
  `internal/auth`, porque tanto `auth` quanto `users` precisam dele — colocá-lo
  dentro de `auth` criaria um import cycle (`auth` já importa `users`).
- **Helpers de JSON HTTP isolados em `internal/apiutil`** (`WriteJSON`,
  `WriteError`, `DecodeJSON`), fora de `internal/httpserver`: todo handler
  de domínio (users, subjects, questions, imagens, ...) precisa desses
  helpers, mas `httpserver` importa todos esses pacotes para montar as
  rotas — se os helpers ficassem em `httpserver`, qualquer handler que os
  usasse fecharia um import cycle. Mesmo raciocínio do ponto anterior,
  aplicado de novo na Fase 5 ao perceber o mesmo problema em escala maior.
- Sem CRUD de disciplinas: a lista é fixa (seed), só há leitura
  (`GET /api/disciplines`) para popular seletores.
- Autorização por papel é sempre validada no backend
  (`auth.RequireRole`) — nunca apenas escondendo botões no frontend
  (seção 20/35).
- **Detecção de assunto duplicado via `pg_trgm`** (extensão nativa do
  Postgres, sem novo serviço/infra): ao criar um assunto, o backend busca
  nomes iguais ou parecidos na mesma disciplina com `similarity()`; se
  achar algo, responde 409 com a lista de parecidos em vez de criar direto,
  e o frontend pede confirmação (`confirmDuplicate: true`) para seguir
  mesmo assim. Duplicata exata (case-insensitive) é sempre bloqueada por um
  índice único, mesmo com confirmação (seção 14).
- Criar/editar/excluir assunto tem regras de autorização diferentes:
  ADMIN faz tudo em qualquer disciplina; ELABORADOR só cria na própria
  disciplina; ninguém além de ADMIN edita ou exclui, pois um assunto é
  compartilhado entre todos os professores da disciplina (seção 14).
- **Uma aplicação pode ter mais de um caderno de prova** (ex.: Caderno 1 e
  Caderno 2, o mais comum, mas o número não é fixo). Cada caderno
  (`application_booklets`) tem sua própria configuração de seleção
  (`booklet_configurations`/`booklet_quota_rules`), seu próprio
  congelamento (seção 27 — congelar um caderno não afeta os demais da mesma
  aplicação) e seu próprio PDF (`generated_documents.booklet_id`). A
  numeração impressa das questões é por caderno: um caderno com 80 questões
  é numerado de 1 a 80, sem lacunas — a atribuição sequencial de posição
  fica a cargo do serviço de geração (Fase 10).
- **Enunciado/comando/alternativas ficam em `jsonb` opaco para o backend**:
  o Go nunca interpreta a estrutura ProseMirror, só valida que não é vazio
  e guarda/devolve como `json.RawMessage`. Isso manteve o conhecimento do
  formato do editor inteiramente no frontend (seção 8) — a Fase 5 trocou o
  editor (textarea → TipTap de verdade) sem nenhuma mudança no backend nem
  migração de dados; `prosemirror.ts` hoje só tem o documento vazio usado
  ao criar uma questão nova.
- **`revision_number` só sobe quando o conteúdo de fato muda** (seção
  5.4): o backend compara enunciado/comando/alternativas antes/depois numa
  transação (com `SELECT ... FOR UPDATE` para evitar corrida) e só
  incrementa se algo mudou de verdade — importante porque o autosave
  (Fase 6) chama o mesmo `PUT` reenviando o conteúdo inteiro a cada
  salvamento, mesmo quando nada mudou desde o último.
- **Regra das 5 alternativas validada duas vezes**: o handler rejeita com
  400 antes de tocar no banco (mensagem clara: quantas alternativas
  vieram, qual posição está repetida, quantas estão marcadas como
  corretas); a trigger de constraint do Postgres (seção 7) é a segunda
  camada, para o caso de outro código futuro escrever direto na tabela.
- **Busca textual via `::text ILIKE`** nos campos jsonb (`GET
  /api/questions?search=`): simples, sem extensão nova, mas sem índice
  (varredura sequencial) e sensível ao formato JSON, não só ao texto
  visível. Aceitável na escala atual (seção 2.3); se o volume de questões
  crescer muito, revisar para uma coluna de texto derivado indexada.
- **Disciplina da questão é imutável após a criação** — não é um requisito
  citado na especificação, então não foi exposta uma forma de "mover" uma
  questão de disciplina; só assunto/ano/dificuldade/status são editáveis
  depois de criada.
- **TipTap v3**, incluindo `@tiptap/extension-mathematics` (KaTeX) e
  `@tiptap/extension-table` — todas MIT/gratuitas (confirmado antes de
  adotar, seção "Editor" da spec). Sem heading (`StarterKit.configure({
  heading: false })`): questões de prova não precisam de títulos/seções, e
  menos botões mantém o editor simples (seção 10 — não virar um Word).
- **Upload de imagem é síncrono e simples** (seção 2.1/2.2): um único
  arquivo por requisição, tipo verificado pelos bytes reais
  (`http.DetectContentType`, nunca pela extensão do nome enviado pelo
  cliente), nome gerado no servidor (hex aleatório) — path traversal e
  disfarce de extensão ficam descartados por construção, não por
  validação de string.
- **Arquivo servido publicamente em `/uploads/`, sem exigir sessão**: a
  seção 13 pede explicitamente para não criar autorização complexa para
  imagens; o nome aleatório e imprevisível já é a proteção. Isso também
  simplifica a Fase 10 (Chromium não precisa de cookie de sessão para
  carregar as imagens ao gerar o PDF).
- **Fórmula: um único painel para os dois modos da seção 11** (visual e
  LaTeX) em vez de duas telas separadas — os botões de "entrada visual"
  só inserem os mesmos trechos de LaTeX que o modo direto aceita, então
  reaproveitar a mesma caixa de texto e o mesmo preview evita duplicar
  código de renderização.
- **Autosave com debounce de 2s e uma trava de "save em voo"** (seção 18):
  um `useEffect` reagenda um `setTimeout` a cada mudança em qualquer campo
  do conjunto (metadados + conteúdo + alternativas), cancelando o anterior
  — nunca uma requisição por tecla. Duas flags (`savingRef`/`pendingRef`)
  garantem que, se uma edição nova chega enquanto um PUT anterior ainda
  está em voo, ela só dispara depois que o primeiro termina, sempre lendo
  o estado mais atual via um ref espelhado a cada render (não os valores
  "congelados" do fechamento onde o timer foi criado).
- **Biblioteca de imagens sem endpoint próprio de detalhe**: `GET
  /api/images` já devolve tudo que a grade precisa (url, nome, tamanho,
  data); não foi criado um `GET /api/images/{id}` porque nada nesta fase
  precisa dele.
- **Um único componente de grade (`ImageGrid`) reaproveitado em dois
  lugares**: a página `/imagens` e o seletor "Escolher da biblioteca"
  dentro do editor. Evita manter duas implementações da mesma busca
  paginada com resultado visual idêntico.

### Dois bugs pegos na revisão final desta fase (antes de qualquer teste manual)

Como não há Go/Node/Docker neste ambiente para compilar, a revisão de
código é a única rede de segurança — vale registrar o que ela pegou:

- **Loop de autosave infinito**: a primeira versão do efeito de debounce
  incluía `question` nas dependências. Como `performSave` troca a
  referência de `question` a cada save bem-sucedido (para atualizar
  `revisionNumber`/`updatedAt`), isso reagendava outro autosave a cada
  ciclo — a questão ficaria salvando sozinha de 2 em 2 segundos para
  sempre, mesmo sem nenhuma edição nova. Corrigido removendo `question`
  das dependências (só os campos de conteúdo/metadados precisam estar lá).
- **`mountedRef` que ficava `false` para sempre em desenvolvimento**: o
  padrão `useRef(true)` + `useEffect(() => () => { ref.current = false })`
  parece certo, mas quebra sob o StrictMode do React: em dev, o React roda
  setup→cleanup→setup logo na primeira montagem para pegar efeitos mal
  limpos, e como o setup nunca reafirmava `ref.current = true`, o cleanup
  do meio deixava a flag presa em `false` — `performSave` nunca mais
  atualizaria a tela após um save, silenciosamente, só em desenvolvimento.
  Corrigido reafirmando `mountedRef.current = true` dentro do próprio
  setup do efeito.

### Fase 8 — Assistente de IA

- **AES-256-GCM com chave externa ao Postgres** (seção 17): a chave mestra
  de 32 bytes vem só de `API_KEY_ENCRYPTION_SECRET` (variável de ambiente);
  um dump do banco sozinho não permite decifrar nenhuma API Key. Cifração
  fica em `internal/security` (mesmo pacote do hash de senha — já é o lugar
  das primitivas sem dependência de domínio).
- **`API_KEY_ENCRYPTION_SECRET` obrigatório para qualquer inicialização**,
  não só para usar IA: `config.Load()` é uma função só, sem casos especiais
  por subcomando — `create-admin` também exige a variável. Na prática isso
  não trava ninguém: o `docker-compose.yml` já recusa subir o container
  sem ela (`${API_KEY_ENCRYPTION_SECRET:?...}`), então por definição o
  container só está de pé se a variável já existe.
- **Endpoints "minha conta" (`SetOwnOpenAIKey`/`OwnOpenAIKeyStatus`) vivem
  em `internal/auth`, não em `internal/users`**: `users` não pode importar
  `auth` (import cycle, `auth` já importa `users`), mas `auth` já tem
  `CurrentUser` e já importa `users`/`security` — mesma classe de problema
  das duas decisões de import cycle anteriores, resolvida do mesmo jeito
  (colocar a funcionalidade no pacote que já tem as dependências certas,
  em vez de forçar uma nova dependência que fecharia um ciclo).
- **Cliente OpenAI só com a stdlib** (`net/http`), sem SDK — é uma única
  chamada HTTP (Chat Completions com `response_format: json_object`), não
  justifica uma dependência inteira.
- **`ReviewResult` deliberadamente simples** (`summary` + `issues[]` +
  `suggestions[]`) em vez de um campo rígido por critério da seção 16 — os
  critérios (clareza, ambiguidade, pistas, adequação ao ano, etc.) entram
  no *prompt* de cada alvo (`statement`/`command`/`alternatives`/`full`),
  guiando a análise sem forçar a IA a preencher campos que não se aplicam
  a todo caso.
- **Texto extraído no frontend via `editor.getText()`**, nunca no backend:
  mantém a decisão da Fase 4/5 de o Go nunca interpretar a estrutura
  ProseMirror. `RichTextEditor` ganhou um `onEditorReady` para o pai poder
  chamar `getText()` sob demanda (só quando um botão de IA é clicado), sem
  duplicar esse texto em estado a cada tecla digitada.
- Modelo da OpenAI configurável via `OPENAI_MODEL` (padrão
  `gpt-4o-mini`) — evita hardcode caso a OpenAI descontinue o modelo.

### Fase 9 — Aplicações e cadernos

- **Configuração padrão como tabela singleton** (`default_configurations`
  + tabelas de anos/cotas associadas), com upsert em vez de uma constraint
  especial de unicidade: o repositório sempre lê "a primeira linha", cria
  se não existir, atualiza se existir. Simples e suficiente, já que só
  existe uma configuração padrão por definição (seção 22).
- **Cota é sempre uma linha "folha", nunca um resumo** (seção 23): decidi
  que uma linha de cota (disciplina + opcionalmente assunto/dificuldade) é
  sempre um critério final que soma ao total — nunca uma linha
  "Matemática: 10" coexistindo com linhas mais específicas
  "Matemática/Difícil: 2" por baixo dela. Isso elimina a ambiguidade de
  como validar "soma das cotas == total de questões" (seção 23): a soma é
  sempre de todas as linhas, sem precisar decidir quais são "resumo" e
  quais são "detalhe". Quem quiser Matemática dividida por dificuldade
  cria 3 linhas (Fácil/Média/Difícil) em vez de 1+3.
- **Verificação de disponibilidade (seção 24) por linha, independente**:
  cada cota é checada sozinha contra `count(*)` de questões elegíveis: não
  há resolução de conflito entre cotas que concorreriam pela mesma questão
  (um problema de alocação tipo bipartite matching). Assumido que cotas não
  se sobrepõem, que é como a tela de configuração as constrói. Documentado
  aqui para o caso de alguém notar contagens "otimistas" com cotas
  desenhadas de propósito para se sobrepor.
- **Congelamento (`is_frozen`) já bloqueia edição de configuração desde
  já**, embora nada ainda defina `is_frozen = true` (isso é Fase 10, no
  momento da geração) — o campo e a checagem já existem para não precisar
  mexer no fluxo de edição de novo quando a geração chegar.
- **Aplicações e cadernos restritos a ADMIN/GESTOR**; ELABORADOR não
  aparece em nenhuma parte deste fluxo (seção 20) — nem o link de menu é
  mostrado para ele.
- **`applications.description` virou `NOT NULL DEFAULT ''`** na migration:
  o Go escaneia essa coluna direto para `string` (não `*string`); a coluna
  já nunca recebia `NULL` de fato pelos meus próprios caminhos de escrita,
  mas deixar o schema garantir isso é mais robusto que confiar só na
  aplicação — achado nesta revisão final, corrigido antes de qualquer
  linha de código rodar contra um banco real.

### Fase 10 — Geração de PDF

- **A única exceção deliberada ao "jsonb opaco para o backend"** (decisão
  da Fase 4/5): gerar PDF em background, no servidor, exige produzir HTML a
  partir do JSON ProseMirror em algum lugar — não dá para empurrar isso
  para o navegador do professor, porque a geração roda numa goroutine sem
  nenhum navegador aberto. `internal/pdf/render.go` converte o subconjunto
  fixo de nós/marcas que o editor habilita (o mesmo conjunto configurado em
  `RichTextEditor.tsx`) — nós desconhecidos têm o conteúdo interno
  preservado em vez de descartado, para nunca perder texto silenciosamente
  se o editor ganhar uma extensão nova no futuro sem atualizar o renderer.
- **Fórmulas continuam renderizadas pelo KaTeX de verdade, só que dentro do
  próprio Chromium**: o HTML gerado carrega KaTeX via CDN
  (`cdn.jsdelivr.net`) e roda `katex.render()` em cada placeholder `.math`
  antes do `chromedp` imprimir — a mesma biblioteca usada no editor,
  seguindo exatamente o fluxo da seção 12 ("TipTap → HTML → renderização
  matemática → Chromium → PDF"). **Limitação conhecida**: isso exige que o
  container do backend tenha acesso de saída à internet no momento da
  geração; se a rede estiver bloqueada, o PDF ainda é gerado (o script
  captura a falha e libera a impressão do mesmo jeito), só que com as
  fórmulas em branco. Vender os arquivos do KaTeX dentro da imagem Docker
  do backend eliminaria essa dependência — não fiz isso porque exigiria
  acoplar o build do backend ao build do frontend (onde o KaTeX já é uma
  dependência via npm) só para copiar 3 arquivos estáticos, e o projeto não
  tem hoje uma cadeia de build compartilhada entre os dois Dockerfiles.
- **Cabeçalho/rodapé/numeração de página via parâmetros nativos do
  `Page.printToPDF` do Chrome** (`headerTemplate`/`footerTemplate` do CDP),
  não CSS: o Chromium não implementa `@page { @top-center {...} }` da
  especificação CSS Paged Media (isso é coisa de Prince/WeasyPrint) — a via
  suportada de verdade é a API nativa do protocolo, que o `chromedp` expõe
  como `page.PrintToPDF().WithHeaderTemplate(...)`.
- **`is_frozen` vira `true` dentro da mesma transação que grava o
  snapshot** (seleção + snapshot + congelamento — seções 25/26/27 inteiras
  como uma unidade atômica): ou a prova inteira foi selecionada e
  congelada, ou nada mudou. Reflete a garantia da seção 24 de nunca deixar
  uma geração pela metade.
- **Gerar de novo um caderno já congelado reaproveita o mesmo snapshot**
  em vez de sortear outras questões — só refaz o HTML/Chromium. Assim, se
  o Chromium falhar por qualquer motivo transitório (rede fora do ar para
  o KaTeX, timeout), tentar de novo produz a mesma prova, não uma diferente.
- **Job em background com uma goroutine simples**, sem fila (seção 30
  pede exatamente isso). O contexto passado para o trabalho é
  `context.Background()`, não o da requisição HTTP — o contexto da
  requisição é cancelado assim que a resposta 202 é enviada, e usá-lo por
  engano mataria a geração no meio.
- **Download do PDF exige sessão (ADMIN/GESTOR)**, diferente das imagens
  (seção 13, propositalmente sem autorização): uma prova é conteúdo
  sensível antes de ser aplicada, então aqui a proteção por sessão faz
  sentido onde para imagem não fazia.
- **Versão do `chromedp` fixada sem alta confiança de teste**: como não há
  Go/Docker neste ambiente, não consegui rodar a geração de ponta a ponta
  nem confirmar 100% a assinatura exata de `page.PrintToPDF().Do(ctx)` (2
  ou 3 valores de retorno) em tempo de compilação real. Vale conferir os
  logs do backend com atenção especial nessa área ao testar.

### Fase 11 — Refinamento

- **Testes cobrem só lógica pura, sem banco** (`ParseAlternatives`,
  `RenderHTML`/`BuildDocument`, `EncryptAPIKey`/`DecryptAPIKey`,
  `HashPassword`/`CheckPassword`, `canAccessDiscipline`,
  `AvailabilityItem.Sufficient`) — são as regras mais críticas da seção 43
  e não exigem infraestrutura para rodar (`go test ./...` funciona sem
  Postgres). Testes de integração (permissões ponta a ponta, autosave,
  congelamento, geração de PDF completa) ficam como próximo passo natural,
  exigindo um banco de teste — não implementados aqui por não haver como
  executá-los e confirmar que passam neste ambiente.

## Testes

Automatizados (Go, sem necessidade de banco — rodam com `go test ./...`
dentro do container ou com Go instalado localmente):

- `internal/questions`: regra das 5 alternativas/1 correta (seção 7) e
  `canAccessDiscipline` (seção 15/43).
- `internal/security`: hash de senha e cifra AES-256-GCM da API Key
  (round-trip, chave errada falha, ciphertext adulterado falha).
- `internal/pdf`: conversão ProseMirror → HTML (negrito, listas, escape de
  texto, fórmulas, nós desconhecidos) e montagem do documento final.
- `internal/booklets`: limite de suficiência da validação de
  disponibilidade (seção 24).

Não executados neste ambiente (sem Go instalado) — rode `go test ./...`
depois do primeiro build para confirmar.

## Pós-lançamento: acesso pela rede local, autocadastro e correção de build

Ajustes feitos depois da Fase 11, a pedido do usuário, ao testar o
primeiro `docker compose up`:

- **Correção de build**: `pgx/v5 v5.9.2` (fixado por causa da
  CVE-2026-33816) exige Go 1.25+; o Dockerfile do backend usava
  `golang:1.23-bookworm`. Corrigido para `golang:1.25-bookworm`, e
  `go.mod` atualizado para `go 1.25`. Esse é exatamente o tipo de
  problema que eu já havia sinalizado como risco não verificável sem um
  toolchain Go real neste ambiente.
- **Correção de YAML**: `docker-compose.yml` tinha um `${VAR:?mensagem}`
  cuja mensagem continha `: ` (dois-pontos seguido de espaço) sem aspas —
  o YAML interpretou isso como um mapeamento novo. Corrigido colocando o
  valor entre aspas e removendo o dois-pontos do meio da frase.
- **Acesso pela rede local (LAN)**: frontend e backend agora publicam
  suas portas explicitamente em `0.0.0.0` (todas as interfaces), não só
  `127.0.0.1` — necessário porque o Docker roda num servidor acessado
  remotamente, e o BoraIF precisa ser aberto de outra máquina da mesma
  rede pelo IP do servidor. O Postgres continua restrito a
  `127.0.0.1` do servidor, de propósito (nenhum motivo para expor um
  banco de dados à rede local inteira). Documentado em `RUNNINGDOCKER.MD`
  (seção 5.1) e `running.md`.
- **Autocadastro de elaboradores com aprovação do ADMIN** (requisito não
  previsto na especificação original, adicionado depois): antes, só um
  ADMIN podia criar usuários. Agora existe `POST /api/auth/signup`,
  público, que só cria contas ELABORADOR (nunca ADMIN/GESTOR — o papel
  nem é aceito na requisição). A conta nasce com `active = false` e a
  coluna nova `users.pending_approval = true` (migration
  `0003_user_approval.sql`). Enquanto pendente, o login é recusado com
  uma mensagem específica ("aguardando aprovação"), verificada só *depois*
  de confirmar a senha certa — para não vazar se um e-mail está cadastrado
  para quem errou a senha. O ADMIN aprova (`POST /api/users/{id}/approve`)
  ou recusa (`POST /api/users/{id}/reject`, que exclui a conta) numa seção
  nova no topo de `/admin/usuarios`; qualquer edição pelo formulário
  administrativo comum também limpa `pending_approval` (uma vez que o
  ADMIN mexeu na conta, ela deixou de estar "pendente" nesse sentido).
  `GET /api/disciplines` deixou de exigir sessão (a tela pública de
  cadastro precisa listar disciplinas antes do login).
- **Bug real na primeira subida em produção**: `backend/migrations/`
  chegou ao servidor do usuário com um `._0001_init.sql` — metadado do
  tipo "AppleDouble" que o macOS cria ao lado de arquivos em certos tipos
  de cópia/transferência (zip pelo Finder, unidade sem suporte a atributos
  estendidos, etc.). Como `//go:embed *.sql` embutiu esse arquivo junto
  com o de verdade, o `goose` tentou interpretar `._0001_init.sql` como
  nome de migration e quebrou ao tentar ler o prefixo numérico
  (`strconv.ParseInt: parsing ".": invalid syntax`) — o backend entrou em
  loop de crash e nunca chegou a criar o schema. Corrigido com
  `backend/.dockerignore` e `frontend/.dockerignore` (excluindo `._*` e
  `.DS_Store` do contexto de build dali em diante) e reforço equivalente
  no `.gitignore` — mas o arquivo que já estava no servidor do usuário
  precisou ser apagado manualmente (`find . -name "._*" -delete`) antes
  do rebuild, já que o `.dockerignore` só afeta builds futuros.
- **Efeito colateral do bug acima**: o frontend também entrou em
  crash-loop, por um motivo diferente e que valia a pena corrigir de
  qualquer forma: o Nginx resolve o hostname `backend` do `proxy_pass`
  **na inicialização**, uma vez só; como o backend estava sempre
  reiniciando, no instante em que o Nginx tentava subir o DNS interno do
  Docker às vezes não conseguia resolver `backend`, e o Nginx recusava
  iniciar (`host not found in upstream "backend"`) — um `emerg` fatal,
  não um 502 comum. Corrigido em `frontend/nginx.conf` movendo o alvo do
  `proxy_pass` para uma variável (`set $backend_upstream backend:8080;`)
  com um `resolver 127.0.0.11` (o DNS interno do Docker) — isso faz o
  Nginx resolver o host a cada requisição, não só na subida, então agora
  o pior caso é um 502 em `/api/` enquanto o backend estiver fora, nunca
  o Nginx inteiro recusando iniciar. Esse é o padrão comum para Nginx
  atrás de um Compose onde o serviço de trás pode não estar pronto ainda.
- **Regressão causada pelo próprio fix acima**: a primeira versão trocou
  `proxy_pass http://backend:8080/api/;` por
  `proxy_pass http://$backend_upstream/api/;` (variável + `/api/` no
  final). Só que, com host resolvido por variável, o Nginx não faz mais a
  troca de prefixo baseada na `location` — o comportamento deixa de ser
  claramente documentado quando ainda existe um caminho depois da
  variável. Resultado real observado: **todo** `/api/*` passou a
  responder 404 do próprio backend Go (`http.ServeMux` não achava rota
  nenhuma — o corpo "404 page not found\n", 19 bytes, é o padrão do Go, e
  isso é como percebi que a requisição chegava no backend com o caminho
  errado, não que o Nginx nem estava repassando). Corrigido tirando
  completamente o caminho do `proxy_pass`
  (`proxy_pass http://$backend_upstream;`, sem `/api/` no final): sem
  nenhum caminho ali, o comportamento do Nginx é repassar a URL original
  inteira sem modificar — documentado e sem ambiguidade, com ou sem
  variável. Fica a lição: variável em `proxy_pass` e caminho em
  `proxy_pass` não se misturam bem juntos.
- **O bug do `._0001_init.sql` voltou** numa transferência de arquivos
  posterior, mesmo já existindo `backend/.dockerignore` — ou o arquivo
  reapareceu numa cópia nova do projeto pro servidor e o `.dockerignore`
  não acompanhou, ou algo no fluxo de transferência do usuário recria
  esses metadados a cada vez. Em vez de depender só de higiene de arquivo
  externa ao código (que já falhou duas vezes), a correção definitiva
  ficou em `internal/db/db.go`: um `fs.FS` que embrulha o `embed.FS` das
  migrations e esconde do `goose`, em tempo de execução, qualquer entrada
  cujo nome não comece com dígitos seguidos de `.sql` — não importa se o
  arquivo indesejado chegou a ser embutido no binário ou não. Isso torna o
  backend imune a essa classe inteira de problema
  independentemente de como os arquivos cheguem ao servidor.
- **Assistente de IA sem log de erro**: o handler de revisão por IA
  (`internal/ai/handlers.go`) devolvia uma mensagem genérica para
  qualquer falha ao chamar a OpenAI, mas nunca registrava o erro real em
  lugar nenhum — tornando impossível diagnosticar de fora (rede
  bloqueada, chave sem crédito, erro específico da OpenAI, etc., todos
  pareciam a mesma coisa). Adicionado `log.Printf` com o erro completo
  antes de responder a mensagem genérica ao professor; a mensagem que ele
  vê continua propositalmente sem detalhe técnico.

## Roadmap

**Todas as 11 fases da especificação foram implementadas:**
~~1. Fundação~~ · ~~2. Usuários e permissões~~ · ~~3. Assuntos~~ ·
~~4. CRUD de questões~~ · ~~5. Editor TipTap~~ · ~~6. Autosave~~ ·
~~7. Biblioteca de imagens~~ · ~~8. Assistente de IA~~ ·
~~9. Aplicações e cadernos~~ · ~~10. Geração de PDF~~ ·
~~11. Refinamento~~ (testes automatizados das regras críticas;
performance, UX e segurança já vinham sendo tratadas fase a fase, não só
no final).

O que fica como próximo passo natural, fora do escopo coberto por esta
sessão (ver seção "Testes" acima e `architecture.md`):

- Testes de integração com banco de teste (permissões ponta a ponta,
  congelamento, geração de PDF completa).
- Vendorizar os arquivos do KaTeX na imagem do backend, removendo a
  dependência de rede externa na hora de gerar PDF.
- Validação manual de ponta a ponta com Docker/Chromium reais — não foi
  possível neste ambiente (sem Go/Node/Docker instalados).
