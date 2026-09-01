-- Assuntos iniciais para as 13 disciplinas do Ensino Médio (seção 14).
--
-- Este script NÃO é uma migration (não fica em backend/migrations/, não
-- roda sozinho na subida do backend) — é para ser executado manualmente,
-- quando/se você quiser popular a base de assuntos de uma vez, em vez de
-- cadastrar um por um pela tela. Idempotente: pode rodar mais de uma vez
-- sem duplicar nada (ON CONFLICT casa com o mesmo índice único que a
-- aplicação usa para impedir nomes repetidos/case-insensitive na mesma
-- disciplina).
--
-- Como rodar (a partir da raiz do projeto, com os containers no ar):
--   docker compose exec -T postgres psql -U SEU_POSTGRES_USER -d SEU_POSTGRES_DB < scripts/seed_subjects.sql
--
-- (troque SEU_POSTGRES_USER/SEU_POSTGRES_DB pelos valores do seu .env —
-- os padrões do .env.example são "boraif" para os dois).

INSERT INTO subjects (discipline_id, name)
SELECT d.id, s.name
FROM (VALUES
    -- Língua Portuguesa
    ('LINGUA_PORTUGUESA', 'Gramática'),
    ('LINGUA_PORTUGUESA', 'Literatura'),
    ('LINGUA_PORTUGUESA', 'Interpretação de Texto'),

    -- Língua Inglesa
    ('LINGUA_INGLESA', 'Gramática'),
    ('LINGUA_INGLESA', 'Compreensão Textual'),
    ('LINGUA_INGLESA', 'Vocabulário'),

    -- Física
    ('FISICA', 'Mecânica'),
    ('FISICA', 'Termologia'),
    ('FISICA', 'Óptica'),
    ('FISICA', 'Ondulatória'),
    ('FISICA', 'Eletromagnetismo'),

    -- Química
    ('QUIMICA', 'Química Geral'),
    ('QUIMICA', 'Físico-Química'),
    ('QUIMICA', 'Química Orgânica'),
    ('QUIMICA', 'Química Inorgânica'),

    -- Redação (disciplina própria — um assunto só é suficiente)
    ('REDACAO', 'Redação'),

    -- História
    ('HISTORIA', 'História Geral'),
    ('HISTORIA', 'História do Brasil'),
    ('HISTORIA', 'Idade Antiga e Medieval'),
    ('HISTORIA', 'Idade Moderna'),
    ('HISTORIA', 'Idade Contemporânea'),

    -- Geografia
    ('GEOGRAFIA', 'Geografia Física'),
    ('GEOGRAFIA', 'Geografia Humana'),
    ('GEOGRAFIA', 'Geopolítica'),
    ('GEOGRAFIA', 'Cartografia'),
    ('GEOGRAFIA', 'Geografia do Brasil'),

    -- Matemática (nomes do Ensino Médio, não da graduação — ex.: "Matrizes,
    -- Determinantes e Sistemas Lineares" em vez de "Álgebra Linear")
    ('MATEMATICA', 'Funções'),
    ('MATEMATICA', 'Trigonometria'),
    ('MATEMATICA', 'Matemática Financeira'),
    ('MATEMATICA', 'Matrizes, Determinantes e Sistemas Lineares'),
    ('MATEMATICA', 'Geometria Plana'),
    ('MATEMATICA', 'Geometria Espacial'),
    ('MATEMATICA', 'Geometria Analítica'),
    ('MATEMATICA', 'Estatística e Probabilidade'),
    ('MATEMATICA', 'Progressões (PA e PG)'),
    ('MATEMATICA', 'Números Complexos e Polinômios'),

    -- Biologia
    ('BIOLOGIA', 'Citologia'),
    ('BIOLOGIA', 'Genética'),
    ('BIOLOGIA', 'Ecologia'),
    ('BIOLOGIA', 'Fisiologia Humana'),
    ('BIOLOGIA', 'Evolução'),

    -- Artes
    ('ARTES', 'Artes Visuais'),
    ('ARTES', 'Música'),
    ('ARTES', 'Teatro'),
    ('ARTES', 'Dança'),
    ('ARTES', 'História da Arte'),

    -- Educação Física
    ('EDUCACAO_FISICA', 'Esportes'),
    ('EDUCACAO_FISICA', 'Ginástica'),
    ('EDUCACAO_FISICA', 'Jogos e Lutas'),
    ('EDUCACAO_FISICA', 'Saúde e Qualidade de Vida'),

    -- Filosofia
    ('FILOSOFIA', 'Filosofia Antiga'),
    ('FILOSOFIA', 'Filosofia Moderna'),
    ('FILOSOFIA', 'Filosofia Contemporânea'),
    ('FILOSOFIA', 'Ética'),
    ('FILOSOFIA', 'Política'),

    -- Sociologia
    ('SOCIOLOGIA', 'Fundamentos da Sociologia'),
    ('SOCIOLOGIA', 'Cultura e Sociedade'),
    ('SOCIOLOGIA', 'Cidadania e Direitos'),
    ('SOCIOLOGIA', 'Movimentos Sociais'),
    ('SOCIOLOGIA', 'Trabalho e Sociedade')
) AS s(discipline_code, name)
JOIN disciplines d ON d.code = s.discipline_code
ON CONFLICT (discipline_id, lower(name)) DO NOTHING;

-- Conferência rápida: quantos assuntos cada disciplina ficou tendo.
SELECT d.name AS disciplina, count(s.id) AS total_assuntos
FROM disciplines d
LEFT JOIN subjects s ON s.discipline_id = d.id
GROUP BY d.name
ORDER BY d.name;
