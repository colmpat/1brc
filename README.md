# 1brc

## Devlog

#### v1.1
This time, I play with the buffer size of the channel. By sending data in chunks, we can reduce the amount of
time we spend scheduling and locking. Running with different buffer sizes, we can see that the time it takes
is as follows:
     
| Buffer Size | Time (s) |
|-------------|----------|
512 | 1m41.843237119s
1024 | 1m39.829944599s
2048 | 1m38.840931314s
4096 | 1m37.272231156s
8192 | 1m36.567135776s
16384 | 1m38.780832584s

from here, we'll go with 8192 as our buffer size. Now looking at pprof, we can see that we are still
spending lots of time waiting for channels to be ready in `runtime.recv.goready.func1`

Adding a monitor to the main channel length it seems that it is always empty, meaning that we are processing
data as fast as we can. This means that we should focus in on producing data faster from the file read path!


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
