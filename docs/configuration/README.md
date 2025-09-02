# TimeMachine CLI Configuration System Documentation

## Overview

This directory contains comprehensive documentation for TimeMachine CLI's production-grade configuration management system. The system is built on Viper with security-first design principles, comprehensive validation, and enterprise-ready features.

## Documentation Structure

### 📐 [Architecture Guide](ARCHITECTURE.md)
**Target Audience**: System architects, senior developers, DevOps engineers

Comprehensive overview of the configuration system's design, architecture components, and implementation details. Covers:
- Security-first architecture principles
- Configuration loading hierarchy and precedence
- Component relationships and data flow
- Performance characteristics and optimization
- Future enhancement roadmap

### 📖 [Configuration Reference](REFERENCE.md)
**Target Audience**: End users, system administrators, developers

Complete reference for all configuration options, formats, and usage examples. Includes:
- All configuration sections with detailed explanations
- Environment variable reference
- Configuration file formats and locations
- CLI command usage and examples
- Production and development configuration templates

### 👨‍💻 [Developer Integration Guide](DEVELOPER_GUIDE.md)
**Target Audience**: Application developers, contributors, integrators

Detailed guide for integrating with and extending the configuration system. Covers:
- Integration patterns and best practices
- Adding new configuration options
- Custom validation implementation
- Testing strategies and examples
- Migration procedures and compatibility

### 🔒 [Security Guide](SECURITY.md)
**Target Audience**: Security engineers, DevSecOps teams, compliance officers

Comprehensive security documentation covering threat models, vulnerabilities, and best practices. Includes:
- Security features and vulnerability prevention
- Threat model and attack vector analysis
- Compliance guidelines and audit requirements
- Incident response procedures
- Security monitoring and validation

### 🔧 [Troubleshooting Guide](TROUBLESHOOTING.md)
**Target Audience**: Support engineers, system administrators, developers

Complete troubleshooting reference with solutions for common issues. Covers:
- Quick diagnostic procedures
- Common configuration problems and solutions
- Validation error reference
- Performance troubleshooting
- Advanced debugging techniques

## Quick Start

### For End Users
1. **Create Configuration**: `timemachine config init`
2. **Validate Setup**: `timemachine config validate`
3. **View Settings**: `timemachine config show`
4. **Customize**: Edit `timemachine.yaml` or use environment variables

### For Developers
1. **Read**: [Developer Integration Guide](DEVELOPER_GUIDE.md)
2. **Integrate**: Use `AppState` for configuration access
3. **Extend**: Follow patterns for adding new options
4. **Test**: Use provided test utilities and patterns

### For Security Teams
1. **Review**: [Security Guide](SECURITY.md)
2. **Assess**: Security features and threat model
3. **Implement**: Security best practices and monitoring
4. **Audit**: Configuration compliance and validation

## Configuration System Features

### 🎯 **Production-Ready**
- **Multi-source configuration**: Files, environment variables, defaults with clear precedence
- **Comprehensive validation**: Type safety, range checking, security validation
- **Enterprise integration**: CLI tools, debugging utilities, audit trails

### 🛡️ **Security-First Design**
- **Path traversal prevention**: Multiple validation layers against directory traversal attacks
- **Environment variable security**: Explicit whitelist prevents injection attacks
- **Secure permissions**: Configuration files use restrictive 0600 permissions
- **Input sanitization**: All configuration values undergo strict validation

### 🔧 **Developer-Friendly**
- **Type-safe access**: Strongly typed configuration structs
- **Hot reload support**: Architecture supports runtime configuration updates
- **Comprehensive testing**: 100+ test cases with full coverage
- **Clean integration**: AppState pattern for dependency injection

### 📊 **Monitoring & Operations**
- **Validation utilities**: Built-in configuration validation and troubleshooting
- **Debug capabilities**: Comprehensive logging and diagnostic tools
- **Performance optimization**: Efficient loading and memory management
- **Migration support**: Version compatibility and upgrade procedures

