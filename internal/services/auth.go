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
	audit    *repository.AuditRepository
}

func NewAuthService(users *repository.UserRepository, sessions *repository.SessionRepository, audits ...*repository.AuditRepository) *AuthService {
	var audit *repository.AuditRepository
	if len(audits) > 0 {
		audit = audits[0]
	}
	return &AuthService{users: users, sessions: sessions, audit: audit}
}

func (s *AuthService) ListUsers(ctx context.Context) ([]models.User, error) {
	return s.users.List(ctx)
}

func (s *AuthService) CreateUser(ctx context.Context, actorID int64, username, password, role string) (models.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || len(password) < 8 || (role != "admin" && role != "staff") {
		return models.User{}, fmt.Errorf("%w: user", ErrValidation)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, err
	}
	user, err := s.users.Create(ctx, username, string(hash), role)
	if err != nil {
		return user, err
	}
	if s.audit != nil {
		record := user.ID
		if err := s.audit.Insert(ctx, models.AuditEntry{UserID: actorID, Action: "user_created", Module: "users", RecordID: &record, Description: user.Username}); err != nil {
			return models.User{}, err
		}
	}
	return user, nil
}

func (s *AuthService) UpdateUser(ctx context.Context, actorID, id int64, username, password, role string) error {
	username = strings.TrimSpace(username)
	if username == "" || (role != "admin" && role != "staff") || (password != "" && len(password) < 8) {
		return fmt.Errorf("%w: user", ErrValidation)
	}
	var hash *string
	if password != "" {
		v, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		h := string(v)
		hash = &h
	}
	if err := s.users.Update(ctx, id, username, role, hash); err != nil {
		return err
	}
	if s.audit != nil {
		record := id
		if err := s.audit.Insert(ctx, models.AuditEntry{UserID: actorID, Action: "user_updated", Module: "users", RecordID: &record, Description: username}); err != nil {
			return err
		}
	}
	return nil
}

func (s *AuthService) SetUserActive(ctx context.Context, actorID, id int64, active bool) error {
	if err := s.users.SetActive(ctx, id, active); err != nil {
		return err
	}
	if s.audit != nil {
		record := id
		action := "user_deactivated"
		if active {
			action = "user_activated"
		}
		if err := s.audit.Insert(ctx, models.AuditEntry{UserID: actorID, Action: action, Module: "users", RecordID: &record}); err != nil {
			return err
		}
	}
	return nil
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
