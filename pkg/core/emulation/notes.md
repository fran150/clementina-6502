# Performance Measurements
Performance measurements from `console.Tick()` call in `emulator.Tick()` method:
- **7.29 MHz**: Upper limit without calling this function
- **5.9 MHz**: When commenting ticker.Tick call in console.Tick method
- **5.5 MHz**: When commenting only ticker.Tick contents but leaving the call (0.4 MHz overhead from call/interface)
- **2.9 MHz**: With window manager returning copy of maps
- **3.6 MHz**: Using Go's new enumerator functions
- **4.1 MHz**: Returning function for each iteration loop
- **4.3 MHz**: Returning map reference directly (allows direct alteration)
