package main

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamotypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"github.com/golang-jwt/jwt/v5"
	"go.followtheprocess.codes/gatekeeper/internal/aws"
	"go.followtheprocess.codes/problem"
	"go.followtheprocess.codes/test"
)

func TestHandler(t *testing.T) {
	headers, publicKey := fakeAuthHeader(t)

	okSSM := ssmReturning(publicKey)
	okDynamo := fakeDynamo{output: &dynamodb.PutItemOutput{}}
	okSTS := stsReturning("AKIAEXAMPLE", "secret-key", "session-token", "arn:aws:sts::123456789123:assumed-role/r/s")

	tests := []struct {
		ssm     aws.SSMClient                    // Fake SSM client
		dynamo  aws.DynamoDBClient               // Fake DDB client
		sts     aws.STSClient                    // Fake STS client
		name    string                           // Name of the test case
		request events.LambdaFunctionURLRequest  // Request into the handler
		want    events.LambdaFunctionURLResponse // Expected response
	}{
		{
			name:    "happy path",
			ssm:     okSSM,
			dynamo:  okDynamo,
			sts:     okSTS,
			request: events.LambdaFunctionURLRequest{Headers: headers},
			want: okResponse(
				`{"AWSAccessKeyId":"AKIAEXAMPLE","AWSSecretAccessKey":"secret-key","AWSSessionToken":"session-token"}`,
			),
		},
		{
			name:    "missing authorization header",
			ssm:     okSSM,
			dynamo:  okDynamo,
			sts:     okSTS,
			request: events.LambdaFunctionURLRequest{Headers: map[string]string{}},
			// Detail is opaque so we never leak why auth failed.
			want: problemResponse(http.StatusForbidden, http.StatusText(http.StatusForbidden)),
		},
		{
			name:    "malformed authorization header",
			ssm:     okSSM,
			dynamo:  okDynamo,
			sts:     okSTS,
			request: events.LambdaFunctionURLRequest{Headers: map[string]string{"Authorization": "garbage"}},
			want:    problemResponse(http.StatusForbidden, http.StatusText(http.StatusForbidden)),
		},
		{
			name:    "lowercase garbage",
			ssm:     okSSM,
			dynamo:  okDynamo,
			sts:     okSTS,
			request: events.LambdaFunctionURLRequest{Headers: map[string]string{"authorization": "garbage"}},
			want:    problemResponse(http.StatusForbidden, http.StatusText(http.StatusForbidden)),
		},
		{
			name:    "public key unavailable",
			ssm:     fakeSSM{err: fmt.Errorf("parameter not found")},
			dynamo:  okDynamo,
			sts:     okSTS,
			request: events.LambdaFunctionURLRequest{Headers: headers},
			want:    problemResponse(http.StatusForbidden, http.StatusText(http.StatusForbidden)),
		},
		{
			name:    "replayed token",
			ssm:     okSSM,
			dynamo:  fakeDynamo{err: &dynamotypes.ConditionalCheckFailedException{Message: new("jti exists")}},
			sts:     okSTS,
			request: events.LambdaFunctionURLRequest{Headers: headers},
			want:    problemResponse(http.StatusForbidden, http.StatusText(http.StatusForbidden)),
		},
		{
			name:    "dynamo unavailable",
			ssm:     okSSM,
			dynamo:  fakeDynamo{err: fmt.Errorf("dynamo is down")},
			sts:     okSTS,
			request: events.LambdaFunctionURLRequest{Headers: headers},
			want:    problemResponse(http.StatusInternalServerError, "dynamodb putitem: dynamo is down"),
		},
		{
			name:    "assume role fails",
			ssm:     okSSM,
			dynamo:  okDynamo,
			sts:     fakeSTS{err: fmt.Errorf("access denied")},
			request: events.LambdaFunctionURLRequest{Headers: headers},
			want:    problemResponse(http.StatusInternalServerError, "assume release upload role: access denied"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Handler{
				logger: slog.New(slog.DiscardHandler),
				ssm:    tt.ssm,
				dynamo: tt.dynamo,
				sts:    tt.sts,
				env: Env{
					NonceTableName:       "test-nonce",
					ReleaseUploadRoleARN: "arn:aws:iam::123456789123:role/not-a-real-role",
					ReleaseBucketName:    "releases-plz",
					LogLevel:             slog.LevelInfo,
				},
			}

			got, err := handler.Handle(t.Context(), tt.request)
			test.Ok(t, err) // Handler never returns an error, it bakes it into the response.

			// Marshal both to JSON so we can diff nicely.
			gotJSON, err := json.MarshalIndent(got, "", "  ")
			test.Ok(t, err)

			wantJSON, err := json.MarshalIndent(tt.want, "", "  ")
			test.Ok(t, err)

			test.DiffBytes(t, gotJSON, wantJSON)
		})
	}
}

