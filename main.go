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
	"sync"
	"time"
)

func timer(name string) func() {
	start := time.Now()
	return func() {
		fmt.Fprintf(os.Stderr, "\n%s took %v\n", name, time.Since(start))
	}
}

// min, max, sum, count
type Results map[string][4]int

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

	garbage := make([]byte, 0, 4096)
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

		body := rdBuf[start:end]

		leading := rdBuf[0:start]
		trailing := rdBuf[end:n]
		garb := make([]byte, start+n-end)
		copy(garb, append(leading, trailing...))

		garbage = append(garbage, garb...)

		if len(body) > 0 {
			c <- string(body)
		}
	}
	if len(garbage) > 0 {
		c <- string(garbage)
	}
}

func lenMonitor(c <-chan string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Fprintf(os.Stderr, "buffer length: %d\n", len(c))
	}
}

func consumer(c <-chan string, r chan<- Results) {
	results := make(Results)

	// possible states: false=parse-station, true=parse-float
	state := true
	strbuf := make([]rune, 128)
	si := 0
	temp := 0
	neg := false
	for chunk := range c {
		for _, char := range chunk {
			if state { // parse-station
				if char == ';' {
					state = false
					continue
				}
				strbuf[si] = char
				si++
			} else { // parse-float
				if char == '\n' { // update-dict
					station := string(strbuf[:si])

					if r, ok := results[station]; !ok {
						results[station] = [4]int{
							temp,
							temp,
							temp,
							1,
						}
					} else {
						if temp < r[0] {
							r[0] = temp
						}
						if temp > r[1] {
							r[1] = temp
						}
						r[2] += temp
						r[3]++
					}

					state = true
					si = 0
					temp = 0
					neg = false
					continue
				} else if char == '.' {
					continue
				}

				temp *= 10
				if neg {
					temp -= int(char - '0')
				} else {
					temp += int(char - '0')
				}
			}
		}
	}

	r <- results
}

func merge(r <-chan Results) Results {
	res := make(Results)
	for result := range r {
		for k, v := range result {
			if r, ok := res[k]; !ok {
				res[k] = v
			} else {
				if v[0] < r[0] {
					r[0] = v[0]
				}
				if v[1] > r[1] {
					r[1] = v[1]
				}
				r[2] += v[2]
				r[3] += v[3]
			}
		}
	}
	return res
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

	bufflenStr := os.Getenv("BUFFLEN")
	buflen, err := strconv.Atoi(bufflenStr)
	if err != nil {
		buflen = 4096 * 4096
	}
	workerstr := os.Getenv("WORKERS")
	workers, err := strconv.Atoi(workerstr)
	if err != nil {
		workers = 17
	}

	defer timer(fmt.Sprintf("main with buflen=%d and workers=%d", buflen, workers))()
	filePath := os.Args[1]

	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	c := make(chan string, 100)
	r := make(chan Results, 100)

	go producer(f, c)

	wg := sync.WaitGroup{}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			consumer(c, r)
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(r)
	}()

	results := merge(r)

	printResults(results)
}
