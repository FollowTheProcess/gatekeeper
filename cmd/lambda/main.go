// Package gatekeeper is a custom CI -> AWS auth mechanism enabling OIDC-like flows on non-OIDC enabled CI systems.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
)

func main() {
	env := LoadEnv()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     env.LogLevel,
	})))

	if err := run(env); err != nil {
		slog.Error("gatekeeper exited with error", slog.Any("err", err))
		os.Exit(1)
	}
}

func run(env Env) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("loading aws config: %w", err)
	}

	// Add spans for AWS SDK calls
	otelaws.AppendMiddlewares(&cfg.APIOptions)

	handler := Handler{
		logger: slog.Default(),
		ssm:    ssm.NewFromConfig(cfg),
		dynamo: dynamodb.NewFromConfig(cfg),
		sts:    sts.NewFromConfig(cfg),
		env:    env,
	}

	tp, err := xrayconfig.NewTracerProvider(ctx)
	if err != nil {
		return fmt.Errorf("creating new tracing provider: %w", err)
	}

	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer shutdownCancel()

		if err := tp.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutting down tracer provider", slog.Any("err", err))
		}
	}()

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(xray.Propagator{})

	lambda.StartWithOptions(
		otellambda.InstrumentHandler(handler.Handle, xrayconfig.WithRecommendedOptions(tp)...),
		lambda.WithContext(ctx),
		lambda.WithEnableSIGTERM(cancel),
	)

	return nil
}
