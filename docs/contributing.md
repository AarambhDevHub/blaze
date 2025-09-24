# Contributing to Blaze

Thank you for your interest in contributing to Blaze, a blazing fast Go web framework inspired by Axum and Actix Web!  This document provides comprehensive guidelines

## Overview

Blaze is a lightweight, high-performance web framework for Go that features advanced routing, middleware support, WebSocket connections, HTTP/2 compatibility, TLS security, multipart form handlin

## Getting Started

### Prerequisites

- Go 1.24.0 or later
- Git for version control
- Basic understanding of Go programming
- Familiarity with web frameworks and HTTP protocols

### Development Environment Setup

1. **Fork and Clone the Repository**
   ```bash
   git clone https://github.com/AarambhDevHub/blaze.git
   cd blaze
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   go mod tidy
   ```

3. **Verify Installation**
   ```bash
   go build ./...
   go test ./...
   ```

## Project Structure

Understanding the project structure

```
blaze/
├── examples/
│   └── basic.go              # Example applications
├── pkg/
│   └── blaze/
│       ├── app.go            # Core application logic
│       ├── context.go        # Request/Response context handling
│       ├── router.go         # Advanced routing system
│       ├── middleware.go     # Built-in middleware
│       ├── websocket.go      # WebSocket support
│       ├── http2.go          # HTTP/2 implementation
│       ├── tls.go            # TLS/SSL configuration
│       ├── cache_middleware.go # Caching system
│       ├── multipart.go      # File upload handling
│       └── error.go          # Error handling utilities
├── docs/                     # Documentation
├── go.mod                    # Go module definition
├── go.sum                    # Dependency checksums
├── README.md                 # Project overview
├── LICENSE                   # Project license
└── Makefile                  # Build automation
```

## How to Contribute

### Types of Contributions

We welcome various types of contributions:

- **Bug Reports**: Help identify and document issues
- **Feature Requests**: Propose new functionality
- **Code Contributions**: Implement features, fix bugs, optimize performance
- **Documentation**: Improve guides, API documentation, examples
- **Testing**: Add test cases, improve test coverage
- **Performance Improvements**: Optimize existing code
- **Security Enhancements**: Identify and fix security vulnerabilities

### Reporting Issues

When reporting bugs or requesting features:

1. **Check Existing Issues**: Search existing issues to avoid duplicates
2. **Use Issue Templates**: Follow the provided templates when available
3. **Provide Context**: Include Go version, OS, and relevant code snippets
4. **Minimal Reproduction**: Create the smallest possible example that demonstrates the issue

