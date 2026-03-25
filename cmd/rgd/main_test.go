package main

import (
	"bytes"
	"testing"

	"github.com/k-shibuki/reinguard/internal/rgdcli"
)

func TestRun_version(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	app := rgdcli.NewApp("testver")
	app.Writer = &buf
	if err := app.Run([]string{"rgd", "version"}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("testver")) {
		t.Errorf("expected output to contain 'testver', got: %s", buf.String())
	}
}
