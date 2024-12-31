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

### Profiling

For profiling the CPU I'm using

```
go test -benchmem -run=^$ -bench ^BenchmarkProcessor$ github.com/fran150/clementina6502/tests -cpuprofile clementina6502.prof
```

and then:

```
go tool pprof -http :8080  clementina6502.prof
```