package inputvalidate

import "testing"

func TestNormalizeDDMMYYYY(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty", input: "", want: ""},
		{name: "valid", input: "20/04/2026", want: "20/04/2026"},
		{name: "trimmed", input: " 20/04/2026 ", want: "20/04/2026"},
		{name: "wrong separator", input: "20-04-2026", wantErr: true},
		{name: "wrong order", input: "2026/04/20", wantErr: true},
		{name: "single digit", input: "2/4/2026", wantErr: true},
		{name: "invalid date", input: "31/02/2026", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeDDMMYYYY(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}
