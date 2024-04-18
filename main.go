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

		leading := rdBuf[0:start]
		trailing := rdBuf[end:n]
		body := rdBuf[start:end]
		garbage = append(garbage, leading...)
		garbage = append(garbage, trailing...)

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

func consumer(c <-chan string) Results {
	results := make(Results)

	// possible states: false=parse-station, true=parse-float
	state := true
	stationBuf := make([]byte, 128)
	si := 0
	temp := byte(0)
	neg := false
	for chunk := range c {
		//fmt.Fprintln(os.Stderr, chunk)
		for i := range chunk {
			if state { // parse-station
				char := chunk[i]

				//fmt.Fprintf(os.Stderr, "chunk: %q\n", char)

				if char == ';' {
					state = false
					continue
				}
				stationBuf[si] = char
				si++
			} else { // parse-float
				char := chunk[i]
				if char == '\n' { // update-dict
					station := string(stationBuf[:si])
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
				} else if char == '.' {
					continue
				}

				temp *= 10
				if neg {
					temp -= char - 48
				} else {
					temp += char - 48
				}
			}
		}
	}

	/*
				split := 0
				for line[split] != ';' {
					split++
				}

				split++
				neg := false
				if line[split] == '-' {
					split++
					neg = true
				}
				end := split + 4 // XX.X

				temp := parseFloat(line[split:end], neg)

				if r, ok := results[station]; !ok {
					results[station] = Result{
						Min:   temp,
						Max:   temp,
						Sum:   temp,
						Count: 1,
					}
				} else {
					r.Min = math.Min(r.Min, temp)
					r.Max = math.Max(r.Max, temp)
					r.Sum += temp
					r.Count++
				}
			}
		}
	*/
	return results
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

	go producer(f, c)
	results := consumer(c)
	printResults(results)
}
