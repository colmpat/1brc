package main

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestMain(t *testing.T) {
	PFX := "test/samples/"
	cases := []string{
		"measurements-1",
		"measurements-10",
		"measurements-10000-unique-keys",
		"measurements-2",
		"measurements-20",
		"measurements-3",
		"measurements-boundaries",
		"measurements-complex-utf8",
		"measurements-dot",
		"measurements-rounding",
		"measurements-short",
		"measurements-shortest",
	}

	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			path := PFX + c
			// Save original stdout
			old := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Call the main function
			os.Args = []string{"main", path + ".txt"}
			main()

			// Restore the original stdout
			w.Close()
			os.Stdout = old

			// Read captured output
			var buf bytes.Buffer
			io.Copy(&buf, r)

			// Expected output
			expected, err := os.ReadFile(path + ".out")
			if err != nil {
				t.Fatalf("Error reading expected output: %s", err)
			}

			// Compare actual output with expected output
			actual := buf.String()
			exp := string(expected)
			if actual != exp {
				t.Fatalf("The output does not match the expected output. expected: %q got %q", exp, actual)
			}
		})
	}
}
