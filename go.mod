module go.followtheprocess.codes/gatekeeper

go 1.26

require (
	github.com/aws/aws-lambda-go v1.54.0
	github.com/aws/aws-sdk-go-v2/config v1.32.32
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.20.55
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.62.1
	github.com/aws/aws-sdk-go-v2/service/ssm v1.73.1
	github.com/aws/aws-sdk-go-v2/service/sts v1.45.1
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/google/uuid v1.6.0
	go.followtheprocess.codes/cli v0.21.1
	go.followtheprocess.codes/log v1.2.2
	go.followtheprocess.codes/msg v1.10.0
	go.followtheprocess.codes/problem v1.1.0
	go.followtheprocess.codes/test v1.4.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda v0.69.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig v0.69.0
	go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws v0.69.0
	go.opentelemetry.io/contrib/propagators/aws v1.44.0
	go.opentelemetry.io/otel v1.44.0
)

require (
	github.com/aws/aws-sdk-go-v2 v1.43.1 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.31 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.32 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.32 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.32 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.33 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.36.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.14 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.12.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.32 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.5.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.42.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.46.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.33.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.38.1 // indirect
	github.com/aws/smithy-go v1.27.5 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/go-logr/logr v1.4.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0 // indirect
	go.followtheprocess.codes/diff v0.2.0 // indirect
	go.followtheprocess.codes/hue v1.2.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/detectors/aws/lambda v0.69.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.44.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.44.0 // indirect
	go.opentelemetry.io/otel/metric v1.44.0 // indirect
	go.opentelemetry.io/otel/sdk v1.44.0 // indirect
	go.opentelemetry.io/otel/trace v1.44.0 // indirect
	go.opentelemetry.io/proto/otlp v1.11.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/term v0.45.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260723215102-3fe39f3c1018 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260723215102-3fe39f3c1018 // indirect
	google.golang.org/grpc v1.82.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
