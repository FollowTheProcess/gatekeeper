// Package auth handles checking and validating the incoming auth token.
package auth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// validProjectRegex is a check for a valid project name.
	//
	// It limits projects to alphanumerics and hyphens and less than 64 chars.
	validProjectRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]{0,64}$`)

	// validVersionRegex is a check for a valid version number.
	validVersionRegex = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:[-+][a-zA-Z0-9.-]+)?$`)
)

const (
	// MaxTokenDuration is the maximum amount of time we expect a token to be valid for, i.e.
	// the maximum allowable difference between the 'iat' and 'exp' claims.
	MaxTokenDuration = 15 * time.Minute

	// Audience is the intended audience for all internally minted tokens.
	Audience = "get.followtheprocess.codes"

	// Issuer is the issuer for all internally minted tokens.
	Issuer = "gatekeeper"
)

// Token is an auth token for gatekeeper.
type Token struct {
	Expires  time.Time
	IssuedAt time.Time
	Project  string
	Version  string
	ID       string
}

// ReplayGuard is a DynamoDB record that records a claimed jti to
// prevent token replay.
type ReplayGuard struct {
	TTL time.Time `dynamodbav:"ttl,unixtime"`
	JTI string    `dynamodbav:"jti"`
}

// ParseFromHeaders grabs the 'Authorization' HTTP header if it exists, and
// parses the value into a valid token.
//
// The returned token has been validated and has it's signature verified by
// fetching a project-specific public key using the fetcher.
//
// The header is looked up in a case-insensitive way as lambda function URL
// requests lowercase all header names.
func ParseFromHeaders(ctx context.Context, headers map[string]string, fetcher Fetcher) (Token, error) {
	var auth string

	// Lambda function URL requests normalise header names to lowercase
	// so we look it up in a case-insensitive way so we're not overly coupled
	// to any specific implementation
	for name, value := range headers {
		if strings.EqualFold(name, "authorization") {
			auth = value

			break
		}

		return Token{}, errors.New("missing 'Authorization' header")
	}

	// Expected: 'Bearer <token>'
	// The usual convention is for "Bearer" to be title-cased. However, there's no
	// strict rule around this, and it's best to follow the robustness principle here.
	if len(auth) < 7 || !strings.EqualFold(auth[:7], "bearer ") {
		return Token{}, fmt.Errorf("no token in 'Authorization' header: %s", auth)
	}

	tokenString := auth[7:]

	token, err := jwt.Parse(
		tokenString,
		keyFetcher(ctx, fetcher),
		jwt.WithExpirationRequired(),
		jwt.WithAudience(Audience),
		jwt.WithIssuedAt(),
		jwt.WithIssuer(Issuer),
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
	)
	if err != nil {
		return Token{}, fmt.Errorf("jwt could not be parsed or validated: %w", err)
	}

	subject, err := token.Claims.GetSubject()
	if err != nil {
		return Token{}, fmt.Errorf("could not get sub claim from JWT: %w", err)
	}

	// Check the token is sufficiently short lived
	iss, err := token.Claims.GetIssuedAt()
	if err != nil {
		return Token{}, fmt.Errorf("could not get iss claim from JWT: %w", err)
	}

	exp, err := token.Claims.GetExpirationTime()
	if err != nil {
		return Token{}, fmt.Errorf("could not get exp claim from JWT: %w", err)
	}

	window := exp.Sub(iss.Time)

	if window > MaxTokenDuration {
		return Token{}, fmt.Errorf("jwt is valid for too long (%s) max allowed %s", window, MaxTokenDuration)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return Token{}, fmt.Errorf("unexpected claims type %T", token.Claims)
	}

	jti, ok := claims["jti"]
	if !ok {
		return Token{}, errors.New("no jti claim in token")
	}

	jtiStr, ok := jti.(string)
	if !ok {
		return Token{}, fmt.Errorf("unexpected jti claim type %T, expected string", jti)
	}

	ver, ok := claims["ver"]
	if !ok {
		return Token{}, errors.New("no ver claim in token")
	}

	verStr, ok := ver.(string)
	if !ok {
		return Token{}, fmt.Errorf("unexpected ver claim type %T, expected string", ver)
	}

	if !validVersionRegex.MatchString(verStr) {
		return Token{}, fmt.Errorf("invalid version %q", verStr)
	}

	tok := Token{
		Expires:  exp.Time,
		IssuedAt: iss.Time,
		Project:  subject,
		Version:  verStr,
		ID:       jtiStr,
	}

	return tok, nil
}

// Sign turns the project release token into a full JWT adding extra claims
// as necessary and signs it with the provided private key.
//
// The private key should be the PEM encoded private key, i.e. the
// '---- BEGIN PRIVATE KEY ----' form.
//
// Sign returns the signed JWT string, ready for use in an Authorization header.
func Sign(tok Token, key string) (string, error) {
	token, err := toJWT(tok)
	if err != nil {
		return "", fmt.Errorf("translate to JWT: %w", err)
	}

	key = strings.TrimSpace(key)              // Strip any leading/trailing space
	key = strings.ReplaceAll(key, `\n`, "\n") // Un-escaping \n, common in env vars

	priv, err := jwt.ParseEdPrivateKeyFromPEM([]byte(key))
	if err != nil {
		return "", fmt.Errorf("could not parse private key: %w", err)
	}

	signed, err := token.SignedString(priv)
	if err != nil {
		return "", fmt.Errorf("could not sign JWT: %w", err)
	}

	return signed, nil
}

// toJWT converts an in-house [Token] to a JWT.
func toJWT(in Token) (*jwt.Token, error) {
	if in.Expires.Sub(in.IssuedAt) > MaxTokenDuration {
		return nil, fmt.Errorf("token is valid for too long, max allowed is %s", MaxTokenDuration)
	}

	out := jwt.NewWithClaims(
		jwt.SigningMethodEdDSA,
		jwt.MapClaims{
			"iss": Issuer,
			"sub": in.Project,
			"ver": in.Version,
			"aud": Audience,
			"jti": in.ID,
			"iat": in.IssuedAt.Unix(),
			"exp": in.Expires.Unix(),
		},
	)

	return out, nil
}

func keyFetcher(ctx context.Context, fetcher Fetcher) jwt.Keyfunc {
	return func(t *jwt.Token) (any, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		claims, ok := t.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("unexpected claims type, expected jwt.MapClaims, got %T", claims)
		}

		sub, err := claims.GetSubject()
		if err != nil {
			return nil, fmt.Errorf("could not get sub claim: %w", err)
		}

		if !validProjectRegex.MatchString(sub) {
			return nil, fmt.Errorf("invalid sub %q, must match the regex %q", sub, validProjectRegex.String())
		}

		pem, err := fetcher.Fetch(ctx, sub)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch public key: %w", err)
		}

		pub, err := jwt.ParseEdPublicKeyFromPEM(pem)
		if err != nil {
			return nil, fmt.Errorf("failed to decode public key PEM: %w", err)
		}

		return pub, nil
	}
}