**Bug Report Template**:
```go
### Description
Brief description of the issue

### Steps to Reproduce
1. Step one
2. Step two
3. Step three

### Expected Behavior
What should happen

### Actual Behavior
What actually happens

### Environment
- Go version: 
- OS: 
- Blaze version: 

### Code Sample
```go
// Minimal reproduction code
```

### Pull Request Process

1. **Create a Feature Branch**
   ```
   git checkout -b feature/your-feature-name
   git checkout -b bugfix/issue-description
   ```

2. **Make Your Changes**
   - Follow coding standards (see below)
   - Add appropriate tests
   - Update documentation if needed
   - Ensure backwards compatibility when possible

3. **Test Your Changes**
   ```
   go test ./...
   go vet ./...
   go fmt ./...
   ```

4. **Commit Your Changes**
   ```
   git add .
   git commit -m "feat: add new middleware for request logging"
   ```

5. **Push and Create PR**
   ```
   git push origin your-branch-name
   ```

## Coding Standards

### Go Conventions

Follow standard Go conventions and the existing codebase style: [file:1]

1. **Naming Conventions**
   - Use `PascalCase` for exported functions, types, and variables
   - Use `camelCase` for unexported functions and variables
   - Interface names should end with `-er` when appropriate
   - Package names should be lowercase, single words

2. **Code Organization**
   - Group related functionality in the same file
   - Keep functions focused and single-purpose
   - Use meaningful variable and function names
   - Add comments for exported functions and complex logic

3. **Error Handling**
   - Always handle errors explicitly
   - Use custom error types when appropriate
   - Provide meaningful error messages
   - Follow the existing error handling patterns in the codebase

### Framework-Specific Guidelines

1. **Context Handling**
   - Always use the Blaze `*Context` type for handlers
   - Utilize context locals for request-scoped data
   - Implement proper timeout handling

2. **Middleware Development**
   - Follow the `MiddlewareFunc` signature: `func(HandlerFunc) HandlerFunc`
   - Chain middleware properly
   - Handle edge cases and errors gracefully
   - Document middleware behavior and usage

3. **Router Contributions**
   - Maintain performance characteristics of the radix tree implementation
   - Support parameter extraction and wildcard routing
   - Preserve route precedence rules

4. **HTTP/2 and TLS**
   - Maintain compatibility with both HTTP/1.1 and HTTP/2
   - Implement proper TLS configuration and security practices
   - Support modern cipher suites and protocols

### Code Quality Requirements

1. **Testing**
   - Write unit tests for new functionality
   - Maintain or improve test coverage
   - Include integration tests for complex features
   - Use table-driven tests where appropriate

2. **Documentation**
   - Add GoDoc comments for all exported functions and types
   - Update relevant documentation files
   - Include usage examples in comments
   - Keep documentation up-to-date with code changes

3. **Performance**
   - Consider performance implications of changes
   - Add benchmarks for performance-critical code
   - Avoid unnecessary allocations
   - Use profiling tools to validate optimizations

## Testing Guidelines

### Running Tests

```
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run specific test
go test -run TestSpecificFunction ./pkg/blaze
```

### Writing Tests

1. **Test Structure**
   ```
   func TestFeatureName(t *testing.T) {
       // Arrange
       app := blaze.New()
       
       // Act
       result := doSomething()
       
       // Assert
       if result != expected {
           t.Errorf("expected %v, got %v", expected, result)
       }
   }
   ```

2. **Table-Driven Tests**
   ```
   func TestMultipleCases(t *testing.T) {
       tests := []struct {
           name     string
           input    string
           expected string
       }{
           {"case1", "input1", "expected1"},
           {"case2", "input2", "expected2"},
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               result := processInput(tt.input)
               assert.Equal(t, tt.expected, result)
           })
       }
   }
   ```

3. **HTTP Testing**
   ```
   func TestHTTPHandler(t *testing.T) {
       app := blaze.New()
       app.GET("/test", func(c *blaze.Context) error {
           return c.Text("Hello, World!")
       })
       
       req := httptest.NewRequest("GET", "/test", nil)
       resp := httptest.NewRecorder()
       
       // Test logic here
   }
   ```

## Documentation

### API Documentation

- Use GoDoc format for all public APIs
- Include usage examples
- Document parameters, return values, and possible errors
- Keep documentation concise but comprehensive

### User Guides

When contributing to user-facing documentation:

- Use clear, step-by-step instructions
- Include complete, working code examples
- Explain concepts before showing implementation
- Link to related documentation sections

### Example Format

```
// CacheMiddleware creates middleware for HTTP response caching.
// It supports configurable TTL, eviction strategies, and memory limits.
//
// Usage:
//   app.Use(blaze.CacheMiddleware(blaze.DefaultCacheOptions()))
//
// Options can be customized:
//   opts := &blaze.CacheOptions{
//       DefaultTTL: 5 * time.Minute,
//       MaxSize:    100 * 1024 * 1024, // 100MB
//   }
//   app.Use(blaze.CacheMiddleware(opts))
func CacheMiddleware(opts *CacheOptions) MiddlewareFunc {
    // Implementation
}
```

## Performance Considerations

### Benchmarking

When making performance-related changes:

1. **Write Benchmarks**
   ```
   func BenchmarkFeature(b *testing.B) {
       for i := 0; i < b.N; i++ {
           // Benchmark code
       }
   }
   ```

2. **Profile Your Code**
   ```
   go test -bench=. -cpuprofile=cpu.prof
   go tool pprof cpu.prof
   ```

3. **Memory Optimization**
   - Minimize allocations in hot paths
   - Reuse objects when possible
   - Use object pools for frequently allocated types

### FastHTTP Integration

Since Blaze is built on FastHTTP, consider: [file:1]

- Zero-copy operations where possible
- Proper buffer management
- Connection pooling and reuse
- Efficient header handling

## Security Guidelines

### Security Best Practices

1. **Input Validation**
   - Always validate and sanitize user input
   - Use parameterized queries for database operations
   - Implement proper CSRF protection

2. **TLS/SSL Implementation**
   - Use secure cipher suites and protocols
   - Implement proper certificate validation
   - Support modern TLS versions (1.2+)

3. **Authentication & Authorization**
   - Implement secure session management
   - Use proper password hashing
   - Follow OAuth/JWT best practices

### Reporting Security Issues

For security vulnerabilities:

- **Do not** create public GitHub issues
- Email security concerns privately
- Provide detailed reproduction steps
- Allow reasonable time for fixes before disclosure

## Community and Communication

### Getting Help

- **GitHub Discussions**: For general questions and discussions
- **Issues**: For bug reports and feature requests
- **Documentation**: Check existing docs first

### Code Reviews

All contributions go through code review:

- Be respectful and constructive
- Focus on code quality and maintainability
- Explain the reasoning behind suggestions
- Be open to feedback and suggestions

### Commit Message Format

Use conventional commit format:

```
type(scope): description

