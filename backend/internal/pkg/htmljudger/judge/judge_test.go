package judge

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/Wei-Shaw/sub2api/internal/pkg/htmljudger/assets"
)

func TestNormalizeTextMatchesTrainedVectorizer(t *testing.T) {
	cases := map[string]string{
		"A\nB":       "a\nb", // a single whitespace character is preserved
		"a \t\n b":   "a b",
		"x  y":       "x y",
		"x\x1c\x1dy": "x y", // Python str.isspace includes information separators
		"İX":         "i̇x",
		"<SVG>":      "<svg>",
	}
	for input, expected := range cases {
		if got := normalizeText(input); got != expected {
			t.Errorf("normalizeText(%q) = %q, want %q", input, got, expected)
		}
	}
}

// Expected margins were computed by the Python reference judge (judge.py) on
// the same synthetic model.
func TestMarginMatchesPythonReference(t *testing.T) {
	vocabulary := []string{"<sv", "svg", "vg>", "g> ", "> <", "abc", "i̇x", "a b", "a\nb", "<svg>"}
	s := &scorer{model: modelFile{
		Representation: "char_ngrams", NgramRange: []int{3, 5}, Vocabulary: vocabulary,
		IDF:       []float64{1.5, 2.0, 1.25, 3.0, 0.5, 1.0, 2.5, 1.75, 2.25, 4.0},
		Coef:      []float64{0.8, -0.4, 0.3, 1.2, -2.0, 0.6, -0.9, 0.45, -1.1, 0.7},
		Intercept: 0.25,
	}, index: map[string]int32{}}
	for i, term := range vocabulary {
		s.index[term] = int32(i)
	}
	cases := map[string]float64{
		"<svg> <svg>":       1.2328234414326733,
		"<SVG>   abc":       1.5870934350202015,
		"A\nB a  b İX":      -0.78852424559836454,
		"zzz":               0.25,
		"<svg>  abc\x1cabc": 1.6212705895585104,
	}
	for html, expected := range cases {
		if got := s.margin(html); math.Abs(got-expected) > 1e-12 {
			t.Errorf("margin(%q) = %v, want %v", html, got, expected)
		}
	}
}

func TestLoadRejectsMissingOrUnpinnedAssets(t *testing.T) {
	if _, err := LoadDir(""); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("empty dir: got %v", err)
	}
	dir := t.TempDir()
	if _, err := LoadDir(dir); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("missing files: got %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ModelFile), []byte(`{"representation":"char_ngrams"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadDir(dir); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("unpinned model: got %v", err)
	}

	// The real model with a tampered prompts.json must also be refused.
	model, err := assets.FS.ReadFile(ModelFile)
	if err != nil {
		t.Fatal(err)
	}
	references, err := assets.FS.ReadFile(ReferencesFile)
	if err != nil {
		t.Fatal(err)
	}
	tampered := fstest.MapFS{
		ModelFile:      {Data: model},
		ReferencesFile: {Data: references},
		PromptsFile:    {Data: []byte(`["x"]`)},
	}
	if _, err := Load(tampered); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("tampered prompts: got %v", err)
	}
	// prompts.json is optional.
	delete(tampered, PromptsFile)
	j, err := Load(tampered)
	if err != nil {
		t.Fatalf("without prompts.json: %v", err)
	}
	if j.BankPrompts() != nil {
		t.Fatal("BankPrompts should be nil without prompts.json")
	}
}

