package drift

import (
	"testing"
	"time"
)

func TestFormuladParseFixture(t *testing.T) {
	html := `<!DOCTYPE html><html><head><title>HIGH STAKES | Formula DRIFT</title></head><body>
<span>SEP 24 - SEP 26</span>
<span>RD 7</span>
<span>#FDLV</span>
<span>LAS VEGAS, NEVADA, USA</span>
</body></html>`
	start, end, ok := parseDateRange(html, 2026)
	if !ok {
		t.Fatal("dates")
	}
	if start.Day() != 24 {
		t.Fatalf("start %v", start)
	}
	if !end.After(time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("end %v", end)
	}
	if m := roundRE.FindStringSubmatch(html); len(m) != 2 || m[1] != "7" {
		t.Fatalf("round %v", m)
	}
	if m := hashtagRE.FindStringSubmatch(html); len(m) != 2 || m[1] != "FDLV" {
		t.Fatalf("hashtag %v", m)
	}
	if m := locationRE.FindStringSubmatch(html); len(m) != 2 || m[1] != "LAS VEGAS, NEVADA, USA" {
		t.Fatalf("location %v", m)
	}
}

func TestFormuladLocationMixedCase(t *testing.T) {
	html := `<span>Atlanta, Georgia, USA</span>`
	if m := locationRE.FindStringSubmatch(html); len(m) != 2 || m[1] != "Atlanta, Georgia, USA" {
		t.Fatalf("location %v", m)
	}
}
