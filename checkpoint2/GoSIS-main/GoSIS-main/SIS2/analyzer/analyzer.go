package analyzer

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"textanalyzer-cp1/task"
)

type Result struct {
	WordCount         int
	CharCount         int
	CharCountNoSpaces int
	LongestWord       string
	AverageWordLength float64
}

func (r Result) Summary() string {
	return fmt.Sprintf(
		"words=%d chars=%d chars_no_spaces=%d longest_word=%q avg_word_length=%.2f",
		r.WordCount, r.CharCount, r.CharCountNoSpaces, r.LongestWord, r.AverageWordLength,
	)
}

type TextAnalyzer struct{}

func New() *TextAnalyzer {
	return &TextAnalyzer{}
}

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