func loadEmbedded(t testing.TB) *Judger {
	t.Helper()
	j, err := Load(assets.FS)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func TestEmbeddedAssetsDescribeTheFrozenV6Judge(t *testing.T) {
	j := loadEmbedded(t)
	if j.Version() != "v6-20261029" {
		t.Errorf("version %q", j.Version())
	}
	if j.Threshold() != 0.33922200916651957 || j.RawThreshold() != -0.16967199352767035 {
		t.Errorf("thresholds %v %v", j.Threshold(), j.RawThreshold())
	}
	if low, high := j.Band(); low != 0.2 || high != 0.5 {
		t.Errorf("band [%v, %v)", low, high)
	}
	prompts := j.BankPrompts()
	if len(prompts) != 200 || len(j.references.Prompts) != 200 {
		t.Fatalf("bank: %d prompts, %d references", len(prompts), len(j.references.Prompts))
	}
	pool := 0
	for _, prompt := range prompts {
		reference, ok := j.references.Prompts[PromptKey(prompt)]
		if !ok {
			t.Fatalf("prompt without reference: %q", prompt)
		}
		if reference.InPool {
			pool++
		}
	}
	if pool != 100 {
		t.Errorf("pool prompts %d, want 100", pool)
	}
	prompts[0] = "mutated"
	if j.BankPrompts()[0] == "mutated" {
		t.Error("BankPrompts must return a copy")
	}
}

// A synthetic judge whose one vocabulary term never matches "zzz", so margin
// equals the intercept, against bank references 0..1.
func syntheticJudge(intercept float64) *Judger {
	return &Judger{
		scorer: scorer{model: modelFile{Representation: "char_ngrams", NgramRange: []int{3, 3},
			Vocabulary: []string{"abc"}, IDF: []float64{1}, Coef: []float64{0}, Intercept: intercept, Threshold: 0.1},
			index: map[string]int32{"abc": 0}},
		references: referencesFile{Version: "test", Threshold: 0.33922200916651957, Band: []float64{0.2, 0.5},
			Prompts: map[string]referenceEntry{PromptKey("bank"): {Low: 0, High: 1, InPool: true}}},
	}
}

func TestProvisionalVerdictsInsideTheBorderlineBand(t *testing.T) {
	cases := []struct {
		z          float64
		quality    Quality
		borderline bool
		class      string
	}{
		{0.19, Normal, false, "normal"},
		{0.2, Normal, true, "borderline"},
		{0.339, Normal, true, "borderline"},
		{0.33922200916651957, Degraded, true, "borderline"},
		{0.49, Degraded, true, "borderline"},
		{0.5, Degraded, false, "degraded"},
	}
	for _, c := range cases {
		got := syntheticJudge(c.z).Judge("bank", "zzz")
		if got.Quality != c.quality || got.Borderline != c.borderline || got.Class() != c.class || got.Method != MethodPool {
			t.Errorf("z=%v: got %+v", c.z, got)
		}
		if got.Score == nil || math.Abs(*got.Score-c.z) > 1e-12 {
			t.Errorf("z=%v: score %v", c.z, got.Score)
		}
	}

	// Prompts outside the bank use the raw threshold and never carry the flag.
	outside := syntheticJudge(0.3).Judge("manual prompt", "zzz")
	if outside.Quality != Degraded || outside.Borderline || outside.Method != MethodRaw || outside.Score != nil {
		t.Errorf("outside the bank: got %+v", outside)
	}
}

func TestCustomReferences(t *testing.T) {
	if _, err := NewReference([]float64{-1}, []float64{1, 2}); !errors.Is(err, ErrBadReference) {
		t.Errorf("one normal margin: %v", err)
	}
	if _, err := NewReference([]float64{-1, -0.5}, []float64{0, 0.2}); !errors.Is(err, ErrBadReference) {
		t.Errorf("gap below minimum: %v", err)
	}
	ref, err := NewReference([]float64{-2, -1}, []float64{1, 2})
	if err != nil || ref.Low != -1.5 || ref.High != 1.5 {
		t.Fatalf("NewReference: %+v %v", ref, err)
	}

	j := loadEmbedded(t)
	if _, err := Load(assets.FS, WithReference(j.BankPrompts()[0], ref)); !errors.Is(err, ErrBadReference) {
		t.Errorf("overriding a bank prompt: %v", err)
	}
	if _, err := Load(assets.FS, WithReference("p", Reference{Low: 0, High: math.NaN()})); !errors.Is(err, ErrBadReference) {
		t.Errorf("NaN reference: %v", err)
	}

	custom, err := Load(assets.FS, WithReference("my prompt", ref))
	if err != nil {
		t.Fatal(err)
	}
	html := "<svg><use href=\"#a\"/></svg>"
	got := custom.Judge("my prompt", html)
	margin := custom.Margin(html)
	want := (margin - ref.Low) / (ref.High - ref.Low)
	if got.Method != MethodCustom || got.Score == nil || *got.Score != want || got.Margin != margin {
		t.Errorf("custom: got %+v, want z %v", got, want)
	}
	if other := custom.Judge("other prompt", html); other.Method != MethodRaw {
		t.Errorf("other prompt: %+v", other)
	}
	// Bank verdicts are unchanged by custom references.
	bank := j.BankPrompts()[0]
	if a, b := j.Judge(bank, html), custom.Judge(bank, html); a.Method != b.Method || *a.Score != *b.Score {
		t.Errorf("bank verdict changed: %+v vs %+v", a, b)
	}
}

func TestConcurrentUseIsSafeAndDeterministic(t *testing.T) {
	j := loadEmbedded(t)
	prompts := j.BankPrompts()
	html, err := os.ReadFile("../cmd/demo/samples/pelican-degraded.html")
	if err != nil {
		t.Fatal(err)
	}
	want := make([]Result, 16)
	for i := range want {
		want[i] = j.Judge(prompts[i], string(html))
	}
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for round := 0; round < 4; round++ {
				for i := range want {
					got := j.Judge(prompts[i], string(html))
					if got.Margin != want[i].Margin || *got.Score != *want[i].Score || got.Quality != want[i].Quality {
						t.Errorf("concurrent result differs for prompt %d", i)
					}
				}
			}
		}()
	}
	wg.Wait()
}

func BenchmarkJudge(b *testing.B) {
	j := loadEmbedded(b)
	html, err := os.ReadFile("../cmd/demo/samples/pelican-normal.html")
	if err != nil {
		b.Fatal(err)
	}
	prompt := j.BankPrompts()[0]
	b.SetBytes(int64(len(html)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j.Judge(prompt, string(html))
	}
}
