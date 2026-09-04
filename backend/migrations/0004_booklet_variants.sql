-- +goose Up

-- "Tipos de prova" (requisito acrescentado depois da especificação
-- original): um caderno pode ter de 1 a 4 tipos — o mesmo conjunto de
-- questões para todos, só a ordem (agrupada por disciplina, nunca
-- misturando disciplinas) e a ordem das alternativas mudam de tipo para
-- tipo. Cada tipo tem seu próprio gabarito, gravado no banco
-- independentemente de qualquer PDF existir.

ALTER TABLE booklet_configurations
    ADD COLUMN variant_count smallint NOT NULL DEFAULT 2 CHECK (variant_count BETWEEN 1 AND 4);

CREATE TABLE booklet_variants (
    id              bigserial PRIMARY KEY,
    booklet_id      bigint NOT NULL REFERENCES application_booklets(id) ON DELETE CASCADE,
    variant_number  smallint NOT NULL CHECK (variant_number BETWEEN 1 AND 4),
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (booklet_id, variant_number)
);

-- Uma linha por questão por tipo: onde ela é impressa naquele tipo
-- (position_in_variant), em que ordem as 5 alternativas aparecem
-- (alternative_order — as letras ORIGINAIS do snapshot, na nova ordem de
-- exibição) e qual letra fica correta NAQUELE tipo (correct_letter). Isso
-- É o gabarito: (position_in_variant, correct_letter) de todas as linhas
-- de um tipo, nessa ordem, já é a resposta certa de cada questão impressa.
CREATE TABLE booklet_variant_questions (
    id                  bigserial PRIMARY KEY,
    variant_id          bigint NOT NULL REFERENCES booklet_variants(id) ON DELETE CASCADE,
    snapshot_id         bigint NOT NULL REFERENCES booklet_question_snapshots(id) ON DELETE CASCADE,
    position_in_variant integer NOT NULL CHECK (position_in_variant > 0),
    alternative_order   jsonb NOT NULL,
    correct_letter      char(1) NOT NULL CHECK (correct_letter IN ('A', 'B', 'C', 'D', 'E')),
    UNIQUE (variant_id, position_in_variant),
    UNIQUE (variant_id, snapshot_id)
);

CREATE INDEX idx_variant_questions_variant ON booklet_variant_questions(variant_id);

-- generated_documents passa a existir por VARIANTE, não só por caderno —
-- cada tipo gera sua própria prova (kind=EXAM) e seu próprio gabarito em
-- PDF (kind=ANSWER_KEY). variant_id fica opcional só para não quebrar
-- registros que já existiam antes dessa migration (gerados quando um
-- caderno só tinha um PDF no total, sem o conceito de tipo).
ALTER TABLE generated_documents
    ADD COLUMN variant_id bigint REFERENCES booklet_variants(id) ON DELETE CASCADE,
    ADD COLUMN kind text NOT NULL DEFAULT 'EXAM' CHECK (kind IN ('EXAM', 'ANSWER_KEY'));

CREATE INDEX idx_generated_documents_variant ON generated_documents(variant_id);

-- +goose Down

ALTER TABLE generated_documents
    DROP COLUMN variant_id,
    DROP COLUMN kind;

DROP TABLE IF EXISTS booklet_variant_questions;
DROP TABLE IF EXISTS booklet_variants;

ALTER TABLE booklet_configurations DROP COLUMN variant_count;
