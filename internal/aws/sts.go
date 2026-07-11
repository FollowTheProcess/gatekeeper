package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// STSClient is an interface wrapping access to STS.
type STSClient interface {
	// AssumeRole assumes an IAM role.
	AssumeRole(
		ctx context.Context,
		params *sts.AssumeRoleInput,
		options ...func(*sts.Options),
	) (*sts.AssumeRoleOutput, error)
}
