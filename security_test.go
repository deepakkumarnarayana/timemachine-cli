package main

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
	"unicode"

	"github.com/deepakkumarnarayana/timemachine-cli/internal/security"
)

// SecurityTest focuses on property-based testing and fuzzing for security validation
// These tests generate random inputs to discover edge cases and vulnerabilities

func TestSecurityProperty_ValidateUserInputPath(t *testing.T) {
	// Property: Any user input path should either be rejected safely or accepted as relative path
	property := func(input string) bool {
		result, err := security.ValidateUserInputPath(input)
		
		// If validation succeeds, result must be a safe relative path
		if err == nil {
			// Must not contain path traversal attempts
			if strings.Contains(result, "..") {
				t.Logf("SECURITY FAILURE: Path traversal allowed: %q -> %q", input, result)
				return false
			}
			
			// Must not be absolute path
			if strings.HasPrefix(result, "/") || strings.HasPrefix(result, "\\") {
				t.Logf("SECURITY FAILURE: Absolute path allowed: %q -> %q", input, result)
				return false
			}
			
			// Must not contain Windows drive letters
			if len(result) >= 2 && result[1] == ':' {
				t.Logf("SECURITY FAILURE: Windows absolute path allowed: %q -> %q", input, result)
				return false
			}
		}
		
		// If validation fails, error must be non-nil (secure failure)
		return err != nil || isSecureRelativePath(result)
	}
	
	config := &quick.Config{MaxCount: 1000}
	if err := quick.Check(property, config); err != nil {
		t.Errorf("Security property failed: %v", err)
	}
}

func TestSecurityProperty_ValidateSystemPath(t *testing.T) {
	// Property: System paths can be absolute but must still be safe
	property := func(input string) bool {
		result, err := security.ValidateSystemPath(input)
		
		// If validation succeeds, result must be a clean path
		if err == nil {
			// Must not contain null bytes
			if strings.ContainsRune(result, 0) {
				t.Logf("SECURITY FAILURE: Null byte in system path: %q -> %q", input, result)
				return false
			}
			
			// Must not contain excessive path traversal (more than reasonable nesting)
			if strings.Count(result, "..") > 10 { // Allow some reasonable traversal for system paths
				t.Logf("SECURITY FAILURE: Excessive traversal in system path: %q -> %q", input, result)
				return false
			}
		}
		
		return err != nil || isSecureSystemPath(result)
	}
	
	config := &quick.Config{MaxCount: 1000}
	if err := quick.Check(property, config); err != nil {
		t.Errorf("System path security property failed: %v", err)
	}
}

func TestFuzz_PathTraversalAttacks(t *testing.T) {
	// Test known path traversal patterns with fuzzing variations
	baseAttacks := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"/etc/passwd",
		"C:\\Windows\\System32",
		"\\\\server\\share",
	}
	
	for i := 0; i < 1000; i++ {
		for _, base := range baseAttacks {
			// Generate variations with random modifications
			fuzzed := fuzzPathTraversalString(base)
			
			// Test against user input validation (should always fail)
			_, err := security.ValidateUserInputPath(fuzzed)
			if err == nil {
				t.Errorf("SECURITY VULNERABILITY: Path traversal attack succeeded with input: %q", fuzzed)
			}
		}
	}
}

func TestFuzz_CommandInjectionInHash(t *testing.T) {
	// Test command injection patterns in git hash validation
	injectionPatterns := []string{
		"; rm -rf /",
		"&& echo pwned",
		"| cat /etc/passwd",
		"$(whoami)",
		"`id`",
		"' OR '1'='1",
		"; DROP TABLE users; --",
	}
	
	for i := 0; i < 500; i++ {
		for _, pattern := range injectionPatterns {
			// Create hash-like input with injection
			validHash := generateRandomHexString(8, 40)
			maliciousInput := validHash + pattern
			
			// Test git hash validation
			suite := NewIntegrationTestSuite(t)
			stdout, stderr, exitCode := suite.runTimemachineCmd("inspect", maliciousInput)
			suite.Cleanup()
			
			// Should fail due to hash validation
			if exitCode == 0 {
				t.Errorf("SECURITY VULNERABILITY: Command injection not blocked: %q", maliciousInput)
			}
			
			// Should not contain injection success indicators
			output := stdout + stderr
			if containsInjectionSuccess(output) {
				t.Errorf("SECURITY VULNERABILITY: Command injection executed: %q -> %s", maliciousInput, output)
			}
		}
	}
}

