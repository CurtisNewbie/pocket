package main

import (
	"testing"
	"time"
	"unicode/utf8"
)

func TestTruncateRunes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"ascii fits", "GitHub", 18, "GitHub"},
		{"ascii exact", "abcdefghijklmnopqr", 18, "abcdefghijklmnopqr"},
		{"ascii long", "abcdefghijklmnopqrstuvwxyz", 18, "abcdefghijklmnopqr…"},
		{"multi-byte fits", "认证服务", 6, "认证服务"},
		{"multi-byte long", "认证服务名字很长", 4, "认证服务…"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := truncateRunes(c.in, c.n)
			if got != c.want {
				t.Fatalf("truncateRunes(%q, %d) = %q, want %q", c.in, c.n, got, c.want)
			}
			if !utf8.ValidString(got) {
				t.Fatalf("truncateRunes(%q, %d) = %q, not valid UTF-8", c.in, c.n, got)
			}
		})
	}
}

func TestSecondsCellTextNonPositivePeriod(t *testing.T) {
	now := time.Unix(59, 0)
	for _, period := range []int{0, -30} {
		if got := secondsCellText(period, now); got != "" {
			t.Fatalf("secondsCellText(%d) = %q, want empty", period, got)
		}
	}
}

func TestDeleteConfirmTextNamesEntry(t *testing.T) {
	e := TotpEntry{Issuer: "GitHub", Account: "alice@example.com"}
	if got, want := deleteConfirmText(e), "Deleting GitHub: alice@example.com"; got != want {
		t.Fatalf("deleteConfirmText() = %q, want %q", got, want)
	}

	e.Remark = "work"
	if got, want := deleteConfirmText(e), "Deleting GitHub: alice@example.com (work)"; got != want {
		t.Fatalf("deleteConfirmText() with a remark = %q, want %q", got, want)
	}
}
