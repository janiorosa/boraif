-- +goose Up

-- ==========================================================================
-- Configuração padrão (seção 22): modelo copiado para a configuração de
-- cada novo CADERNO (não da aplicação — seção 21.1), que depois evolui de
-- forma independente. Tratada como uma tabela singleton: sempre há no
-- máximo uma linha, criada/atualizada pelo mesmo endpoint (upsert na
-- aplicação, não com uma constraint especial aqui).
-- ==========================================================================

CREATE TABLE default_configurations (
    id              bigserial PRIMARY KEY,
    total_questions integer NOT NULL DEFAULT 0,
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE default_configuration_grade_years (
    default_configuration_id bigint NOT NULL REFERENCES default_configurations(id) ON DELETE CASCADE,
    grade_year_id             bigint NOT NULL REFERENCES grade_years(id),
    PRIMARY KEY (default_configuration_id, grade_year_id)
);

-- Mesmo modelo de cota "achatado" da seção 23/booklet_quota_rules: cada
-- linha é um critério final (disciplina + opcionalmente assunto/dificuldade)
-- que soma ao total, nunca uma linha-resumo sobre outras linhas mais
-- específicas da mesma disciplina.
CREATE TABLE default_quota_rules (
    id                        bigserial PRIMARY KEY,
    default_configuration_id bigint NOT NULL REFERENCES default_configurations(id) ON DELETE CASCADE,
    discipline_id             bigint NOT NULL REFERENCES disciplines(id),
    subject_id                bigint REFERENCES subjects(id),
    difficulty_id             bigint REFERENCES difficulties(id),
    quantity                  integer NOT NULL CHECK (quantity > 0)
);

CREATE INDEX idx_default_quota_rules_configuration ON default_quota_rules(default_configuration_id);

-- +goose Down

DROP TABLE IF EXISTS default_quota_rules;
DROP TABLE IF EXISTS default_configuration_grade_years;
DROP TABLE IF EXISTS default_configurations;
