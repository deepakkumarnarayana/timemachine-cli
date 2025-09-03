# TimeMachine CLI Shadow Branching Security Analysis

## Executive Summary

This comprehensive security assessment evaluates the shadow branching implementation in TimeMachine CLI, focusing on repository isolation, branch state management, monitoring mechanisms, and concurrent operations. The analysis identifies several security strengths and areas requiring attention.

**Overall Security Posture: GOOD** with some medium-priority recommendations for hardening.

## Architecture Overview

TimeMachine CLI implements a "shadow repository" system using Git's `--git-dir` and `--work-tree` flags to create complete isolation between the main repository and snapshot storage. The system maintains branch synchronization through caching mechanisms and real-time monitoring.

### Key Components Analyzed:
- **GitManager** (`internal/core/git.go`) - Shadow repository operations
- **AppState** (`internal/core/state.go`) - Branch lifecycle and state management  
- **Watcher** (`internal/core/watcher.go`) - Real-time branch monitoring via .git/HEAD
- **Commands** (`internal/commands/*.go`) - Integration patterns and validation

## Security Assessment by Component

### 1. Shadow Repository Isolation (GitManager) 

**Security Rating: HIGH** ✅

#### Strengths:
- **Complete Git Isolation**: Every Git operation uses `--git-dir=.git/timemachine_snapshots --work-tree=.` ensuring complete separation from main repository
- **Consistent Command Pattern**: All Git operations go through `RunCommand()` method with enforced isolation flags
- **Safe Restoration**: Uses `git restore --source=<hash> --worktree` instead of dangerous `git checkout`/`git reset` operations
- **Path Validation**: Shadow repository directory creation uses safe `os.MkdirAll()` with 0755 permissions

#### Potential Security Concerns:
- **Command Injection Risk**: Git commands constructed via string concatenation could be vulnerable if user input isn't properly validated
- **Config Copy Mechanism**: `copyGitConfig()` copies user.name/user.email from main repo but lacks validation

#### Recommendations:
```go
// Add input validation for branch names
func (g *GitManager) SwitchOrCreateShadowBranch(branchName string) error {
    if !isValidBranchName(branchName) {
        return fmt.Errorf("invalid branch name: %s", branchName)
    }
    // ... rest of implementation
}

func isValidBranchName(name string) bool {
    // Git branch name validation
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9/_.-]+$`, name)
    return matched && !strings.HasPrefix(name, "-") && !strings.HasSuffix(name, ".")
}
```

### 2. Branch State Management (AppState)

**Security Rating: HIGH** ✅

#### Strengths:
- **Thread-Safe Operations**: Uses `sync.RWMutex` for protecting branch state access
- **Cache TTL Implementation**: 30-second cache TTL prevents stale state attacks
- **Validation Patterns**: `ValidateBranchState()` and `EnsureValidBranchState()` provide comprehensive validation
- **Error Handling**: Enhanced error messages with user-friendly context

#### Security Analysis:

**Thread Safety Assessment:**
```go
// AppState properly protects critical sections
func (s *AppState) RefreshBranchState() error {
    s.stateMutex.Lock()           // Exclusive lock for writes
    defer s.stateMutex.Unlock()
    // ... critical section operations
}

func (s *AppState) GetBranchContext() (string, string, bool, error) {
    s.stateMutex.RLock()          // Shared lock for reads  
    defer s.stateMutex.RUnlock()
    // ... read operations
}
```

#### Potential Security Concerns:
- **Cache Invalidation**: `InvalidateBranchCache()` is public and could be misused
- **State Corruption**: No integrity checking of branch state data
- **Time-of-Check-Time-of-Use**: Potential TOCTOU race between validation and use

#### Recommendations:
```go
// Add integrity validation
func (s *AppState) ValidateBranchState() error {
    s.stateMutex.RLock()
    defer s.stateMutex.RUnlock()
    
    // Existing validations...
    
    // Add integrity check
    if err := s.validateStateIntegrity(); err != nil {
        return fmt.Errorf("state integrity check failed: %w", err)
    }
    
    return nil
}

