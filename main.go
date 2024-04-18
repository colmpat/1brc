package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
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

type Results map[string]Result

type Result struct {
	Min   float64
	Max   float64
	Sum   float64
	Count int
}

func parseFloat(b []byte, neg bool) float64 {
	//XX.X
	i := 0
	m := 4 // 20.3 == 4 but 8.2 == 3

	val := 0.0
	for i < m {
		if b[i] == '.' {
			if i == 1 {
				m--
			}
			i++
			continue
		}
		val *= 10
		val += float64(b[i] - 48)
		i++
	}

	if neg {
		val *= -0.1
	} else {
		val *= 0.1
	}

	return val
}

func printResults(results Results) {
	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	fmt.Print("{")
	for _, k := range keys[:len(keys)-1] {
		r := results[k]
		avg := r.Sum / float64(r.Count)
		fmt.Printf("%s=%.1f/%.1f/%.1f, ", k, r.Min, avg, r.Max)
	}
	k := keys[len(keys)-1]
	r := results[k]
	avg := r.Sum / float64(r.Count)
	fmt.Printf("%s=%.1f/%.1f/%.1f}\n", k, r.Min, avg, r.Max)
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
	strbuf := make([]rune, 64)
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
					si = 0

					t := float64(temp) / 10.0
					temp = 0
					neg = false
					if r, ok := results[station]; !ok {
						results[station] = Result{
							Min:   t,
							Max:   t,
							Sum:   t,
							Count: 1,
						}
					} else {
						r.Min = math.Min(r.Min, t)
						r.Max = math.Max(r.Max, t)
						r.Sum += t
						r.Count++
					}

					state = true
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
				r.Min = math.Min(r.Min, v.Min)
				r.Max = math.Max(r.Max, v.Min)
				r.Sum += v.Sum
				r.Count += v.Count
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

	defer timer("main")()
	filePath := os.Args[1]

	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	c := make(chan string, 100)
	r := make(chan Results, 100)

	go producer(f, c)

	workerstr := os.Getenv("WORKERS")
	workers, err := strconv.Atoi(workerstr)
	if err != nil {
		workers = 17
	}

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
