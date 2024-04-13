# 1brc

## Devlog

#### v0

Setup the project and got something basic working. Added profiling and a timer mechanism to measure our
approach from here on out. To profile, I'm using golang's built-in profiling tool called `pprof`. This
dumps a profile file called `cpu.prof` which you can visualize with `pprof -http=localhost:8080 cpu.prof`.
This initial version is meant to serve as a baseline for future optimizations!

**Runtime**: 3m41.311502687s
