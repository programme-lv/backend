package plang

import "testing"

func TestProgrLangIDFromFilename(t *testing.T) {
	tests := map[string]string{
		"full.cpp":      "cpp17",
		"partial.PY":    "python3.13",
		"main.go":       "go1.25",
		"Solution.java": "java25",
	}

	for fname, want := range tests {
		got, err := ProgrLangIDFromFilename(fname)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", fname, err)
		}
		if got != want {
			t.Fatalf("%s: expected %s, got %s", fname, want, got)
		}
	}
}

func TestProgrLangIDFromFilenameUnknown(t *testing.T) {
	if _, err := ProgrLangIDFromFilename("solution.txt"); err == nil {
		t.Fatal("expected an error for an unknown extension")
	}
}
