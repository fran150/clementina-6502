# clementina-6502

In construction....

```
       /\      
      /  \     
     / _o \    
    / <(\  \   
   /   />`A \  
  '----------`
  ```

### Port emulation

In order to test the emulator I use:

```
socat -d -d pty,raw,echo=0 pty,raw,echo=0
```

to create pseudo interconnected ports. Ben's computer operates with 8-N-1, 19200 bps

### Profiling

For profiling the CPU I'm using

```
go test -benchmem -run=^$ -bench ^BenchmarkProcessor$ github.com/fran150/clementina6502/tests -cpuprofile clementina6502.prof
```

and then:

```
go tool pprof -http :8080  clementina6502.prof
```