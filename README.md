# 1brc

## Devlog

#### v2.5
Because our state machine marches through the string, that got me thinking that we could use a trie.
We already march through so update a pointer along the way will be WAY faster than parsing to a string 
then hashing for a map lookup (i think).

Let's do it!

**Runtime**: 1m8.107846359s

Slower than the map lookup but I think this is because I'm not using the trie correctly. I'm going to
try to improve the `findChild` function to be more efficient. It works but I think it could be faster!
I also came across some literature that says we can compress the trie to make it faster. I'm going to
try that next as well.

#### v2.4
I was reading an article about faster map lookups and then saw another stach overflow post that made the 
same claim so I added this weird-unintuitive lookup pattern to the map access. This actually made the
lookup way faster... literally have no idea why this works but it does.
 
**Runtime**: 9.795203221s I think this is a 200x speedup from the original?

#### v2.3
Changing the producer to just copy the buffer and send it to the consumer simplified the code a lot.
It allows our state machine to be as simple as we initially planned and just iterate over the buffer and
only process on big inflection points. We got to remove the rune business and just copy a slice of the 
buffer to a string.

**Runtime**: 15.202590219s

#### v2.2
This version takes the parsing logic and leans into a state-machine-approach. This avoids having to read
bytes then iterate again over them -> it will be one pass over the data. This will also let us yank out
any need for a scanner.

This gets us the same runtime which is good and bad. On the one hand, no improvement. On the other hand,
we can optimize ours which we wouldn't be able to if we were use other libs.

Switching to use a string builder instead of the string buffer was noticibly slower. But looking at their
source code I saw how the built the string and it was the same as I did except with the use of the `unsafe`
package for pointer arithmitic. This was WAY faster but didn't work so I'm gonna keep this in the back of
my head to see if we can get this working later.

Notable function usage to look into:
27.8% runtime.mapaccess2_faststr (map lookup)
26.9% runtime.slicebytetostring

Running main with buflen=16777216 and workers=17 took 24.709733555s

Looking at the flame graph, we see that the `slicetorune` function takes up 50% of the time now. This is
annoying becasue I'm only using this to fix the string output format. Maybe I'll try the string builder
again next.

#### v2.1
Looking at the flamegraph for this, the majority of our time is spent processing. The consumer has three
main culprits: map access, scanner.Text(), and parseFloat.

Changing just the float parsing and the scanner.Text() call we reduce the runtime to:

1m48.595598283s


> Note: looking at the parsing logic of the consumer, I think a state-machine could be really fast here.
Something like the following states: `READING_STATION`, `SEMICOLON`, `PARSING_TEMP`, `NEWLINE`. 
Automota theory fast?

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
