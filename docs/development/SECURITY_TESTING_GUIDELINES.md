# Security Testing Guidelines

## 🔒 **Overview**

This document provides comprehensive security testing guidelines for TimeMachine CLI to ensure robust protection against common attack vectors while maintaining usability for legitimate users.

## 📋 **Security Testing Strategy**

### **1. Input Validation Testing**

#### **Git Hash Validation**
```go
// Test Cases for Git Hash Security
func TestGitHashSecurity(t *testing.T) {
    testCases := []struct {
        name        string
        input       string
        shouldFail  bool
        description string
    }{
        {"Valid short hash", "abc123", false, "Standard 6-character hash"},
        {"Valid full hash", "abcdef1234567890abcdef1234567890abcdef12", false, "Full 40-character hash"},
        {"Command injection", "abc123; rm -rf /", true, "Shell command injection attempt"},
        {"SQL injection", "abc123'; DROP TABLE users; --", true, "SQL injection pattern"},
        {"Path traversal", "../../../etc/passwd", true, "Path traversal in hash"},
        {"Script injection", "<script>alert('xss')</script>", true, "Script tag injection"},
        {"Null bytes", "abc123\x00../../etc/passwd", true, "Null byte injection"},
    }
}
```

#### **File Path Validation**
```go
// Test Cases for Path Security  
func TestPathSecurity(t *testing.T) {
    testCases := []struct {
        name        string
        input       string
        shouldFail  bool
        threat      string
    }{
        {"Basic traversal", "../../../etc/passwd", true, "Unix path traversal"},
        {"Windows traversal", "..\\..\\..\\windows\\system32", true, "Windows path traversal"},
        {"Mixed separators", "../..\\../etc/passwd", true, "Mixed separator attack"},
        {"Encoded traversal", "%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd", true, "URL encoded traversal"},
        {"Unicode traversal", "..%c0%af..%c0%af..%c0%afetc%c0%afpasswd", true, "Unicode encoding bypass"},
        {"Absolute paths", "/etc/passwd", true, "Absolute path access"},
        {"UNC paths", "\\\\server\\share\\file", true, "Windows UNC path"},
    }
}
```

### **2. Command Injection Prevention**

#### **Git Command Security**
```go
// Test Cases for Command Injection
func TestCommandInjection(t *testing.T) {
    dangerousInputs := []string{
        "hash; rm -rf /",
        "hash && echo pwned",  
        "hash | cat /etc/passwd",
        "$(rm -rf /)",
        "`rm -rf /`",
        "hash$(whoami)",
        "hash & net user hacker password /add",
    }
    
    // All should be rejected by input validation
    for _, input := range dangerousInputs {
        err := validateGitHash(input)
        if err == nil {
            t.Errorf("Command injection not detected: %s", input)
        }
    }
}
```

### **3. Path Traversal Protection**

#### **Multi-Layer Defense**
1. **Input Sanitization**: Reject `..` patterns and absolute paths
2. **Path Canonicalization**: Use `filepath.Clean()` to normalize paths
3. **Containment Validation**: Ensure resolved paths stay within allowed directories
4. **OS-Specific Checks**: Handle Windows/Unix path differences

```go
// Comprehensive Path Traversal Testing
func TestPathTraversalDefense(t *testing.T) {
    attacks := []struct {
        input       string
        threat      string
        shouldBlock bool
    }{
        {"../../../etc/passwd", "Basic Unix traversal", true},
        {"..\\..\\..\\windows\\system32\\config\\SAM", "Windows traversal", true}, 
        {"....//....//....//etc/passwd", "Double encoding", true},
        {"..%2F..%2F..%2Fetc%2Fpasswd", "URL encoding", true},
        {"file/../../etc/passwd", "Nested traversal", true},
        {"symlink-to-etc/../passwd", "Symlink traversal", true},
    }
}
```

### **4. Race Condition Testing**

#### **Concurrent Operations**
```go
// Test Cases for Race Conditions
func TestConcurrentOperations(t *testing.T) {
    // Test concurrent snapshot creation
    // Test concurrent restore operations  
    // Test file system race conditions
    // Test shadow repository concurrent access
}
```

### **5. Privilege Escalation Prevention**

#### **File System Permissions**
- **Shadow Repository Access**: Ensure shadow repo inherits proper permissions
- **Temporary File Security**: Secure creation of temporary files during operations
- **Hook Script Validation**: Validate Git hooks don't enable privilege escalation

## 🚨 **Known Security Issues**

### **Current Vulnerabilities (To Fix)**

