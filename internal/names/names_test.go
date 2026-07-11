package names_test

import (
	"regexp"
	"testing"

	"go.followtheprocess.codes/gatekeeper/internal/names"
	"go.followtheprocess.codes/test"
)

var nameRegex = regexp.MustCompile(`^[a-z]+-[a-z0-9]+$`)

func TestGet(t *testing.T) {
	for range 100 {
		got := names.Get()
		test.True(t, nameRegex.MatchString(got), test.Context("generated name %q did not match regex", got))
	}
}
