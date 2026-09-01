-- +goose Up

-- Autocadastro de elaboradores com aprovação do administrador (requisito
-- adicionado depois da especificação original, seguindo o mesmo padrão de
-- autorização já usado no resto do sistema): um professor pode se
-- cadastrar sozinho, mas a conta nasce inativa e marcada como pendente —
-- só um ADMIN aprovando (ou recusando) libera ou descarta o acesso.
ALTER TABLE users ADD COLUMN pending_approval boolean NOT NULL DEFAULT false;

-- +goose Down

ALTER TABLE users DROP COLUMN pending_approval;
