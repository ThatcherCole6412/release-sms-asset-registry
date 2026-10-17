package releases

import "testing"

func TestSelectTemplate(t *testing.T) {
	tests := []struct {
		name string
		in   ReleaseInput
		want string
	}{
		{name: "failed build", in: ReleaseInput{BuildEvent: "failed"}, want: "build-failed"},
		{name: "promoted release", in: ReleaseInput{BuildEvent: "succeeded", Operation: "promote"}, want: "release-promoted"},
		{name: "critical diagnostic takes priority", in: ReleaseInput{BuildEvent: "failed", Diagnostic: "critical"}, want: "diagnostic-critical"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SelectTemplate(tt.in); got != tt.want {
				t.Fatalf("SelectTemplate() = %q, want %q", got, tt.want)
			}
		})
	}
}
