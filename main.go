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
func producer(f *os.File, c chan<- []byte) {
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

		res := make([]byte, n)
		copy(res, rdBuf)
		body := res[start:end]

		leading := res[0:start]
		trailing := res[end:n]
		garbage = append(garbage, leading...)
		garbage = append(garbage, trailing...)

		if len(body) > 0 {
			c <- body
		}
	}
	if len(garbage) > 0 {
		c <- garbage
	}
}

func lenMonitor(c <-chan string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Fprintf(os.Stderr, "buffer length: %d\n", len(c))
	}
}

func consumer(c <-chan []byte, r chan<- Results) {
	results := make(Results)

	// possible states: false=parse-station, true=parse-float
	state := true
	si := -1
	se := -1
	temp := 0
	neg := false
	for chunk := range c {
		for i, char := range chunk {
			if state { // parse-station
				if char == ';' {
					state = false // switch to parse-float
					se = i        // save string-end
					continue
				} else if char == '\n' {
					continue
				}

				if si < 0 {
					si = i // save string-start
				}
			} else { // parse-float
				if char == '\n' { // update-dict
					state = true
					neg = false

					if r, ok := results[string(chunk[si:se])]; ok {
						if temp < r[0] {
							r[0] = temp
						}
						if temp > r[1] {
							r[1] = temp
						}
						r[2] += temp
						r[3]++
						temp = 0
						si = -1
						se = -1
						continue
					}

					results[string(chunk[si:se])] = &[4]int{
						temp,
						temp,
						temp,
						1,
					}

					temp = 0
					si = -1
					se = -1
					continue
				} else if char == '.' {
					continue
				} else if char == '-' {
					neg = true
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
	for resultMap := range r {
		for station, values := range resultMap {
			if existingResult, ok := res[station]; !ok {
				res[station] = values
			} else {
				if values[0] < existingResult[0] {
					existingResult[0] = values[0]
				}
				if values[1] > existingResult[1] {
					existingResult[1] = values[1]
				}
				existingResult[2] += values[2]
				existingResult[3] += values[3]
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

	c := make(chan []byte, 100)
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
