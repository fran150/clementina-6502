# Computers Package - Refactored Architecture

This package has been completely refactored to eliminate architectural issues and provide a clean, maintainable design.

## New Architecture Overview

### Core Components

#### 1. **ComputerSystem** - Main Orchestrator
- Composes all components using dependency injection
- Manages lifecycle (start/stop/pause/resume)
- Delegates to specialized components for specific concerns

#### 2. **Core Interfaces**
- `Emulator`: Pure emulation logic (`Tick` method)
- `Renderer`: Display rendering (`Draw` method)  
- `ComputerCore`: Combines emulation and rendering
- `ComputerController`: Lifecycle management
- `SpeedController`: Speed management

#### 3. **Specialized Components**
- `DefaultSpeedController`: Manages emulation speed with non-linear scaling
- `StateManager`: Handles computer state (pause/step/reset)
- `EmulationLoop`: Manages timing and execution loops
- `Console`: Pure business logic for console management
- `TViewConsole`: UI-specific console implementation

### Console Architecture

#### 4. **Console Components**
- `WindowManager`: Manages window lifecycle
- `NavigationManager`: Handles window navigation and history
- `WindowControllers`: Type-safe access to specific window types
- `TViewFramework`: UI framework abstraction

## Benefits Achieved

### ✅ **Eliminated Circular Dependencies**
- No more mutual dependencies between `EmulationLoop` and `Computer`
- Clean dependency graph with clear ownership

### ✅ **Single Responsibility Principle**
- Each component has one clear purpose
- Easy to understand and modify individual components

### ✅ **Interface Segregation**
- Focused interfaces instead of monolithic ones
- Components depend only on what they need

### ✅ **Dependency Inversion**
- High-level modules don't depend on low-level modules
- Both depend on abstractions (interfaces)

### ✅ **Type Safety**
- Compile-time guarantees instead of runtime casting
- Type-safe window controllers eliminate runtime errors

### ✅ **Better Testability**
- All components can be tested in isolation
- Easy to create mock implementations
- Dependencies are injected through interfaces

## Usage Examples

### Creating a Computer System
```go
// Create core computer implementation
computer := &MyComputer{} // implements ComputerCore

// Create speed controller
speedController := NewSpeedController(1.0) // 1 MHz

// Create configuration
config := &EmulationLoopConfig{DisplayFps: 60}

// Compose the system
system := NewComputerSystem(computer, speedController, config)

// Start emulation
context, err := system.Start()
```

### Creating a Console
```go
// Create console with new architecture
console := NewTViewConsole()

// Add windows
console.AddWindow("memory", memoryWindow)
console.AddWindow("cpu", cpuWindow)

// Type-safe access
if controller := console.GetConsole().GetMemoryController("memory"); controller != nil {
    controller.ScrollUp(5) // Compile-time safe
}
```

## Computer Implementations

### Ben Eater Computer (`pkg/computers/beneater/`)
- Implements Ben Eater's 6502 breadboard computer
- Includes LCD display, VIA, ACIA, and serial communication
- Uses the new architecture with `ComputerSystem`

### Clementina Computer (`pkg/computers/clementina/`)
- Implements the Clementina 6502 computer design
- Features extended memory banking and advanced addressing
- Also refactored to use the new architecture

## Migration Notes

The old `BaseComputer` and `BaseConsole` classes have been removed. All implementations now use:

- `ComputerSystem` instead of `BaseComputer`
- `TViewConsole` instead of `BaseConsole`
- Focused interfaces instead of monolithic ones

This provides a much cleaner, more maintainable architecture that follows SOLID principles and eliminates the circular dependencies and tight coupling of the previous design.