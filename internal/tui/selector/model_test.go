package selector

import (
	"sort"
	"testing"

	"github.com/mg/ai-tui/internal/config"
)

// itemPairs extracts the (provider, model) pairs from the selector's list
// items in their displayed order.
func itemPairs(t *testing.T, m Model) []ModelItem {
	t.Helper()
	items := m.list.Items()
	out := make([]ModelItem, 0, len(items))
	for _, it := range items {
		mi, ok := it.(ModelItem)
		if !ok {
			t.Fatalf("list item is not a ModelItem: %T", it)
		}
		out = append(out, mi)
	}
	return out
}

// TestNew_ListsAllProviderModelPairs proves the selector builds one entry per
// (provider, model) pair, not one per provider: a config with a single
// provider that lists two models yields two distinct items.
func TestNew_ListsAllProviderModelPairs(t *testing.T) {
	providers := map[string]config.Provider{
		"openai": {
			Model:  "gpt-4o",
			Models: []string{"gpt-4o", "gpt-4o-mini"},
		},
	}

	m := New(providers)
	pairs := itemPairs(t, m)

	if len(pairs) != 2 {
		t.Fatalf("expected 2 items (one per model), got %d: %+v", len(pairs), pairs)
	}

	got := map[string]string{}
	for _, p := range pairs {
		got[p.ModelName] = p.ProviderName
	}
	if got["gpt-4o"] != "openai" {
		t.Errorf("missing item for gpt-4o under openai; got %v", got)
	}
	if got["gpt-4o-mini"] != "openai" {
		t.Errorf("missing item for gpt-4o-mini under openai; got %v", got)
	}
}

// TestNew_ModelsSortedAlphabetically proves each provider's models are sorted
// so ordering is deterministic regardless of config order.
func TestNew_ModelsSortedAlphabetically(t *testing.T) {
	providers := map[string]config.Provider{
		"p": {
			Models: []string{"zeta", "alpha", "mid"},
		},
	}

	m := New(providers)
	pairs := itemPairs(t, m)

	var models []string
	for _, p := range pairs {
		models = append(models, p.ModelName)
	}
	want := []string{"alpha", "mid", "zeta"}
	if !sort.StringsAreSorted(models) || len(models) != len(want) {
		t.Fatalf("expected sorted %v, got %v", want, models)
	}
	for i, w := range want {
		if models[i] != w {
			t.Errorf("models[%d] = %q, want %q", i, models[i], w)
		}
	}
}

// TestNew_FallsBackToSingleModelField proves that when a provider lists no
// `models` (only the singular `model`), the selector still emits one item for
// it — backward compatibility with old configs.
func TestNew_FallsBackToSingleModelField(t *testing.T) {
	providers := map[string]config.Provider{
		"openai": {Model: "gpt-4"},
	}

	m := New(providers)
	pairs := itemPairs(t, m)

	if len(pairs) != 1 {
		t.Fatalf("expected 1 item from the singular model field, got %d", len(pairs))
	}
	if pairs[0].ProviderName != "openai" || pairs[0].ModelName != "gpt-4" {
		t.Errorf("unexpected item: %+v", pairs[0])
	}
}

// TestNew_ProviderNamesSorted proves providers are listed alphabetically even
// when multiple providers each have multiple models.
func TestNew_ProviderNamesSorted(t *testing.T) {
	providers := map[string]config.Provider{
		"zeta":     {Models: []string{"z1"}},
		"alpha":    {Models: []string{"a1", "a2"}},
		"midpoint": {Models: []string{"m1"}},
	}

	m := New(providers)
	pairs := itemPairs(t, m)

	var names []string
	for _, p := range pairs {
		names = append(names, p.ProviderName)
	}
	if !sort.StringsAreSorted(names) {
		t.Errorf("provider names not sorted: %v", names)
	}
}
