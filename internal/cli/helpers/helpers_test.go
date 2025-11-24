package helpers

import "testing"

func TestSplitKeyValue(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"key=value", []string{"key", "value"}},
		{"name=john", []string{"name", "john"}},
		{"single", []string{"single"}},
		{"with=equals=sign", []string{"with", "equals=sign"}},
		{"empty=", []string{"empty", ""}},
	}

	for _, tt := range tests {
		result := SplitKeyValue(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("SplitKeyValue(%q) returned %d parts, want %d", tt.input, len(result), len(tt.expected))
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("SplitKeyValue(%q)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
			}
		}
	}
}

func TestIntPtr(t *testing.T) {
	val := 42
	ptr := IntPtr(val)

	if ptr == nil {
		t.Fatal("IntPtr returned nil")
	}

	if *ptr != val {
		t.Errorf("IntPtr(%d) = %d, want %d", val, *ptr, val)
	}
}
