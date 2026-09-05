package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"supermarket/internal/models"
	"supermarket/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrBootstrapDisabled  = errors.New("bootstrap disabled")
)

type AuthService struct {
	users    *repository.UserRepository
	sessions *repository.SessionRepository
}

func NewAuthService(users *repository.UserRepository, sessions *repository.SessionRepository) *AuthService {
	return &AuthService{users: users, sessions: sessions}
}

func (s *AuthService) Bootstrap(ctx context.Context, username, password string) (models.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || len(password) < 8 {
		return models.User{}, fmt.Errorf("invalid bootstrap input")
	}
	count, err := s.users.Count(ctx)
	if err != nil {
		return models.User{}, err
	}
	if count != 0 {
		return models.User{}, ErrBootstrapDisabled
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, fmt.Errorf("hash password: %w", err)
	}
	return s.users.Create(ctx, username, string(hash), "admin")
}

func (s *AuthService) Login(ctx context.Context, username, password string) (models.User, string, time.Time, error) {
	id, hash, _, active, err := s.users.GetCredentials(ctx, strings.TrimSpace(username))
	if err != nil || !active {
		return models.User{}, "", time.Time{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return models.User{}, "", time.Time{}, ErrInvalidCredentials
	}
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return models.User{}, "", time.Time{}, err
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return models.User{}, "", time.Time{}, fmt.Errorf("generate session token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	if err := s.sessions.Create(ctx, user.ID, hashToken(token), expiresAt); err != nil {
		return models.User{}, "", time.Time{}, fmt.Errorf("create session: %w", err)
	}
	return user, token, expiresAt, nil
}

func (s *AuthService) UserForToken(ctx context.Context, token string) (models.User, error) {
	id, err := s.sessions.UserIDByTokenHash(ctx, hashToken(token))
	if err != nil {
		return models.User{}, err
	}
	return s.users.GetByID(ctx, id)
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, hashToken(token))
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
