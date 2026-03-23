package schema

import "testing"

func TestListEmbedded_nonEmpty(t *testing.T) {
	t.Parallel()
	// Given: embedded schema FS is present
	// When: ListEmbedded walks embed FS
	names, err := ListEmbedded()
	// Then: multiple schema files are listed
	if err != nil {
		t.Fatal(err)
	}
	if len(names) < 4 {
		t.Fatal(names)
	}
}
