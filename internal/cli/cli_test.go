package cli

import (
	"testing"
)

func TestParseRMArgs(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantTarget   string
		wantValid    bool
	}{
		{
			name:       "too few arguments (len 1)",
			args:       []string{"build"},
			wantTarget: "",
			wantValid:  false,
		},
		{
			name:       "too few arguments (len 2)",
			args:       []string{"build", "rm"},
			wantTarget: "",
			wantValid:  false,
		},
		{
			name:       "too many arguments",
			args:       []string{"build", "rm", "id:123", "extra"},
			wantTarget: "",
			wantValid:  false,
		},
		{
			name:       "invalid prefix",
			args:       []string{"build", "rm", "foo:123"},
			wantTarget: "",
			wantValid:  false,
		},
		{
			name:       "empty id after colon",
			args:       []string{"build", "rm", "id:"},
			wantTarget: "",
			wantValid:  false,
		},
		{
			name:       "empty status after colon",
			args:       []string{"build", "rm", "status:"},
			wantTarget: "",
			wantValid:  false,
		},
		{
			name:       "valid id",
			args:       []string{"build", "rm", "id:G1"},
			wantTarget: "id:G1",
			wantValid:  true,
		},
		{
			name:       "valid status",
			args:       []string{"build", "rm", "status:todo"},
			wantTarget: "status:todo",
			wantValid:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTarget, gotValid := ParseRMArgs(tt.args)
			if gotValid != tt.wantValid {
				t.Errorf("ParseRMArgs() gotValid = %v, want %v", gotValid, tt.wantValid)
			}
			if gotTarget != tt.wantTarget {
				t.Errorf("ParseRMArgs() gotTarget = %q, want %q", gotTarget, tt.wantTarget)
			}
		})
	}
}
