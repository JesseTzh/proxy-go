package acme

import (
	"testing"

	"github.com/proxy-go/proxy-go/internal/testutil"
)

func TestHTTPProviderStoresAndCleansChallenge(t *testing.T) {
	db := testutil.NewDB(t)
	provider := NewHTTPProvider(New(db))

	if err := provider.Present("proxy.example.com", "token", "key-auth"); err != nil {
		t.Fatalf("present challenge: %v", err)
	}
	if got, ok := New(db).GetKeyAuthorization("token"); !ok || got != "key-auth" {
		t.Fatalf("challenge not stored, got %q ok=%v", got, ok)
	}

	if err := provider.CleanUp("proxy.example.com", "token", "key-auth"); err != nil {
		t.Fatalf("cleanup challenge: %v", err)
	}
	if got, ok := New(db).GetKeyAuthorization("token"); ok {
		t.Fatalf("challenge not cleaned up, got %q", got)
	}
}