func TestFuzz_UnicodePathTraversal(t *testing.T) {
	// Test Unicode-based path traversal attacks
	unicodeAttacks := []string{
		"..%c0%af..%c0%af..%c0%afetc%c0%afpasswd",     // UTF-8 overlong encoding
		"..%e0%80%af..%e0%80%af..%e0%80%afetc",        // UTF-8 overlong encoding
		"..%c1%9c..%c1%9c..%c1%9cetc",                 // Another overlong encoding
		"\u002e\u002e\u002f\u002e\u002e\u002f",       // Unicode dot and slash
		"\uff0e\uff0e\uff0f\uff0e\uff0e\uff0f",       // Full-width characters
	}
	
	for _, attack := range unicodeAttacks {
		_, err := security.ValidateUserInputPath(attack)
		if err == nil {
			t.Errorf("SECURITY VULNERABILITY: Unicode path traversal succeeded: %q", attack)
		}
	}
}

func TestProperty_InputSanitizationIdempotency(t *testing.T) {
	// Property: Applying validation twice should yield same result
	property := func(input string) bool {
		// First validation
		result1, err1 := security.ValidateUserInputPath(input)
		if err1 != nil {
			return true // Failed validation is OK
		}
		
		// Second validation on the result
		result2, err2 := security.ValidateUserInputPath(result1)
		if err2 != nil {
			t.Logf("CONSISTENCY FAILURE: Second validation failed on sanitized input: %q -> %q", input, result1)
			return false
		}
		
		// Results should be identical (idempotent)
		if result1 != result2 {
			t.Logf("CONSISTENCY FAILURE: Non-idempotent validation: %q -> %q -> %q", input, result1, result2)
			return false
		}
		
		return true
	}
	
	config := &quick.Config{MaxCount: 1000}
	if err := quick.Check(property, config); err != nil {
		t.Errorf("Idempotency property failed: %v", err)
	}
}

func TestRace_ConcurrentValidation(t *testing.T) {
	// Test concurrent access to validation functions for race conditions
	inputs := []string{
		"../../../etc/passwd",
		"valid/relative/path.txt",  
		"C:\\Windows\\System32",
		"/absolute/path",
		"normal-file.go",
	}
	
	// Run concurrent validations
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("concurrent_%d", i), func(t *testing.T) {
			t.Parallel()
			
			for j := 0; j < 100; j++ {
				input := inputs[j%len(inputs)]
				
				// Test user input validation
				result1, err1 := security.ValidateUserInputPath(input)
				
				// Test system path validation  
				result2, err2 := security.ValidateSystemPath(input)
				
				// Results should be deterministic
				result1b, err1b := security.ValidateUserInputPath(input)
				result2b, err2b := security.ValidateSystemPath(input)
				
				if result1 != result1b || (err1 == nil) != (err1b == nil) {
					t.Errorf("Race condition in ValidateUserInputPath: %q", input)
				}
				
				if result2 != result2b || (err2 == nil) != (err2b == nil) {
					t.Errorf("Race condition in ValidateSystemPath: %q", input)
				}
			}
		})
	}
}

func TestSecurity_NullByteInjection(t *testing.T) {
	// Test null byte injection attacks
	nullByteAttacks := []string{
		"file.txt\x00../../../etc/passwd",
		"safe-file\x00; rm -rf /",
		"normal\x00.php\x00.txt",
		"\x00../etc/passwd",
		"file\x00\x00\x00traversal",
	}
	
	for _, attack := range nullByteAttacks {
		_, err := security.ValidateUserInputPath(attack)
		if err == nil {
			t.Errorf("SECURITY VULNERABILITY: Null byte injection succeeded: %q", attack)
		}
	}
}

func TestSecurity_SymlinkValidation(t *testing.T) {
	// Test symlink-related security (this will be important for future symlink handling)
	symlinkPatterns := []string{
		"symlink-to-etc/../passwd",
		"link/../../../../etc/hosts",
		"./symlink/../../../root",
	}
	
	for _, pattern := range symlinkPatterns {
		_, err := security.ValidateUserInputPath(pattern)
		if err == nil {
			t.Errorf("POTENTIAL SECURITY ISSUE: Symlink traversal pattern allowed: %q", pattern)
		}
	}
}

// Helper functions for security testing

