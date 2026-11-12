// Package judge is a standalone port of the frozen v6 HTML degradation judge.
//
// Given the prompt that was sent to a model and the HTML artifact it returned,
// the judge decides whether the artifact looks like the output of a normal
// model or of a degraded ("dumbed-down") one. It runs fully locally: no network
// calls and no model tokens.
//
// Load the assets once with Load or LoadDir and share the returned *Judger;
// it is immutable and safe for concurrent use.
package judge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Pinned SHA-256 digests of the frozen v6 assets. Load refuses any other
// bytes, so a swapped or partial copy can never score artifacts.
const (
	ModelSHA256      = "fc07b7cc8839bcbf2568ac58a8ac0ae615bfc11579c4343986f9ad2847ba25ff"
	ReferencesSHA256 = "2355c00315f80dd2f3649a5259df1be7fd941452dd8ed8d3a9bf7baf69ac46d1"
	PromptsSHA256    = "3a4de754d5ff6967ee82dd71489d81b3f68a03151bcf6e2efd5d705e4400ec0b"
)

// Asset file names inside the asset directory or fs.FS.
const (
	ModelFile      = "model.json"
	ReferencesFile = "references.json"
	PromptsFile    = "prompts.json" // optional: texts of the reference-bank prompts
)

// Errors returned by Load. Use errors.Is to test for them.
var (
	ErrUnavailable    = errors.New("judge assets are unavailable")
	ErrDigestMismatch = errors.New("judge assets do not match the pinned version")
	ErrMalformed      = errors.New("judge assets are malformed")
	ErrBadReference   = errors.New("invalid custom reference")
)

// Quality is the binary verdict.
type Quality string

const (
	Normal   Quality = "normal"
	Degraded Quality = "degraded"
)

// Method says how a verdict was reached.
type Method string

const (
	// MethodPool: prompt is one of the 100 bank prompts used to fit the v6
	// threshold; its reference means come from the same data.
	MethodPool Method = "v6_pool"
	// MethodBank: prompt is one of the other 100 bank prompts.
	MethodBank Method = "v6_bank"
	// MethodCustom: prompt has a caller-supplied reference (WithReference).
	MethodCustom Method = "v6_custom"
	// MethodRaw: prompt has no reference; the raw V5b margin threshold is used
	// and no borderline band applies.
	MethodRaw Method = "raw_v5b"
)

// Result is one verdict.
type Result struct {
	// Quality is the effective verdict. Inside the borderline band it is a
	// provisional verdict that should be confirmed by a human.
	Quality Quality `json:"quality"`
	// Borderline marks a low-confidence verdict (z inside the band). It is
	// only set for prompts with references, never for MethodRaw.
	Borderline bool   `json:"borderline"`
	Method     Method `json:"method"`
	// Score is the same-prompt position z: 0 at the normal reference mean,
	// 1 at the degraded reference mean. Nil for MethodRaw.
	Score *float64 `json:"score"`
	// Margin is the raw V5b decision value (higher means more degraded).
	Margin float64 `json:"margin"`
	Reason string  `json:"reason"`
}

// Class is the three-class view used by the frozen offline judge:
// "normal", "degraded", or "borderline" when Borderline is set.
func (r Result) Class() string {
	if r.Borderline {
		return "borderline"
	}
	return string(r.Quality)
}

type modelFile struct {
	Representation string    `json:"representation"`
	NgramRange     []int     `json:"ngram_range"`
	Vocabulary     []string  `json:"vocabulary"`
	IDF            []float64 `json:"idf"`
	Coef           []float64 `json:"coef"`
	Intercept      float64   `json:"intercept"`
	Threshold      float64   `json:"threshold"`
}

type referenceEntry struct {
	Low         float64 `json:"low"`
	High        float64 `json:"high"`
	InPool      bool    `json:"in_pool"`
	LocalSHA256 string  `json:"local_sha256"`
	USSHA256    string  `json:"us_sha256"`
}

type referencesFile struct {
	Version   string                    `json:"version"`
	Threshold float64                   `json:"threshold"`
	Band      []float64                 `json:"band"`
	Prompts   map[string]referenceEntry `json:"prompts"`
}

type scorer struct {
	model modelFile
	index map[string]int32
}

// Judger scores HTML artifacts. It is immutable after Load and safe for
// concurrent use by any number of goroutines.
type Judger struct {
	scorer
	references referencesFile
	custom     map[string]Reference
	prompts    []string
}

// Option customizes Load.
type Option func(*loadConfig) error

type loadConfig struct {
	custom map[string]Reference
}

// LoadDir loads the assets from a directory on disk.
func LoadDir(dir string, opts ...Option) (*Judger, error) {
	if dir == "" {
		return nil, fmt.Errorf("%w: asset directory is not configured", ErrUnavailable)
	}
	return Load(os.DirFS(dir), opts...)
}

