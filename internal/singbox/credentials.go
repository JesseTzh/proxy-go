package singbox

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	"github.com/proxy-go/proxy-go/internal/security"
)

type Credentials struct {
	UUID              string
	RealityPrivateKey string
	RealityPublicKey  string
	RealityShortID    string
	Password          string
}

type CredentialGenerator interface {
	UUID(ctx context.Context) (string, error)
	RealityKeyPair(ctx context.Context) (privateKey string, publicKey string, err error)
	ShortID() (string, error)
	Password() (string, error)
}

type CLICredentialGenerator struct {
	Binary string
}

type StaticCredentialGenerator struct {
	Credentials Credentials
}

func (g CLICredentialGenerator) UUID(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, g.Binary, "generate", "uuid").Output()
	if err != nil {
		return uuid.NewString(), nil
	}
	return strings.TrimSpace(string(out)), nil
}

func (g CLICredentialGenerator) RealityKeyPair(ctx context.Context) (string, string, error) {
	out, err := exec.CommandContext(ctx, g.Binary, "generate", "reality-keypair").CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("sing-box generate reality-keypair: %w: %s", err, strings.TrimSpace(string(out)))
	}
	privateKey, publicKey, err := ParseRealityKeyPairOutput(string(out))
	if err != nil {
		return "", "", err
	}
	return privateKey, publicKey, nil
}

func (g CLICredentialGenerator) ShortID() (string, error) {
	return security.RandomHex(4)
}

func (g CLICredentialGenerator) Password() (string, error) {
	return security.RandomHex(16)
}

func (g StaticCredentialGenerator) UUID(context.Context) (string, error) {
	return g.Credentials.UUID, nil
}

func (g StaticCredentialGenerator) RealityKeyPair(context.Context) (string, string, error) {
	return g.Credentials.RealityPrivateKey, g.Credentials.RealityPublicKey, nil
}

func (g StaticCredentialGenerator) ShortID() (string, error) {
	return g.Credentials.RealityShortID, nil
}

func (g StaticCredentialGenerator) Password() (string, error) {
	return g.Credentials.Password, nil
}

func ParseRealityKeyPairOutput(output string) (privateKey string, publicKey string, err error) {
	for _, line := range strings.Split(output, "\n") {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		switch normalizeKeyLabel(key) {
		case "privatekey", "private":
			privateKey = strings.TrimSpace(value)
		case "publickey", "public":
			publicKey = strings.TrimSpace(value)
		}
	}
	if privateKey == "" || publicKey == "" {
		return "", "", fmt.Errorf("sing-box reality-keypair output missing private or public key")
	}
	return privateKey, publicKey, nil
}

func normalizeKeyLabel(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	label = strings.ReplaceAll(label, " ", "")
	label = strings.ReplaceAll(label, "_", "")
	label = strings.ReplaceAll(label, "-", "")
	return label
}
