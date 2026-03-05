package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"bilibili-up-admin/internal/model"
	"bilibili-up-admin/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

const (
	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "admin123456"
)

type AuthService struct {
	users    *repository.AdminUserRepository
	sessions *repository.AdminSessionRepository
}

type LoginResult struct {
	Token              string `json:"token"`
	Username           string `json:"username"`
	MustChangePassword bool   `json:"must_change_password"`
}

func NewAuthService(users *repository.AdminUserRepository, sessions *repository.AdminSessionRepository) *AuthService {
	return &AuthService{users: users, sessions: sessions}
}

func (s *AuthService) EnsureDefaultAdmin(ctx context.Context) error {
	existing, err := s.users.First(ctx)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(DefaultAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.users.Create(ctx, &model.AdminUser{
		Username:           DefaultAdminUsername,
		PasswordHash:       string(hash),
		MustChangePassword: true,
	})
}

func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, errors.New("账号或密码不能为空")
	}

	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("账号或密码错误")
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, errors.New("账号或密码错误")
	}

	rawToken, tokenHash, err := newSessionToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := now.Add(7 * 24 * time.Hour)
	if err := s.sessions.DeleteExpired(ctx); err != nil {
		return nil, err
	}
	if err := s.sessions.Create(ctx, &model.AdminSession{
		AdminUserID: user.ID,
		TokenHash:   tokenHash,
		ExpiresAt:   expiresAt,
		LastSeenAt:  &now,
	}); err != nil {
		return nil, err
	}

	user.LastLoginAt = &now
	if err := s.users.Update(ctx, user); err != nil {
		return nil, err
	}

	return &LoginResult{
		Token:              rawToken,
		Username:           user.Username,
		MustChangePassword: user.MustChangePassword,
	}, nil
}

func (s *AuthService) ValidateSession(ctx context.Context, token string) (*model.AdminUser, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("empty token")
	}
	_, tokenHash := splitAndHashToken(token)
	session, err := s.sessions.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if session == nil || session.ExpiresAt.Before(time.Now()) {
		if session != nil {
			_ = s.sessions.DeleteByTokenHash(ctx, tokenHash)
		}
		return nil, errors.New("session expired")
	}

	user, err := s.users.GetByID(ctx, session.AdminUserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		_ = s.sessions.DeleteByTokenHash(ctx, tokenHash)
		return nil, errors.New("user not found")
	}

	now := time.Now()
	session.LastSeenAt = &now
	_ = s.sessions.Update(ctx, session)

	return user, nil
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	_, tokenHash := splitAndHashToken(token)
	if tokenHash == "" {
		return nil
	}
	return s.sessions.DeleteByTokenHash(ctx, tokenHash)
}

func (s *AuthService) ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword string) error {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("用户不存在")
	}

	if len(strings.TrimSpace(newPassword)) < 8 {
		return errors.New("新密码至少 8 位")
	}

	if currentPassword == "" {
		if !user.MustChangePassword {
			return errors.New("当前密码不能为空")
		}
	} else if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)) != nil {
		return errors.New("当前密码错误")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	user.MustChangePassword = false
	return s.users.Update(ctx, user)
}

func newSessionToken() (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("generate token failed: %w", err)
	}
	token := hex.EncodeToString(raw)
	_, hash := splitAndHashToken(token)
	return token, hash, nil
}

func splitAndHashToken(token string) (string, string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", ""
	}
	sum := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(sum[:])
}