// Load loads and verifies the assets from fsys (for example assets.FS or
// os.DirFS). model.json and references.json are required and must match the
// pinned digests; prompts.json is optional but verified when present.
func Load(fsys fs.FS, opts ...Option) (*Judger, error) {
	var config loadConfig
	for _, opt := range opts {
		if err := opt(&config); err != nil {
			return nil, err
		}
	}
	j := &Judger{custom: config.custom}
	if err := readPinnedJSON(fsys, ModelFile, ModelSHA256, &j.model); err != nil {
		return nil, err
	}
	if err := readPinnedJSON(fsys, ReferencesFile, ReferencesSHA256, &j.references); err != nil {
		return nil, err
	}
	m := j.model
	if m.Representation != "char_ngrams" || len(m.NgramRange) != 2 || m.NgramRange[0] < 1 || m.NgramRange[1] < m.NgramRange[0] ||
		len(m.IDF) != len(m.Vocabulary) || len(m.Coef) != len(m.Vocabulary) || len(j.references.Band) != 2 {
		return nil, ErrMalformed
	}
	if _, err := fs.Stat(fsys, PromptsFile); err == nil {
		if err := readPinnedJSON(fsys, PromptsFile, PromptsSHA256, &j.prompts); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s: %v", ErrUnavailable, PromptsFile, err)
	}
	for key := range j.custom {
		if _, ok := j.references.Prompts[key]; ok {
			return nil, fmt.Errorf("%w: prompt is already in the reference bank", ErrBadReference)
		}
	}
	j.index = make(map[string]int32, len(m.Vocabulary))
	for i, term := range m.Vocabulary {
		j.index[term] = int32(i)
	}
	return j, nil
}

func readPinnedJSON(fsys fs.FS, name, digest string, target any) error {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrUnavailable, name, err)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != digest {
		return fmt.Errorf("%w: %s", ErrDigestMismatch, name)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrMalformed, name, err)
	}
	return nil
}

// PromptKey is the reference-bank key of a prompt: the hex SHA-256 of its
// exact UTF-8 bytes. Any change (whitespace, punctuation) gives another key.
func PromptKey(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return hex.EncodeToString(sum[:])
}

// Version is the reference set version, e.g. "v6-20261029".
func (j *Judger) Version() string { return j.references.Version }

// Threshold is the same-prompt z at or above which an artifact is degraded.
func (j *Judger) Threshold() float64 { return j.references.Threshold }

// Band is the borderline interval [low, high) of z.
func (j *Judger) Band() (low, high float64) { return j.references.Band[0], j.references.Band[1] }

// RawThreshold is the V5b margin threshold used for prompts without references.
func (j *Judger) RawThreshold() float64 { return j.model.Threshold }

// InBank reports whether prompt has a built-in same-prompt reference.
func (j *Judger) InBank(prompt string) bool {
	_, ok := j.references.Prompts[PromptKey(prompt)]
	return ok
}

// BankPrompts returns a copy of the bank prompt texts, or nil when the
// optional prompts.json was not loaded.
func (j *Judger) BankPrompts() []string {
	if j.prompts == nil {
		return nil
	}
	return append([]string(nil), j.prompts...)
}

// Margin returns the raw V5b decision value of html (higher means more
// degraded). It is the input to NewReference.
func (j *Judger) Margin(html string) float64 { return j.margin(html) }

// Judge scores one artifact. Bank prompts are judged against their own
// references; inside the borderline band the threshold still gives a
// provisional verdict and the result is flagged for human review. Other
// prompts fall back to the raw V5b threshold without a band.
func (j *Judger) Judge(prompt, html string) Result {
	margin := j.margin(html)
	key := PromptKey(prompt)
	reference, ok := j.references.Prompts[key]
	if !ok {
		if custom, found := j.custom[key]; found {
			return j.positioned(margin, custom.Low, custom.High, MethodCustom, "custom same-prompt position")
		}
		quality := Normal
		if margin >= j.model.Threshold {
			quality = Degraded
		}
		return Result{Quality: quality, Method: MethodRaw, Margin: margin,
			Reason: fmt.Sprintf("Built-in judge %s: prompt outside the reference bank, raw margin %.3f", j.references.Version, margin)}
	}
	method := MethodBank
	if reference.InPool {
		method = MethodPool
	}
	return j.positioned(margin, reference.Low, reference.High, method, "same-prompt position")
}

func (j *Judger) positioned(margin, low, high float64, method Method, label string) Result {
	z := (margin - low) / (high - low)
	quality := Normal
	if z >= j.references.Threshold {
		quality = Degraded
	}
	band := j.references.Band
	return Result{Quality: quality, Borderline: z >= band[0] && z < band[1], Method: method,
		Score: &z, Margin: margin, Reason: fmt.Sprintf("Built-in judge %s: %s %.3f", j.references.Version, label, z)}
}