// okResponse builds the successful response carrying the given JSON body.
func okResponse(body string) events.LambdaFunctionURLResponse {
	return events.LambdaFunctionURLResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":           "application/json",
			"Cache-Control":          "no-store",
			"X-Content-Type-Options": "nosniff",
		},
		Body: body,
	}
}

// problemResponse builds the error response the handler emits for a given
// status and detail, reusing the real [problem.Problem] serialiser.
func problemResponse(status int, detail string) events.LambdaFunctionURLResponse {
	prob := problem.Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Detail: detail,
		Status: status,
	}

	return events.LambdaFunctionURLResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type":  problem.ContentType,
			"Cache-Control": "no-store",
		},
		Body: string(prob.JSON()),
	}
}

// fakeAuthHeader creates an `authorization: Bearer XXX` with a real, signed JWT
// and returns the public key needed to verify it.
//
// Note: The lambda URL payload lowercases all header names.
func fakeAuthHeader(t *testing.T) (headers map[string]string, publicKey string) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(nil)
	test.Ok(t, err)

	now := time.Now().UTC()

	claims := map[string]any{
		"iss": "gatekeeper",
		"sub": "my-project1",
		"ver": "v1.2.3",
		"aud": "get.followtheprocess.codes",
		"jti": "747477bb-2e1a-40ce-a7c8-f82246d36af0",
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims(claims))

	signed, err := token.SignedString(priv)
	test.Ok(t, err)

	headers = map[string]string{
		"authorization": fmt.Sprintf("Bearer %s", signed),
	}

	// The Fetcher contract returns raw PEM bytes, so encode the public key
	// the same way the SSMFetcher would store it.
	der, err := x509.MarshalPKIXPublicKey(pub)
	test.Ok(t, err)

	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	return headers, string(pemBytes)
}

// fakeSSM is a fake [aws.SSMClient] returning static output.
type fakeSSM struct {
	output *ssm.GetParameterOutput
	err    error
}

func (f fakeSSM) GetParameter(
	_ context.Context,
	_ *ssm.GetParameterInput,
	_ ...func(*ssm.Options),
) (*ssm.GetParameterOutput, error) {
	return f.output, f.err
}

// fakeDynamo is a fake [aws.DynamoDBClient] returning static output.
type fakeDynamo struct {
	output *dynamodb.PutItemOutput
	err    error
}

func (f fakeDynamo) PutItem(
	_ context.Context,
	_ *dynamodb.PutItemInput,
	_ ...func(*dynamodb.Options),
) (*dynamodb.PutItemOutput, error) {
	return f.output, f.err
}

// fakeSTS is a fake [aws.STSClient] returning static output.
type fakeSTS struct {
	output *sts.AssumeRoleOutput
	err    error
}

func (f fakeSTS) AssumeRole(
	_ context.Context,
	_ *sts.AssumeRoleInput,
	_ ...func(*sts.Options),
) (*sts.AssumeRoleOutput, error) {
	return f.output, f.err
}

// ssmReturning builds a fake SSM client that serves publicKey as a String
// parameter, exactly as the real SSMFetcher expects to read it.
func ssmReturning(publicKey string) fakeSSM {
	return fakeSSM{
		output: &ssm.GetParameterOutput{
			Parameter: &ssmtypes.Parameter{
				Type:  ssmtypes.ParameterTypeString,
				Value: new(publicKey),
			},
		},
	}
}

// stsReturning builds a fake STS client that hands back the given credentials.
func stsReturning(accessKey, secretKey, sessionToken, arn string) fakeSTS {
	return fakeSTS{
		output: &sts.AssumeRoleOutput{
			Credentials: &ststypes.Credentials{
				AccessKeyId:     new(accessKey),
				SecretAccessKey: new(secretKey),
				SessionToken:    new(sessionToken),
			},
			AssumedRoleUser: &ststypes.AssumedRoleUser{
				Arn:           new(arn),
				AssumedRoleId: new("an ID"),
			},
		},
	}
}
