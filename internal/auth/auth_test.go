package auth_test

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.followtheprocess.codes/gatekeeper/internal/auth"
	"go.followtheprocess.codes/test"
)

func TestParseFromHeader(t *testing.T) {
	tests := []struct {
		claims  map[string]any // JWT claims to bake into the test token (not iat and exp)
		name    string         // Name of the test case
		wantErr bool           // Whether ParseFromHeader should return an error
		window  time.Duration  // Duration the JWT is not expired for, iat and exp are dynamically generated
	}{
		{
			name:    "no claims",
			claims:  make(map[string]any),
			window:  60 * time.Second,
			wantErr: true,
		},
		{
			name: "valid",
			claims: map[string]any{
				"iss": "gatekeeper",
				"sub": "my-project1",
				"ver": "v1.2.3",
				"aud": "get.followtheprocess.codes",
				"jti": "747477bb-2e1a-40ce-a7c8-f82246d36af0",
			},
			window:  60 * time.Second,
			wantErr: false,
		},
		{
			name: "valid but expired",
			claims: map[string]any{
				"iss": "gatekeeper",
				"sub": "my-project1",
				"ver": "v1.2.3",
				"aud": "get.followtheprocess.codes",
				"jti": "747477bb-2e1a-40ce-a7c8-f82246d36af0",
			},
			window:  0 * time.Second,
			wantErr: true,
		},
		{
			name: "wrong audience",
			claims: map[string]any{
				"iss": "gatekeeper",
				"sub": "my-project1",
				"ver": "v1.2.3",
				"aud": "someone else",
				"jti": "747477bb-2e1a-40ce-a7c8-f82246d36af0",
			},
			window:  60 * time.Second,
			wantErr: true,
		},
		{
			name: "wrong issuer",
			claims: map[string]any{
				"iss": "evil",
				"sub": "my-project1",
				"ver": "v1.2.3",
				"aud": "get.followtheprocess.codes",
				"jti": "747477bb-2e1a-40ce-a7c8-f82246d36af0",
			},
			window:  60 * time.Second,
			wantErr: true,
		},
		{
			name: "too long lived",
			claims: map[string]any{
				"iss": "gatekeeper",
				"sub": "my-project1",
				"ver": "v1.2.3",
				"aud": "get.followtheprocess.codes",
				"jti": "747477bb-2e1a-40ce-a7c8-f82246d36af0",
			},
			window:  1 * time.Hour,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub, priv, err := ed25519.GenerateKey(nil)
			test.Ok(t, err)

			now := time.Now().UTC()

			tt.claims["iat"] = now.Unix()
			tt.claims["exp"] = now.Add(tt.window).Unix()

			token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims(tt.claims))

			signed, err := token.SignedString(priv)
			test.Ok(t, err)

			headers := map[string]string{
				"Authorization": fmt.Sprintf("Bearer %s", signed),
			}

			// The Fetcher contract returns raw PEM bytes, so encode the public key
			// the same way the SSMFetcher would store it.
			der, err := x509.MarshalPKIXPublicKey(pub)
			test.Ok(t, err)

			pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

			fetcher := staticFetcher{key: pemBytes}

			_, err = auth.ParseFromHeaders(t.Context(), headers, fetcher)
			test.WantErr(t, err, tt.wantErr)
		})
	}
}

func TestSign(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second) // claims are second-resolution (Unix())

	tests := []struct {
		name   string     // Name of the test case
		token  auth.Token // Token to sign
		escape bool       // Whether to encode the key with literal \n, as env vars often do
	}{
		{
			name: "valid",
			token: auth.Token{
				Expires:  now.Add(2 * time.Minute),
				IssuedAt: now,
				Project:  "test",
				Version:  "v1.2.3",
				ID:       "arandomstring",
			},
		},
		{
			name: "escaped newlines",
			token: auth.Token{
				Expires:  now.Add(2 * time.Minute),
				IssuedAt: now,
				Project:  "test",
				Version:  "v1.2.3",
				ID:       "arandomstring",
			},
			escape: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pub, priv, err := ed25519.GenerateKey(nil)
			test.Ok(t, err)

			privDER, err := x509.MarshalPKCS8PrivateKey(priv)
			test.Ok(t, err)

			privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

			pubDER, err := x509.MarshalPKIXPublicKey(pub)
			test.Ok(t, err)

			pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

			key := string(privPEM)
			if tt.escape {
				key = strings.ReplaceAll(key, "\n", `\n`)
			}

			signed, err := auth.Sign(tt.token, key)
			test.Ok(t, err)

			headers := map[string]string{"Authorization": "Bearer " + signed}
			got, err := auth.ParseFromHeaders(t.Context(), headers, staticFetcher{key: pubPEM})
			test.Ok(t, err)

			test.True(t, got.IssuedAt.Equal(tt.token.IssuedAt))
			test.True(t, got.Expires.Equal(tt.token.Expires))
			test.Equal(t, got.Project, tt.token.Project)
			test.Equal(t, got.Version, tt.token.Version)
			test.Equal(t, got.ID, tt.token.ID)
		})
	}
}

func TestSignErrors(t *testing.T) {
	now := time.Now().UTC()

	_, priv, err := ed25519.GenerateKey(nil)
	test.Ok(t, err)
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	test.Ok(t, err)

	validKey := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}))

	tests := []struct {
		name  string     // Name of the test case
		token auth.Token // Token to sign
		key   string     // PEM private key to sign with
	}{
		{
			name: "valid too long",
			token: auth.Token{
				IssuedAt: now,
				Expires:  now.Add(auth.MaxTokenDuration + time.Minute),
				Project:  "test",
				Version:  "v1.2.3",
				ID:       "x",
			},
			key: validKey,
		},
		{
			name: "malformed key",
			token: auth.Token{
				IssuedAt: now,
				Expires:  now.Add(time.Minute),
				Project:  "test",
				Version:  "v1.2.3",
				ID:       "x",
			},
			key: "not a pem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := auth.Sign(tt.token, tt.key)
			test.Err(t, err)
		})
	}
}

type staticFetcher struct {
	key []byte
}

func (s staticFetcher) Fetch(ctx context.Context, project string) ([]byte, error) {
	return s.key, nil
}
