package sms

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestFerzzRequestContract(t *testing.T) {
	sender := &FerzzSender{token: "test-token", client: &http.Client{Transport: testTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.String() != "https://gateway.ferzz.ir/send" || r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("unexpected SMS request: %s %s", r.Method, r.URL)
		}
		var body struct {
			Token, Destination, Action string
			Payload                    map[string]string
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Token != "test-token" || body.Destination != "989123456789" || body.Action != "sms" || body.Payload["code"] != "123456" {
			t.Fatal("SMS request does not match provider contract")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})}}
	if err := sender.SendOTP(context.Background(), "989123456789", "123456"); err != nil {
		t.Fatal(err)
	}
}

func TestFerzzRejectsDeliveryFailure(t *testing.T) {
	sender := &FerzzSender{token: "test-token", client: &http.Client{Transport: testTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 401, Body: io.NopCloser(strings.NewReader("private provider response"))}, nil
	})}}
	if err := sender.SendOTP(context.Background(), "989123456789", "123456"); err == nil || strings.Contains(err.Error(), "private") {
		t.Fatal("provider failure was not safely handled")
	}
	sender.token = ""
	if err := sender.SendOTP(context.Background(), "989123456789", "123456"); err == nil {
		t.Fatal("missing token accepted")
	}
}
