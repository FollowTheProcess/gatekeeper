package aws_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"go.followtheprocess.codes/gatekeeper/internal/aws"
	"go.followtheprocess.codes/test"
)

func TestSSMFetcher(t *testing.T) {
	tests := []struct {
		name    string // Name of the test case
		project string // Project to fetch
		want    []byte // Expected output
		wantErr bool   // Expected error
	}{
		{
			name:    "wrong param",
			project: "wrong",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "correct project",
			project: "test",
			want:    []byte("I'm a public key for the test project"),
			wantErr: false,
		},
		{
			name:    "not a string",
			project: "notastring",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "nil",
			project: "nil",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty",
			project: "empty",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newFakeSSMClient()

			fetcher := aws.NewSSMFetcher(client)

			got, err := fetcher.Fetch(t.Context(), tt.project)
			test.WantErr(t, err, tt.wantErr)
			test.DiffBytes(t, got, tt.want)
		})
	}
}

type fakeSSMClient struct {
	params map[string]*ssm.GetParameterOutput
}

func newFakeSSMClient() fakeSSMClient {
	return fakeSSMClient{
		params: map[string]*ssm.GetParameterOutput{
			"wrong": {
				Parameter: &types.Parameter{
					Type:  types.ParameterTypeString,
					Value: new("I'm the wrong param"),
				},
			},
			"/releases/test/public-key": {
				Parameter: &types.Parameter{
					Type:  types.ParameterTypeString,
					Value: new("I'm a public key for the test project"),
				},
			},
			"/releases/notastring/public-key": {
				Parameter: &types.Parameter{
					Type:  types.ParameterTypeStringList,
					Value: new("wrong,param,type"),
				},
			},
			"/releases/nil/public-key": {
				Parameter: &types.Parameter{
					Type:  types.ParameterTypeString,
					Value: nil,
				},
			},
			"/releases/empty/public-key": {
				Parameter: &types.Parameter{
					Type:  types.ParameterTypeString,
					Value: new(""),
				},
			},
		},
	}
}

func (f fakeSSMClient) GetParameter(
	ctx context.Context,
	params *ssm.GetParameterInput,
	options ...func(*ssm.Options),
) (*ssm.GetParameterOutput, error) {
	key := *params.Name

	val, ok := f.params[key]
	if !ok {
		return nil, fmt.Errorf("no such parameter %s", key)
	}

	return val, nil
}
