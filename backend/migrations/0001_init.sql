-- +goose Up

CREATE EXTENSION IF NOT EXISTS citext;

-- ==========================================================================
-- Catálogos (evitam strings soltas espalhadas pelo código)
-- ==========================================================================

CREATE TABLE grade_years (
    id    bigserial PRIMARY KEY,
    code  text NOT NULL UNIQUE,
    name  text NOT NULL,
    sort_order smallint NOT NULL
);

CREATE TABLE difficulties (
    id    bigserial PRIMARY KEY,
    code  text NOT NULL UNIQUE,
    name  text NOT NULL,
    sort_order smallint NOT NULL
);

CREATE TABLE question_statuses (
    id    bigserial PRIMARY KEY,
    code  text NOT NULL UNIQUE,
    name  text NOT NULL,
    -- questões nesse status podem ser selecionadas para geração de prova (seção 25)
    eligible_for_exam boolean NOT NULL DEFAULT false,
    sort_order smallint NOT NULL
);

CREATE TABLE disciplines (
    id    bigserial PRIMARY KEY,
    code  text NOT NULL UNIQUE,
    name  text NOT NULL
);

-- ==========================================================================
-- Usuários e sessões
-- ==========================================================================

CREATE TABLE users (
    id                          bigserial PRIMARY KEY,
    name                        text NOT NULL,
    email                       citext NOT NULL UNIQUE,
    password_hash               text NOT NULL,
    role                        text NOT NULL CHECK (role IN ('ADMIN', 'ELABORADOR', 'GESTOR')),
    -- obrigatório apenas para ELABORADOR (seção 15)
    discipline_id               bigint REFERENCES disciplines(id),
    openai_api_key_ciphertext   bytea,
    openai_api_key_nonce        bytea,
    active                      boolean NOT NULL DEFAULT true,
    created_at                  timestamptz NOT NULL DEFAULT now(),
    updated_at                  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT elaborador_requires_discipline
        CHECK (role <> 'ELABORADOR' OR discipline_id IS NOT NULL)
);

CREATE INDEX idx_users_discipline ON users(discipline_id);

CREATE TABLE sessions (
    id          bigserial PRIMARY KEY,
    user_id     bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  bytea NOT NULL UNIQUE,
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL
);

CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- ==========================================================================
-- Assuntos
--
-- pg_trgm habilita busca por semelhança de nome (seção 14: "evitar
-- duplicações acidentais de assuntos, realizando validação de nomes
-- semelhantes/exatos antes de criar um novo assunto"). Não é fila/cache,
-- é apenas uma extensão nativa do Postgres para comparação de texto — não
-- fere o princípio de simplicidade de infraestrutura (seção 2.1).
-- ==========================================================================

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE subjects (
    id              bigserial PRIMARY KEY,
    discipline_id   bigint NOT NULL REFERENCES disciplines(id),
    name            text NOT NULL,
    created_by      bigint REFERENCES users(id),
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_subjects_discipline ON subjects(discipline_id);

-- Bloqueia duplicata exata (case-insensitive) na mesma disciplina.
CREATE UNIQUE INDEX ux_subjects_discipline_name_ci ON subjects (discipline_id, lower(name));

-- Suporte à busca por nomes semelhantes (não apenas idênticos) via similarity().
CREATE INDEX idx_subjects_name_trgm ON subjects USING gin (name gin_trgm_ops);

-- ==========================================================================
-- Questões
-- ==========================================================================

CREATE TABLE questions (
    id              bigserial PRIMARY KEY,
    discipline_id   bigint NOT NULL REFERENCES disciplines(id),
    subject_id      bigint NOT NULL REFERENCES subjects(id),
    grade_year_id   bigint NOT NULL REFERENCES grade_years(id),
    difficulty_id   bigint NOT NULL REFERENCES difficulties(id),
    status_id       bigint NOT NULL REFERENCES question_statuses(id),
    author_id       bigint NOT NULL REFERENCES users(id),
    -- conteúdo estruturado ProseMirror/TipTap (seção 8), HTML é derivado sob demanda
    statement_json  jsonb NOT NULL,
    command_json    jsonb NOT NULL,
    revision_number integer NOT NULL DEFAULT 1,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_questions_discipline ON questions(discipline_id);
CREATE INDEX idx_questions_subject ON questions(subject_id);
CREATE INDEX idx_questions_grade_year ON questions(grade_year_id);
CREATE INDEX idx_questions_difficulty ON questions(difficulty_id);
CREATE INDEX idx_questions_status ON questions(status_id);
CREATE INDEX idx_questions_author ON questions(author_id);
CREATE INDEX idx_questions_updated_at ON questions(updated_at);
-- filtro composto mais comum na listagem (seção 38/39)
CREATE INDEX idx_questions_discipline_status ON questions(discipline_id, status_id);

-- ==========================================================================
-- Alternativas — regra fundamental da seção 7:
-- exatamente 5 por questão (posições 1..5 = A..E), exatamente 1 correta.
-- ==========================================================================

CREATE TABLE question_alternatives (
    id              bigserial PRIMARY KEY,
    question_id     bigint NOT NULL REFERENCES questions(id) ON DELETE CASCADE,
    position        smallint NOT NULL CHECK (position BETWEEN 1 AND 5),
    content_json    jsonb NOT NULL,
    is_correct      boolean NOT NULL DEFAULT false,
    UNIQUE (question_id, position)
);

CREATE INDEX idx_question_alternatives_question ON question_alternatives(question_id);

-- nenhuma questão pode ter duas alternativas corretas (garantido sempre, não só no fim da transação)
CREATE UNIQUE INDEX ux_question_alternatives_one_correct
    ON question_alternatives(question_id) WHERE is_correct;

-- exatamente 5 linhas e exatamente 1 correta: só pode ser checado de forma agregada,
-- por isso usamos um trigger de constraint adiado para o fim da transação (a aplicação
-- sempre cria/atualiza as 5 alternativas de uma questão dentro de uma única transação).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_question_alternatives() RETURNS trigger AS $$
DECLARE
    affected_question_id bigint;
    total_count int;
    correct_count int;
BEGIN
    IF TG_OP = 'DELETE' THEN
        affected_question_id := OLD.question_id;
    ELSE
        affected_question_id := NEW.question_id;
    END IF;

    SELECT count(*), count(*) FILTER (WHERE is_correct)
      INTO total_count, correct_count
      FROM question_alternatives
      WHERE question_id = affected_question_id;

    IF total_count <> 5 THEN
        RAISE EXCEPTION 'question % must have exactly 5 alternatives, has %',
            affected_question_id, total_count;
    END IF;

    IF correct_count <> 1 THEN
        RAISE EXCEPTION 'question % must have exactly 1 correct alternative, has %',
            affected_question_id, correct_count;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE CONSTRAINT TRIGGER trg_check_question_alternatives
    AFTER INSERT OR UPDATE OR DELETE ON question_alternatives
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION check_question_alternatives();

-- ==========================================================================
-- Imagens (compartilhadas por disciplina — seção 13)
-- ==========================================================================

CREATE TABLE images (
    id              bigserial PRIMARY KEY,
    discipline_id   bigint NOT NULL REFERENCES disciplines(id),
    filename        text NOT NULL,
    path            text NOT NULL,
    mime_type       text NOT NULL,
    size_bytes      bigint NOT NULL,
    uploaded_by     bigint NOT NULL REFERENCES users(id),
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_images_discipline ON images(discipline_id);

-- ==========================================================================
-- Aplicações (temporadas de prova — seção 21) e cadernos
--
-- Uma aplicação (ex.: "2026/1") é a campanha. Cada aplicação pode gerar mais
-- de um CADERNO DE PROVAS (ex.: Caderno 1, Caderno 2, Caderno 3) — o mais
-- comum são 2 cadernos, mas o número não é fixo. Cada caderno tem sua
-- própria configuração (disciplinas/assuntos/dificuldades/quantidades),
-- seu próprio congelamento (seção 27) e seu próprio PDF — não a aplicação
-- como um todo. A numeração impressa das questões (1..N) também é por
-- caderno: um caderno com 80 questões é numerado de 1 a 80.
-- ==========================================================================

CREATE TABLE applications (
    id              bigserial PRIMARY KEY,
    name            text NOT NULL UNIQUE,
    description     text NOT NULL DEFAULT '',
    status          text NOT NULL DEFAULT 'RASCUNHO'
                        CHECK (status IN ('RASCUNHO', 'ATIVA', 'ENCERRADA')),
    creator_id      bigint NOT NULL REFERENCES users(id),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE application_booklets (
    id              bigserial PRIMARY KEY,
    application_id  bigint NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    name            text NOT NULL,
    sort_order      smallint NOT NULL DEFAULT 1,
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (application_id, name)
);

CREATE INDEX idx_booklets_application ON application_booklets(application_id);

CREATE TABLE booklet_configurations (
    id                  bigserial PRIMARY KEY,
    booklet_id          bigint NOT NULL UNIQUE REFERENCES application_booklets(id) ON DELETE CASCADE,
    total_questions     integer NOT NULL CHECK (total_questions > 0),
    -- uma vez congelada (seção 27), a configuração deste caderno não pode
    -- mais ser alterada; outros cadernos da mesma aplicação não são afetados
    is_frozen           boolean NOT NULL DEFAULT false,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE booklet_configuration_grade_years (
    configuration_id  bigint NOT NULL REFERENCES booklet_configurations(id) ON DELETE CASCADE,
    grade_year_id     bigint NOT NULL REFERENCES grade_years(id),
    PRIMARY KEY (configuration_id, grade_year_id)
);

-- Regra de cotas flexível (seção 23): quantidade por disciplina, opcionalmente
-- refinada por assunto e/ou dificuldade, sem obrigar todas as combinações.
CREATE TABLE booklet_quota_rules (
    id                  bigserial PRIMARY KEY,
    configuration_id    bigint NOT NULL REFERENCES booklet_configurations(id) ON DELETE CASCADE,
    discipline_id       bigint NOT NULL REFERENCES disciplines(id),
    subject_id          bigint REFERENCES subjects(id),
    difficulty_id       bigint REFERENCES difficulties(id),
    quantity            integer NOT NULL CHECK (quantity > 0)
);

CREATE INDEX idx_quota_rules_configuration ON booklet_quota_rules(configuration_id);

-- ==========================================================================
-- Snapshot de questões usadas em um caderno (seção 26) — preserva a
-- representação exibida no PDF mesmo que a questão original seja editada
-- depois. position_in_booklet é a numeração impressa (1..N), única e
-- sequencial dentro do caderno.
-- ==========================================================================

CREATE TABLE booklet_question_snapshots (
    id                  bigserial PRIMARY KEY,
    booklet_id          bigint NOT NULL REFERENCES application_booklets(id) ON DELETE CASCADE,
    question_id         bigint REFERENCES questions(id) ON DELETE SET NULL,
    position_in_booklet integer NOT NULL CHECK (position_in_booklet > 0),
    discipline_name     text NOT NULL,
    subject_name        text NOT NULL,
    grade_year_name     text NOT NULL,
    difficulty_name     text NOT NULL,
    statement_json      jsonb NOT NULL,
    command_json        jsonb NOT NULL,
    alternatives_json   jsonb NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (booklet_id, position_in_booklet)
);

CREATE INDEX idx_snapshots_booklet ON booklet_question_snapshots(booklet_id);

-- ==========================================================================
-- Geração de documentos/PDF em background (seção 30) — um PDF por caderno.
-- ==========================================================================

CREATE TABLE generated_documents (
    id              bigserial PRIMARY KEY,
    booklet_id      bigint NOT NULL REFERENCES application_booklets(id) ON DELETE CASCADE,
    status          text NOT NULL DEFAULT 'PENDING'
                        CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED')),
    file_path       text,
    error_message   text,
    requested_by    bigint NOT NULL REFERENCES users(id),
    created_at      timestamptz NOT NULL DEFAULT now(),
    completed_at    timestamptz
);

CREATE INDEX idx_generated_documents_booklet ON generated_documents(booklet_id);

-- ==========================================================================
-- Seed: catálogos fixos (seção 32)
-- ==========================================================================

INSERT INTO grade_years (code, name, sort_order) VALUES
    ('1_ANO', '1º ano', 1),
    ('2_ANO', '2º ano', 2),
    ('3_ANO', '3º ano', 3);

INSERT INTO difficulties (code, name, sort_order) VALUES
    ('FACIL',   'Fácil',   1),
    ('MEDIA',   'Média',   2),
    ('DIFICIL', 'Difícil', 3);

INSERT INTO question_statuses (code, name, eligible_for_exam, sort_order) VALUES
    ('RASCUNHO',    'Rascunho',    false, 1),
    ('EM_REVISAO',  'Em revisão',  false, 2),
    ('EM_TESTE',    'Em teste',    false, 3),
    ('TESTADA',     'Testada',     true,  4),
    ('APROVADA',    'Aprovada',    true,  5),
    ('REJEITADA',   'Rejeitada',   false, 6),
    ('ARQUIVADA',   'Arquivada',   false, 7),
    ('OBSOLETA',    'Obsoleta',    false, 8);

INSERT INTO disciplines (code, name) VALUES
    ('LINGUA_PORTUGUESA', 'Língua Portuguesa'),
    ('LINGUA_INGLESA',    'Língua Inglesa'),
    ('FISICA',            'Física'),
    ('QUIMICA',           'Química'),
    ('REDACAO',           'Redação'),
    ('HISTORIA',          'História'),
    ('GEOGRAFIA',         'Geografia'),
    ('MATEMATICA',        'Matemática'),
    ('BIOLOGIA',          'Biologia'),
    ('ARTES',             'Artes'),
    ('EDUCACAO_FISICA',   'Educação Física'),
    ('FILOSOFIA',         'Filosofia'),
    ('SOCIOLOGIA',        'Sociologia');

-- +goose Down

DROP TABLE IF EXISTS generated_documents;
DROP TABLE IF EXISTS booklet_question_snapshots;
DROP TABLE IF EXISTS booklet_quota_rules;
DROP TABLE IF EXISTS booklet_configuration_grade_years;
DROP TABLE IF EXISTS booklet_configurations;
DROP TABLE IF EXISTS application_booklets;
DROP TABLE IF EXISTS applications;
DROP TABLE IF EXISTS images;
DROP TRIGGER IF EXISTS trg_check_question_alternatives ON question_alternatives;
DROP FUNCTION IF EXISTS check_question_alternatives();
DROP TABLE IF EXISTS question_alternatives;
DROP TABLE IF EXISTS questions;
DROP TABLE IF EXISTS subjects;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS disciplines;
DROP TABLE IF EXISTS question_statuses;
DROP TABLE IF EXISTS difficulties;
DROP TABLE IF EXISTS grade_years;
