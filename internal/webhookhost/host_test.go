package webhookhost

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/justin-napolitano/jayburd-bookkeeping/internal/webhook"
)

// keySource supplies only an ephemeral synthetic public key.
type keySource struct {
	key   webhook.JWK
	calls atomic.Int32
}

func (k *keySource) Get(context.Context, string) (webhook.JWK, error) {
	k.calls.Add(1)
	return k.key, nil
}

// receiverFunc injects committed, uncertain and cancelled receipt outcomes.
type receiverFunc func(context.Context, webhook.Verified) (webhook.Receipt, error)

func (f receiverFunc) Receive(ctx context.Context, v webhook.Verified) (webhook.Receipt, error) {
	return f(ctx, v)
}

// signedFixture creates synthetic notification bytes and maintained-library ES256 signatures in memory.
func signedFixture(t *testing.T) (*keySource, string, string) {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	body := `{"webhook_type":"TRANSACTIONS","webhook_code":"SYNC_UPDATES_AVAILABLE","item_id":"synthetic-item","environment":"sandbox","initial_update_complete":true,"historical_update_complete":false}`
	hash := sha256.Sum256([]byte(body))
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{"iat": time.Now().Unix(), "request_body_sha256": hex.EncodeToString(hash[:])})
	token.Header["kid"] = "synthetic-key"
	signed, err := token.SignedString(k)
	if err != nil {
		t.Fatal(err)
	}
	return &keySource{key: webhook.JWK{Alg: "ES256", Curve: "P-256", ID: "synthetic-key", Type: "EC", Use: "sig", X: base64.RawURLEncoding.EncodeToString(k.X.FillBytes(make([]byte, 32))), Y: base64.RawURLEncoding.EncodeToString(k.Y.FillBytes(make([]byte, 32))), Created: time.Now().Add(-time.Hour).Unix()}}, signed, body
}

// startHost opens only an ephemeral loopback socket and joins shutdown before cleanup completes.
func startHost(t *testing.T, h *Host) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- h.Serve(ctx, l) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(4 * time.Second):
			t.Error("host failed to stop")
		}
	})
	return "http://" + l.Addr().String()
}

// request posts exact synthetic bytes through a real socket and captures only bounded response metadata.
func request(t *testing.T, url, token, body string) int {
	t.Helper()
	r, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Plaid-Verification", token)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "synthetic") || resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("unsafe response")
	}
	if resp.Header.Get("Cache-Control") != "no-store" || resp.Header.Get("X-Content-Type-Options") != "nosniff" || resp.Header.Get("Content-Security-Policy") == "" {
		t.Fatal("missing response safeguards")
	}
	return resp.StatusCode
}

// TestSignedSocketDelivery proves no socket acknowledgement precedes a committed receipt, including retry.
func TestSignedSocketDelivery(t *testing.T) {
	keys, token, body := signedFixture(t)
	var calls atomic.Int32
	var closed atomic.Int32
	h, err := New(keys, receiverFunc(func(ctx context.Context, v webhook.Verified) (webhook.Receipt, error) {
		if _, ok := ctx.Deadline(); !ok || v.ItemID != "synthetic-item" {
			t.Error("missing verified bounded context")
		}
		if calls.Add(1) == 1 {
			return webhook.Receipt{}, errors.New("synthetic-private-diagnostic")
		}
		return webhook.Receipt{State: "duplicate", ReceiptID: "00000000-0000-4000-8000-000000000001", StreamCount: 1}, nil
	}), func() { closed.Add(1) })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		h.Close()
		h.Close()
		if closed.Load() != 1 {
			t.Error("cleanup repeated")
		}
	})
	base := startHost(t, h)
	if request(t, base+"/webhooks/plaid", token, body) != 503 {
		t.Fatal("uncertain receipt acknowledged")
	}
	if request(t, base+"/webhooks/plaid", token, body) != 204 {
		t.Fatal("durable retry rejected")
	}
	if calls.Load() != 2 || keys.calls.Load() != 1 {
		t.Fatal("unexpected receipt/key work")
	}
}

