# Contributing to Clementina 6502

Thank you for your interest in contributing to Clementina 6502! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Documentation Guidelines](#documentation-guidelines)
- [Testing Guidelines](#testing-guidelines)
- [Pull Request Process](#pull-request-process)

## Code of Conduct

Please be respectful and considerate of others when contributing to this project. We aim to foster an inclusive and welcoming community.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR-USERNAME/clementina-6502.git`
3. Add the upstream repository: `git remote add upstream https://github.com/fran150/clementina-6502.git`
4. Create a branch for your changes: `git checkout -b feature/your-feature-name`

## Development Workflow

1. Make your changes
2. Run tests: `make test`
3. Check code coverage: `make coverage`
4. Ensure your code follows Go best practices

## Documentation Guidelines

We follow standard Go documentation practices:

### General Principles

1. Every exported (capitalized) symbol must have a documentation comment
2. Package-level documentation should provide an overview
3. Examples should demonstrate common use cases
4. Use complete sentences that start with the symbol name

### Package Documentation

Each package should have a package comment:

```go
// Package emulator provides a cycle-accurate 6502 emulator.
//
// It emulates the complete Ben Eater's 6502 computer with all peripherals
// including the 65C02S CPU, 65C22S VIA, 65C51N ACIA, and HD44780U LCD.
package emulator
```

### Types, Functions, and Methods

Document all exported symbols:

```go
// CPU represents a 65C02S processor with all registers and flags.
// It implements the complete instruction set and handles interrupts.
type CPU struct {
    // Fields...
}

// Reset initializes the CPU to its power-on state.
// It sets the program counter to the reset vector and clears registers.
func (c *CPU) Reset() {
    // Implementation...
}
```

### Examples

Include examples to demonstrate usage:

```go
// Example_basic demonstrates basic usage of the emulator.
func Example_basic() {
    // Create a new emulator
    emu := emulator.New()
    
    // Load a ROM file
    err := emu.LoadROM("examples/hello.bin")
    if err != nil {
        fmt.Println("Error loading ROM:", err)
        return
    }
    
    // Run the emulator
    emu.Run(emulator.RunOptions{
        ClockSpeed: 1000000, // 1 MHz
    })
    
    // Output:
    // Emulator running at 1 MHz
}
```

### Documentation Format

- Use full sentences with proper punctuation
- Start with the name of the thing being documented
- Explain what it does, not how it does it
- Include any important constraints or side effects
- Document parameters and return values for functions

## Testing Guidelines

1. Write tests for all new functionality
2. Aim for high code coverage (at least 80%)
3. Include both unit tests and integration tests where appropriate
4. Use table-driven tests for testing multiple scenarios

## Pull Request Process

1. Update the README.md with details of changes if applicable
2. Update the documentation for any changed functionality
3. Ensure all tests pass and code coverage is maintained or improved
4. Submit your pull request with a clear description of the changes
5. Address any feedback from code reviews

Thank you for contributing to Clementina 6502!
