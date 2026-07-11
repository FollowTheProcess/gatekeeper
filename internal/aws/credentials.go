package aws

import (
	"fmt"
	"strings"
)

// Credentials holds the AWS credentials for an assumed session.
//
//nolint:tagliatelle // "Aws" looks dumb, acronyms should be upper.
type Credentials struct {
	AWSAccessKeyID     string `json:"AWSAccessKeyId"`
	AWSSecretAccessKey string `json:"AWSSecretAccessKey"`
	AWSSessionToken    string `json:"AWSSessionToken"`
}

// Export returns the `export ...` lines for the set of credentials.
func (c Credentials) Export() string {
	b := &strings.Builder{}

	fmt.Fprintf(b, "export AWS_ACCESS_KEY_ID=%s\n", quote(c.AWSAccessKeyID))
	fmt.Fprintf(b, "export AWS_SECRET_ACCESS_KEY=%s\n", quote(c.AWSSecretAccessKey))
	fmt.Fprintf(b, "export AWS_SESSION_TOKEN=%s\n", quote(c.AWSSessionToken))

	return b.String()
}

// quote returns a shell singled-quoted string of input, ensuring
// no expansions, commands etc. can be smuggled in via credentials.
//
// Probably not needed but eval(<something from the network>) should
// invoke caution regardless.
func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
