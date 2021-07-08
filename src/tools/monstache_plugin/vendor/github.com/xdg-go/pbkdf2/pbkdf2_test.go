// Copyright 2021 by David A. Golden. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package pbkdf2

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"strings"
	"testing"
)

type testVector struct {
	n string
	p string
	s string
	c int
	l int
	o string
}

type testKeyGen struct {
	h       func() hash.Hash
	vectors []testVector
}

var cases = []testKeyGen{
	{
		h: sha1.New,
		vectors: []testVector{
			// RFC 6070 test vectors
			{
				n: "SHA-1 1 iter",
				p: "password",
				s: "salt",
				c: 1,
				l: 20,
				o: "0c 60 c8 0f 96 1f 0e 71 f3 a9 b5 24 af 60 12 06 2f e0 37 a6",
			},
			{
				n: "SHA-1 2 iters",
				p: "password",
				s: "salt",
				c: 2,
				l: 20,
				o: "ea 6c 01 4d c7 2d 6f 8c cd 1e d9 2a ce 1d 41 f0 d8 de 89 57",
			},
			{
				n: "SHA-1 4096 iters",
				p: "password",
				s: "salt",
				c: 4096,
				l: 20,
				o: "4b 00 79 01 b7 65 48 9a be ad 49 d9 26 f7 21 d0 65 a4 29 c1",
			},
			{
				n: "SHA-1 4096 iters, longer pw/salt/dk_length",
				p: "passwordPASSWORDpassword",
				s: "saltSALTsaltSALTsaltSALTsaltSALTsalt",
				c: 4096,
				l: 25,
				o: "3d 2e ec 4f e4 1c 84 9b 80 c8 d8 36 62 c0 e4 4a 8b 29 1a 96 4c f2 f0 70 38",
			},
			{
				n: "SHA-1 4096 iters, embedded nulls, short dk",
				p: "pass\x00word",
				s: "sa\x00lt",
				c: 4096,
				l: 16,
				o: "56 fa 6a a7 55 48 09 9d cc 37 d7 f0 34 25 e0 c3",
			},
			// Additional test vectors
			{
				n: "SHA-1 3 iters",
				p: "password",
				s: "salt",
				c: 3,
				l: 20,
				o: "6b 4e 26 12 5c 25 cf 21 ae 35 ea d9 55 f4 79 ea 2e 71 f6 ff",
			},
		},
	},
	{
		h: sha256.New224,
		vectors: []testVector{
			// SHA-224 vectors from Crypt::PBKDF2/PBKDF2::Tiny
			{
				n: "SHA-224 1 iter",
				p: "password",
				s: "salt",
				c: 1,
				l: 28,
				o: "3c 19 8c bd b9 46 4b 78 57 96 6b d0 5b 7b c9 2b c1 cc 4e 6e 63 15 5d 4e 49 05 57 fd",
			},
			{
				n: "SHA-224 1000 iter",
				p: "password",
				s: "salt",
				c: 1000,
				l: 28,
				o: "d3 bc f3 20 fd 91 89 08 ea fc aa 46 0f af 40 e2 01 f6 50 8d 4e 6f 3d 9c 1c 0a bd 30",
			},
		},
	},
	{
		h: sha256.New,
		vectors: []testVector{
			// SHA-256 vectors from https://stackoverflow.com/questions/5130513/pbkdf2-hmac-sha2-test-vectors
			{
				n: "SHA-256 1 iter",
				p: "password",
				s: "salt",
				c: 1,
				l: 32,
				o: "12 0f b6 cf fc f8 b3 2c 43 e7 22 52 56 c4 f8 37 a8 65 48 c9 2c cc 35 48 08 05 98 7c b7 0b e1 7b",
			},
			{
				n: "SHA-256 2 iters",
				p: "password",
				s: "salt",
				c: 2,
				l: 32,
				o: "ae 4d 0c 95 af 6b 46 d3 2d 0a df f9 28 f0 6d d0 2a 30 3f 8e f3 c2 51 df d6 e2 d8 5a 95 47 4c 43",
			},
			{
				n: "SHA-256 4096 iter",
				p: "password",
				s: "salt",
				c: 4096,
				l: 32,
				o: "c5 e4 78 d5 92 88 c8 41 aa 53 0d b6 84 5c 4c 8d 96 28 93 a0 01 ce 4e 11 a4 96 38 73 aa 98 13 4a",
			},
			// Too many iterations
			// {
			// 	p: "password",
			// 	s: "salt",
			// 	c: 16777216,
			// 	l: 32,
			// 	o: "cf 81 c6 6f e8 cf c0 4d 1f 31 ec b6 5d ab 40 89 f7 f1 79 e8 9b 3b 0b cb 17 ad 10 e3 ac 6e ba 46",
			// },
			{
				n: "SHA-256 4096 iters, longer pw/salt/dk_length",
				p: "passwordPASSWORDpassword",
				s: "saltSALTsaltSALTsaltSALTsaltSALTsalt",
				c: 4096,
				l: 40,
				o: "34 8c 89 db cb d3 2b 2f 32 d8 14 b8 11 6e 84 cf 2b 17 34 7e bc 18 00 18 1c 4e 2a 1f b8 dd 53 e1 c6 35 51 8c 7d ac 47 e9",
			},
			{
				n: "SHA-256 4096 iters, embedded nulls, short dk",
				p: "pass\x00word",
				s: "sa\x00lt",
				c: 4096,
				l: 16,
				o: "89 b6 9d 05 16 f8 29 89 3c 69 62 26 65 0a 86 87",
			},
			// SHA-256 vectors from Crypt::PBKDF2/PBKDF2::Tiny
			{
				n: "SHA-256 1 iter, 2",
				p: "password",
				s: "salt",
				c: 1,
				l: 32,
				o: "12 0f b6 cf fc f8 b3 2c 43 e7 22 52 56 c4 f8 37 a8 65 48 c9 2c cc 35 48 08 05 98 7c b7 0b e1 7b ",
			},
			{
				n: "SHA-256 1000 iter",
				p: "password",
				s: "salt",
				c: 1000,
				l: 32,
				o: "63 2c 28 12 e4 6d 46 04 10 2b a7 61 8e 9d 6d 7d 2f 81 28 f6 26 6b 4a 03 26 4d 2a 04 60 b7 dc b3 ",
			},
		},
	},
	{
		h: sha512.New384,
		vectors: []testVector{
			// SHA-384 vectors from Crypt::PBKDF2/PBKDF2::Tiny
			{
				n: "SHA-384 1 iter",
				p: "password",
				s: "salt",
				c: 1,
				l: 48,
				o: "c0 e1 4f 06 e4 9e 32 d7 3f 9f 52 dd f1 d0 c5 c7 19 16 09 23 36 31 da dd 76 a5 67 db 42 b7 86 76 b3 8f c8 00 cc 53 dd b6 42 f5 c7 44 42 e6 2b e4 ",
			},
			{
				n: "SHA-384 1000 iter",
				p: "password",
				s: "salt",
				c: 1000,
				l: 48,
				o: "3b d3 7e 22 36 94 1d 4a 77 b1 b5 b7 14 c6 f9 13 fa bb 6b 08 41 a6 d7 d8 65 6b 99 d6 11 e9 00 fe 06 ed b9 3b 5b 80 9e fa a9 67 8b 63 5c e5 13 e0 ",
			},
		},
	},
	{
		h: sha512.New,
		vectors: []testVector{
			// SHA-512 vectors from Crypt::PBKDF2/PBKDF2::Tiny
			{
				n: "SHA-512 1 iter",
				p: "password",
				s: "salt",
				c: 1,
				l: 64,
				o: "86 7f 70 cf 1a de 02 cf f3 75 25 99 a3 a5 3d c4 af 34 c7 a6 69 81 5a e5 d5 13 55 4e 1c 8c f2 52 c0 2d 47 0a 28 5a 05 01 ba d9 99 bf e9 43 c0 8f 05 02 35 d7 d6 8b 1d a5 5e 63 f7 3b 60 a5 7f ce ",
			},
			{
				n: "SHA-512 1000 iter",
				p: "password",
				s: "salt",
				c: 1000,
				l: 64,
				o: "af e6 c5 53 07 85 b6 cc 6b 1c 64 53 38 47 31 bd 5e e4 32 ee 54 9f d4 2f b6 69 57 79 ad 8a 1c 5b f5 9d e6 9c 48 f7 74 ef c4 00 7d 52 98 f9 03 3c 02 41 d5 ab 69 30 5e 7b 64 ec ee b8 d8 34 cf ec ",
			},
		},
	},
}

func TestKey(t *testing.T) {
	for _, c := range cases {
		c := c
		for _, v := range c.vectors {
			v := v
			t.Run(v.n, func(t *testing.T) {
				t.Parallel()
				expected, err := hex.DecodeString(strings.Replace(v.o, " ", "", -1))
				if err != nil {
					t.Fatalf("error decoding expected output: %v", err)
				}
				key := Key([]byte(v.p), []byte(v.s), v.c, v.l, c.h)
				if !bytes.Equal(expected, key) {
					t.Errorf("incorrect derived key\n  Got: %s\n Want: %s\n", keyTuples(key), v.o)
				}
			})
		}
	}
}

func keyTuples(key []byte) string {
	var xs []string
	for len(key) > 0 {
		xs = append(xs, hex.EncodeToString(key[0:1]))
		key = key[1:]
	}
	return strings.Join(xs, " ")
}
