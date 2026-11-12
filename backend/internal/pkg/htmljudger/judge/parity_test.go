package judge

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"math"
	"os"
	"testing"
)

type parityCase struct {
	Case    string `json:"case"`
	Prompt  string `json:"prompt"`
	HTML    string `json:"html"`
	HTMLB64 string `json:"html_b64"`
}

type recordedResult struct {
	Case       string   `json:"case"`
	Quality    string   `json:"quality"`
	Borderline bool     `json:"borderline"`
	Method     string   `json:"method"`
	Score      *float64 `json:"score"`
	Margin     float64  `json:"margin"`
	Reason     string   `json:"reason"`
}

func readJSONL[T any](t *testing.T, path string) []T {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1<<20), 64<<20)
	var rows []T
	for scanner.Scan() {
		var row T
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil {
			t.Fatalf("%s line %d: %v", path, len(rows)+1, err)
		}
		rows = append(rows, row)
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	return rows
}

// TestParityWithNewAPIJudge replays synthetic inputs (cases.jsonl) through
// this package and requires bit-identical results to the outputs recorded
// from the original New API service judge (newapi-recorded.jsonl, see
// testdata/parity/README.md for how they were recorded).
func TestParityWithNewAPIJudge(t *testing.T) {
	j := loadEmbedded(t)
	cases := readJSONL[parityCase](t, "testdata/parity/cases.jsonl")
	recorded := readJSONL[recordedResult](t, "testdata/parity/newapi-recorded.jsonl")
	if len(cases) == 0 || len(cases) != len(recorded) {
		t.Fatalf("%d cases, %d recorded results", len(cases), len(recorded))
	}
	methods := map[Method]int{}
	borderline := 0
	for i, c := range cases {
		want := recorded[i]
		if want.Case != c.Case {
			t.Fatalf("row %d: case %q recorded as %q", i, c.Case, want.Case)
		}
		html := c.HTML
		if c.HTMLB64 != "" {
			raw, err := base64.StdEncoding.DecodeString(c.HTMLB64)
			if err != nil {
				t.Fatal(err)
			}
			html = string(raw)
		}
		got := j.Judge(c.Prompt, html)
		if string(got.Quality) != want.Quality || got.Borderline != want.Borderline || string(got.Method) != want.Method ||
			got.Reason != want.Reason || math.Float64bits(got.Margin) != math.Float64bits(want.Margin) {
			t.Errorf("%s: got %+v, recorded %+v", c.Case, got, want)
			continue
		}
		switch {
		case (got.Score == nil) != (want.Score == nil):
			t.Errorf("%s: score presence differs", c.Case)
		case got.Score != nil && math.Float64bits(*got.Score) != math.Float64bits(*want.Score):
			t.Errorf("%s: score %v, recorded %v", c.Case, *got.Score, *want.Score)
		}
		methods[got.Method]++
		if got.Borderline {
			borderline++
		}
	}
	// The fixture must keep exercising every branch.
	if methods[MethodPool] == 0 || methods[MethodBank] == 0 || methods[MethodRaw] == 0 || borderline == 0 {
		t.Errorf("fixture coverage: methods %v, borderline %d", methods, borderline)
	}
	t.Logf("parity rows %d (methods %v, borderline %d): bit-identical", len(cases), methods, borderline)
}

type pythonParityRow struct {
	Case    string   `json:"case"`
	Prompt  string   `json:"prompt"`
	HTML    string   `json:"html"`
	Margin  float64  `json:"margin"`
	Z       *float64 `json:"z"`
	Method  string   `json:"method"`
	Quality string   `json:"quality"`
}

// TestParityWithFrozenPythonJudge compares against the frozen Python v6
// judge on a private fixture (real model outputs, not shipped). Run with
// JUDGE_PYTHON_PARITY_FILE=<parity.jsonl>. Python and Go sum floats
// differently, so margins agree to 1e-9 rather than bit-for-bit.
func TestParityWithFrozenPythonJudge(t *testing.T) {
	path := os.Getenv("JUDGE_PYTHON_PARITY_FILE")
	if path == "" {
		t.Skip("JUDGE_PYTHON_PARITY_FILE is not set")
	}
	j := loadEmbedded(t)
	rows := readJSONL[pythonParityRow](t, path)
	if len(rows) == 0 {
		t.Fatal("empty parity file")
	}
	maxDiff := 0.0
	for _, row := range rows {
		got := j.Judge(row.Prompt, row.HTML)
		maxDiff = math.Max(maxDiff, math.Abs(got.Margin-row.Margin))
		if math.Abs(got.Margin-row.Margin) > 1e-9 || string(got.Method) != row.Method {
			t.Errorf("%s: margin %v method %s, python %v %s", row.Case, got.Margin, got.Method, row.Margin, row.Method)
		}
		// The frozen judge labels the band "borderline"; this judge gives the
		// threshold's provisional verdict there and flags it instead.
		if got.Class() != row.Quality {
			t.Errorf("%s: class %s, python %s", row.Case, got.Class(), row.Quality)
		}
		if row.Quality == "borderline" && row.Z != nil {
			want := Normal
			if *row.Z >= j.Threshold() {
				want = Degraded
			}
			if got.Quality != want {
				t.Errorf("%s: provisional %s, want %s", row.Case, got.Quality, want)
			}
		}
		switch {
		case (row.Z == nil) != (got.Score == nil):
			t.Errorf("%s: score presence differs", row.Case)
		case row.Z != nil && math.Abs(*row.Z-*got.Score) > 1e-9:
			t.Errorf("%s: z %v, python %v", row.Case, *got.Score, *row.Z)
		}
	}
	t.Logf("python parity rows %d, max margin difference %.3g", len(rows), maxDiff)
}
