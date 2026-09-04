# Arquitetura do BoraIF

## Visão geral

O BoraIF é um **monólito modular**: um único backend, um único frontend e
um único banco de dados, sem microservices, sem fila de mensagens
(RabbitMQ/Kafka), sem Redis e sem service mesh. Essa escolha é
deliberada — o sistema é usado por poucas pessoas simultaneamente (1-4
professores/gestores), então a complexidade de uma arquitetura
distribuída não traria benefício real, só mais pontos de falha e mais
trabalho de manutenção.

```
┌─────────────┐      /api/*       ┌─────────────┐      SQL      ┌──────────────┐
│  frontend   │ ───proxy Nginx──▶ │   backend   │ ────────────▶ │  PostgreSQL  │
│ React + TS  │                   │     Go      │                │              │
│  (Nginx)    │ ◀──────HTML───────│  net/http   │                │              │
└─────────────┘                   └──────┬──────┘                └──────────────┘
                                          │ controla
                                          ▼
                                   ┌─────────────┐
                                   │  Chromium   │  (headless, dentro do
                                   │  headless   │   próprio container backend)
                                   └─────────────┘
```

Três containers Docker: `frontend`, `backend`, `postgres`. O Chromium
usado para gerar PDF roda *dentro* do container do backend (não é um
quarto container) — o backend Go o controla via protocolo DevTools
(biblioteca `chromedp`).

## Stack tecnológica

### Backend

| Tecnologia | Uso |
|---|---|
| **Go 1.23** | linguagem do backend inteiro |
| **`net/http`** (biblioteca padrão) | roteamento HTTP — sem framework externo (Go 1.22+ já suporta padrões de rota com método, ex.: `"GET /api/questions/{id}"`) |
| **PostgreSQL 16** | banco de dados relacional |
| **`pgx/v5`** | driver/cliente PostgreSQL (sem ORM — SQL escrito à mão, por controle e simplicidade) |
| **`goose`** | migrations versionadas, embutidas no binário via `embed.FS` e aplicadas automaticamente na subida |
| **`golang.org/x/crypto/bcrypt`** | hash de senha |
| **AES-256-GCM** (biblioteca padrão `crypto/aes`+`crypto/cipher`) | cifra das API Keys da OpenAI dos professores |
| **`chromedp`** | controla um Chromium headless para gerar PDF |
| **OpenAI Chat Completions API** | assistente de revisão de questões (chamado via HTTP puro, sem SDK) |

### Frontend

| Tecnologia | Uso |
|---|---|
| **React 18 + TypeScript** | interface |
| **Vite** | build e servidor de desenvolvimento |
| **React Router** | navegação entre telas |
| **TipTap 3** (sobre ProseMirror) | editor de texto rico das questões |
| **KaTeX** | renderização de fórmulas matemáticas (no editor e, via Chromium, no PDF) |
| **Nginx** | serve os arquivos estáticos em produção e faz proxy de `/api/*` para o backend |

### Infraestrutura

| Tecnologia | Uso |
|---|---|
| **Docker + Docker Compose** | empacotamento e orquestração local dos 3 containers |
| **Volume Docker nomeado** (`postgres_data`) | persistência do banco, independente do ciclo de vida dos containers de aplicação |
| **Bind mounts** (`./uploads`, `./generated`) | imagens e PDFs ficam no disco do host, sobrevivendo a qualquer rebuild |

Não há: Kubernetes, service mesh, API gateway separado, event bus, Redis,
RabbitMQ, Kafka, ORM, ou qualquer SDK de terceiros além dos listados
acima. Sempre que havia a opção entre uma solução com menos dependências e
outra mais "completa", este projeto escolheu a mais simples — ver a seção
"Regra para decisões técnicas" em `especificacoes.md`.

## Como o backend é organizado

Cada domínio do negócio é um pacote Go em `backend/internal/`, sem camadas
artificiais (sem "controller/service/repository" genéricos por cima de
tudo) — cada pacote geralmente tem `model.go` (tipos), `repository.go`
(acesso ao Postgres) e `handlers.go` (HTTP), e às vezes mais arquivos
quando o domínio pede (ex.: `pdf/render.go`, `pdf/chromium.go`).

