// Package gatekeeper implements the functionality of the gatekeeper
// cli tool.
//
// The CLI in package cmd dispatches to the exported members of this package.
package gatekeeper

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"go.followtheprocess.codes/gatekeeper/internal/auth"
	"go.followtheprocess.codes/gatekeeper/internal/aws"
	"go.followtheprocess.codes/gatekeeper/internal/names"
	"go.followtheprocess.codes/log"
)

// Gatekeeper represents the current state of the tool.
type Gatekeeper struct {
	stdin  io.Reader    // Program inputs here (e.g. prompts)
	stdout io.Writer    // Normal program output
	stderr io.Writer    // Logs and errors
	logger *log.Logger  // The logger, threaded through the entire app
	client *http.Client // HTTP client
}

// New returns a new [Gatekeeper].
func New(debug bool, stdin io.Reader, stdout, stderr io.Writer) Gatekeeper {
	level := log.LevelInfo
	if debug {
		level = log.LevelDebug
	}

	logger := log.New(stderr, log.WithLevel(level), log.Prefix("gatekeeper"))

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return Gatekeeper{
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		logger: logger,
		client: client,
	}
}

// KeysOptions holds the flags and args for the keys subcommand.
type KeysOptions struct {
	Name  string // Name to use for the key pair e.g. {name} and {name}.pub
	Path  string // Path in which to save the keys, defaults to "."
	Debug bool   // Enable debug logging
}

// TODO: Add a --upload option that creates the SSM param

// Keys handles the keys subcommand.
func (g Gatekeeper) Keys(ctx context.Context, options KeysOptions) error {
	start := time.Now()

	public, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		return fmt.Errorf("could not generate key pair: %w", err)
	}

	name := options.Name
	if name == "" {
		name = names.Get()
	}

	path := options.Path
	if path == "" {
		path = "."
	}

	privPath := filepath.Join(path, name)
	pubPath := filepath.Join(path, name+".pub")

	privDER, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return fmt.Errorf("could not marshal private key: %w", err)
	}

	pubDER, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		return fmt.Errorf("could not marshal public key: %w", err)
	}

	priv := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})
	pub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	// Ensure the directory exists
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("could not create directory: %w", err)
	}

	if err := os.WriteFile(privPath, priv, 0o600); err != nil {
		return fmt.Errorf("could not write private key: %w", err)
	}

	//nolint:gosec // 0644 is convention for public keys
	if err := os.WriteFile(pubPath, pub, 0o644); err != nil {
		return fmt.Errorf("could not write public key: %w", err)
	}

	g.logger.Info(
		"Generated public/private keypair",
		slog.String("algorithm", "ed25519"),
		slog.Duration("took", time.Since(start)),
		slog.String("name", name),
	)

	return nil
}

// AuthOptions holds the flags and args for the auth subcommand.
type AuthOptions struct {
	URL      string        // URL to the auth server
	Project  string        // Project to release
	Version  string        // Version of the project
	Debug    bool          // Enable debug logging
	Duration time.Duration // Requested duration of the session
}

// Auth handles the auth subcommand.
func (g Gatekeeper) Auth(ctx context.Context, options AuthOptions) error {
	logger := g.logger.Prefixed("auth").With(
		slog.String("project", options.Project),
		slog.String("version", options.Version),
		slog.Duration("duration", options.Duration),
	)

	if options.Duration > auth.MaxTokenDuration {
		return fmt.Errorf("duration %s exceeds maximum allowed: %s", options.Duration, auth.MaxTokenDuration)
	}

	logger.Info(
		"authenticating release session",
		slog.String("url", options.URL),
	)

	// 1) Mint a JWT and sign it using the private key (from env var)
	privateKey := os.Getenv("GATEKEEPER_PRIVATE_KEY")
	if privateKey == "" {
		return errors.New("missing 'GATEKEEPER_PRIVATE_KEY' env var")
	}

	now := time.Now().UTC()

	tok := auth.Token{
		IssuedAt: now,
		Expires:  now.Add(options.Duration),
		Project:  options.Project,
		Version:  options.Version,
		ID:       uuid.NewString(),
	}

	signed, err := auth.Sign(tok, privateKey)
	if err != nil {
		return fmt.Errorf("failed to mint new token: %w", err)
	}

	// 2) Hit the lambda URL with it and get the response
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, options.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	request.Header.Add("Authorization", "Bearer "+signed)

	start := time.Now()

	response, err := g.client.Do(request)
	if err != nil || response == nil {
		return fmt.Errorf("failed to execute http request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	logger.Debug(
		"requested auth session",
		slog.String("url", options.URL),
		slog.String("status", response.Status),
		slog.Duration("took", time.Since(start)),
	)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP response status %s: %s", response.Status, string(body))
	}

	// 3) Print the `export AWS...` lines to stdout
	var creds aws.Credentials
	if err := json.Unmarshal(body, &creds); err != nil {
		return fmt.Errorf("invalid response body: %w", err)
	}

	fmt.Fprint(g.stdout, creds.Export())

	return nil
}
