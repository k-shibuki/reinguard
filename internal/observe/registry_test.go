package observe

import (
	"errors"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/config"
)

func TestProviderRegistry_Register_duplicate(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	if err := r.Register("a", func(map[string]any) (Provider, error) {
		return &stubProv{id: "a"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	err := r.Register("a", func(map[string]any) (Provider, error) {
		return &stubProv{id: "a"}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("want duplicate error, got %v", err)
	}
}

func TestProviderRegistry_Register_nilFactory(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	err := r.Register("x", nil)
	if err == nil || !strings.Contains(err.Error(), "nil factory") {
		t.Fatalf("got %v", err)
	}
}

func TestProviderRegistry_Register_zeroValue(t *testing.T) {
	t.Parallel()
	var r ProviderRegistry
	if err := r.Register("a", func(map[string]any) (Provider, error) {
		return &stubProv{id: "a"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	out, err := r.Build([]config.ProviderSpec{{ID: "a", Enabled: true}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("got %d providers", len(out))
	}
}

func TestProviderRegistry_Build_unknownProvider(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	_, err := r.Build([]config.ProviderSpec{{ID: "nope", Enabled: true}})
	if err == nil || !strings.Contains(err.Error(), "unknown provider") {
		t.Fatalf("got %v", err)
	}
}

func TestProviderRegistry_Build_emptyEnabledID(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	_, err := r.Build([]config.ProviderSpec{{ID: "   ", Enabled: true}})
	if err == nil || !strings.Contains(err.Error(), "empty id") {
		t.Fatalf("got %v", err)
	}
}

func TestProviderRegistry_Build_duplicateEnabledID(t *testing.T) {
	t.Parallel()
	reg := DefaultRegistry()
	_, err := reg.Build([]config.ProviderSpec{
		{ID: "git", Enabled: true},
		{ID: "git", Enabled: true},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("got %v", err)
	}
}

func TestProviderRegistry_Build_skipsDisabled(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	if err := r.Register("a", func(map[string]any) (Provider, error) {
		return &stubProv{id: "a", frag: Fragment{Signals: map[string]any{"k": 1}}}, nil
	}); err != nil {
		t.Fatal(err)
	}
	m, err := r.Build([]config.ProviderSpec{
		{ID: "a", Enabled: false},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 0 {
		t.Fatalf("got %v", m)
	}
}

func TestProviderRegistry_Build_factoryError(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	if err := r.Register("bad", func(map[string]any) (Provider, error) {
		return nil, errors.New("boom")
	}); err != nil {
		t.Fatal(err)
	}
	_, err := r.Build([]config.ProviderSpec{{ID: "bad", Enabled: true}})
	if err == nil || !strings.Contains(err.Error(), "build provider") {
		t.Fatalf("got %v", err)
	}
}

func TestProviderRegistry_Build_nilProvider(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	if err := r.Register("nilp", func(map[string]any) (Provider, error) {
		return nil, nil
	}); err != nil {
		t.Fatal(err)
	}
	_, err := r.Build([]config.ProviderSpec{{ID: "nilp", Enabled: true}})
	if err == nil || !strings.Contains(err.Error(), "returned nil") {
		t.Fatalf("got %v", err)
	}
}

func TestProviderRegistry_Build_idMismatch(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	if err := r.Register("x", func(map[string]any) (Provider, error) {
		return &stubProv{id: "wrong"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	_, err := r.Build([]config.ProviderSpec{{ID: "x", Enabled: true}})
	if err == nil || !strings.Contains(err.Error(), "returned provider id") {
		t.Fatalf("got %v", err)
	}
}

func TestDefaultRegistry_Build_githubOptionsRoundTrip(t *testing.T) {
	t.Parallel()
	wantBase := "https://api.example.com/"
	reg := DefaultRegistry()
	m, err := reg.Build([]config.ProviderSpec{
		{ID: "github", Enabled: true, Options: map[string]any{"api_base": wantBase}},
	})
	if err != nil {
		t.Fatal(err)
	}
	gp, ok := m["github"].(*GitHubProvider)
	if !ok {
		t.Fatalf("got %T", m["github"])
	}
	if gp.APIBase != wantBase {
		t.Fatalf("APIBase=%q want %q", gp.APIBase, wantBase)
	}
}

func TestDefaultRegistry_shallowCopyOptions(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	var captured map[string]any
	if err := reg.Register("mut", func(opts map[string]any) (Provider, error) {
		captured = opts
		if opts != nil {
			opts["injected"] = true
		}
		return &stubProv{id: "mut"}, nil
	}); err != nil {
		t.Fatal(err)
	}
	specOpts := map[string]any{"a": 1}
	_, err := reg.Build([]config.ProviderSpec{{ID: "mut", Enabled: true, Options: specOpts}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := specOpts["injected"]; ok {
		t.Fatal("factory mutated original spec options map")
	}
	if captured == nil {
		t.Fatal("expected captured opts")
	}
	if _, ok := captured["injected"]; !ok {
		t.Fatal("expected mutation only on copy passed to factory")
	}
}
