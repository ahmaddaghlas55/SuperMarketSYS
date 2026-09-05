package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"supermarket/internal/models"
)

var ErrNotFound = errors.New("not found")

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) Count(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

func (r *UserRepository) Create(ctx context.Context, username, passwordHash, role string) (models.User, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO users(username, password_hash, role) VALUES (?, ?, ?)`,
		username, passwordHash, role)
	if err != nil {
		return models.User{}, fmt.Errorf("create user: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return models.User{}, fmt.Errorf("read user id: %w", err)
	}
	return r.GetByID(ctx, id)
}

func (r *UserRepository) GetCredentials(ctx context.Context, username string) (int64, string, string, bool, error) {
	var id int64
	var passwordHash, role string
	var active int
	err := r.db.QueryRowContext(ctx,
		`SELECT id, password_hash, role, active FROM users WHERE username = ?`,
		username).Scan(&id, &passwordHash, &role, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", "", false, ErrNotFound
	}
	if err != nil {
		return 0, "", "", false, fmt.Errorf("get credentials: %w", err)
	}
	return id, passwordHash, role, active == 1, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (models.User, error) {
	var user models.User
	var active int
	var created string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, role, active, created_at FROM users WHERE id = ?`, id).
		Scan(&user.ID, &user.Username, &user.Role, &active, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, ErrNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("get user: %w", err)
	}
	user.Active = active == 1
	user.CreatedAt, err = time.Parse("2006-01-02 15:04:05", created)
	if err != nil {
		return models.User{}, fmt.Errorf("parse user created_at: %w", err)
	}
	return user, nil
}
