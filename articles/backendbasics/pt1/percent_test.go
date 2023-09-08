package backendbasics

import "testing"

func TestEscape(t *testing.T) {
	for unescaped, escaped := range map[string]string{
		"hello world":   "hello%20world",
		"hello%20world": "hello%2520world",
	} {
		if got := Escape(escaped); got != escaped {
			t.Errorf("Escape(%q) = %q, want %q", escaped, got, escaped)
		}
		if got := Escape(unescaped); got != escaped {
			t.Errorf("Escape(%q) = %q, want %q", unescaped, got, escaped)
		}
		if got := Unescape(escaped); got != unescaped {
			t.Errorf("Escape(%q) = %q, want %q", escaped, got, escaped)
		}
		if got := Unescape(Escape(unescaped)); got != unescaped {
			t.Errorf("Unescape(Escape(%q)) = %q, want %q", unescaped, got, unescaped)
		}
		if got := Escape(Unescape(escaped)); got != escaped {
			t.Errorf("Escape(Unescape(%q)) = %q, want %q", escaped, got, escaped)

		}
	}
}
