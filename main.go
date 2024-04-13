package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"runtime/pprof"
	"slices"
	"strconv"
	"strings"
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

func parseFloat(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	return f
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
	fmt.Printf("%s=%.1f/%.1f/%.1f}", k, r.Min, avg, r.Max)
}

func producer(c chan<- []string) {
	defer close(c)

	buflenstr := os.Getenv("BUFFLEN")
	if buflenstr == "" {
		buflenstr = "8192"
	}
	buflen, err := strconv.Atoi(buflenstr)
	if err != nil {
		panic(err)
	}

	filePath := "data/measurements.txt"

	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lines := make([]string, 0, buflen)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) == cap(lines) {
			c <- lines
			lines = make([]string, 0, buflen)
		}
	}
	if len(lines) > 0 {
		c <- lines
	}
}

func lenMonitor(c <-chan []string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Fprintf(os.Stderr, "len(c)=%d\n", len(c))
		case _, ok := <-c:
			if !ok {
				return
			}
		}
	}
}

func consumer(c <-chan []string) Results {
	results := make(Results)
	for lines := range c {
		for _, line := range lines {
			station, tempStr, _ := strings.Cut(line, ";")
			temp := parseFloat(tempStr)

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
	filePath := "data/measurements.txt"

	f, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	c := make(chan []string, 128)
	go lenMonitor(c)
	go producer(c)
	results := consumer(c)

	printResults(results)
}
