package analyzer

import (
	"math"
	"testing"
)

func TestTextAnalyzer_Process(t *testing.T) {
	// Table-driven test covering the core statistics TextAnalyzer must
	// compute correctly (requirement: "correctness of the produced
	// result").
	tests := []struct {
		name           string
		input          string
		wantWords      int
		wantChars      int
		wantCharsNoSp  int
		wantLongest    string
		wantAvgWordLen float64
	}{
		{
			name:           "spec example",
			input:          "Hello Go world",
			wantWords:      3,
			wantChars:      14,
			wantCharsNoSp:  12,
			wantLongest:    "Hello",
			wantAvgWordLen: 4.0,
		},
		{
			name:           "single word",
			input:          "Go",
			wantWords:      1,
			wantChars:      2,
			wantCharsNoSp:  2,
			wantLongest:    "Go",
			wantAvgWordLen: 2.0,
		},
		{
			name:           "punctuation attached to words",
			input:          "One, two, three!",
			wantWords:      3,
			wantChars:      16,
			wantCharsNoSp:  14,
			wantLongest:    "three!",
			wantAvgWordLen: 14.0 / 3.0,
		},
		{
			name:           "collapses repeated whitespace",
			input:          "a    b",
			wantWords:      2,
			wantChars:      6,
			wantCharsNoSp:  2,
			wantLongest:    "a", // tie on length: first word wins
			wantAvgWordLen: 1.0,
		},
	}

	a := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := a.Process(tt.input)
			if err != nil {
				t.Fatalf("Process(%q) returned unexpected error: %v", tt.input, err)
			}

			r, ok := res.(Result)
			if !ok {
				t.Fatalf("Process(%q) returned result of type %T, want analyzer.Result", tt.input, res)
			}

			if r.WordCount != tt.wantWords {
				t.Errorf("WordCount = %d, want %d", r.WordCount, tt.wantWords)
			}
			if r.CharCount != tt.wantChars {
				t.Errorf("CharCount = %d, want %d", r.CharCount, tt.wantChars)
			}
			if r.CharCountNoSpaces != tt.wantCharsNoSp {
				t.Errorf("CharCountNoSpaces = %d, want %d", r.CharCountNoSpaces, tt.wantCharsNoSp)
			}
			if r.LongestWord != tt.wantLongest {
				t.Errorf("LongestWord = %q, want %q", r.LongestWord, tt.wantLongest)
			}
			if math.Abs(r.AverageWordLength-tt.wantAvgWordLen) > 0.001 {
				t.Errorf("AverageWordLength = %f, want %f", r.AverageWordLength, tt.wantAvgWordLen)
			}

			if r.Summary() == "" {
				t.Error("Summary() returned an empty string")
			}
		})
	}
}

func TestTextAnalyzer_Process_InvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty string", input: ""},
		{name: "whitespace only", input: "   \t\n  "},
	}

	a := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := a.Process(tt.input); err == nil {
				t.Fatalf("Process(%q) = nil error, want an error", tt.input)
			}
		})
	}
}