// TestSocketIsolation rejects private API routes, malformed signatures and altered bodies before persistence.
func TestSocketIsolation(t *testing.T) {
	keys, token, body := signedFixture(t)
	var calls atomic.Int32
	h, _ := New(keys, receiverFunc(func(context.Context, webhook.Verified) (webhook.Receipt, error) {
		calls.Add(1)
		return webhook.Receipt{}, nil
	}), nil)
	base := startHost(t, h)
	for _, path := range []string{"/v1/workspace", "/healthz", "/webhooks/plaid?x=1", "/webhooks/%70laid", "/webhooks/plaid/"} {
		if request(t, base+path, token, body) != 404 {
			t.Fatalf("route exposed: %s", path)
		}
	}
	if keys.calls.Load() != 0 {
		t.Fatal("route denial resolved key")
	}
	c, err := net.DialTimeout("tcp", strings.TrimPrefix(base, "http://"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(c, "OPTIONS * HTTP/1.1\r\nHost: synthetic.local\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	options, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		t.Fatal(err)
	}
	options.Body.Close()
	if options.StatusCode != 404 {
		t.Fatal("general OPTIONS bypassed exact route")
	}
	if request(t, base+"/webhooks/plaid", "invalid", body) != 401 {
		t.Fatal("invalid signature accepted")
	}
	if request(t, base+"/webhooks/plaid", token, body+" ") != 401 {
		t.Fatal("changed body accepted")
	}
	if request(t, base+"/webhooks/plaid", token, strings.Repeat("x", 65537)) != 413 {
		t.Fatal("oversize body accepted")
	}
	if calls.Load() != 0 {
		t.Fatal("unverified input reached receiver")
	}
}

// TestHostConfigurationAndBounds pins stricter-than-provider limits and rejects incomplete composition.
func TestHostConfigurationAndBounds(t *testing.T) {
	keys, _, _ := signedFixture(t)
	r := receiverFunc(func(context.Context, webhook.Verified) (webhook.Receipt, error) { return webhook.Receipt{}, nil })
	if _, err := New(nil, r, nil); err != ErrConfiguration {
		t.Fatal("missing keys accepted")
	}
	if _, err := New(keys, nil, nil); err != ErrConfiguration {
		t.Fatal("missing receiver accepted")
	}
	if _, err := NewSQL(keys, nil, nil); err != ErrConfiguration {
		t.Fatal("missing restricted pool accepted")
	}
	h, _ := New(keys, r, nil)
	srv := h.server(context.Background())
	if srv.ReadHeaderTimeout != 2*time.Second || srv.ReadTimeout != 5*time.Second || srv.WriteTimeout != 8*time.Second || srv.IdleTimeout != 15*time.Second || srv.MaxHeaderBytes != 16384 {
		t.Fatal("transport bounds changed without review")
	}
	if !srv.DisableGeneralOptionsHandler {
		t.Fatal("server bypasses exact route for OPTIONS star")
	}
	if err := h.Serve(nil, nil); err != ErrConfiguration {
		t.Fatal("incomplete lifecycle accepted")
	}
}

// TestStalledBodyRead proves an ordinary socket cannot retain admission indefinitely without supplying its body.
func TestStalledBodyRead(t *testing.T) {
	keys, token, _ := signedFixture(t)
	var calls atomic.Int32
	h, _ := New(keys, receiverFunc(func(context.Context, webhook.Verified) (webhook.Receipt, error) {
		calls.Add(1)
		return webhook.Receipt{}, nil
	}), nil)
	base := startHost(t, h)
	c, err := net.DialTimeout("tcp", strings.TrimPrefix(base, "http://"), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.SetDeadline(time.Now().Add(7 * time.Second)); err != nil {
		t.Fatal(err)
	}
	_, err = fmt.Fprintf(c, "POST /webhooks/plaid HTTP/1.1\r\nHost: synthetic.local\r\nContent-Type: application/json\r\nPlaid-Verification: %s\r\nContent-Length: 100\r\n\r\n", token)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		t.Fatal("bounded body read failed to return rejection")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 400 || calls.Load() != 0 || keys.calls.Load() != 0 {
		t.Fatal("incomplete body reached authenticated work")
	}
}

// TestShutdownCancelsReceipt proves process cancellation reaches in-flight durable work before cleanup.
func TestShutdownCancelsReceipt(t *testing.T) {
	keys, token, body := signedFixture(t)
	entered := make(chan struct{})
	cancelled := make(chan struct{})
	h, _ := New(keys, receiverFunc(func(ctx context.Context, _ webhook.Verified) (webhook.Receipt, error) {
		close(entered)
		<-ctx.Done()
		close(cancelled)
		return webhook.Receipt{}, ctx.Err()
	}), nil)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- h.Serve(ctx, l) }()
	response := make(chan int, 1)
	go func() { response <- request(t, "http://"+l.Addr().String()+"/webhooks/plaid", token, body) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("receiver did not start")
	}
	cancel()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("receiver was not cancelled")
	}
	if <-response != 503 {
		t.Fatal("cancelled receipt acknowledged")
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	l2, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l2.Close()
	if err := h.Serve(context.Background(), l2); err != ErrAlreadyServed {
		t.Fatal("host reused")
	}
}
