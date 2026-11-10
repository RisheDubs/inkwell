package journal

import (
	"errors"
	"testing"
)

func TestCursorRoundTrip(t *testing.T) {
	c := Cursor{EntryDate: "2026-11-09", ID: "6730a1b2c3d4e5f601234567"}
	got, err := DecodeCursor(c.Encode())
	if err != nil {
		t.Fatalf("DecodeCursor() error = %v", err)
	}
	if got != c {
		t.Errorf("DecodeCursor(Encode(c)) = %+v, want %+v", got, c)
	}
}

func TestDecodeCursorRejectsTampering(t *testing.T) {
	for _, s := range []string{
		"",
		"not-base64!!",
		Cursor{EntryDate: "", ID: "x"}.Encode(),
		Cursor{EntryDate: "2026-11-09", ID: ""}.Encode(),
		Cursor{EntryDate: "09/11/2026", ID: "x"}.Encode(),
		"eyJkIjoxfQ", // {"d":1}: valid base64 and JSON, wrong types
	} {
		var verr *ValidationError
		if _, err := DecodeCursor(s); !errors.As(err, &verr) {
			t.Errorf("DecodeCursor(%q) error = %v, want a *ValidationError", s, err)
		}
	}
}

func TestNormaliseLimit(t *testing.T) {
	for in, want := range map[int]int{0: 20, -5: 20, 1: 1, 50: 50, 100: 100, 1000: 100} {
		if got := NormaliseLimit(in); got != want {
			t.Errorf("NormaliseLimit(%d) = %d, want %d", in, got, want)
		}
	}
}