1. **Path Traversal in Restore Command** ⚠️
   ```bash
   # Currently VULNERABLE - needs fix
   ./timemachine restore abc123 --files "../../../etc/passwd" --force
   ```
   - **Impact**: Could potentially access files outside project directory
   - **Priority**: HIGH - Immediate fix required
   - **Fix**: Implement proper path validation in restore command

2. **Symlink Following** ⚠️
   ```bash
   # Potential issue with symlinks
   ln -s /etc/passwd project-file.txt  
   ./timemachine restore abc123 --files project-file.txt
   ```
   - **Impact**: Could follow symlinks to sensitive files
   - **Priority**: MEDIUM - Validate symlink handling

### **Security Improvements Implemented** ✅

1. **Git Hash Validation** ✅
   - Regex validation prevents command injection
   - Limited to valid hex characters and length

2. **Input Sanitization** ✅  
   - `ValidateUserInputPath()` rejects path traversal attempts
   - `ValidateSystemPath()` allows absolute paths for system operations only

3. **Shadow Repository Isolation** ✅
   - Complete isolation from main Git repository
   - Uses separate `.git/timemachine_snapshots/` directory

## 🔧 **Security Testing Implementation**

### **Property-Based Testing**
```go
// Generate random inputs to test security boundaries
func TestSecurityProperties(t *testing.T) {
    quick.Check(func(input string) bool {
        // Any user input should either be accepted safely or rejected clearly
        err := validateUserInput(input)
        return err != nil || isSafeInput(input)
    }, nil)
}
```

### **Fuzzing Integration**
```bash
# Use Go's built-in fuzzing for security testing
go test -fuzz=FuzzGitHash
go test -fuzz=FuzzFilePath  
go test -fuzz=FuzzCommandInput
```

### **Security Test Automation**
```go
// Automated security regression testing
func TestSecurityRegression(t *testing.T) {
    // Run all known attack patterns
    // Ensure no previously fixed vulnerabilities return
    // Test with various input encodings
}
```

## 🎯 **Testing Priorities**

### **Phase 1: Critical Fixes** (Immediate)
1. ✅ **Function Naming Clarity** - Completed
2. ✅ **Integration Testing** - Completed  
3. ✅ **Behavior-Driven Tests** - Completed
4. 🚧 **Fix Path Traversal in Restore** - In Progress
5. 🔄 **Property-Based Security Tests** - Pending

### **Phase 2: Enhanced Security** (Next Sprint)
1. **Symlink Validation** - Add symlink following protection
2. **File Permission Auditing** - Ensure secure file creation
3. **Cross-Platform Security** - Windows/Unix security parity
4. **Security Documentation** - User security guidelines

### **Phase 3: Advanced Protection** (Future)
1. **Rate Limiting** - Prevent snapshot spam attacks
2. **Resource Limits** - Prevent disk exhaustion attacks  
3. **Audit Logging** - Security event logging
4. **Sandboxing** - Container-based isolation

## 📚 **Security References**

### **Attack Vectors to Test**
- **OWASP Top 10** compliance
- **CWE-22**: Path Traversal  
- **CWE-78**: Command Injection
- **CWE-79**: Cross-Site Scripting (if applicable)
- **CWE-367**: Race Conditions

### **Security Tools Integration**
```bash
# Static Analysis
gosec ./...

# Dependency Scanning  
nancy sleuth

# Fuzzing
go test -fuzz=Fuzz -fuzztime=10m

# Penetration Testing
# Manual security testing with specialized tools
```

## 🎪 **Security Testing Workflow**

### **Pre-Commit Testing**
1. Run security unit tests
2. Execute static analysis (gosec)  
3. Validate input sanitization
4. Check for hardcoded secrets

### **CI/CD Integration**
1. Automated security regression tests
2. Dependency vulnerability scanning
3. Fuzzing on long-running builds
4. Security-focused integration tests

### **Release Security Validation**
1. Full security test suite execution
2. Manual penetration testing
3. Security review of new features
4. Documentation security updates

---

## 💡 **Key Security Principles**

1. **Defense in Depth**: Multiple layers of validation
2. **Fail Securely**: Invalid inputs fail safely with clear errors
3. **Least Privilege**: Operations use minimum required permissions
4. **Input Validation**: All user inputs are validated and sanitized
5. **Secure by Default**: Default configuration is secure
6. **Clear Security Boundaries**: System vs user input validation separation

This document ensures TimeMachine CLI maintains robust security while providing excellent user experience for AI-assisted development workflows.