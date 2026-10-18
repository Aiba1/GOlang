// Package analyzer implements the actual processing logic for this
// application's chosen domain: text analysis. It satisfies
// task.Processor so package service can use it without knowing
// anything about how the analysis is done.
package analyzer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"textanalyzer-cp1/task"
)

// Result holds the statistics produced by analyzing a piece of text.
// It implements task.Result.
type Result struct {
	WordCount         int
	CharCount         int // total runes, including whitespace
	CharCountNoSpaces int // runes that are not whitespace
	LongestWord       string
	AverageWordLength float64
}

// Summary implements task.Result with a value receiver: Result is a
// small, immutable snapshot of statistics, so there is no reason to
// pay for an extra indirection to read it.
func (r Result) Summary() string {
	return fmt.Sprintf(
		"words=%d chars=%d chars_no_spaces=%d longest_word=%q avg_word_length=%.2f",
		r.WordCount, r.CharCount, r.CharCountNoSpaces, r.LongestWord, r.AverageWordLength,
	)
}

// TextAnalyzer processes raw text input into a Result. It holds no
// state of its own, but is a struct (rather than a bare function) so
// that it has a clear, named type that can be passed around and so
// that it could grow configuration (e.g. a stop-word list) later
// without changing its exported API.
type TextAnalyzer struct{}

// New returns a ready-to-use TextAnalyzer.
func New() *TextAnalyzer {
	return &TextAnalyzer{}
}

// Process implements task.Processor. It computes word and character
// statistics for input. Process assumes the caller has already
// rejected empty/whitespace-only input (package task's Store does
// this at creation time); it still guards against that case directly
// so TextAnalyzer is safe to use on its own, e.g. in tests.
func (a *TextAnalyzer) Process(input string) (task.Result, error) {
	if strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("analyzer: cannot process empty or whitespace-only input")
	}

	words := strings.Fields(input)

	charCount := utf8.RuneCountInString(input)

	noSpaces := 0
	for _, r := range input {
		if !unicode.IsSpace(r) {
			noSpaces++
		}
	}

	var longest string
	totalWordRunes := 0
	for _, w := range words {
		n := utf8.RuneCountInString(w)
		totalWordRunes += n
		if n > utf8.RuneCountInString(longest) {
			longest = w
		}
	}

	var avg float64
	if len(words) > 0 {
		avg = float64(totalWordRunes) / float64(len(words))
	}

	return Result{
		WordCount:         len(words),
		CharCount:         charCount,
		CharCountNoSpaces: noSpaces,
		LongestWord:       longest,
		AverageWordLength: avg,
	}, nil
}
