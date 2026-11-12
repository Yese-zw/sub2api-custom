package judge

import (
	"math"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The functions in this file are the scoring core of the frozen v6 judge and
// are kept line-for-line equivalent to the original implementation. Do not
// "simplify" them: the parity tests require bit-identical margins.

// isPythonSpace reports whether r is whitespace for Python str.isspace, which
// also treats the ASCII information separators (U+001C..U+001F) as spaces.
func isPythonSpace(r rune) bool {
	return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f)
}

// normalizeText mirrors the trained vectorizer: Python lower() followed by
// collapsing every run of two or more whitespace characters into one space.
// A single whitespace character is preserved as-is.
func normalizeText(html string) string {
	var lowered strings.Builder
	lowered.Grow(len(html))
	for _, r := range html {
		if r == 'İ' { // Python lowers U+0130 to "i" + combining dot above.
			lowered.WriteString("i̇")
			continue
		}
		lowered.WriteRune(unicode.ToLower(r))
	}
	text := lowered.String()
	var out strings.Builder
	out.Grow(len(text))
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if !isPythonSpace(r) {
			out.WriteString(text[i : i+size])
			i += size
			continue
		}
		j, count := i+size, 1
		for j < len(text) {
			next, nextSize := utf8.DecodeRuneInString(text[j:])
			if !isPythonSpace(next) {
				break
			}
			j, count = j+nextSize, count+1
		}
		if count > 1 {
			out.WriteByte(' ')
		} else {
			out.WriteString(text[i:j])
		}
		i = j
	}
	return out.String()
}

// margin is the V5b decision value: sublinear TF-IDF over vocabulary character
// n-grams, L2-normalized, dotted with the logistic regression coefficients.
func (m *scorer) margin(html string) float64 {
	text := normalizeText(html)
	offsets := make([]int, 0, len(text)+1)
	for i := range text {
		offsets = append(offsets, i)
	}
	offsets = append(offsets, len(text))
	runes := len(offsets) - 1
	counts := make(map[int32]int)
	low, high := m.model.NgramRange[0], m.model.NgramRange[1]
	for n := low; n <= high && n <= runes; n++ {
		for i := 0; i+n <= runes; i++ {
			if index, ok := m.index[text[offsets[i]:offsets[i+n]]]; ok {
				counts[index]++
			}
		}
	}
	if len(counts) == 0 {
		return m.model.Intercept
	}
	indices := make([]int32, 0, len(counts))
	for index := range counts {
		indices = append(indices, index)
	}
	sort.Slice(indices, func(a, b int) bool { return indices[a] < indices[b] })
	weights := make([]float64, len(indices))
	var norm neumaierSum
	for k, index := range indices {
		weights[k] = (1 + math.Log(float64(counts[index]))) * m.model.IDF[index]
		norm.add(weights[k] * weights[k])
	}
	length := math.Sqrt(norm.value())
	var dot neumaierSum
	for k, index := range indices {
		dot.add(weights[k] / length * m.model.Coef[index])
	}
	return m.model.Intercept + dot.value()
}

// neumaierSum is compensated summation. It keeps the result order-independent
// to ~1 ulp, close to Python's math.fsum used when the reference margins were
// computed.
type neumaierSum struct{ sum, compensation float64 }

func (s *neumaierSum) add(value float64) {
	total := s.sum + value
	if math.Abs(s.sum) >= math.Abs(value) {
		s.compensation += (s.sum - total) + value
	} else {
		s.compensation += (value - total) + s.sum
	}
	s.sum = total
}

func (s *neumaierSum) value() float64 { return s.sum + s.compensation }
