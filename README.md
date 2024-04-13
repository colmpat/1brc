# 1brc

## Devlog


#### v1.0
Made a producer and consumer model with a channel. Not as good as I was hoping for, but it's a start.
Looking at prof, we spend more time doing scheduling/locking than we do actually processing the data.
That means we should increase workers? I'm not sure. I'll try that next.

**Runtime**: 5m6.327792616s

#### v0

Setup the project and got something basic working. Added profiling and a timer mechanism to measure our
approach from here on out. To profile, I'm using golang's built-in profiling tool called `pprof`. This
dumps a profile file called `cpu.prof` which you can visualize with `pprof -http=localhost:8080 cpu.prof`.
This initial version is meant to serve as a baseline for future optimizations!

**Runtime**: 3m41.311502687s