func isSecureRelativePath(path string) bool {
	return !strings.Contains(path, "..") && 
		   !strings.HasPrefix(path, "/") &&
		   !strings.HasPrefix(path, "\\") &&
		   len(path) >= 2 && path[1] != ':' // No Windows drive letters
}

func isSecureSystemPath(path string) bool {
	return !strings.ContainsRune(path, 0) && // No null bytes
		   strings.Count(path, "..") <= 10    // Reasonable traversal limit
}

func fuzzPathTraversalString(base string) string {
	// Create variations of path traversal strings
	variations := []func(string) string{
		func(s string) string { return strings.ReplaceAll(s, "/", "\\") },
		func(s string) string { return strings.ReplaceAll(s, "..", "....") },
		func(s string) string { return strings.ToUpper(s) },
		func(s string) string { return s + strings.Repeat("/", rand.Intn(10)) },
		func(s string) string { return strings.Repeat("./", rand.Intn(5)) + s },
		func(s string) string { return s + "\x00extra" }, // Null byte injection
	}
	
	variation := variations[rand.Intn(len(variations))]
	return variation(base)
}

func generateRandomHexString(minLen, maxLen int) string {
	length := minLen + rand.Intn(maxLen-minLen+1)
	chars := "0123456789abcdef"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func containsInjectionSuccess(output string) bool {
	// Check for signs that command injection succeeded
	indicators := []string{
		"uid=", "gid=",           // Output from `id` command
		"total ",                 // Output from `ls -l`
		"root:x:",               // /etc/passwd content
		"pwned",                 // Test injection indicator
		"ECHO is",               // Windows echo output
	}
	
	outputLower := strings.ToLower(output)
	for _, indicator := range indicators {
		if strings.Contains(outputLower, strings.ToLower(indicator)) {
			return true
		}
	}
	return false
}

// Fuzzing functions for Go 1.18+ (if available)

func FuzzValidateUserInputPath(f *testing.F) {
	// Seed with known problematic inputs
	testInputs := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"/etc/hosts", 
		"C:\\Windows\\System32",
		"normal-file.txt",
		"",
		".",
		"..",
		"./file.txt",
		"path/to/file.go",
	}
	
	for _, input := range testInputs {
		f.Add(input)
	}
	
	f.Fuzz(func(t *testing.T, input string) {
		result, err := security.ValidateUserInputPath(input)
		
		// If validation succeeds, result must be safe
		if err == nil {
			if strings.Contains(result, "..") ||
			   strings.HasPrefix(result, "/") ||
			   strings.HasPrefix(result, "\\") ||
			   (len(result) >= 2 && result[1] == ':') {
				t.Errorf("SECURITY FAILURE: Unsafe path allowed: %q -> %q", input, result)
			}
		}
		
		// Should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("PANIC in ValidateUserInputPath with input: %q", input)
			}
		}()
	})
}

func FuzzValidateSystemPath(f *testing.F) {
	// Seed with various system path inputs
	testInputs := []string{
		"/usr/local/bin",
		"C:\\Program Files",
		"/home/user/.git",
		"./relative/path",
		"../parent/directory",
		"",
		"/",
		"\\",
	}
	
	for _, input := range testInputs {
		f.Add(input)
	}
	
	f.Fuzz(func(t *testing.T, input string) {
		result, err := security.ValidateSystemPath(input)
		
		// If validation succeeds, result must not contain null bytes
		if err == nil {
			if strings.ContainsRune(result, 0) {
				t.Errorf("SECURITY FAILURE: Null byte in system path: %q -> %q", input, result)
			}
		}
		
		// Should never panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("PANIC in ValidateSystemPath with input: %q", input)
			}
		}()
	})
}

// Advanced property-based testing

func TestProperty_NoControlCharacters(t *testing.T) {
	// Property: Validated paths should not contain control characters
	property := func(input string) bool {
		result, err := security.ValidateUserInputPath(input)
		if err != nil {
			return true // Rejection is fine
		}
		
		// Check for control characters in result
		for _, r := range result {
			if unicode.IsControl(r) && r != '\t' { // Allow tab but not other control chars
				t.Logf("SECURITY FAILURE: Control character in validated path: %q -> %q", input, result)
				return false
			}
		}
		return true
	}
	
	config := &quick.Config{MaxCount: 1000}
	if err := quick.Check(property, config); err != nil {
		t.Errorf("Control character property failed: %v", err)
	}
}