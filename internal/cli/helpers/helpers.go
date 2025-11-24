package helpers

// SplitKeyValue splits a "key=value" string into [key, value]
// If no '=' is found, returns [input]
func SplitKeyValue(pair string) []string {
	parts := make([]string, 0, 2)
	idx := -1
	for i, ch := range pair {
		if ch == '=' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return []string{pair}
	}
	parts = append(parts, pair[:idx])
	parts = append(parts, pair[idx+1:])
	return parts
}

// IntPtr returns a pointer to an int value
func IntPtr(i int) *int {
	return &i
}
