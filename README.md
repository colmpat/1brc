# 1brc

## Devlog

#### v2.1
Looking at the flamegraph for this, the majority of our time is spent processing. The consumer has three
main culprits: map access, scanner.Text(), and parseFloat.

#### v2.0
Now that I have some quick file processing I added one layer on top of this: boundary awareness. When we
slurp up this large buffer, we often read into the middle of the line. To solve this, I make a garage bin
in the producer that holds these endcaps. When reading to this chunk boundary, I walk backwards from the 
end until I hit a newline and forwards from the start until I hit a newline and add these segments to the 
garbage bin. The piece inbetween is added to the channel and the garbage bin is added to the channel once
at the very end.

This version also brings tests in from the challenge repo which helped to validate this garbage bin logic.

**Runtime**: 2m5.236455412s

#### v1.2
I want to drill into the fastest way to read the file so this will be
based on no processing logic. If we can find the aboslute fastest way to read the file, we can
then add processing logic on top of that. I'm going to try a few different methods to read the file

| Buffer Size | Scanner Time (s) | f.Read() Time (s) |
|-------------|----------| ----------|
1024 | 1m0.858337389s | 19.097867371s | 
2048 | 47.164890247s | 13.358809107s | 
4096 | 41.066632658s | 9.070102125s | 
8192 | 36.921121763s | 6.123235963s | 
16384 | 34.536465545s | 5.688912133s | 
32768 | 33.757824443s | 5.368416801s | 
65536 | 33.094731985s | 5.035294434s | 
131072 | 32.489679904s | 5.246261899s | 
262144 | 31.971283174s | 5.258327858s | 
524288 | 32.064310861s | 4.796567423s | 
1048576 | ~ | 4.691419723s | 
2097152 | ~ | 4.718754142s | 
4194304 | ~ | 4.742348154s | 
8388608 | ~ | 4.732217901s
16777216 | ~ | 4.778489242s
33554432 | ~ | 4.80144076s

For now, I want to get something working in the faster range of outputs aboce and optimize on this buffer speed a little later.


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
