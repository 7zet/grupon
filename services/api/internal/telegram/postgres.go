package telegram

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool} }
func (s *PostgresStore) SetLanguage(ctx context.Context, chatID int64, language string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO telegram_chats(chat_id,language) VALUES($1,$2) ON CONFLICT(chat_id) DO UPDATE SET language=EXCLUDED.language,updated_at=now()`, chatID, language)
	return err
}
func (s *PostgresStore) Language(ctx context.Context, chatID int64) (string, error) {
	var language string
	err := s.pool.QueryRow(ctx, `SELECT language FROM telegram_chats WHERE chat_id=$1`, chatID).Scan(&language)
	return language, err
}
