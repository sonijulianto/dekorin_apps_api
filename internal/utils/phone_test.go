package utils

import "testing"

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "081234567890", expected: "081234567890"},
		{input: "+6281234567890", expected: "081234567890"},
		{input: "6281234567890", expected: "081234567890"},
		{input: "0812-3456-7890", expected: "081234567890"},
		{input: "+62 812-3456-7890", expected: "081234567890"},
		{input: "0857 1234 5678", expected: "085712345678"},
	}

	for _, tc := range tests {
		actual := NormalizePhone(tc.input)
		if actual != tc.expected {
			t.Errorf("NormalizePhone(%q) = %q; expected %q", tc.input, actual, tc.expected)
		}
	}
}
