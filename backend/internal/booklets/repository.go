package booklets

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound         = errors.New("booklet not found")
	ErrDuplicateName    = errors.New("booklet with this name already exists in the application")
	ErrInvalidReference = errors.New("invalid application/discipline/subject/difficulty/grade year reference")
	ErrFrozen           = errors.New("booklet configuration is frozen")
	ErrQuotaMismatch    = errors.New("sum of quota rules does not match total questions")
	ErrInvalidVariants  = errors.New("variant count must be between 1 and 4")
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) ListForApplication(ctx context.Context, applicationID int64) ([]Booklet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, application_id, name, sort_order
		FROM application_booklets
		WHERE application_id = $1
		ORDER BY sort_order
	`, applicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Booklet
	for rows.Next() {
		var b Booklet
		if err := rows.Scan(&b.ID, &b.ApplicationID, &b.Name, &b.SortOrder); err != nil {
			return nil, err
		}
		result = append(result, b)
	}
	return result, rows.Err()
}

func (r *Repository) FindByID(ctx context.Context, id int64) (Booklet, error) {
	var b Booklet
	err := r.pool.QueryRow(ctx, `
		SELECT id, application_id, name, sort_order FROM application_booklets WHERE id = $1
	`, id).Scan(&b.ID, &b.ApplicationID, &b.Name, &b.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return Booklet{}, ErrNotFound
	}
	return b, err
}

// Create grava um novo caderno da aplicação e já copia a configuração
// padrão (seção 22) para a configuração dele, de forma independente —
// alterar o caderno depois não afeta o padrão, e vice-versa.
func (r *Repository) Create(ctx context.Context, applicationID int64, name string) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	var sortOrder int16
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(sort_order), 0) + 1 FROM application_booklets WHERE application_id = $1
	`, applicationID).Scan(&sortOrder); err != nil {
		return 0, err
	}

	var bookletID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO application_booklets (application_id, name, sort_order)
		VALUES ($1, $2, $3) RETURNING id
	`, applicationID, name, sortOrder).Scan(&bookletID)
	if isUniqueViolation(err) {
		return 0, ErrDuplicateName
	}
	if isForeignKeyViolation(err) {
		return 0, ErrInvalidReference
	}
	if err != nil {
		return 0, err
	}

	// seção 22: total_questions da configuração padrão, ou 1 (mínimo aceito
	// pela constraint) se nenhum padrão foi definido ainda — o
	// gestor/admin ajusta antes de gerar qualquer coisa.
	var hasDefault bool
	var defaultConfigID int64
	var defaultTotal int
	err = tx.QueryRow(ctx, `SELECT id, total_questions FROM default_configurations ORDER BY id LIMIT 1`).
		Scan(&defaultConfigID, &defaultTotal)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		defaultTotal = 1
	case err != nil:
		return 0, err
	default:
		hasDefault = true
		if defaultTotal <= 0 {
			defaultTotal = 1
		}
	}

	var configID int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO booklet_configurations (booklet_id, total_questions, is_frozen)
		VALUES ($1, $2, false) RETURNING id
	`, bookletID, defaultTotal).Scan(&configID); err != nil {
		return 0, err
	}

	if hasDefault {
		if _, err := tx.Exec(ctx, `
			INSERT INTO booklet_configuration_grade_years (configuration_id, grade_year_id)
			SELECT $1, grade_year_id FROM default_configuration_grade_years WHERE default_configuration_id = $2
		`, configID, defaultConfigID); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO booklet_quota_rules (configuration_id, discipline_id, subject_id, difficulty_id, quantity)
			SELECT $1, discipline_id, subject_id, difficulty_id, quantity FROM default_quota_rules WHERE default_configuration_id = $2
		`, configID, defaultConfigID); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return bookletID, nil
}

func (r *Repository) GetConfiguration(ctx context.Context, bookletID int64) (Configuration, error) {
	var cfg Configuration
	cfg.BookletID = bookletID
	err := r.pool.QueryRow(ctx, `
		SELECT id, total_questions, variant_count, is_frozen FROM booklet_configurations WHERE booklet_id = $1
	`, bookletID).Scan(&cfg.ID, &cfg.TotalQuestions, &cfg.VariantCount, &cfg.IsFrozen)
	if errors.Is(err, pgx.ErrNoRows) {
		return Configuration{}, ErrNotFound
	}
	if err != nil {
		return Configuration{}, err
	}

	years, err := r.gradeYearsFor(ctx, "booklet_configuration_grade_years", "configuration_id", cfg.ID)
	if err != nil {
		return Configuration{}, err
	}
	cfg.GradeYearIDs = years

	rules, err := r.quotaRulesFor(ctx, "booklet_quota_rules", "configuration_id", cfg.ID)
	if err != nil {
		return Configuration{}, err
	}
	cfg.QuotaRules = rules

	return cfg, nil
}

// UpdateConfiguration substitui os anos e as cotas do caderno por completo
// (o conjunto é pequeno e mantido pelo gestor/admin numa tela só — apagar e
// reinserir dentro da transação é mais simples que calcular um diff).
func (r *Repository) UpdateConfiguration(ctx context.Context, bookletID int64, totalQuestions, variantCount int, gradeYearIDs []int64, quotas []QuotaRule) error {
	sum := 0
	for _, q := range quotas {
		sum += q.Quantity
	}
	if sum != totalQuestions {
		return fmt.Errorf("%w: soma das cotas é %d, total informado é %d", ErrQuotaMismatch, sum, totalQuestions)
	}
	if variantCount < 1 || variantCount > 4 {
		return ErrInvalidVariants
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var configID int64
	var isFrozen bool
	err = tx.QueryRow(ctx, `
		SELECT id, is_frozen FROM booklet_configurations WHERE booklet_id = $1
	`, bookletID).Scan(&configID, &isFrozen)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if isFrozen {
		return ErrFrozen
	}

	if _, err := tx.Exec(ctx, `
		UPDATE booklet_configurations SET total_questions = $1, variant_count = $2, updated_at = now() WHERE id = $3
	`, totalQuestions, variantCount, configID); err != nil {
		return err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM booklet_configuration_grade_years WHERE configuration_id = $1`, configID); err != nil {
		return err
	}
	for _, gy := range gradeYearIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO booklet_configuration_grade_years (configuration_id, grade_year_id) VALUES ($1, $2)
		`, configID, gy); err != nil {
			if isForeignKeyViolation(err) {
				return ErrInvalidReference
			}
			return err
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM booklet_quota_rules WHERE configuration_id = $1`, configID); err != nil {
		return err
	}
	for _, q := range quotas {
		if _, err := tx.Exec(ctx, `
			INSERT INTO booklet_quota_rules (configuration_id, discipline_id, subject_id, difficulty_id, quantity)
			VALUES ($1, $2, $3, $4, $5)
		`, configID, q.DisciplineID, q.SubjectID, q.DifficultyID, q.Quantity); err != nil {
			if isForeignKeyViolation(err) {
				return ErrInvalidReference
			}
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetDefaultConfiguration lê a configuração padrão (seção 22). Devolve uma
// Configuration vazia (sem erro) se nenhum padrão foi definido ainda.
func (r *Repository) GetDefaultConfiguration(ctx context.Context) (Configuration, error) {
	var cfg Configuration
	err := r.pool.QueryRow(ctx, `SELECT id, total_questions FROM default_configurations ORDER BY id LIMIT 1`).
		Scan(&cfg.ID, &cfg.TotalQuestions)
	if errors.Is(err, pgx.ErrNoRows) {
		return Configuration{}, nil
	}
	if err != nil {
		return Configuration{}, err
	}

	years, err := r.gradeYearsFor(ctx, "default_configuration_grade_years", "default_configuration_id", cfg.ID)
	if err != nil {
		return Configuration{}, err
	}
	cfg.GradeYearIDs = years

	rules, err := r.quotaRulesFor(ctx, "default_quota_rules", "default_configuration_id", cfg.ID)
	if err != nil {
		return Configuration{}, err
	}
	cfg.QuotaRules = rules

	return cfg, nil
}

// SetDefaultConfiguration faz upsert da linha singleton (seção 22): cria se
// ainda não existir, atualiza se já existir. Alterar o padrão nunca toca
// nos cadernos já criados — eles já têm sua própria cópia independente.
func (r *Repository) SetDefaultConfiguration(ctx context.Context, totalQuestions int, gradeYearIDs []int64, quotas []QuotaRule) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var id int64
	err = tx.QueryRow(ctx, `SELECT id FROM default_configurations ORDER BY id LIMIT 1`).Scan(&id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if err := tx.QueryRow(ctx, `
			INSERT INTO default_configurations (total_questions) VALUES ($1) RETURNING id
		`, totalQuestions).Scan(&id); err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		if _, err := tx.Exec(ctx, `
			UPDATE default_configurations SET total_questions = $1, updated_at = now() WHERE id = $2
		`, totalQuestions, id); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM default_configuration_grade_years WHERE default_configuration_id = $1`, id); err != nil {
		return err
	}
	for _, gy := range gradeYearIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO default_configuration_grade_years (default_configuration_id, grade_year_id) VALUES ($1, $2)
		`, id, gy); err != nil {
			if isForeignKeyViolation(err) {
				return ErrInvalidReference
			}
			return err
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM default_quota_rules WHERE default_configuration_id = $1`, id); err != nil {
		return err
	}
	for _, q := range quotas {
		if _, err := tx.Exec(ctx, `
			INSERT INTO default_quota_rules (default_configuration_id, discipline_id, subject_id, difficulty_id, quantity)
			VALUES ($1, $2, $3, $4, $5)
		`, id, q.DisciplineID, q.SubjectID, q.DifficultyID, q.Quantity); err != nil {
			if isForeignKeyViolation(err) {
				return ErrInvalidReference
			}
			return err
		}
	}

	return tx.Commit(ctx)
}

// ValidateAvailability implementa a seção 24: para cada linha de cota do
// caderno, conta quantas questões elegíveis (status com eligible_for_exam,
// seção 25) existem para aquele critério dentre os anos configurados.
//
// Simplificação deliberada (documentada no README): a contagem é feita por
// linha de forma independente, sem resolver sobreposição entre linhas que
// concorreriam pela mesma questão — o modelo assume cotas não sobrepostas
// (seção 23), que é como a tela de configuração as constrói.
func (r *Repository) ValidateAvailability(ctx context.Context, bookletID int64) ([]AvailabilityItem, error) {
	var configID int64
	if err := r.pool.QueryRow(ctx, `
		SELECT id FROM booklet_configurations WHERE booklet_id = $1
	`, bookletID).Scan(&configID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	years, err := r.gradeYearsFor(ctx, "booklet_configuration_grade_years", "configuration_id", configID)
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT d.name, sub.name, df.name, qr.discipline_id, qr.subject_id, qr.difficulty_id, qr.quantity
		FROM booklet_quota_rules qr
		JOIN disciplines d ON d.id = qr.discipline_id
		LEFT JOIN subjects sub ON sub.id = qr.subject_id
		LEFT JOIN difficulties df ON df.id = qr.difficulty_id
		WHERE qr.configuration_id = $1
	`, configID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []AvailabilityItem
	for rows.Next() {
		var disciplineName string
		var subjectName, difficultyName *string
		var disciplineID int64
		var subjectID, difficultyID *int64
		var quantity int
		if err := rows.Scan(&disciplineName, &subjectName, &difficultyName,
			&disciplineID, &subjectID, &difficultyID, &quantity); err != nil {
			return nil, err
		}

		available, err := r.countAvailable(ctx, disciplineID, subjectID, difficultyID, years)
		if err != nil {
			return nil, err
		}

		item := AvailabilityItem{DisciplineName: disciplineName, Requested: quantity, Available: available}
		if subjectName != nil {
			item.SubjectName = *subjectName
		}
		if difficultyName != nil {
			item.DifficultyName = *difficultyName
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) countAvailable(ctx context.Context, disciplineID int64, subjectID, difficultyID *int64, gradeYearIDs []int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*)
		FROM questions q
		JOIN question_statuses st ON st.id = q.status_id
		WHERE q.discipline_id = $1
		  AND ($2::bigint IS NULL OR q.subject_id = $2)
		  AND ($3::bigint IS NULL OR q.difficulty_id = $3)
		  AND q.grade_year_id = ANY($4)
		  AND st.eligible_for_exam
	`, disciplineID, subjectID, difficultyID, gradeYearIDs).Scan(&count)
	return count, err
}

func (r *Repository) gradeYearsFor(ctx context.Context, table, column string, id int64) ([]int64, error) {
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`SELECT grade_year_id FROM %s WHERE %s = $1`, table, column), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var years []int64
	for rows.Next() {
		var y int64
		if err := rows.Scan(&y); err != nil {
			return nil, err
		}
		years = append(years, y)
	}
	return years, rows.Err()
}

func (r *Repository) quotaRulesFor(ctx context.Context, table, column string, id int64) ([]QuotaRule, error) {
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, discipline_id, subject_id, difficulty_id, quantity FROM %s WHERE %s = $1
	`, table, column), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []QuotaRule
	for rows.Next() {
		var q QuotaRule
		if err := rows.Scan(&q.ID, &q.DisciplineID, &q.SubjectID, &q.DifficultyID, &q.Quantity); err != nil {
			return nil, err
		}
		rules = append(rules, q)
	}
	return rules, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
