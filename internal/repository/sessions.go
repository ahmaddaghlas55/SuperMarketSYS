package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type SessionRepository struct{ db *sql.DB }

func NewSessionRepository(db *sql.DB) *SessionRepository { return &SessionRepository{db: db} }

func (r *SessionRepository) Create(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO auth_sessions(user_id, token_hash, expires_at) VALUES (?, ?, ?)`,
		userID, tokenHash, expiresAt.UTC().Format(time.RFC3339))
	return err
}

func (r *SessionRepository) UserIDByTokenHash(ctx context.Context, tokenHash string) (int64, error) {
	var userID int64
	var expiresAt string
	err := r.db.QueryRowContext(ctx, `
		SELECT s.user_id, s.expires_at
		FROM auth_sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND u.active = 1`, tokenHash).Scan(&userID, &expiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("get session: %w", err)
	}
	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return 0, fmt.Errorf("parse session expiry: %w", err)
	}
	if !time.Now().Before(expiry) {
		return 0, ErrNotFound
	}
	return userID, nil
}

func (r *SessionRepository) Delete(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM auth_sessions WHERE token_hash = ?`, tokenHash)
	return err
}
