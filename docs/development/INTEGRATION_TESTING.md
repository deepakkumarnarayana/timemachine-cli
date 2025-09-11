# Integration Testing Guide

## Overview

This document describes the comprehensive integration test suite created to prevent issues like the inspect command bug that was missed by unit tests.

## Why Integration Tests Were Critical

### The Bug We Fixed
The inspect command was failing with the error:
```
Error: failed to validate snapshot hash: path must be local and relative
```

**Root Cause**: `sanitizeGitPath()` was using `security.SanitizeUserInputPath()` which only allows relative paths, but `state.ShadowRepoDir` is an absolute path.

**Why Unit Tests Missed It**:
- Unit tests tested individual functions in isolation
- Unit tests had **incorrect expectations** (expected absolute paths to be rejected)
- No tests actually executed the CLI commands as users would

### The Fix
Changed `sanitizeGitPath()` to use `security.SanitizeGitPath()` which allows absolute paths for system directories.

## Integration Test Coverage

### 🔧 **TestInitCommand**
- ✅ Successful initialization
- ✅ Already initialized scenario
- ✅ Verifies shadow repository creation
- ✅ Verifies .gitignore updates

### ⚙️ **TestStatusCommand** 
- ✅ Status before init
- ✅ Status after init
- ✅ Displays correct project information

### 📋 **TestListCommand**
- ✅ List before init (graceful message)
- ✅ List after init (shows snapshots)
- ✅ List with limit flag
- ✅ Hash format validation

### 👁️ **TestShowCommand**
- ✅ Show with full hash
- ✅ Show with short hash (the critical test!)
- ✅ Nonexistent hash handling
- ✅ Invalid hash validation

### 🔍 **TestInspectCommand** (Critical - This Caught the Bug!)
- ✅ **Inspect with short hash** (would have failed before fix!)
- ✅ **Inspect with full hash** (would have failed before fix!)
- ✅ Inspect with --stats flag
- ✅ Inspect with --diff flag
- ✅ Inspect with file filter (matches found)
- ✅ Inspect with file filter (no matches)
- ✅ Inspect without hash (uses latest)
- ✅ Nonexistent hash handling
- ✅ **Malicious file filter rejection** (security test)
- ✅ Search all functionality

### 🔄 **TestRestoreCommand**
- ✅ Restore specific file with --files flag
- ✅ Restore nonexistent file
- ✅ Malicious path rejection (security test)
- ✅ File content verification after restore

### 🧹 **TestCleanCommand**
- ✅ Basic help functionality test

### 🔗 **TestCrossCommandIntegration**
- ✅ Complete workflow: init → status → list → show → inspect
- ✅ Tests command interactions

### ❌ **TestErrorHandling**
- ✅ Invalid command handling
- ✅ Missing arguments handling

### 🛡️ **TestSecurityValidation** (Critical Security Tests)
- ✅ Path traversal attacks (`../../../etc/passwd`)
- ✅ Windows absolute paths (`C:\Windows\System32`)
- ✅ UNC paths (`\\server\share\file.txt`)
- ✅ Hash injection attacks (`abc123; rm -rf /`)
- ✅ Command injection (`$(rm -rf /)`, backticks)

## Key Insights from Integration Testing

### 1. **Test Behavior Alignment Issues Found**
- Status message: Expected "not initialized" vs actual "Status: Not initialized"
- List command: Returns exit code 0 with error message (not failure)
- Show command: Different error message format than expected
- Restore command: Uses `--files` flag, not positional args
- Inspect filter: Shows "(filtered for: ...)" only when no matches found

### 2. **Security Validation Works**
All malicious inputs are properly rejected:
- Path traversal attempts
- Absolute path attempts  
- Command injection attempts
- Hash injection attempts

### 3. **The Critical Tests That Would Have Caught the Bug**
```go
t.Run("inspect_with_short_hash", func(t *testing.T) {
    shortHash := hash[:8]
    stdout, stderr, exitCode := suite.runTimemachineCmd("inspect", shortHash)
    
    suite.expectSuccess(stdout, stderr, exitCode, "inspect", shortHash)
    // This would have FAILED before our fix!
})
```

## Running Integration Tests

### Run All Tests
```bash
go test -v integration_test.go
```

### Run Specific Test Categories
```bash
# Test just the inspect command (the critical one)
go test -v -run TestInspectCommand integration_test.go

# Test security validation
go test -v -run TestSecurityValidation integration_test.go

# Test cross-command integration
go test -v -run TestCrossCommandIntegration integration_test.go
```

### Expected Results
All tests should pass:
```
PASS
ok  	command-line-arguments	26.297s
```

## Integration vs Unit Test Strategy

### When to Use Integration Tests
- ✅ Testing actual CLI command execution
- ✅ Testing complete user workflows
- ✅ Testing command interactions
- ✅ Testing real file system operations
- ✅ Testing security input validation end-to-end

### When to Use Unit Tests
- ✅ Testing individual function logic
- ✅ Testing error conditions in isolation
- ✅ Testing algorithm correctness
- ✅ Fast feedback during development

### The Lesson: Both Are Essential
- **Unit tests**: Catch logic errors in individual functions
- **Integration tests**: Catch system-level issues and misaligned behavior

## Future Improvements

### 1. Add More Command Coverage
- `start` command (challenging - long-running process)
- `config` command variations
- More `clean` command scenarios

### 2. Add Performance Tests
```go
func TestPerformanceWithLargeRepository(t *testing.T) {
    // Test with 1000+ snapshots
    // Measure command execution time
}
```

### 3. Add Cross-Platform Tests
- Test Windows-specific behavior
- Test different file path formats
- Test different Git configurations

### 4. Add Property-Based Testing
```go
func TestSanitizationAlwaysRejectsTraversal(t *testing.T) {
    // Generate random inputs with traversal patterns
    // Assert all are rejected
}
```

## Continuous Integration

Add to CI pipeline:
```yaml
- name: Run Integration Tests
  run: go test -v integration_test.go
  timeout-minutes: 5
```

## Conclusion

The integration test suite has proven invaluable:
1. **Would have caught the inspect command bug** immediately
2. **Validates complete user workflows** work as expected
3. **Ensures security validation** works end-to-end
4. **Catches behavior misalignment** between tests and implementation
5. **Provides confidence** in releases

**Key Takeaway**: Integration tests are not optional - they catch critical bugs that unit tests miss by testing the system as users actually use it.