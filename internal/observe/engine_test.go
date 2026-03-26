package observe

import (
	"context"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/config"
)

type stubProv struct {
	err  error
	id   string
	frag Fragment
}

func (s *stubProv) ID() string { return s.id }

func (s *stubProv) Collect(ctx context.Context, opts Options) (Fragment, error) {
	return s.frag, s.err
}

func TestEngine_Collect_serial(t *testing.T) {
	t.Parallel()
	// Given: two stub providers enabled
	e := &Engine{Providers: map[string]Provider{
		"a": &stubProv{id: "a", frag: Fragment{Signals: map[string]any{"x": 1}}},
		"b": &stubProv{id: "b", frag: Fragment{Signals: map[string]any{"y": 2}}},
	}}
	root := config.Root{
		Providers: []config.ProviderSpec{
			{ID: "a", Enabled: true},
			{ID: "b", Enabled: true},
		},
	}
	// When: Collect with Serial
	signals, diags, _, err := e.Collect(context.Background(), &root, Options{Serial: true, WorkDir: t.TempDir()})
	// Then: merged signals
	if err != nil {
		t.Fatal(err)
	}
	aMap, ok := signals["a"].(map[string]any)
	if !ok {
		t.Fatalf("expected signals[a] map, got %T", signals["a"])
	}
	if aMap["x"] != 1 {
		t.Fatalf("%v", signals)
	}
	if len(diags) != 0 {
		t.Fatal(diags)
	}
}

func TestEngine_Collect_parallelSameSignals(t *testing.T) {
	t.Parallel()
	// Given: one enabled stub provider
	e := &Engine{Providers: map[string]Provider{
		"a": &stubProv{id: "a", frag: Fragment{Signals: map[string]any{"v": 1}}},
	}}
	root := config.Root{Providers: []config.ProviderSpec{{ID: "a", Enabled: true}}}
	// When: Collect runs without Serial
	signals, _, _, err := e.Collect(context.Background(), &root, Options{Serial: false, WorkDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	aMap, ok := signals["a"].(map[string]any)
	if !ok {
		t.Fatalf("expected signals[a] map, got %T", signals["a"])
	}
	// Then: merged fragment under provider id
	if aMap["v"] != 1 {
		t.Fatalf("%v", signals)
	}
}

func TestEngine_Collect_nilRoot(t *testing.T) {
	t.Parallel()
	// Given: nil root
	e := NewEngine()
	// When: Collect runs
	_, _, _, err := e.Collect(context.Background(), nil, Options{Serial: true})
	// Then: nil config root error
	if err == nil || !strings.Contains(err.Error(), "nil config root") {
		t.Fatalf("got err=%v", err)
	}
}

func TestNewEngineFromConfig(t *testing.T) {
	t.Parallel()
	//nolint:govet // table-driven case struct; fieldalignment is not worth obfuscating field order
	tests := []struct {
		name         string
		wantErr      string
		wantGit      bool
		wantNoGitHub bool
		specs        []config.ProviderSpec
	}{
		{
			name: "git_only",
			specs: []config.ProviderSpec{
				{ID: "git", Enabled: true},
			},
			wantGit:      true,
			wantNoGitHub: true,
		},
		{
			name:    "unknown_provider",
			wantErr: "unknown provider",
			specs: []config.ProviderSpec{
				{ID: "unknown", Enabled: true},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Given: provider specs (see subtest name)
			// When: NewEngineFromConfig builds the engine
			e, err := NewEngineFromConfig(tt.specs)
			// Then: error or providers match expectations
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if e == nil || e.Providers == nil {
				t.Fatal("nil engine")
			}
			if tt.wantGit {
				if _, ok := e.Providers["git"]; !ok {
					t.Fatal("missing git provider")
				}
			}
			if tt.wantNoGitHub {
				if _, ok := e.Providers["github"]; ok {
					t.Fatal("unexpected github provider")
				}
			}
		})
	}
}

func TestEngine_unknownProvider(t *testing.T) {
	t.Parallel()
	// Given: config references missing provider
	e := &Engine{Providers: map[string]Provider{}}
	root := config.Root{Providers: []config.ProviderSpec{{ID: "nope", Enabled: true}}}
	// When: Collect runs
	_, diags, deg, err := e.Collect(context.Background(), &root, Options{Serial: true})
	// Then: degraded with diagnostic
	if err != nil {
		t.Fatal(err)
	}
	if !deg {
		t.Fatal("expected degraded")
	}
	if len(diags) != 1 || diags[0].Code != "provider_failed" || diags[0].Provider != "nope" {
		t.Fatalf("%+v", diags)
	}
}
