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

	results := make(Results)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
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

	printResults(results)
}
