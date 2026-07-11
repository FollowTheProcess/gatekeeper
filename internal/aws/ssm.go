package aws

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// SSMClient is an interface wrapping access to SSM.
type SSMClient interface {
	// GetParameter gets a parameter by name from SSM.
	GetParameter(
		ctx context.Context,
		params *ssm.GetParameterInput,
		options ...func(*ssm.Options),
	) (*ssm.GetParameterOutput, error)
}

// SSMFetcher is a Fetcher that retrieves the public key
// from an SSM parameter.
type SSMFetcher struct {
	client SSMClient
}

// NewSSMFetcher creates a new [SSMFetcher].
func NewSSMFetcher(client SSMClient) SSMFetcher {
	return SSMFetcher{
		client: client,
	}
}

// Fetch fetches a public key from SSM.
func (s SSMFetcher) Fetch(ctx context.Context, project string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	paramPath := path.Join("/releases", project, "public-key")

	output, err := s.client.GetParameter(ctx, &ssm.GetParameterInput{
		Name: new(paramPath),
	})
	if err != nil || output == nil {
		return nil, fmt.Errorf("could not fetch SSM parameter: %w", err)
	}

	if output.Parameter.Type != types.ParameterTypeString {
		return nil, fmt.Errorf(
			"SSM parameter has incorrect type, expected %s, got %s",
			types.ParameterTypeString,
			output.Parameter.Type,
		)
	}

	value := output.Parameter.Value
	if value == nil {
		return nil, errors.New("SSM parameter value was nil")
	}

	actual := *value
	if len(actual) == 0 {
		return nil, errors.New("SSM parameter value was empty")
	}

	return []byte(actual), nil
}
