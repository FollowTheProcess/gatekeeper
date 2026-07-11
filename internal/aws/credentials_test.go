package aws_test

import (
	"testing"

	"go.followtheprocess.codes/gatekeeper/internal/aws"
	"go.followtheprocess.codes/test"
)

func TestCredentialsExport(t *testing.T) {
	tests := []struct {
		name  string          // Name of the test case
		creds aws.Credentials // Credentials in
		want  string          // Expected output
	}{
		{
			name:  "empty",
			creds: aws.Credentials{},
			want: "export AWS_ACCESS_KEY_ID=''\n" +
				"export AWS_SECRET_ACCESS_KEY=''\n" +
				"export AWS_SESSION_TOKEN=''\n",
		},
		{
			name: "simple",
			creds: aws.Credentials{
				AWSAccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				AWSSecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				AWSSessionToken:    "FwoGZXIvYXdzEXAMPLE",
			},
			want: "export AWS_ACCESS_KEY_ID='AKIAIOSFODNN7EXAMPLE'\n" +
				"export AWS_SECRET_ACCESS_KEY='wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY'\n" +
				"export AWS_SESSION_TOKEN='FwoGZXIvYXdzEXAMPLE'\n",
		},
		{
			name: "injection is neutralised",
			creds: aws.Credentials{
				AWSAccessKeyID:     "AKIA$(rm -rf /)",
				AWSSecretAccessKey: "secret`whoami`",
				AWSSessionToken:    "tok;echo pwned",
			},
			want: "export AWS_ACCESS_KEY_ID='AKIA$(rm -rf /)'\n" +
				"export AWS_SECRET_ACCESS_KEY='secret`whoami`'\n" +
				"export AWS_SESSION_TOKEN='tok;echo pwned'\n",
		},
		{
			name: "embedded single quote is escaped",
			creds: aws.Credentials{
				AWSAccessKeyID:     "a'b",
				AWSSecretAccessKey: "c'd",
				AWSSessionToken:    "e'f",
			},
			want: `export AWS_ACCESS_KEY_ID='a'\''b'` + "\n" +
				`export AWS_SECRET_ACCESS_KEY='c'\''d'` + "\n" +
				`export AWS_SESSION_TOKEN='e'\''f'` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			test.Diff(t, tt.creds.Export(), tt.want)
		})
	}
}
