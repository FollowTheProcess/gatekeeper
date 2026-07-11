package aws

import (
	"encoding/json"
	"fmt"
	"time"
)

// InlineUploadPolicy creates a scoped inline role policy JSON for a particular
// bucket, project and version that is bounded by exp via a "DateLessThan" condition
// so the inline credentials expire when the JWT does.
func InlineUploadPolicy(bucket, project, version string, exp time.Time) (string, error) {
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
				Condition: condition{
					DateLessThan: map[string]string{
						"aws:CurrentTime": exp.UTC().Format(time.RFC3339),
					},
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
	Effect    string    `json:"Effect"`
	Condition condition `json:"Condition"`
	Action    []string  `json:"Action"`
	Resource  []string  `json:"Resource"`
}

type condition struct {
	DateLessThan map[string]string `json:"DateLessThan"`
}
