// Command gen writes the synthetic parity inputs (cases.jsonl). It is
// deterministic: re-running it yields byte-identical output. Expected outputs
// are recorded separately by running the original New API judge on these
// inputs (see ../README.md).
//
//	go run ./judge/testdata/parity/gen > judge/testdata/parity/cases.jsonl
package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

type parityCase struct {
	Case    string `json:"case"`
	Prompt  string `json:"prompt"`
	HTML    string `json:"html,omitempty"`
	HTMLB64 string `json:"html_b64,omitempty"` // for inputs that are not valid UTF-8
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func read(path string) string {
	data, err := os.ReadFile(path)
	must(err)
	return string(data)
}

func main() {
	var prompts []string
	must(json.Unmarshal([]byte(read("assets/prompts.json")), &prompts))
	samples := []string{"pelican-normal", "pelican-borderline", "pelican-degraded"}
	texts := map[string]string{}
	for _, name := range samples {
		texts[name] = read("cmd/demo/samples/" + name + ".html")
	}
	normalLines := strings.Split(texts["pelican-normal"], "\n")
	degradedTokens := strings.SplitAfter(texts["pelican-degraded"], "> ")

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	emit := func(c parityCase) {
		if !utf8.ValidString(c.HTML) {
			c.HTMLB64, c.HTML = base64.StdEncoding.EncodeToString([]byte(c.HTML)), ""
		}
		must(enc.Encode(c))
	}

	offBank := "请用 SVG 画一只鹈鹕骑自行车的动画"
	for _, name := range samples {
		emit(parityCase{Case: "sample:" + name + ":bank", Prompt: prompts[0], HTML: texts[name]})
		emit(parityCase{Case: "sample:" + name + ":offbank", Prompt: offBank, HTML: texts[name]})
		emit(parityCase{Case: "sample:" + name + ":bank-trailing-space", Prompt: prompts[0] + " ", HTML: texts[name]})
	}

	// One blended artifact per bank prompt: a seeded mix of normal-style lines
	// and degraded-style tokens, so z spans both sides of the threshold and
	// the borderline band. Each is also scored under an off-bank prompt.
	rng := rand.New(rand.NewSource(20261110))
	for i, prompt := range prompts {
		share := float64(i%20) / 19
		var b strings.Builder
		for k := 0; k < 60; k++ {
			if rng.Float64() < share {
				b.WriteString(degradedTokens[rng.Intn(len(degradedTokens))])
			} else {
				b.WriteString(normalLines[rng.Intn(len(normalLines))])
				b.WriteByte('\n')
			}
		}
		html := b.String()
		emit(parityCase{Case: "blend:" + strconv.Itoa(i+1) + ":bank", Prompt: prompt, HTML: html})
		emit(parityCase{Case: "blend:" + strconv.Itoa(i+1) + ":offbank", Prompt: prompt + "（变体）", HTML: html})
	}

	// Text normalization edge cases.
	edges := map[string]string{
		"empty":            "",
		"one-rune":         "<",
		"two-runes":        "<s",
		"no-vocabulary":    "zzz",
		"whitespace-only":  " \t\n\r\n  ",
		"single-newline":   "<svg>\n<use href=\"#a\"/>",
		"crlf":             "<svg>\r\n<use href=\"#a\"/>\r\n</svg>",
		"nbsp-run":         "<svg>  <rect fill=\"rgba(0,0,0,0.5)\"/>",
		"info-separators":  "<svg>\x1c\x1d<g>\x1e</g>\x1f</svg>",
		"dotted-capital-i": "İX <SVG VIEWBOX=\"0 0 10 10\"> İİ",
		"uppercase":        "<SVG><USE HREF=\"#WHEEL\"/><!-- THE WHEEL. --></SVG>",
		"cjk-only":         "播放 暂停 继续 动画 鹈鹕 自行车",
		"emoji":            "<svg>🚲🐦 <text>🚲</text></svg>",
		"combining":        "é å <desc>café</desc>",
		"ideographic-sp":   "<svg>　　<g/> <g/> </svg>",
		"invalid-utf8":     "<svg>\xff\xfe<rect/>\xc3</svg>",
		"tabs-mixed":       "<style>\t:root {\t--sky: #fff; }\t\t.a { fill: rgba(255,255,255,0.2); }</style>",
	}
	for _, key := range sortedKeys(edges) {
		emit(parityCase{Case: "edge:" + key, Prompt: prompts[1], HTML: edges[key]})
		emit(parityCase{Case: "edge:" + key + ":offbank", Prompt: offBank, HTML: edges[key]})
	}

	// Random strings over an alphabet of HTML, whitespace and non-ASCII runes.
	alphabet := []rune("<>/=\"'#:;{}()., -_\t\n\r 　\x1cabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789İİı鹈鹕动画暂停🚲")
	fragments := []string{"<use href=\"#", "rgba(255,255,255,", "<feMerge>", "<!-- ", ". -->", "transform-box: fill-box;", "<stop offset=\".6\"", "@keyframes", "<desc>", "100vw", "<meta ", " />\n"}
	for i := 0; i < 80; i++ {
		var b strings.Builder
		for k, n := 0, 5+rng.Intn(400); k < n; k++ {
			if rng.Intn(8) == 0 {
				b.WriteString(fragments[rng.Intn(len(fragments))])
			} else {
				b.WriteRune(alphabet[rng.Intn(len(alphabet))])
			}
		}
		emit(parityCase{Case: "random:" + strconv.Itoa(i+1), Prompt: prompts[rng.Intn(len(prompts))], HTML: b.String()})
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