## Configuration Sections

| Section | Purpose | Key Settings |
|---------|---------|--------------|
| **Log** | Logging behavior | Level, format, file output |
| **Watcher** | File watching | Debounce delay, file limits, ignore patterns |
| **Cache** | Performance caching | Memory limits, TTL, LRU settings |
| **Git** | Repository management | Cleanup thresholds, GC settings, commit limits |
| **UI** | User interface | Progress indicators, colors, output formats |

## Environment Variables

All configuration options can be overridden using `TIMEMACHINE_*` environment variables:

```bash
# Core configuration
export TIMEMACHINE_LOG_LEVEL=debug
export TIMEMACHINE_WATCHER_DEBOUNCE=2s
export TIMEMACHINE_CACHE_MAX_MEMORY=50
export TIMEMACHINE_GIT_AUTO_GC=true
export TIMEMACHINE_UI_COLOR=true
```

See [Configuration Reference](REFERENCE.md) for complete environment variable documentation.

## CLI Commands

```bash
# Configuration management
timemachine config init [--global] [--force]    # Create default config
timemachine config show [--format yaml|json]    # Display current config
timemachine config get <key>                    # Get specific value
timemachine config set <key> <value>            # Set configuration value
timemachine config validate                     # Validate configuration
```

## Common Use Cases

### Development Environment Setup
```yaml
log:
  level: debug
  format: json

watcher:
  debounce_delay: 1s
  ignore_patterns:
    - "node_modules/**"
    - "*.tmp"

cache:
  max_memory_mb: 25
  ttl: 30m
```

### Production Environment Setup
```yaml
log:
  level: warn
  format: json
  file: "/var/log/timemachine/app.log"

watcher:
  debounce_delay: 5s
  max_watched_files: 200000

cache:
  max_entries: 20000
  max_memory_mb: 100
  ttl: 2h

ui:
  color_output: false
  pager: never
```

### CI/CD Environment Setup
```yaml
log:
  level: info
  format: json

watcher:
  debounce_delay: 3s
  max_watched_files: 10000

cache:
  max_entries: 1000
  max_memory_mb: 10

ui:
  progress_indicators: false
  color_output: false
  table_format: json
```

## Getting Help

### Documentation Priority
1. **Quick Issues**: [Troubleshooting Guide](TROUBLESHOOTING.md)
2. **Configuration Questions**: [Configuration Reference](REFERENCE.md)
3. **Security Concerns**: [Security Guide](SECURITY.md)
4. **Development Help**: [Developer Guide](DEVELOPER_GUIDE.md)
5. **Architecture Questions**: [Architecture Guide](ARCHITECTURE.md)

### Diagnostic Commands
```bash
# Health check
timemachine config validate

# Current configuration
timemachine config show

# Debug information
export TIMEMACHINE_LOG_LEVEL=debug
timemachine config validate

# Environment check
env | grep TIMEMACHINE_
```

### Support Information
- **Issue Tracking**: GitHub Issues
- **Documentation**: This directory
- **Examples**: See each guide for specific examples
- **Security Issues**: Follow responsible disclosure via security contact

## Contributing

When contributing to the configuration system:

1. **Read**: [Developer Integration Guide](DEVELOPER_GUIDE.md)
2. **Follow**: Security best practices from [Security Guide](SECURITY.md)
3. **Test**: Add comprehensive test coverage
4. **Document**: Update relevant documentation sections
5. **Validate**: Ensure all validation rules are covered

## Version History

- **v1.0.0**: Initial production release with complete configuration system
- **Security Fix**: Removed `AutomaticEnv()` vulnerability (environment variable injection prevention)
- **Feature Complete**: All 5 configuration sections with comprehensive validation

---

This documentation represents a production-grade configuration system designed for enterprise use with security, reliability, and maintainability as core principles.