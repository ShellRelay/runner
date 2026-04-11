package main

import "testing"

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		// Equal
		{name: "equal same format", a: "v1.2.3", b: "v1.2.3", want: 0},
		{name: "equal no v prefix", a: "1.2.3", b: "1.2.3", want: 0},
		{name: "equal mixed prefix", a: "v1.2.3", b: "1.2.3", want: 0},

		// Less than
		{name: "patch less", a: "v1.2.3", b: "v1.2.4", want: -1},
		{name: "minor less", a: "v1.2.8", b: "v1.3.0", want: -1},
		{name: "major less", a: "v1.9.9", b: "v2.0.0", want: -1},

		// Greater than
		{name: "patch greater", a: "v1.2.4", b: "v1.2.3", want: 1},
		{name: "minor greater", a: "v1.3.0", b: "v1.2.8", want: 1},
		{name: "major greater", a: "v2.0.0", b: "v1.9.9", want: 1},

		// Edge cases
		{name: "zero versions", a: "v0.0.0", b: "v0.0.0", want: 0},
		{name: "zero vs one", a: "v0.0.0", b: "v0.0.1", want: -1},
		{name: "large numbers", a: "v10.20.30", b: "v10.20.30", want: 0},
		{name: "large major wins", a: "v10.0.0", b: "v9.99.99", want: 1},

		// Missing parts (shorter version strings)
		{name: "short vs full equal", a: "v1.0", b: "v1.0.0", want: 0},
		{name: "short vs full less", a: "v1.0", b: "v1.0.1", want: -1},
		{name: "single digit", a: "v1", b: "v1.0.0", want: 0},

		// dev version (non-numeric → Atoi returns 0)
		{name: "dev equals 0.0.0", a: "vdev", b: "v0.0.0", want: 0},
		{name: "dev less than 0.0.1", a: "vdev", b: "v0.0.1", want: -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := compareSemver(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("compareSemver(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