body (optional)

footer (optional)
```

**Types:**
- `feat`: New features
- `fix`: Bug fixes
- `docs`: Documentation changes
- `style`: Code formatting changes
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Build process or auxiliary tool changes

**Examples:**
```
feat(router): add support for route groups with shared middleware

fix(websocket): resolve memory leak in connection pooling

docs(api): update middleware documentation with usage examples
```

## Release Process

### Versioning

We follow Semantic Versioning (SemVer):

- **MAJOR**: Incompatible API changes
- **MINOR**: Backwards-compatible functionality additions
- **PATCH**: Backwards-compatible bug fixes

### Release Checklist

For maintainers preparing releases:

1. Update version numbers
2. Update CHANGELOG.md
3. Run full test suite
4. Performance regression testing
5. Update documentation
6. Create release notes
7. Tag release in Git

## Development Workflow

### Feature Development

1. **Planning Phase**
   - Discuss feature in GitHub issues
   - Get consensus on approach
   - Consider backward compatibility

2. **Implementation Phase**
   - Create feature branch
   - Implement core functionality
   - Add comprehensive tests
   - Update documentation

3. **Review Phase**
   - Submit pull request
   - Address review feedback
   - Ensure CI passes
   - Get maintainer approval

### Bug Fixes

1. **Reproduction**
   - Create minimal test case
   - Understand root cause
   - Consider edge cases

2. **Fix Implementation**
   - Implement targeted fix
   - Add regression tests
   - Verify fix doesn't break existing functionality

3. **Validation**
   - Test with original reproduction case
   - Run full test suite
   - Consider performance impact

## Advanced Topics

### Extending Core Functionality

When adding major new features:

1. **Architecture Considerations**
   - Maintain separation of concerns
   - Consider plugin architecture
   - Plan for extensibility

2. **Integration Points**
   - Work with existing middleware system
   - Maintain HTTP/2 compatibility
   - Consider TLS implications

### Performance Profiling

Use Go's built-in profiling tools:

```
# CPU profiling
go test -bench=. -cpuprofile=cpu.prof

# Memory profiling
go test -bench=. -memprofile=mem.prof

# Block profiling
go test -bench=. -blockprofile=block.prof
```

### Cross-Platform Considerations

Ensure contributions work across platforms:

- Test on multiple operating systems
- Consider endianness and architecture differences
- Handle file path separators correctly
- Test with different Go versions

## Troubleshooting

### Common Issues

1. **Build Failures**
   - Ensure Go version compatibility
   - Check dependency versions
   - Clear module cache: `go clean -modcache`

2. **Test Failures**
   - Check for race conditions: `go test -race`
   - Verify test isolation
   - Check timing-dependent tests

3. **Import Issues**
   - Verify module path
   - Check Go version requirements
   - Ensure proper module initialization

### Debug Tips

- Use `go vet` to catch common issues
- Enable race detection during development
- Use debugging tools like Delve
- Add strategic logging for complex issues

## Recognition

Contributors will be:

- Added to the CONTRIBUTORS file
- Mentioned in release notes for significant contributions
- Credited in documentation where appropriate

## Questions?

If you have questions not covered in this guide:

- Check existing GitHub issues and discussions
- Create a new discussion for general questions
- Refer to the project documentation
- Contact maintainers for complex architectural questions

Thank you for contributing to Blaze! Your efforts help make this framework better for the entire Go community.