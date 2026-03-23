package main

import (
	"testing"
)

func TestRun_version(t *testing.T) {
	t.Parallel()
	if err := run([]string{"rgd", "version"}, "testver"); err != nil {
		t.Fatal(err)
	}
}
