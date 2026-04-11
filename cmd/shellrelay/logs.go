package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

func cmdLogs(args []string) {
	fs := flag.NewFlagSet("logs", flag.ExitOnError)
	nLines := fs.Int("n", 50, "Number of lines to show (0 = all)")
	follow := fs.Bool("f", false, "Follow log output (like tail -f)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: shellrelay logs [-n <lines>] [-f]\n\nShow the daemon log file.\n\nOptions:\n")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	path, err := logFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "logs: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No log file found. Start the daemon with: shellrelay start")
			return
		}
		fmt.Fprintf(os.Stderr, "logs: open: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if *nLines > 0 && !*follow {
		printLastN(f, *nLines)
	} else {
		// Print all (or tail then follow)
		if *nLines > 0 {
			printLastN(f, *nLines)
			// Seek to end for follow
			f.Seek(0, io.SeekEnd)
		} else {
			io.Copy(os.Stdout, f)
		}
		if *follow {
			followFile(f)
		}
	}
}

// printLastN prints the last n lines of a file using a proper ring buffer.
func printLastN(f *os.File, n int) {
	scanner := bufio.NewScanner(f)
	ring := make([]string, n)
	pos := 0
	total := 0
	for scanner.Scan() {
		ring[pos%n] = scanner.Text()
		pos++
		total++
	}
	// Determine how many lines to print and starting index
	count := total
	if count > n {
		count = n
	}
	start := (pos - count) % n
	if start < 0 {
		start += n
	}
	for i := 0; i < count; i++ {
		fmt.Println(ring[(start+i)%n])
	}
}

// followFile reads new content appended to the file, sleeping between polls.
func followFile(f *os.File) {
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			fmt.Print(line)
		}
		if err != nil {
			if err == io.EOF {
				time.Sleep(250 * time.Millisecond)
				continue
			}
			return
		}
	}
}
