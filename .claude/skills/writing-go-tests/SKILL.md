---
name: writing-go-tests
description: Write Go tests following project conventions. Use when creating test files, writing unit or integration tests, choosing mocks, or setting up test fixtures. Covers test naming, assertions, mock usage, table-driven patterns, and common pitfalls.
---

# Writing Go Tests

Project-specific test conventions for this codebase.

## Critical Rules

- **Always use mockery-generated `Moq*` mocks** when one exists for the interface. Never hand-roll a mock struct for an interface that has a `*_mock.go` file.
- **NEVER edit `*_mock.go` files manually** — not even for "simple" signature changes. Always regenerate by running `mockery` (no args) from the module root. If `mockery` fails due to stale mock contents, delete the offending `*_mock.go` file and re-run `mockery`.

## Mock Selection Guide

| Situation | Use | Import |
|-----------|-----|--------|
| Interface has a `*_mock.go` file | `Moq*` struct from that file | Same package (in-package tests) |
| Interface has NO mock file | Inline mock struct in test file | N/A |

Before creating an inline mock, check if a `*_mock.go` file exists in the interface's package:

```bash
ls cova/<package-path>/*_mock.go
```

## Test Naming (testdox style)

A test name must be **self-describing** — reading it alone should be enough to understand what the test verifies, without looking at the code.

Test names are sentences in CamelCase that read as documentation when spaces are inserted between words.
This is the [testdox](https://github.com/bitfield/gotestdox) convention — tools render them as readable sentences automatically.

### Format

Test names **must** use BDD-style phrasing: describe what is being done and what should happen.
Use "doing X should Y" as the base structure, and append "when Z" for conditional behavior.

- **General behavior:** `Test_DoingXShouldY` or `Test_DoingXShouldYWhenZ`
- **Function-specific:** `TestFunctionName_DoingXShouldY` or `TestFunctionName_DoingXShouldYWhenZ`

The first underscore separates the function/type name from the descriptive sentence.
For general tests not tied to a specific function, `Test_` acts as the prefix and the rest is the sentence.

### Examples

```go
func Test_ProcessingValidInputShouldReturnSuccess(t *testing.T)
func Test_LoadingEmptyConfigShouldFallBackToDefaults(t *testing.T)
func TestHandleInput_ReadingInputShouldCloseItAfterwards(t *testing.T)
func TestNewClient_CreatingClientShouldReturnErrorWhenConfigIsMissing(t *testing.T)
```

Bad — doesn't use BDD-style phrasing:

```go
func Test_LoadConfig(t *testing.T)
func Test_ValidInputReturnsSuccess(t *testing.T)
func TestNewClient_Error(t *testing.T)
```

### Table-driven subtest names

Subtest names must make sense when combined with the parent test name, since testdox concatenates them (e.g., `TestParse_ParsingInputShouldSucceed/WhenInputIsValidJSON`).
The parent test carries the "doing X should Y" phrasing; subtests provide the varying condition in CamelCase:

```go
func TestParse_ParsingInputShouldSucceed(t *testing.T) {
    tests := []struct {
        name  string
        input string
    }{
        {"WhenInputIsValidJSON", `{"key":"val"}`},
        {"WhenInputIsEmptyObject", `{}`},
    }
    ...
}
```

## Assertions

Use `testify/require` for all assertions. When expecting errors, match by keyword, not full message:

```go
// Good: checks for key error indicator
require.Error(t, err)
require.Contains(t, err.Error(), "not found")

// Bad: matches entire error message
require.EqualError(t, err, "file could not be found in the path /config/nonexistent.yaml")
```

## Unit Tests

Unit tests verify a single function or method in isolation.

- Use mocks to isolate the function being tested.
- Place unit tests in the same package as the code being tested.
- Each test verifies a single behavior.

```go
package mypackage

func TestProcess_ProcessingValidInputShouldReturnResult(t *testing.T) {
    // Arrange
    mock := &MoqDependency{
        DoWorkFunc: func(ctx context.Context, input string) (string, error) {
            return "result", nil
        },
    }
    svc := NewMyService(mock)

    // Act
    result, err := svc.Process(ctx, "input")

    // Assert
    require.NoError(t, err)
    require.Equal(t, "result", result)
}
```

## Table-Driven Tests

Use when testing multiple scenarios of the same function:

```go
func TestDetermineVerbosity_SettingFlagsShouldReturnCorrectLevel(t *testing.T) {
    tests := []struct {
        name     string
        verbose  bool
        extra    bool
        expected VerbosityLevel
    }{
        {"WhenUsingDefaults", false, false, VerbosityNormal},
        {"WhenSettingVerboseFlag", true, false, VerbosityVerbose},
        {"WhenSettingBothFlags", true, true, VerbosityExtraVerbose},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := determineVerbosity(tt.verbose, tt.extra)
            require.Equal(t, tt.expected, result)
        })
    }
}
```

## Integration Tests

Integration tests verify interaction between components, including OS-dependent interactions.
They may use a mix of real dependencies and mocks as appropriate.

- Allow opting out with `testing.Short()`.
- Place in the test package (e.g., `mypackage_test` for `mypackage` package).
- Follow testdox naming (same as unit tests).

```go
package mypackage_test

func TestProcess_ProcessingRequestShouldReturnResultWhenDependencyIsAvailable(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    svc := NewMyService(/* real dependencies */)
    result, err := svc.Process(ctx, "input")
    require.NoError(t, err)
    require.NotEmpty(t, result)
}
```

## Tech Stack

- `testify/require` — assertions (never `assert` for error checks)
- `mockery` with moq template — mock generation (see `.mockery.yml`)
