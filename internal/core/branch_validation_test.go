package core

import (
	"testing"
)

func TestIsValidBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
		reason   string
	}{
		// Valid branch names (should return true)
		{"simple", "main", true, "simple branch name"},
		{"with-hyphen", "feature-login", true, "hyphens are allowed"},
		{"with-underscore", "bug_fix_123", true, "underscores are allowed"},
		{"with-slash", "feature/login", true, "slashes are allowed"},
		{"with-number", "release-1.2.3", true, "numbers and dots are allowed"},
		{"complex-valid", "feature/user-auth_v2.1", true, "complex but valid name"},
		{"github-style", "feat/add-user-authentication", true, "common GitHub naming"},
		{"numeric-start", "123-feature", true, "can start with numbers"},

		// Invalid branch names (should return false)
		{"empty", "", false, "empty names not allowed"},
		{"too-long", string(make([]byte, 300)), false, "names over 255 chars not allowed"},
		{"leading-dot", ".hidden", false, "cannot start with dot"},
		{"trailing-dot", "feature.", false, "cannot end with dot"},
		{"consecutive-dots", "feature..name", false, "consecutive dots not allowed"},
		{"consecutive-slashes", "feature//name", false, "consecutive slashes not allowed"},
		{"leading-slash", "/feature", false, "cannot start with slash"},
		{"trailing-slash", "feature/", false, "cannot end with slash"},
		{"at-sequence", "feature@{upstream}", false, "@{ sequence not allowed"},
		{"lock-suffix", "feature.lock", false, ".lock suffix not allowed"},
		{"reserved-head", "HEAD", false, "HEAD is reserved"},
		{"reserved-at", "@", false, "@ is reserved"},
		{"invalid-chars", "feature:name", false, "colon not allowed"},
		{"space", "feature name", false, "spaces not allowed"},
		{"tilde", "feature~1", false, "tilde not allowed"},
		{"caret", "feature^1", false, "caret not allowed"},
		{"question", "feature?", false, "question mark not allowed"},
		{"asterisk", "feature*", false, "asterisk not allowed"},
		{"bracket", "feature[1]", false, "brackets not allowed"},
		{"backslash", "feature\\name", false, "backslash not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("isValidBranchName(%q) = %v; expected %v (%s)", 
					tt.input, result, tt.expected, tt.reason)
			}
		})
	}
}

// TestCommonRealWorldBranchNames tests branch names commonly used in real projects
func TestCommonRealWorldBranchNames(t *testing.T) {
	validBranchNames := []string{
		// Common naming conventions
		"main",
		"master", 
		"develop",
		"development",
		"staging",
		"production",
		
		// Feature branches
		"feature/user-authentication",
		"feature/payment-integration", 
		"feature-login-page",
		"feat/add-dark-mode",
		
		// Bug fix branches
		"bugfix/login-error",
		"bug/memory-leak",
		"fix/header-alignment",
		"hotfix/security-patch",
		
		// Release branches
		"release/v1.2.3",
		"release-2.0.0",
		"rel/beta-1",
		
		// Personal/contributor branches
		"john/working-branch",
		"alice-feature-work",
		"contributor_123/fix",
		
		// Version/environment branches
		"v1.0",
		"v2.1.0-beta",
		"test-environment",
		"integration-tests",
	}

	for _, branchName := range validBranchNames {
		t.Run(branchName, func(t *testing.T) {
			if !isValidBranchName(branchName) {
				t.Errorf("Common branch name %q should be valid but was rejected", branchName)
			}
		})
	}
}