package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/colmpat/1brc/pkg/trie"
)

const (
	BUFFLEN = 4096 * 4096
	WORKERS = 17
)

func timer(name string) func() {
	start := time.Now()
	return func() {
		fmt.Fprintf(os.Stderr, "\n%s took %v\n", name, time.Since(start))
	}
}

// min, max, sum, count
type Results map[string]*[4]int

func printResults(results Results) {
	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	fmt.Print("{")
	for _, k := range keys[:len(keys)-1] {
		r := results[k]
		min := float64(r[0]) / 10.0
		max := float64(r[1]) / 10.0
		avg := float64(r[2]) / float64(r[3]) / 10.0
		fmt.Printf("%s=%.1f/%.1f/%.1f, ", k, min, avg, max)
	}
	k := keys[len(keys)-1]
	r := results[k]
	min := float64(r[0]) / 10.0
	max := float64(r[1]) / 10.0
	avg := float64(r[2]) / float64(r[3]) / 10.0
	fmt.Printf("%s=%.1f/%.1f/%.1f}\n", k, min, avg, max)
}

// reads the entire file into memory
func producer(f *os.File, c chan<- string) {
	defer close(c)

	bufflenStr := os.Getenv("BUFFLEN")
	buflen, err := strconv.Atoi(bufflenStr)
	if err != nil {
		buflen = 4096 * 4096
	}

	rdBuf := make([]byte, buflen)

	garbage := strings.Builder{}
	garbage.Grow(buflen)
	for {
		n, err := f.Read(rdBuf)
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			panic(err)
		}

		start := 1
		end := n
		for rdBuf[start-1] != '\n' {
			start++
		}
		for rdBuf[end-1] != '\n' {
			end--
		}

		bsb := strings.Builder{}
		bsb.Write(rdBuf[start:end])

		garbage.Write(rdBuf[0:start])
		garbage.Write(rdBuf[end:n])

		if bsb.Len() > 0 {
			c <- bsb.String()
		}
	}
	if garbage.Len() > 0 {
		c <- garbage.String()
	}
}

func lenMonitor(c <-chan string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Fprintf(os.Stderr, "buffer length: %d\n", len(c))
	}
}

func trieParser(cin <-chan string, cout chan<- *trie.Trie) {
	tr := trie.NewTrie()
	var np = tr.Root

	// possible states: false=parse-station, true=parse-float
	state := true
	temp := 0
	neg := false
	for chunk := range cin {
		for _, r := range chunk {
			if state { // parse-station
				if r == ';' {
					state = false // switch to parse-float
					continue
				} else if r == '\n' {
					continue
				}
				np = np.GetOrInsertChild(r)
			} else { // parse-float
				if r == '\n' { // update-dict
					np.Update(temp)

					np = tr.Root
					state = true
					neg = false
					temp = 0
					continue
				} else if r == '.' {
					continue
				} else if r == '-' {
					neg = true
					continue
				}

				temp *= 10
				if neg {
					temp -= int(r - '0')
				} else {
					temp += int(r - '0')
				}
			}
		}
	}

	cout <- tr
}

func main() {
	pf, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	defer pf.Close()
	if err := pprof.StartCPUProfile(pf); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
	defer pprof.StopCPUProfile()

	defer timer(fmt.Sprintf("main with buflen=%d and workers=%d", BUFFLEN, WORKERS))()
	filePath := os.Args[1]

	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	chunkchan := make(chan string, WORKERS)
	triechan := make(chan *trie.Trie, WORKERS)

	go producer(f, chunkchan)

	wg := sync.WaitGroup{}
	wg.Add(WORKERS)
	for i := 0; i < WORKERS; i++ {
		go func() {
			trieParser(chunkchan, triechan)
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(triechan)
	}()

	t := trie.NewTrie()
	for tr := range triechan {
		t.Merge(tr)
	}
	t.Write(os.Stdout)
}
