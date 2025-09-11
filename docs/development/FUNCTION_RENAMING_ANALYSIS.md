# Function Renaming Analysis & Design

## Current State Analysis

### Function Inventory
| Current Name | Location | Purpose | Allows Absolute | Issues |
|---|---|---|---|---|
| `sanitizeGitPath()` | `inspect.go` | Local wrapper for system paths | ✅ Yes | Confusing name vs shared function |
| `sanitizeGitPath()` | `git.go` | Local wrapper for system paths | ✅ Yes | Duplicate name, unclear purpose |
| `sanitizeFilePath()` | `inspect.go` | User file filter validation | ❌ No | Purpose not clear from name |
| `SanitizeGitPath()` | `security/path.go` | System path validation | ✅ Yes | Similar name to local functions |
| `SanitizeUserInputPath()` | `security/path.go` | User input validation | ❌ No | Long name, unclear vs GitPath |

### Usage Patterns Identified
```go
// System paths (absolute paths OK):
sanitizeGitPath(state.ShadowRepoDir)    // ~/.../project/.git/timemachine_snapshots  
sanitizeGitPath(state.ProjectRoot)      // ~/.../project
sanitizeGitPath(state.GitDir)          // ~/.../project/.git

// User inputs (relative only):
sanitizeFilePath(fileFilter)           // "src/main.go" from user --file flag
```

## Proposed New Naming Scheme

### Design Principles
1. **Purpose-Driven Names**: Function name clearly indicates what it validates
2. **Security Level Clear**: Obvious which functions are for system vs user input
3. **No Naming Conflicts**: Local vs shared functions clearly distinguished
4. **Consistent Patterns**: Similar naming convention across all functions

### New Function Names

#### Shared Functions (security/path.go)
```go
// Current → New
SanitizeGitPath()        → ValidateSystemPath()      // For system directories (absolute OK)
SanitizeUserInputPath()  → ValidateUserInputPath()   // For user inputs (relative only)
```

#### Local Wrapper Functions
```go
// Current → New  
sanitizeGitPath()    → validateSystemGitDir()    // System Git directories
sanitizeFilePath()   → validateUserFileFilter()  // User file filters
```

#### Additional Validation Functions
```go
validateGitHash()    → validateGitHash()         // Keep existing - already clear
```

## Detailed Function Mapping

### 1. System Path Validation
**Purpose**: Validate paths for shadow repo, project root, git directories
**Security**: Allows absolute paths (required for system operations)
```go
// Before:
func sanitizeGitPath(path string) (string, error) {
    return security.SanitizeGitPath(path)
}

// After:  
func validateSystemGitDir(path string) (string, error) {
    return security.ValidateSystemPath(path)
}
```

### 2. User File Filter Validation  
**Purpose**: Validate file filters provided by user (--file flag)
**Security**: Only allows relative paths (security requirement)
```go
// Before:
func sanitizeFilePath(path string) (string, error) {
    // Complex validation logic...
}

// After:
func validateUserFileFilter(path string) (string, error) {
    // Same validation logic, clearer purpose
}
```

### 3. Shared Security Functions
**Purpose**: Core validation logic used by wrappers
**Security**: Clear distinction between system vs user input
```go
// Before:
func SanitizeGitPath(path string) (string, error) {
    // Allows absolute paths for system use
}
func SanitizeUserInputPath(path string) (string, error) {
    // Relative paths only for user input
}

// After:
func ValidateSystemPath(path string) (string, error) {
    // Allows absolute paths - clearly for system use
}
func ValidateUserInputPath(path string) (string, error) {
    // Relative paths only - clearly for user input
}
```

## Benefits of New Naming

### ✅ Clear Purpose
- `validateSystemGitDir()` - Obviously for system Git directories
- `validateUserFileFilter()` - Obviously for user-provided file filters
- `ValidateSystemPath()` - Obviously for system paths (absolute OK)
- `ValidateUserInputPath()` - Obviously for user input (relative only)

### ✅ Security Clarity  
- "System" functions → Absolute paths allowed
- "User" functions → Relative paths only  
- No confusion about which to use when

### ✅ No Name Conflicts
- Local vs shared functions clearly distinguished
- No duplicate function names with different purposes
- Consistent naming patterns

### ✅ Maintainability
- New developers immediately understand purpose
- Code review easier with clear function names
- Future refactoring less error-prone

## Implementation Plan

### Phase 1: Rename Shared Functions (security/path.go)
1. `SanitizeGitPath()` → `ValidateSystemPath()`
2. `SanitizeUserInputPath()` → `ValidateUserInputPath()`  
3. Update all imports and usage

### Phase 2: Rename Local Wrappers
1. `sanitizeGitPath()` → `validateSystemGitDir()`
2. `sanitizeFilePath()` → `validateUserFileFilter()`
3. Update all call sites

### Phase 3: Update Tests & Documentation
1. Rename test functions to match
2. Update test error messages
3. Update documentation references
4. Update integration tests

### Phase 4: Validation
1. Run all tests to ensure no breakage
2. Run integration tests
3. Manual testing of key commands
4. Update CLAUDE.md with new patterns

## Risk Mitigation

### Low Risk Refactoring
- Only changing function names, no logic changes
- Comprehensive integration tests will catch any issues
- Each phase can be validated independently

### Rollback Plan
- Git branch for each phase allows easy rollback
- Integration tests provide confidence for each change
- Can be done incrementally with testing at each step

## Success Criteria

- [ ] All function names clearly indicate purpose
- [ ] No naming conflicts between local/shared functions
- [ ] All tests pass with new names
- [ ] Integration tests pass
- [ ] Documentation updated to reflect new patterns
- [ ] Future developers can immediately understand function purpose