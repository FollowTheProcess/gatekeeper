package aws

import (
	"encoding/json"
	"fmt"
)

// InlineUploadPolicy creates a scoped inline role policy JSON for a particular
// bucket, project and version.
func InlineUploadPolicy(bucket, project, version string) (string, error) {
	doc := policyDocument{
		Version: "2012-10-17",
		Statement: []statement{
			{
				Effect: "Allow",
				Action: []string{
					"s3:PutObject",
					"s3:AbortMultipartUpload",
				},
				Resource: []string{
					fmt.Sprintf("arn:aws:s3:::%s/%s/%s/*", bucket, project, version),
				},
			},
		},
	}

	out, err := json.Marshal(doc)
	if err != nil {
		return "", fmt.Errorf("marshal session policy: %w", err)
	}

	return string(out), nil
}

type policyDocument struct {
	Version   string      `json:"Version"`
	Statement []statement `json:"Statement"`
}

type statement struct {
	Effect   string   `json:"Effect"`
	Action   []string `json:"Action"`
	Resource []string `json:"Resource"`
}
