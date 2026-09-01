// Session implementa autenticação por sessão server-side: o cookie carrega
// somente um token aleatório de alta entropia, e o backend guarda no banco o
// hash SHA-256 desse token (nunca o token em si). Isso permite revogar uma
// sessão instantaneamente (ex.: desativar um usuário) sem a complexidade de
// refresh tokens de um esquema JWT, e sem depender de Redis.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sessionDuration = 7 * 24 * time.Hour

var ErrSessionNotFound = errors.New("session not found or expired")

type SessionStore struct {
	pool *pgxpool.Pool
}

func NewSessionStore(pool *pgxpool.Pool) *SessionStore {
	return &SessionStore{pool: pool}
}

// Create gera um novo token de sessão para o usuário e retorna o token bruto
// (a ser colocado no cookie). Apenas o hash do token é persistido.
func (s *SessionStore) Create(ctx context.Context, userID int64) (token string, expiresAt time.Time, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", time.Time{}, err
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	expiresAt = time.Now().Add(sessionDuration)

	hash := hashToken(token)
	_, err = s.pool.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, hash, expiresAt)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

// UserIDForToken resolve um token de cookie para o id do usuário, se a sessão
// existir e ainda não tiver expirado.
func (s *SessionStore) UserIDForToken(ctx context.Context, token string) (int64, error) {
	hash := hashToken(token)
	var userID int64
	err := s.pool.QueryRow(ctx, `
		SELECT user_id FROM sessions
		WHERE token_hash = $1 AND expires_at > now()
	`, hash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrSessionNotFound
	}
	return userID, err
}

func (s *SessionStore) Revoke(ctx context.Context, token string) error {
	hash := hashToken(token)
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hash)
	return err
}

func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