Dois pacotes existem *só* para evitar ciclos de import, e vale entender o
porquê: `internal/apiutil` (helpers de resposta JSON) e
`internal/security` (hash de senha + cifra AES) não têm nenhuma
dependência de domínio, porque tanto pacotes "de baixo" (ex.: `users`)
quanto o `httpserver` (que importa *todos* os domínios para montar as
rotas) precisam deles — se esses helpers vivessem dentro de um pacote de
domínio qualquer, o Go recusaria compilar por causa do ciclo. Esse padrão
se repetiu três vezes durante o desenvolvimento (ver `DEVELOPMENT.md`)
e é a razão de esses dois pacotes existirem separados.

A autorização por papel (ADMIN/ELABORADOR/GESTOR) e por disciplina é
sempre validada no backend, nunca só escondendo botão no frontend.

## Como o frontend é organizado

Uma pasta por área de funcionalidade em `frontend/src/pages/` (questões,
assuntos, imagens, aplicações, contas), mais um pacote de componentes
compartilhados em `frontend/src/components/` — o mais importante deles é
o editor de texto rico (`components/editor/`), usado de forma idêntica
para o enunciado, o comando e cada uma das cinco alternativas de toda
questão.

Não há Redux nem TanStack Query — estado é local por página (React
`useState`/`useEffect`), com cuidado deliberado para não disparar
requisições HTTP duplicadas (efeitos com dependências corretas, e um
"debounce" no autosave e na busca textual).

## Pastas importantes (não é a árvore inteira — só o que importa)

```
boraif/
├── backend/
│   ├── cmd/server/          # ponto de entrada: sobe o servidor OU roda "create-admin"
│   ├── internal/
│   │   ├── apiutil/         # helpers de resposta JSON HTTP (sem dependência de domínio)
│   │   ├── security/        # hash de senha + cifra AES-256-GCM (sem dependência de domínio)
│   │   ├── auth/            # login/logout/sessão + "minha conta" (API Key da OpenAI)
│   │   ├── users/           # CRUD de usuários e papéis
│   │   ├── disciplines/     # leitura das 13 disciplinas fixas
│   │   ├── subjects/        # assuntos (com detecção de nome duplicado/parecido via pg_trgm)
│   │   ├── catalogs/        # anos, dificuldades, status de questão (leitura)
│   │   ├── questions/       # CRUD de questões, alternativas, autosave
│   │   ├── images/          # upload e biblioteca de imagens por disciplina
│   │   ├── ai/              # cliente OpenAI + prompts do assistente de revisão
│   │   ├── applications/    # CRUD de aplicações (campanhas de prova)
│   │   ├── booklets/        # cadernos, configuração, cotas, disponibilidade
│   │   ├── pdf/             # seleção de questões, snapshot, HTML, Chromium, PDF
│   │   └── httpserver/      # monta o roteador HTTP juntando todos os domínios acima
│   └── migrations/          # SQL versionado (goose), embutido no binário
├── frontend/
│   └── src/
│       ├── pages/           # uma pasta por área (questions, subjects, images, applications, account, admin)
│       ├── components/
│       │   ├── editor/      # o editor de texto rico (TipTap) e suas ferramentas
│       │   └── AppLayout.tsx  # cabeçalho/menu compartilhado
│       ├── api/client.ts    # wrapper único sobre fetch (cookies, tratamento de erro)
│       └── auth/AuthContext.tsx  # sessão do usuário logado, carregada uma única vez
├── uploads/                 # imagens enviadas (persistente, fora dos containers)
├── generated/               # PDFs gerados (persistente, fora dos containers)
├── docker-compose.yml       # define os 3 containers e o volume do Postgres
└── especificacoes.md        # documento de requisitos original do projeto
```

## Fluxos principais

### Editor de questões → banco

