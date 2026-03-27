package schema

import "testing"

func TestNewCompiler(t *testing.T) {
	t.Parallel()
	// Given/When: compiler loads embedded schemas
	c, err := NewCompiler()
	// Then: root schema compiles
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Compile(URIReinguardConfig); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Compile(URILabelsConfig); err != nil {
		t.Fatal(err)
	}
}
