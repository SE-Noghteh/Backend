package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"zedlancer/backend/bootstrap"
)

type FerzzSender struct {
	token  string
	client *http.Client
}

func NewFerzzSender() *FerzzSender {
	return &FerzzSender{token: bootstrap.Run().Env.SMS.Token, client: &http.Client{Timeout: 10 * time.Second}}
}
func (s *FerzzSender) SendOTP(ctx context.Context, destination, code string) error {
	if s.token == "" {
		return fmt.Errorf("SMS token is not configured")
	}
	request := struct {
		Token       string            `json:"token"`
		Destination string            `json:"destination"`
		Action      string            `json:"action"`
		Payload     map[string]string `json:"payload"`
	}{s.token, destination, "sms", map[string]string{"code": code}}
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://gateway.ferzz.ir/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("SMS delivery failed")
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("SMS delivery returned status %d", response.StatusCode)
	}
	return nil
}
