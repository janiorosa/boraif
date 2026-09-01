package users

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound            = errors.New("user not found")
	ErrEmailTaken          = errors.New("email already in use")
	ErrAPIKeyNotConfigured = errors.New("openai api key not configured")
	ErrInvalidReference    = errors.New("invalid discipline reference")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, email, password_hash, role, discipline_id, active, pending_approval
		FROM users
		WHERE email = $1
	`, email).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.DisciplineID, &u.Active, &u.PendingApproval)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}

func (r *Repository) FindByID(ctx context.Context, id int64) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, email, password_hash, role, discipline_id, active, pending_approval
		FROM users
		WHERE id = $1
	`, id).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.DisciplineID, &u.Active, &u.PendingApproval)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return u, nil
}

// List retorna todos os usuários para a tela de administração. O volume é
// pequeno (seção 2.3 — poucos usuários simultâneos, sem necessidade de
// paginação aqui como existe na listagem de questões).
func (r *Repository) List(ctx context.Context) ([]User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, email, password_hash, role, discipline_id, active, pending_approval
		FROM users
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.Role, &u.DisciplineID, &u.Active, &u.PendingApproval); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

// Create insere um novo usuário. Usado pelo comando de bootstrap do
// administrador (cmd/server create-admin), pelo endpoint administrativo de
// criação de usuários e pelo autocadastro de elaboradores — cada chamador
// decide explicitamente Active/PendingApproval, sem valor padrão implícito.
func (r *Repository) Create(ctx context.Context, u User) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash, role, discipline_id, active, pending_approval)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, u.Name, u.Email, u.PasswordHash, u.Role, u.DisciplineID, u.Active, u.PendingApproval).Scan(&id)
	switch {
	case isUniqueViolation(err):
		return 0, ErrEmailTaken
	case isForeignKeyViolation(err):
		return 0, ErrInvalidReference
	default:
		return id, err
	}
}

// Update altera dados administrativos do usuário (nome, e-mail, papel,
// disciplina, ativo/inativo). A senha é alterada separadamente por
// UpdatePassword, pois nem toda edição envolve trocar a senha. Qualquer
// edição pelo formulário administrativo também limpa PendingApproval: se um
// ADMIN está mexendo nesse usuário diretamente, o cadastro deixou de estar
// "pendente" nesse sentido — aprovar/recusar de verdade continua sendo feito
// por Approve/RejectPending, que têm sua própria regra de negócio.
func (r *Repository) Update(ctx context.Context, u User) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users
		SET name = $1, email = $2, role = $3, discipline_id = $4, active = $5,
		    pending_approval = false, updated_at = now()
		WHERE id = $6
	`, u.Name, u.Email, u.Role, u.DisciplineID, u.Active, u.ID)
	if isUniqueViolation(err) {
		return ErrEmailTaken
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) UpdatePassword(ctx context.Context, id int64, passwordHash string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2
	`, passwordHash, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Approve libera o acesso de um autocadastro pendente. Só afeta linhas que
// de fato estão pendentes — não serve para "reativar" uma conta desativada
// pelo caminho normal (isso é o Update comum, com Active=true).
func (r *Repository) Approve(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET active = true, pending_approval = false, updated_at = now()
		WHERE id = $1 AND pending_approval = true
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RejectPending descarta um autocadastro pendente (exclui a linha). Só
// afeta linhas pendentes, de propósito — nunca serve como "excluir usuário"
// genérico para contas já aprovadas/ativas.
func (r *Repository) RejectPending(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1 AND pending_approval = true`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetAPIKey grava a API Key da OpenAI do professor já cifrada (seção 17) —
// este repositório nunca vê nem manuseia a chave em texto puro; isso é
// responsabilidade exclusiva do chamador (handler), usando internal/security.
func (r *Repository) SetAPIKey(ctx context.Context, userID int64, ciphertext, nonce []byte) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET openai_api_key_ciphertext = $1, openai_api_key_nonce = $2, updated_at = now()
		WHERE id = $3
	`, ciphertext, nonce, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// APIKeyFor devolve o ciphertext/nonce cifrados de um usuário, para o
// assistente de IA descriptografar sob demanda (nunca antes disso).
func (r *Repository) APIKeyFor(ctx context.Context, userID int64) (ciphertext, nonce []byte, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT openai_api_key_ciphertext, openai_api_key_nonce FROM users WHERE id = $1
	`, userID).Scan(&ciphertext, &nonce)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if ciphertext == nil || nonce == nil {
		return nil, nil, ErrAPIKeyNotConfigured
	}
	return ciphertext, nonce, nil
}

// HasAPIKey indica só se uma chave está configurada, sem nunca expor nem a
// chave nem o ciphertext — usado pela tela de configuração do professor.
func (r *Repository) HasAPIKey(ctx context.Context, userID int64) (bool, error) {
	var hasKey bool
	err := r.pool.QueryRow(ctx, `
		SELECT openai_api_key_ciphertext IS NOT NULL FROM users WHERE id = $1
	`, userID).Scan(&hasKey)
	return hasKey, err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
