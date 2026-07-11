package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamotypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	ststypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	"go.followtheprocess.codes/gatekeeper/internal/auth"
	"go.followtheprocess.codes/gatekeeper/internal/aws"
	"go.followtheprocess.codes/problem"
)

// TODO: Add an S3Client and ListObjects so we can check if the release already exists
// If so return a 409 Problem JSON

// TODO: Cache SSM params, maybe with the lambda extension or something?

var errReplayCheckFailed = errors.New("token failed replay check")

// Handler is the lambda function handler.
type Handler struct {
	logger *slog.Logger
	ssm    aws.SSMClient
	dynamo aws.DynamoDBClient
	sts    aws.STSClient
	env    Env
}

func (h Handler) Handle(
	ctx context.Context,
	request events.LambdaFunctionURLRequest,
) (events.LambdaFunctionURLResponse, error) {
	//nolint:revive // We actually want to temporarily modify the handler logger
	h.logger = h.logger.With(
		slog.String("source_ip", request.RequestContext.HTTP.SourceIP),
		slog.String("request_id", request.RequestContext.RequestID),
	)

	h.logger.Info("Handling request")

	// 1) Parse and validate the token, fetching sub from the SSM param
	//    make sure we return 403 on any error (even ssm param not found)
	token, err := auth.ParseFromHeaders(ctx, request.Headers, aws.NewSSMFetcher(h.ssm))
	if err != nil {
		// Problem detail is intentionally opaque to avoid leaking auth details
		// but log line will have the real error
		return h.Error(http.StatusForbidden, err, problem.Detail("Forbidden"))
	}

	// 2) Put jti into Dynamo with ttl=exp, fail if it already exists
	// preventing a token replay
	if err = h.replayCheck(ctx, token); err != nil {
		if errors.Is(err, errReplayCheckFailed) {
			return h.Error(
				http.StatusForbidden,
				err,
				problem.Detail("Forbidden"),
			)
		}

		// Normal, non replay related error
		return h.Error(http.StatusInternalServerError, err)
	}

	h.logger.Info(
		"Token validated",
		slog.String("id", token.ID),
		slog.String("project", token.Project),
		slog.String("version", token.Version),
		slog.Time("issued", token.IssuedAt),
		slog.Time("expires", token.Expires),
	)

	// TODO: I want to make sure it can't overwrite an existing version
	// maybe object lock at some point in the future once it's all stable
	// and works

	// An inline session policy narrowing the (already tight) IAM role policy
	// from PutObject on the entire bucket, to PutObject on specifically this
	// project and this version (from the token claims)
	//
	// So now all an assumer of this role can do is PutObject to
	// `{bucket}/project/version/*` for up to the (low) max duration. Nice and secure!
	policy, err := aws.InlineUploadPolicy(h.env.ReleaseBucketName, token.Project, token.Version)
	if err != nil {
		return h.Error(http.StatusInternalServerError, fmt.Errorf("marshal inline policy: %w", err))
	}

	output, err := h.assumeReleaseRole(ctx, token, policy)
	if err != nil {
		return h.Error(http.StatusInternalServerError, err)
	}

	h.logger.Info(
		"Assumed upload role",
		slog.String("arn", *output.AssumedRoleUser.Arn),
		slog.String("id", *output.AssumedRoleUser.AssumedRoleId),
	)

	response := aws.Credentials{
		AWSAccessKeyID:     *output.Credentials.AccessKeyId,
		AWSSecretAccessKey: *output.Credentials.SecretAccessKey,
		AWSSessionToken:    *output.Credentials.SessionToken,
	}

	responseBody, err := json.Marshal(response)
	if err != nil {
		return h.Error(http.StatusInternalServerError, err)
	}

	return events.LambdaFunctionURLResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":           "application/json",
			"Cache-Control":          "no-store",
			"X-Content-Type-Options": "nosniff",
		},
		Body: string(responseBody),
	}, nil
}

// Error crafts a LambdaFunctionURLResponse with a problem JSON
// describing the non-nil error.
//
// The signature is such that you can directly return it from a lambda
// function handler, but the returned error will always be nil so that
// your response is always sent.
//
// If a lambda handler returns a non-nil error, aws kills it and you
// don't get the nice response body you wanted for an API-like interaction
// where even errors generate valid JSON responses.
func (h Handler) Error(status int, err error, options ...problem.Option) (events.LambdaFunctionURLResponse, error) {
	prob := problem.Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Detail: err.Error(),
		Status: status,
	}

	for _, option := range options {
		option(&prob)
	}

	h.logger.Error("lambda returned an error", slog.Any("err", err))

	return events.LambdaFunctionURLResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type":  problem.ContentType,
			"Cache-Control": "no-store",
		},
		Body: string(prob.JSON()),
	}, nil
}

// replayCheck handles the token replay checking via jti existing
// in a DynamoDB table.
func (h Handler) replayCheck(ctx context.Context, token auth.Token) error {
	guard := auth.ReplayGuard{
		JTI: token.ID,
		TTL: token.Expires,
	}

	item, err := attributevalue.MarshalMap(guard)
	if err != nil {
		return fmt.Errorf("marshal replay guard: %w", err)
	}

	_, err = h.dynamo.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           new(h.env.NonceTableName),
		Item:                item,
		ConditionExpression: new("attribute_not_exists(jti)"),
	})
	if err != nil {
		if _, ok := errors.AsType[*dynamotypes.ConditionalCheckFailedException](err); ok {
			// jti already claimed, this is a replay attempt -> Forbidden
			return fmt.Errorf("%w: jti %s is already used: %w", errReplayCheckFailed, token.ID, err)
		}

		// A normal error
		return fmt.Errorf("dynamodb putitem: %w", err)
	}

	return nil
}

// assumeReleaseRole assumes the release role based on the validated token.
func (h Handler) assumeReleaseRole(
	ctx context.Context,
	token auth.Token,
	policy string,
) (*sts.AssumeRoleOutput, error) {
	output, err := h.sts.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         new(h.env.ReleaseUploadRoleARN),
		RoleSessionName: new(fmt.Sprintf("release-%s-%s", token.Project, token.Version)),
		DurationSeconds: new(int32(auth.MaxTokenDuration.Seconds())),
		Policy:          new(policy),
		SourceIdentity:  new(fmt.Sprintf("%s@%s", token.Project, token.Version)),
		Tags: []ststypes.Tag{
			{Key: new("project"), Value: new(token.Project)},
			{Key: new("version"), Value: new(token.Version)},
			{Key: new("jti"), Value: new(token.ID)},
		},
	})
	if err != nil || output == nil {
		return nil, fmt.Errorf("assume release upload role: %w", err)
	}

	return output, nil
}
