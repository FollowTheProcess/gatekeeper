package gatekeeper_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.followtheprocess.codes/gatekeeper/internal/aws"
	"go.followtheprocess.codes/gatekeeper/internal/gatekeeper"
	"go.followtheprocess.codes/problem"
	"go.followtheprocess.codes/test"
)

func TestKeys(t *testing.T) {
	dir := t.TempDir()

	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	gk := gatekeeper.New(false, stdin, stdout, stderr)

	options := gatekeeper.KeysOptions{
		Name: "silly-deterministic",
		Path: dir,
	}

	err := gk.Keys(t.Context(), options)
	test.Ok(t, err)

	pubPath := filepath.Join(dir, options.Name+".pub")
	privPath := filepath.Join(dir, options.Name)

	// Files should exist
	_, err = os.Stat(pubPath)
	test.Ok(t, err)

	_, err = os.Stat(privPath)
	test.Ok(t, err)

	// They should look like keys
	pub, err := os.ReadFile(pubPath)
	test.Ok(t, err)

	priv, err := os.ReadFile(privPath)
	test.Ok(t, err)

	test.True(t, bytes.Contains(pub, []byte("-----BEGIN PUBLIC KEY-----")))
	test.True(t, bytes.Contains(pub, []byte("-----END PUBLIC KEY-----")))
	test.True(t, bytes.Contains(priv, []byte("-----BEGIN PRIVATE KEY-----")))
	test.True(t, bytes.Contains(priv, []byte("-----END PRIVATE KEY-----")))
}

func TestAuth(t *testing.T) {
	tests := []struct {
		name    string                 // Name of the test case
		stdout  string                 // Expected stdout
		backend http.HandlerFunc       // Fake backend, respond however we like for the test
		options gatekeeper.AuthOptions // Options to pass to gatekeeper
		wantErr bool                   // Whether Auth should return an error
	}{
		{
			name: "happy",
			backend: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				response := aws.Credentials{
					AWSAccessKeyID:     "defforeal",
					AWSSecretAccessKey: "blah",
					AWSSessionToken:    "arealtokenlol",
				}

				test.Ok(t, json.NewEncoder(w).Encode(response))
			}),
			options: gatekeeper.AuthOptions{
				Project:  "test",
				Version:  "v1.2.3",
				Debug:    false,
				Duration: 2 * time.Minute,
			},
			stdout:  "export AWS_ACCESS_KEY_ID='defforeal'\nexport AWS_SECRET_ACCESS_KEY='blah'\nexport AWS_SESSION_TOKEN='arealtokenlol'",
			wantErr: false,
		},
		{
			name: "sad",
			backend: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", problem.ContentType)
				w.WriteHeader(http.StatusTeapot)
				prob := problem.Problem{
					Type:     "about:blank",
					Title:    "Wrong",
					Detail:   "Something went wrong sorry",
					Instance: "blah",
					Status:   http.StatusTeapot,
				}
				_, err := w.Write(prob.JSON())
				test.Ok(t, err)
			}),
			options: gatekeeper.AuthOptions{
				Project:  "test",
				Version:  "v1.2.3",
				Debug:    false,
				Duration: 2 * time.Minute,
			},
			stdout:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdin := &bytes.Buffer{}
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			gk := gatekeeper.New(false, stdin, stdout, stderr)

			srv := httptest.NewServer(tt.backend)
			t.Cleanup(srv.Close)

			// Give it an actual private key via the env var
			_, priv, err := ed25519.GenerateKey(nil)
			test.Ok(t, err)

			privDER, err := x509.MarshalPKCS8PrivateKey(priv)
			test.Ok(t, err)

			privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})

			t.Setenv("GATEKEEPER_PRIVATE_KEY", string(privPEM))

			tt.options.URL = srv.URL
			err = gk.Auth(t.Context(), tt.options)

			test.WantErr(t, err, tt.wantErr)
			test.Diff(t, stdout.String(), tt.stdout)
		})
	}
}
