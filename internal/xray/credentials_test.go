package xray

import (
	"context"
	"testing"
)

func TestParseX25519Output(t *testing.T) {
	privateKey, publicKey, err := ParseX25519Output("Private key: abc\nPublic key: def\n")
	if err != nil {
		t.Fatalf("parse x25519 output: %v", err)
	}
	if privateKey != "abc" || publicKey != "def" {
		t.Fatalf("unexpected keys: %q %q", privateKey, publicKey)
	}
}

func TestStaticCredentialGeneratorForServiceTests(t *testing.T) {
	gen := StaticCredentialGenerator{Credentials: Credentials{
		UUID:              "11111111-1111-1111-1111-111111111111",
		RealityPrivateKey: "private-key",
		RealityPublicKey:  "public-key",
		RealityShortID:    "abcd1234",
	}}
	id, err := gen.UUID(context.Background())
	if err != nil {
		t.Fatalf("uuid: %v", err)
	}
	privateKey, publicKey, err := gen.RealityKeyPair(context.Background())
	if err != nil {
		t.Fatalf("key pair: %v", err)
	}
	shortID, err := gen.ShortID()
	if err != nil {
		t.Fatalf("short id: %v", err)
	}
	if id != gen.Credentials.UUID || privateKey != gen.Credentials.RealityPrivateKey || publicKey != gen.Credentials.RealityPublicKey || shortID != gen.Credentials.RealityShortID {
		t.Fatalf("unexpected credentials: %q %q %q %q", id, privateKey, publicKey, shortID)
	}
}
