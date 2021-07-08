package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/klauspost/compress/s2"
	"github.com/klauspost/compress/s2/cmd/internal/readahead"
)

var (
	safe   = flag.Bool("safe", false, "Do not overwrite output files")
	verify = flag.Bool("verify", false, "Verify files, but do not write output")
	stdout = flag.Bool("c", false, "Write all output to stdout. Multiple input files will be concatenated")
	remove = flag.Bool("rm", false, "Delete source file(s) after successful decompression")
	quiet  = flag.Bool("q", false, "Don't write any output to terminal, except errors")
	bench  = flag.Int("bench", 0, "Run benchmark n times. No output will be written")
	help   = flag.Bool("help", false, "Display help")

	version = "(dev)"
	date    = "(unknown)"
)

func main() {
	flag.Parse()
	r := s2.NewReader(nil)

	// No args, use stdin/stdout
	args := flag.Args()
	if len(args) == 0 || *help {
		_, _ = fmt.Fprintf(os.Stderr, "s2 decompress v%v, built at %v.\n\n", version, date)
		_, _ = fmt.Fprintf(os.Stderr, "Copyright (c) 2011 The Snappy-Go Authors. All rights reserved.\n"+
			"Copyright (c) 2019 Klaus Post. All rights reserved.\n\n")
		_, _ = fmt.Fprintln(os.Stderr, `Usage: s2d [options] file1 file2

Decompresses all files supplied as input. Input files must end with '.s2' or '.snappy'.
Output file names have the extension removed. By default output files will be overwritten.
Use - as the only file name to read from stdin and write to stdout.

Wildcards are accepted: testdir/*.txt will compress all files in testdir ending with .txt
Directories can be wildcards as well. testdir/*/*.txt will match testdir/subdir/b.txt

File names beginning with 'http://' and 'https://' will be downloaded and decompressed.
Extensions on downloaded files are ignored. Only http response code 200 is accepted.

Options:`)
		flag.PrintDefaults()
		os.Exit(0)
	}
	if len(args) == 1 && args[0] == "-" {
		r.Reset(os.Stdin)
		if !*verify {
			_, err := io.Copy(os.Stdout, r)
			exitErr(err)
		} else {
			_, err := io.Copy(ioutil.Discard, r)
			exitErr(err)
		}
		return
	}
	var files []string

	for _, pattern := range args {
		if isHTTP(pattern) {
			files = append(files, pattern)
			continue
		}

		found, err := filepath.Glob(pattern)
		exitErr(err)
		if len(found) == 0 {
			exitErr(fmt.Errorf("unable to find file %v", pattern))
		}
		files = append(files, found...)
	}

	*quiet = *quiet || *stdout

	if *bench > 0 {
		debug.SetGCPercent(10)
		for _, filename := range files {
			switch {
			case strings.HasSuffix(filename, ".s2"):
			case strings.HasSuffix(filename, ".snappy"):
			default:
				if !isHTTP(filename) {
					fmt.Println("Skipping", filename)
					continue
				}
			}

			func() {
				if !*quiet {
					fmt.Print("Reading ", filename, "...")
				}
				// Input file.
				file, size, _ := openFile(filename)
				b := make([]byte, size)
				_, err := io.ReadFull(file, b)
				exitErr(err)
				file.Close()

				for i := 0; i < *bench; i++ {
					if !*quiet {
						fmt.Print("\nDecompressing...")
					}
					r.Reset(bytes.NewBuffer(b))
					start := time.Now()
					output, err := io.Copy(ioutil.Discard, r)
					exitErr(err)
					if !*quiet {
						elapsed := time.Since(start)
						ms := elapsed.Round(time.Millisecond)
						mbPerSec := (float64(output) / (1024 * 1024)) / (float64(elapsed) / (float64(time.Second)))
						pct := float64(output) * 100 / float64(len(b))
						fmt.Printf(" %d -> %d [%.02f%%]; %v, %.01fMB/s", len(b), output, pct, ms, mbPerSec)
					}
				}
				fmt.Println("")
			}()
		}
		os.Exit(0)
	}

	for _, filename := range files {
		dstFilename := cleanFileName(filename)
		switch {
		case strings.HasSuffix(filename, ".s2"):
			dstFilename = strings.TrimSuffix(dstFilename, ".s2")
		case strings.HasSuffix(filename, ".snappy"):
			dstFilename = strings.TrimSuffix(dstFilename, ".snappy")
		default:
			if !isHTTP(filename) {
				fmt.Println("Skipping", filename)
				continue
			}
		}
		if *verify {
			dstFilename = "(verify)"
		}

		func() {
			var closeOnce sync.Once
			if !*quiet {
				fmt.Print("Decompressing ", filename, " -> ", dstFilename)
			}
			// Input file.
			file, _, mode := openFile(filename)
			defer closeOnce.Do(func() { file.Close() })
			rc := rCounter{in: file}
			src, err := readahead.NewReaderSize(&rc, 2, 4<<20)
			exitErr(err)
			defer src.Close()
			if *safe {
				_, err := os.Stat(dstFilename)
				if !os.IsNotExist(err) {
					exitErr(errors.New("destination files exists"))
				}
			}
			var out io.Writer
			switch {
			case *verify:
				out = ioutil.Discard
			case *stdout:
				out = os.Stdout
			default:
				dstFile, err := os.OpenFile(dstFilename, os.O_CREATE|os.O_WRONLY, mode)
				exitErr(err)
				defer dstFile.Close()
				bw := bufio.NewWriterSize(dstFile, 4<<20)
				defer bw.Flush()
				out = bw
			}
			r.Reset(src)
			start := time.Now()
			output, err := io.Copy(out, r)
			exitErr(err)
			if !*quiet {
				elapsed := time.Since(start)
				mbPerSec := (float64(output) / (1024 * 1024)) / (float64(elapsed) / (float64(time.Second)))
				pct := float64(output) * 100 / float64(rc.n)
				fmt.Printf(" %d -> %d [%.02f%%]; %.01fMB/s\n", rc.n, output, pct, mbPerSec)
			}
			if *remove && !*verify {
				closeOnce.Do(func() {
					file.Close()
					if !*quiet {
						fmt.Println("Removing", filename)
					}
					err := os.Remove(filename)
					exitErr(err)
				})
			}
		}()
	}
}

func openFile(name string) (rc io.ReadCloser, size int64, mode os.FileMode) {
	if isHTTP(name) {
		resp, err := http.Get(name)
		exitErr(err)
		if resp.StatusCode != http.StatusOK {
			exitErr(fmt.Errorf("unexpected response status code %v, want 200 OK", resp.Status))
		}
		return resp.Body, resp.ContentLength, os.ModePerm
	}
	file, err := os.Open(name)
	exitErr(err)
	st, err := file.Stat()
	exitErr(err)
	return file, st.Size(), st.Mode()
}

func cleanFileName(s string) string {
	if isHTTP(s) {
		s = strings.TrimPrefix(s, "http://")
		s = strings.TrimPrefix(s, "https://")
		s = strings.Map(func(r rune) rune {
			switch r {
			case '\\', '/', '*', '?', ':', '|', '<', '>', '~':
				return '_'
			}
			if r < 20 {
				return '_'
			}
			return r
		}, s)
	}
	return s
}

func isHTTP(name string) bool {
	return strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://")
}

func exitErr(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nERROR:", err.Error())
		os.Exit(2)
	}
}

type rCounter struct {
	n  int
	in io.Reader
}

func (w *rCounter) Read(p []byte) (n int, err error) {
	n, err = w.in.Read(p)
	w.n += n
	return n, err

}
