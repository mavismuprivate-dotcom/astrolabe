package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"

	"astrolabe/internal/storage"
)

var (
	ErrInvalidPhone = errors.New("invalid phone")
	ErrInvalidCode  = errors.New("invalid code")
)

var phonePattern = regexp.MustCompile(`^1\d{10}$`)

type CodeSender interface {
	SendLoginCode(ctx context.Context, phone string, code string) error
}

type CodeGenerator func() (string, error)

type Service struct {
	store    storage.AuthStore
	sender   CodeSender
	now      func() time.Time
	generate CodeGenerator
}

func NewService(store storage.AuthStore, sender CodeSender, now func() time.Time, generate CodeGenerator) *Service {
	if now == nil {
		now = time.Now
	}
	if generate == nil {
		generate = defaultCodeGenerator
	}
	return &Service{
		store:    store,
		sender:   sender,
		now:      now,
		generate: generate,
	}
}

func (s *Service) RequestCode(ctx context.Context, phone string) error {
	phone = normalizePhone(phone)
	if !phonePattern.MatchString(phone) {
		return ErrInvalidPhone
	}
	code, err := s.generate()
	if err != nil {
		return err
	}
	if err := s.store.SaveLoginCode(ctx, phone, hashCode(phone, code), s.now().UTC().Add(10*time.Minute)); err != nil {
		return err
	}
	if s.sender != nil {
		return s.sender.SendLoginCode(ctx, phone, code)
	}
	return nil
}

func (s *Service) VerifyCode(ctx context.Context, phone string, code string) (storage.User, storage.AuthSession, error) {
	phone = normalizePhone(phone)
	if !phonePattern.MatchString(phone) {
		return storage.User{}, storage.AuthSession{}, ErrInvalidPhone
	}
	if len(code) != 6 {
		return storage.User{}, storage.AuthSession{}, ErrInvalidCode
	}

	ok, err := s.store.ConsumeLoginCode(ctx, phone, hashCode(phone, code), s.now().UTC())
	if err != nil {
		return storage.User{}, storage.AuthSession{}, err
	}
	if !ok {
		return storage.User{}, storage.AuthSession{}, ErrInvalidCode
	}

	user, err := s.store.GetOrCreateUserByPhone(ctx, phone)
	if err != nil {
		return storage.User{}, storage.AuthSession{}, err
	}
	session, err := s.store.CreateAuthSession(ctx, user.ID, s.now().UTC().Add(30*24*time.Hour))
	if err != nil {
		return storage.User{}, storage.AuthSession{}, err
	}
	return user, session, nil
}

func (s *Service) CurrentUser(ctx context.Context, sessionID string) (storage.User, error) {
	if sessionID == "" {
		return storage.User{}, storage.ErrAuthSessionNotFound
	}
	return s.store.GetUserByAuthSession(ctx, sessionID, s.now().UTC())
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.store.DeleteAuthSession(ctx, sessionID)
}

func normalizePhone(phone string) string {
	return regexp.MustCompile(`\D+`).ReplaceAllString(phone, "")
}

func hashCode(phone string, code string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", phone, code)))
	return hex.EncodeToString(sum[:])
}