O conteúdo de cada campo (enunciado, comando, cada alternativa) é salvo
como **JSON do ProseMirror/TipTap**, não como HTML solto — isso permite
reconstruir o HTML sob demanda (para visualização ou PDF) sem perder
estrutura. O backend trata esse JSON como opaco: ele não interpreta a
estrutura do editor, só valida que não está vazio.

### Autosave

O frontend debouncia 2 segundos após a última edição em qualquer campo e
manda o conjunto inteiro (metadados + 7 campos de conteúdo) num único
`PUT`, nunca campo a campo. Uma trava evita duas requisições de save
simultâneas.

### Geração de PDF (a única exceção ao "JSON opaco")

Como a geração roda em background, sem navegador aberto, o backend
*precisa* converter o JSON do ProseMirror em HTML sozinho — é a única
exceção deliberada à regra acima, isolada em `internal/pdf/render.go`.
Fluxo completo:

```
questão salva (JSON ProseMirror por campo)
        │  (na primeira geração de um caderno)
        ▼
seleção aleatória das questões elegíveis por cota (seção 25),
agrupadas em blocos contíguos por disciplina (não mais embaralhadas juntas)
        │
        ▼
snapshot congelado (seção 26) + configuração travada (seção 27)
        │
        ▼
até 4 "tipos de prova" gerados (seção 21.2): mesmas questões, cada tipo
reordena só DENTRO de cada bloco de disciplina + embaralha as 5
alternativas de cada questão — o resultado (posição impressa × letra
correta) já É o gabarito daquele tipo, gravado em booklet_variant_questions
        │  (isso tudo é só banco — roda síncrono, dentro do handler HTTP)
        ▼
JSON → HTML (internal/pdf/render.go), um documento por tipo × {prova, gabarito}
        │
        ▼
documento HTML completo, com placeholders de fórmula (prova) ou a lista
"Q# - Letra" em colunas via CSS column-width (gabarito)
        │
        ▼
Chromium headless carrega o HTML, roda o KaTeX (via CDN) sobre os
placeholders, e só então o backend manda imprimir em PDF (chromedp)
        │
        ▼
PDF salvo em disco + registro de status atualizado (COMPLETED)
```

A etapa de seleção/congelamento/geração de tipos roda dentro da própria
requisição HTTP que pediu a geração (é rápida — só banco). Só a
renderização de PDF em si, por documento, roda depois em background, uma
de cada vez — um caderno com N tipos gera 2×N PDFs (prova + gabarito por
tipo) numa única geração. O gabarito também pode ser baixado em CSV sem
passar por nada disso: é montado na hora, direto do banco.

### Autenticação

Sessão guardada no Postgres (tabela `sessions`), identificada por um
cookie `httpOnly` com um token aleatório — não é JWT. Essa escolha
permite revogar uma sessão instantaneamente (por exemplo, ao desativar um
usuário) sem a complexidade de gerenciar refresh tokens, e sem precisar de
Redis para guardar sessão em memória compartilhada.

## Decisões de arquitetura mais importantes (resumo)

Cada uma destas é detalhada, com o raciocínio completo, em
`DEVELOPMENT.md`:

- Sessão server-side em vez de JWT.
- Migrations com `goose`, embutidas no binário, aplicadas automaticamente.
- Catálogos (disciplinas, anos, dificuldades, status) como tabelas, nunca
  strings soltas no código.
- Cadernos de prova como entidade própria dentro de uma aplicação — uma
  aplicação pode ter mais de um caderno, cada um com configuração,
  congelamento e numeração próprios.
- Cota de seleção de questões sempre como linha "folha" (nunca uma linha
  resumindo outras mais específicas), para a validação "soma das cotas =
  total" nunca ser ambígua.
- API Key da OpenAI cifrada com AES-256-GCM, chave mestra fora do banco.
- Upload de imagem sem autorização complexa (só separação por disciplina)
  e servido publicamente por nome de arquivo imprevisível; download de PDF
  já gerado exige sessão, por ser conteúdo mais sensível antes da
  aplicação da prova.
