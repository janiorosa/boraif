# Banco de dados do BoraIF

PostgreSQL 16. O schema completo é criado e mantido por migrations
versionadas (`backend/migrations/*.sql`, formato `goose`), aplicadas
automaticamente sempre que o backend inicia — não é preciso rodar nada
manualmente. Este documento explica cada tabela e cada coluna.

## Catálogos fixos

Tabelas de valores fixos (populadas pelo seed, seção 32 da especificação),
usadas para nunca espalhar strings soltas pelo código.

### `grade_years` — anos do Ensino Médio

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `code` | text, único | código estável (`1_ANO`, `2_ANO`, `3_ANO`) |
| `name` | text | nome exibido ("1º ano", "2º ano", "3º ano") |
| `sort_order` | smallint | ordem de exibição |

### `difficulties` — dificuldades

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `code` | text, único | `FACIL`, `MEDIA`, `DIFICIL` |
| `name` | text | nome exibido |
| `sort_order` | smallint | ordem de exibição |

### `question_statuses` — status de uma questão

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `code` | text, único | `RASCUNHO`, `EM_REVISAO`, `EM_TESTE`, `TESTADA`, `APROVADA`, `REJEITADA`, `ARQUIVADA`, `OBSOLETA` |
| `name` | text | nome exibido |
| `eligible_for_exam` | boolean | se `true`, questões nesse status podem ser sorteadas para uma prova (só `APROVADA` e `TESTADA`, por padrão) |
| `sort_order` | smallint | ordem de exibição |

### `disciplines` — disciplinas

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `code` | text, único | ex.: `MATEMATICA`, `FISICA` |
| `name` | text | nome exibido |

As 13 disciplinas do Ensino Médio já vêm cadastradas pelo seed; não há
tela para criar novas.

## Usuários e sessão

### `users` — usuários do sistema

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `name` | text | nome |
| `email` | citext, único | login (case-insensitive) |
| `password_hash` | text | hash bcrypt da senha — **nunca a senha em texto puro** |
| `role` | text | `ADMIN`, `ELABORADOR` ou `GESTOR` |
| `discipline_id` | bigint, FK → `disciplines.id`, opcional | obrigatório para ELABORADOR; sempre nulo para ADMIN/GESTOR (constraint garante isso) |
| `openai_api_key_ciphertext` | bytea, opcional | API Key da OpenAI cifrada (AES-256-GCM) |
| `openai_api_key_nonce` | bytea, opcional | nonce usado na cifra — sem ele não dá para decifrar |
| `active` | boolean | usuário desativado não consegue logar |
| `pending_approval` | boolean | `true` só para autocadastros (ELABORADOR) ainda não revisados por um ADMIN — enquanto `true`, o login é recusado mesmo com senha certa |
| `created_at` / `updated_at` | timestamptz | auditoria |

Um professor pode se cadastrar sozinho (`POST /api/auth/signup`, sem
sessão) — a conta nasce com `active = false` e `pending_approval = true`;
só um ADMIN aprovando ou recusando (excluindo a conta) libera ou descarta
o acesso. Contas criadas pelo próprio ADMIN já nascem `active = true` e
`pending_approval = false`.

A chave usada para cifrar `openai_api_key_ciphertext` **não fica no
banco** — vem da variável de ambiente `API_KEY_ENCRYPTION_SECRET`. Mesmo
um dump completo do banco não permite recuperar as API Keys sem essa
chave externa.

### `sessions` — sessões de login ativas

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `user_id` | bigint, FK → `users.id` | dono da sessão |
| `token_hash` | bytea, único | hash SHA-256 do token do cookie — o token em si nunca é gravado |
| `created_at` | timestamptz | quando a sessão foi criada |
| `expires_at` | timestamptz | quando a sessão expira (7 dias após criação) |

## Assuntos

### `subjects` — assuntos de cada disciplina

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `discipline_id` | bigint, FK → `disciplines.id` | disciplina dona do assunto |
| `name` | text | nome do assunto (ex.: "Mecânica dos Fluidos") |
| `created_by` | bigint, FK → `users.id`, opcional | quem criou |
| `created_at` | timestamptz | auditoria |

