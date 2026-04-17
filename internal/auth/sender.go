package auth

import (
	"context"
	"log"
)

type LogCodeSender struct {
	Logger *log.Logger
}

func (s LogCodeSender) SendLoginCode(_ context.Context, phone string, code string) error {
	if s.Logger != nil {
		s.Logger.Printf("auth login code for %s: %s", phone, code)
	}
	return nil
}
