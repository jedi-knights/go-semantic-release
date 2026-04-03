package domain_test

import (
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestStep_String(t *testing.T) {
	tests := []struct {
		step domain.Step
		want string
	}{
		{domain.StepVerifyConditions, "verifyConditions"},
		{domain.StepAnalyzeCommits, "analyzeCommits"},
		{domain.StepVerifyRelease, "verifyRelease"},
		{domain.StepGenerateNotes, "generateNotes"},
		{domain.StepPrepare, "prepare"},
		{domain.StepPublish, "publish"},
		{domain.StepAddChannel, "addChannel"},
		{domain.StepSuccess, "success"},
		{domain.StepFail, "fail"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.step.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