Um índice único em `(discipline_id, lower(name))` impede duas grafias
idênticas (ignorando maiúsculas/minúsculas) na mesma disciplina. Um índice
trigram (`pg_trgm`) em `name` permite buscar nomes *parecidos* (não só
idênticos) antes de criar um assunto novo.

## Questões

### `questions` — a questão em si

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `discipline_id` | bigint, FK → `disciplines.id` | disciplina (imutável após criada) |
| `subject_id` | bigint, FK → `subjects.id` | assunto |
| `grade_year_id` | bigint, FK → `grade_years.id` | ano |
| `difficulty_id` | bigint, FK → `difficulties.id` | dificuldade |
| `status_id` | bigint, FK → `question_statuses.id` | status atual |
| `author_id` | bigint, FK → `users.id` | quem criou |
| `statement_json` | jsonb | **enunciado**, em formato ProseMirror/TipTap |
| `command_json` | jsonb | **comando** (a pergunta em si), em formato ProseMirror/TipTap |
| `revision_number` | integer | sobe só quando o *conteúdo* muda de fato (não a cada autosave que reenvia o mesmo texto) |
| `created_at` / `updated_at` | timestamptz | auditoria |

Enunciado e comando ficam em `jsonb` (não em HTML solto) porque o BoraIF
guarda a estrutura do editor, não uma renderização — o HTML é produzido
sob demanda quando necessário (visualização, PDF).

### `question_alternatives` — as cinco alternativas de cada questão

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `question_id` | bigint, FK → `questions.id`, `ON DELETE CASCADE` | questão dona |
| `position` | smallint (1 a 5) | 1=A, 2=B, 3=C, 4=D, 5=E |
| `content_json` | jsonb | texto da alternativa, em formato ProseMirror/TipTap |
| `is_correct` | boolean | se é a alternativa correta |

Regras impostas pelo próprio banco (não só pela aplicação):

- Um índice único garante no máximo uma alternativa correta por questão.
- Uma *trigger* (`check_question_alternatives`), adiada até o fim da
  transação, garante que toda questão termina com **exatamente 5**
  alternativas e **exatamente 1** marcada como correta — nunca 4, nunca 6,
  nunca 0 ou 2 corretas. Não existe uma sexta linha/coluna para "a
  resposta certa": ela é sempre uma propriedade de uma das cinco.

## Imagens

### `images` — biblioteca de imagens por disciplina

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `discipline_id` | bigint, FK → `disciplines.id` | disciplina dona (compartilhada entre todos os professores dela) |
| `filename` | text | nome original do arquivo enviado |
| `path` | text | caminho relativo no filesystem (`images/{código da disciplina}/{nome aleatório}.ext`) |
| `mime_type` | text | tipo detectado pelos bytes reais do arquivo, nunca pela extensão informada |
| `size_bytes` | bigint | tamanho do arquivo |
| `uploaded_by` | bigint, FK → `users.id` | quem enviou (sem controle de "dono" além disso — qualquer professor da disciplina reusa) |
| `created_at` | timestamptz | quando foi enviada |

O arquivo em si fica no filesystem (pasta `uploads/`), não no banco — só a
referência fica aqui.

## Aplicações e cadernos

Uma **aplicação** é uma campanha de prova (ex.: "2026/1"). Cada aplicação
pode ter um ou mais **cadernos** — o mais comum são 2, mas pode ser
qualquer quantidade. Cada caderno tem sua própria seleção de questões,
configuração e PDF.

### `applications` — a campanha

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `name` | text, único | ex.: "2026/1" |
| `description` | text | descrição livre (nunca nula — string vazia por padrão) |
| `status` | text | `RASCUNHO`, `ATIVA` ou `ENCERRADA` — informativo, não trava edição dos cadernos |
| `creator_id` | bigint, FK → `users.id` | quem criou |
| `created_at` / `updated_at` | timestamptz | auditoria |

### `application_booklets` — os cadernos de uma aplicação

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `application_id` | bigint, FK → `applications.id`, `ON DELETE CASCADE` | aplicação dona |
| `name` | text | ex.: "Caderno 1" (único dentro da mesma aplicação) |
| `sort_order` | smallint | ordem de exibição |