func (s *AppState) validateStateIntegrity() error {
    // Validate branch names contain only safe characters
    if !isValidBranchName(s.CurrentBranch) || !isValidBranchName(s.ShadowBranch) {
        return fmt.Errorf("invalid characters in branch names")
    }
    return nil
}
```

### 3. Real-time Branch Monitoring (Watcher)

**Security Rating: MEDIUM** ⚠️

#### Strengths:
- **File System Isolation**: Monitors only `.git/HEAD` for branch changes
- **Thread-Safe Branch Tracking**: Uses `branchMutex` for branch change detection
- **Graceful Error Handling**: Branch monitoring failures don't crash the application

#### Security Concerns:
- **File System Race Conditions**: `.git/HEAD` monitoring could race with Git operations
- **Branch Name Validation**: No validation of branch names received from Git HEAD
- **Signal Handling**: Branch change events processed without rate limiting

#### Critical Security Issue Found:
```go
// In handleBranchChange() - lacks branch name validation
func (w *Watcher) handleBranchChange() {
    currentBranch, err := w.gitManager.GetCurrentBranch()
    if err != nil {
        fmt.Printf("Warning: failed to get current branch after HEAD change: %v\n", err)
        return
    }
    
    // SECURITY ISSUE: No validation of currentBranch content
    w.lastBranch = currentBranch  // Could contain malicious content
    w.state.CurrentBranch = currentBranch
}
```

#### Recommendations:
```go
func (w *Watcher) handleBranchChange() {
    w.branchMutex.Lock()
    defer w.branchMutex.Unlock()

    currentBranch, err := w.gitManager.GetCurrentBranch()
    if err != nil {
        fmt.Printf("Warning: failed to get current branch after HEAD change: %v\n", err)
        return
    }

    // ADD: Branch name validation
    if !isValidBranchName(currentBranch) {
        fmt.Printf("Warning: invalid branch name detected: %s\n", currentBranch)
        return
    }

    // ADD: Rate limiting for branch changes
    if time.Since(w.lastBranchChangeTime) < time.Second {
        return // Ignore rapid branch changes
    }
    w.lastBranchChangeTime = time.Now()

    // Rest of implementation...
}
```

### 4. Command Integration Security

**Security Rating: HIGH** ✅

#### Strengths:
- **Input Validation**: Commands use consistent validation patterns (e.g., `validateGitHash()`, `sanitizeFilePath()`)
- **Branch State Validation**: All commands call `EnsureValidBranchState()` before operations
- **Defense in Depth**: Multiple validation layers in `inspect.go`
- **Cross-platform Path Security**: Comprehensive path traversal prevention

#### Excellent Security Implementation Example:
```go
// From inspect.go - exemplary defense-in-depth approach
func sanitizeFilePath(path string) (string, error) {
    if path == "" {
        return "", nil
    }
    
    // Multiple validation layers
    if strings.Contains(path, "..") {
        return "", fmt.Errorf("path traversal not allowed")
    }
    
    if len(path) >= 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
        return "", fmt.Errorf("absolute paths not allowed")
    }
    
    if !filepath.IsLocal(path) {
        return "", fmt.Errorf("path must be local and relative")
    }
    
    return filepath.Clean(path), nil
}
```

#### Minor Security Concerns:
- **Hash Validation Inconsistency**: Some commands validate hashes, others don't
- **Error Information Disclosure**: Detailed Git error messages could leak information

## Race Condition Analysis

### Identified Race Conditions:

#### 1. Branch Switch During Operations ⚠️
**Scenario**: User switches branches while TimeMachine operations are in progress

**Risk**: Medium - Could result in operations on wrong branch or state corruption

**Mitigation**: Existing mutex protection in AppState provides good protection, but add operation-level locking:

```go
type OperationLock struct {
    mu sync.Mutex
    activeOps map[string]bool
}

func (g *GitManager) CreateSnapshot(message string) error {
    g.operationLock.Lock()
    defer g.operationLock.Unlock()
    
    // Ensure branch hasn't changed during operation
    if err := g.State.EnsureValidBranchState(); err != nil {
        return fmt.Errorf("branch state changed during operation: %w", err)
    }
    
    // ... rest of implementation
}
```

#### 2. Concurrent File Watcher Events ✅
**Assessment**: Well handled by debouncer mechanism and thread-safe event processing

#### 3. Shadow Repository Access ✅
**Assessment**: Git's internal locking mechanisms protect against corruption

## Attack Vector Analysis

### Potential Attack Scenarios:

#### 1. Malicious Branch Names 
**Risk**: HIGH if unvalidated branch names are used in commands
**Current Status**: Partially mitigated by Git's own validation
**Recommendation**: Add explicit validation

#### 2. Shadow Repository Corruption
**Risk**: LOW - Isolated from main repository, limited blast radius
**Current Status**: Well isolated

#### 3. State Cache Poisoning
**Risk**: MEDIUM - Malicious processes could manipulate branch state
**Current Status**: TTL and validation provide some protection
**Recommendation**: Add integrity checking

#### 4. File System Race Conditions
**Risk**: LOW-MEDIUM - Timing attacks on `.git/HEAD` monitoring
**Current Status**: Basic mutex protection
**Recommendation**: Enhanced rate limiting

## Security Recommendations

### Priority 1 (High Impact)

1. **Add Branch Name Validation**
```go
func isValidBranchName(name string) bool {
    if name == "" || len(name) > 255 {
        return false
    }
    
    // Git branch name rules
    matched, _ := regexp.MatchString(`^[a-zA-Z0-9/_.-]+$`, name)
    return matched && 
           !strings.HasPrefix(name, "-") && 
           !strings.HasPrefix(name, ".") &&
           !strings.HasSuffix(name, ".") &&
           !strings.Contains(name, "..")
}
```

2. **Implement Operation-Level Locking**
```go
type GitManager struct {
    State *AppState
    operationMutex sync.Mutex // Add operation-level protection
}
```

### Priority 2 (Medium Impact)

3. **Add State Integrity Validation**
4. **Implement Rate Limiting for Branch Changes**
5. **Enhanced Error Sanitization**

### Priority 3 (Low Impact)

6. **Audit Logging for Security Events**
7. **Enhanced Monitoring Capabilities**

## Conclusion

The TimeMachine CLI shadow branching implementation demonstrates **strong security architecture** with excellent isolation mechanisms and thread safety. The major strengths include:

- Complete Git repository isolation using `--git-dir`/`--work-tree` flags
- Robust thread safety with proper mutex usage
- Comprehensive input validation in command handlers
- Defense-in-depth security patterns

The identified security concerns are primarily **medium-priority hardening opportunities** rather than critical vulnerabilities. The implementation shows security-conscious design patterns throughout.

**Overall Security Assessment: GOOD** ✅

The system is production-ready with recommended security enhancements that would elevate it to excellent security posture.

---

*Security Analysis conducted on: 2025-09-03*  
*Analyzed Components: GitManager, AppState, Watcher, Command Integration*  
*Assessment Methodology: Code review, attack vector analysis, race condition assessment*