// Copyright (c) 2019 Klaus Post. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package s2

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"strings"
	"testing"

	"github.com/klauspost/compress/zip"
)

func testOptions(t testing.TB) map[string][]WriterOption {
	var testOptions = map[string][]WriterOption{
		"default": {},
		"better":  {WriterBetterCompression()},
		"best":    {WriterBestCompression()},
		"none":    {WriterUncompressed()},
	}

	x := make(map[string][]WriterOption)
	cloneAdd := func(org []WriterOption, add ...WriterOption) []WriterOption {
		y := make([]WriterOption, len(org)+len(add))
		copy(y, org)
		copy(y[len(org):], add)
		return y
	}
	for name, opt := range testOptions {
		x[name] = opt
		x[name+"-c1"] = cloneAdd(opt, WriterConcurrency(1))
	}
	testOptions = x
	x = make(map[string][]WriterOption)
	for name, opt := range testOptions {
		x[name] = opt
		if !testing.Short() {
			x[name+"-4k-win"] = cloneAdd(opt, WriterBlockSize(4<<10))
			x[name+"-4M-win"] = cloneAdd(opt, WriterBlockSize(4<<20))
		}
	}
	testOptions = x
	x = make(map[string][]WriterOption)
	for name, opt := range testOptions {
		x[name] = opt
		x[name+"-pad-min"] = cloneAdd(opt, WriterPadding(2), WriterPaddingSrc(zeroReader{}))
		if !testing.Short() {
			x[name+"-pad-8000"] = cloneAdd(opt, WriterPadding(8000), WriterPaddingSrc(zeroReader{}))
			x[name+"-pad-max"] = cloneAdd(opt, WriterPadding(4<<20), WriterPaddingSrc(zeroReader{}))
		}
	}
	testOptions = x
	return testOptions
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

func TestEncoderRegression(t *testing.T) {
	data, err := ioutil.ReadFile("testdata/enc_regressions.zip")
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	// Same as fuzz test...
	test := func(t *testing.T, data []byte) {
		if testing.Short() && len(data) > 10000 {
			t.SkipNow()
		}
		for name, opts := range testOptions(t) {
			t.Run(name, func(t *testing.T) {
				var buf bytes.Buffer
				dec := NewReader(nil)
				enc := NewWriter(&buf, opts...)

				comp := Encode(make([]byte, MaxEncodedLen(len(data))), data)
				decoded, err := Decode(nil, comp)
				if err != nil {
					t.Error(err)
					return
				}
				if !bytes.Equal(data, decoded) {
					t.Error("block decoder mismatch")
					return
				}
				if mel := MaxEncodedLen(len(data)); len(comp) > mel {
					t.Error(fmt.Errorf("MaxEncodedLen Exceed: input: %d, mel: %d, got %d", len(data), mel, len(comp)))
					return
				}
				comp = EncodeBetter(make([]byte, MaxEncodedLen(len(data))), data)
				decoded, err = Decode(nil, comp)
				if err != nil {
					t.Error(err)
					return
				}
				if !bytes.Equal(data, decoded) {
					t.Error("block decoder mismatch")
					return
				}
				if mel := MaxEncodedLen(len(data)); len(comp) > mel {
					t.Error(fmt.Errorf("MaxEncodedLen Exceed: input: %d, mel: %d, got %d", len(data), mel, len(comp)))
					return
				}

				// Test writer.
				n, err := enc.Write(data)
				if err != nil {
					t.Error(err)
					return
				}
				if n != len(data) {
					t.Error(fmt.Errorf("Write: Short write, want %d, got %d", len(data), n))
					return
				}
				err = enc.Close()
				if err != nil {
					t.Error(err)
					return
				}
				// Calling close twice should not affect anything.
				err = enc.Close()
				if err != nil {
					t.Error(err)
					return
				}
				comp = buf.Bytes()
				if enc.pad > 0 && len(comp)%enc.pad != 0 {
					t.Error(fmt.Errorf("wanted size to be mutiple of %d, got size %d with remainder %d", enc.pad, len(comp), len(comp)%enc.pad))
					return
				}
				dec.Reset(&buf)
				got, err := ioutil.ReadAll(dec)
				if err != nil {
					t.Error(err)
					return
				}
				if !bytes.Equal(data, got) {
					t.Error("block (reset) decoder mismatch")
					return
				}

				// Test Reset on both and use ReadFrom instead.
				buf.Reset()
				enc.Reset(&buf)
				n2, err := enc.ReadFrom(bytes.NewBuffer(data))
				if err != nil {
					t.Error(err)
					return
				}
				if n2 != int64(len(data)) {
					t.Error(fmt.Errorf("ReadFrom: Short read, want %d, got %d", len(data), n2))
					return
				}
				err = enc.Close()
				if err != nil {
					t.Error(err)
					return
				}
				if enc.pad > 0 && buf.Len()%enc.pad != 0 {
					t.Error(fmt.Errorf("wanted size to be mutiple of %d, got size %d with remainder %d", enc.pad, buf.Len(), buf.Len()%enc.pad))
					return
				}
				dec.Reset(&buf)
				got, err = ioutil.ReadAll(dec)
				if err != nil {
					t.Error(err)
					return
				}
				if !bytes.Equal(data, got) {
					t.Error("frame (reset) decoder mismatch")
					return
				}
			})
		}
	}
	for _, tt := range zr.File {
		if !strings.HasSuffix(t.Name(), "") {
			continue
		}
		t.Run(tt.Name, func(t *testing.T) {
			r, err := tt.Open()
			if err != nil {
				t.Error(err)
				return
			}
			b, err := ioutil.ReadAll(r)
			if err != nil {
				t.Error(err)
				return
			}
			test(t, b[:len(b):len(b)])
		})
	}
}

func TestWriterPadding(t *testing.T) {
	n := 100
	if testing.Short() {
		n = 5
	}
	rng := rand.New(rand.NewSource(0x1337))
	d := NewReader(nil)

	for i := 0; i < n; i++ {
		padding := (rng.Int() & 0xffff) + 1
		src := make([]byte, (rng.Int()&0xfffff)+1)
		for i := range src {
			src[i] = uint8(rng.Uint32()) & 3
		}
		var dst bytes.Buffer
		e := NewWriter(&dst, WriterPadding(padding))
		// Test the added padding is invisible.
		_, err := io.Copy(e, bytes.NewBuffer(src))
		if err != nil {
			t.Fatal(err)
		}
		err = e.Close()
		if err != nil {
			t.Fatal(err)
		}
		err = e.Close()
		if err != nil {
			t.Fatal(err)
		}

		if dst.Len()%padding != 0 {
			t.Fatalf("wanted size to be mutiple of %d, got size %d with remainder %d", padding, dst.Len(), dst.Len()%padding)
		}
		var got bytes.Buffer
		d.Reset(&dst)
		_, err = io.Copy(&got, d)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(src, got.Bytes()) {
			t.Fatal("output mismatch")
		}

		// Try after reset
		dst.Reset()
		e.Reset(&dst)
		_, err = io.Copy(e, bytes.NewBuffer(src))
		if err != nil {
			t.Fatal(err)
		}
		err = e.Close()
		if err != nil {
			t.Fatal(err)
		}
		if dst.Len()%padding != 0 {
			t.Fatalf("wanted size to be mutiple of %d, got size %d with remainder %d", padding, dst.Len(), dst.Len()%padding)
		}

		got.Reset()
		d.Reset(&dst)
		_, err = io.Copy(&got, d)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(src, got.Bytes()) {
			t.Fatal("output mismatch after reset")
		}
	}
}

func BenchmarkWriterRandom(b *testing.B) {
	rng := rand.New(rand.NewSource(1))
	// Make max window so we never get matches.
	data := make([]byte, 4<<20)
	for i := range data {
		data[i] = uint8(rng.Intn(256))
	}

	for name, opts := range testOptions(b) {
		w := NewWriter(ioutil.Discard, opts...)
		b.Run(name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			b.SetBytes(int64(len(data)))
			for i := 0; i < b.N; i++ {
				err := w.EncodeBuffer(data)
				if err != nil {
					b.Fatal(err)
				}
			}
			// Flush output
			w.Flush()
		})
		w.Close()
	}
}