### `booklet_configurations` — configuração de seleção de um caderno

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `booklet_id` | bigint, FK → `application_booklets.id`, único, `ON DELETE CASCADE` | caderno dono (relação 1 para 1) |
| `total_questions` | integer | quantas questões o caderno deve ter no total |
| `variant_count` | smallint, `1` a `4`, padrão `2` | quantos "tipos de prova" o caderno terá (seção 21.2) — mesmas questões em todos, só a ordem muda |
| `is_frozen` | boolean | `true` depois que as questões já foram selecionadas/geradas — a partir daí, a configuração não pode mais ser alterada |
| `created_at` / `updated_at` | timestamptz | auditoria |

### `booklet_configuration_grade_years` — anos incluídos num caderno

| Coluna | Tipo | Descrição |
|---|---|---|
| `configuration_id` | bigint, FK → `booklet_configurations.id`, `ON DELETE CASCADE` | configuração dona |
| `grade_year_id` | bigint, FK → `grade_years.id` | ano incluído |

Chave primária composta pelas duas colunas — um caderno pode incluir mais
de um ano (ex.: prova conjunta de 1º e 2º ano).

### `booklet_quota_rules` — quantas questões de cada critério

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `configuration_id` | bigint, FK → `booklet_configurations.id`, `ON DELETE CASCADE` | configuração dona |
| `discipline_id` | bigint, FK → `disciplines.id` | disciplina da cota |
| `subject_id` | bigint, FK → `subjects.id`, opcional | assunto (se quiser restringir) |
| `difficulty_id` | bigint, FK → `difficulties.id`, opcional | dificuldade (se quiser restringir) |
| `quantity` | integer, `> 0` | quantas questões desse critério entram no caderno |

Cada linha é um critério **final** (nunca uma linha-resumo por cima de
outras mais específicas) — o total de uma disciplina é sempre a soma das
linhas dela, nunca uma linha própria coexistindo com outras mais
detalhadas. Isso evita ambiguidade ao validar que a soma de todas as
linhas bate com `total_questions`.

### `booklet_question_snapshots` — questões já selecionadas e congeladas

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `booklet_id` | bigint, FK → `application_booklets.id`, `ON DELETE CASCADE` | caderno dono |
| `question_id` | bigint, FK → `questions.id`, `ON DELETE SET NULL`, opcional | questão original (a referência pode virar nula se a questão original for excluída depois — o snapshot continua íntegro) |
| `position_in_booklet` | integer, `> 0` | numeração impressa (1, 2, 3, ... até `total_questions`) — única por caderno |
| `discipline_name`, `subject_name`, `grade_year_name`, `difficulty_name` | text | nomes capturados no momento da geração (não mudam se o cadastro original mudar depois) |
| `statement_json`, `command_json` | jsonb | cópia congelada do enunciado/comando da questão naquele momento |
| `alternatives_json` | jsonb | cópia congelada das 5 alternativas (posição, conteúdo, qual é a correta) |
| `created_at` | timestamptz | quando o snapshot foi criado |

Esta é a peça-chave da "seção 26" da especificação: garante que o PDF de
uma prova já gerada nunca muda, mesmo que a questão original seja editada
ou até excluída depois. As posições aqui são a ordem "canônica", agrupada
por disciplina em blocos contíguos — cada tipo de prova (abaixo) reordena
só *dentro* desses blocos, nunca entre eles.

### `booklet_variants` — os "tipos de prova" de um caderno (seção 21.2)

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `booklet_id` | bigint, FK → `application_booklets.id`, `ON DELETE CASCADE` | caderno dono |
| `variant_number` | smallint, `1` a `4` | qual tipo ("Tipo 1", "Tipo 2", ...) — único por caderno |
| `created_at` | timestamptz | quando foi gerado |

Criados de uma vez, junto com o snapshot, na primeira geração de PDF do
caderno (`variant_count` linhas). Nunca é criado ou removido depois disso.

### `booklet_variant_questions` — o gabarito de cada tipo

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `variant_id` | bigint, FK → `booklet_variants.id`, `ON DELETE CASCADE` | tipo dono |
| `snapshot_id` | bigint, FK → `booklet_question_snapshots.id`, `ON DELETE CASCADE` | qual questão (a mesma em todos os tipos do caderno) |
| `position_in_variant` | integer, `> 0` | numeração impressa *daquele tipo* — única por tipo |
| `alternative_order` | jsonb | as letras originais do snapshot (`alternatives_json`), na nova ordem de exibição daquele tipo — ex.: `["C","A","E","B","D"]` |
| `correct_letter` | char(1), `A`–`E` | letra correta *naquele tipo* |

