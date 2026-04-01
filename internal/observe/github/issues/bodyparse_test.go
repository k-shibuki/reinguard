package issues

import (
	"reflect"
	"testing"
)

func TestParseSections_h2Only(t *testing.T) {
	t.Parallel()
	body := "## One\n\n### Three\n\n## Two\n"
	got := ParseSections(body)
	want := []string{"One", "Two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestParseSections_skipsEmptyAndHashes(t *testing.T) {
	t.Parallel()
	body := "##\n##  \n## ###not\n## OK\n"
	got := ParseSections(body)
	want := []string{"OK"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v want %#v", got, want)
	}
}

func TestParseSections_noHeaders(t *testing.T) {
	t.Parallel()
	if ParseSections("") != nil {
		t.Fatal("want nil")
	}
	if ParseSections("no headers\n") != nil {
		t.Fatal("want nil")
	}
}

func TestHasBlockers(t *testing.T) {
	t.Parallel()
	if !HasBlockers("Blocked by #99") {
		t.Fatal("want true")
	}
	if HasBlockers("blocked by #1") {
		t.Fatal("want false for wrong case")
	}
	if HasBlockers("") {
		t.Fatal("want false")
	}
}