`(position_in_variant, correct_letter)` de todas as linhas de um tipo,
nessa ordem, **é** o gabarito daquele tipo — não existe uma tabela
"gabarito" separada, porque seria uma duplicata exata desses dados. Fica
gravado no banco independente de qualquer PDF ter sido gerado.

### `generated_documents` — histórico de gerações de PDF

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `booklet_id` | bigint, FK → `application_booklets.id`, `ON DELETE CASCADE` | caderno dono |
| `variant_id` | bigint, FK → `booklet_variants.id`, `ON DELETE CASCADE`, opcional | tipo de prova dono (nulo só em linhas de antes dos tipos de prova existirem) |
| `kind` | text, padrão `EXAM` | `EXAM` (prova) ou `ANSWER_KEY` (gabarito em PDF) |
| `status` | text | `PENDING` → `PROCESSING` → `COMPLETED` ou `FAILED` |
| `file_path` | text, opcional | caminho do PDF gerado (só depois de `COMPLETED`) |
| `error_message` | text, opcional | motivo da falha (só depois de `FAILED`) |
| `requested_by` | bigint, FK → `users.id` | quem pediu a geração |
| `created_at` | timestamptz | quando foi pedida |
| `completed_at` | timestamptz, opcional | quando terminou (sucesso ou falha) |

Cada clique em "Gerar PDF" cria um par de linhas (`EXAM` + `ANSWER_KEY`)
por tipo de prova do caderno — um caderno com 2 tipos gera 4 linhas por
geração. O gabarito em CSV não passa por aqui: é montado na hora, direto
de `booklet_variant_questions`, sem gerar nenhum registro.

## Configuração padrão

Modelo copiado para a configuração de todo caderno novo (seção 22), para
o gestor não precisar preencher tudo do zero toda vez. Existe **no máximo
uma linha** em `default_configurations` (tratada como uma tabela
"singleton").

### `default_configurations`

| Coluna | Tipo | Descrição |
|---|---|---|
| `id` | bigserial (PK) | identificador |
| `total_questions` | integer | total padrão sugerido |
| `updated_at` | timestamptz | última alteração |

### `default_configuration_grade_years`

Mesma estrutura de `booklet_configuration_grade_years`, mas ligada a
`default_configurations` em vez de a um caderno específico.

### `default_quota_rules`

Mesma estrutura de `booklet_quota_rules`, mas ligada a
`default_configurations`.

Alterar a configuração padrão **nunca** muda cadernos já criados — cada
caderno recebe sua própria cópia independente no momento em que é criado.

## Índices que valem destacar

Além das chaves primárias/estrangeiras (sempre indexadas), o schema cria
índices específicos para os filtros mais usados nas listagens:

- `questions`: por disciplina, assunto, ano, dificuldade, status, autor,
  data de atualização, e um índice composto `(discipline_id, status_id)`
  (a combinação mais comum na tela de listagem).
- `subjects`: índice trigram (`pg_trgm`) em `name`, para busca por
  semelhança.
- `sessions`: por usuário e por data de expiração.
- `images`, `booklet_question_snapshots`, `generated_documents`,
  `application_booklets`: por disciplina/caderno/aplicação, conforme o
  caso — sempre a coluna usada para "listar tudo de X".
- `booklet_variant_questions`: por `variant_id` (montar a prova/gabarito de
  um tipo inteiro de uma vez); `generated_documents`: também por
  `variant_id`.

## Diagrama simplificado de relações

```
disciplines ──┬── users (elaborador)
              ├── subjects ── questions ── question_alternatives
              ├── images
              └── (via quota_rules) booklet_quota_rules / default_quota_rules

applications ── application_booklets ── booklet_configurations ──┬── booklet_configuration_grade_years
                       │                                          └── booklet_quota_rules
                       ├── booklet_question_snapshots ── booklet_variant_questions
                       ├── booklet_variants ── booklet_variant_questions
                       │                    └── generated_documents (variant_id)
                       └── generated_documents (booklet_id)

grade_years / difficulties / question_statuses → referenciadas por
questions, booklet_configuration_grade_years, booklet_quota_rules, etc.
```
