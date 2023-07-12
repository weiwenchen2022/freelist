// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package freelist_test

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"

	"github.com/weiwenchen2022/freelist"
)

type P struct {
	X, Y, Z int
	Name    string
}

type Q struct {
	X, Y *int32
	Name string

	next *Q // for free list
}

var freeQ = &freelist.FreeList[Q]{
	New: func() *Q {
		return new(Q)
	},
	Reset: func(q *Q) {
		*q = Q{}
	},
}

func Example() {
	// Initialize the encoder and decoder. Normally enc and dec would be
	// bound to network connections and the encoder and decoder would
	// run in different processes.
	var network bytes.Buffer        // Stand-in for a network connection
	enc := gob.NewEncoder(&network) // Will write to network.
	dec := gob.NewDecoder(&network) // Will read from network.

	// Encode (send) some values.
	if err := enc.Encode(P{3, 4, 5, "Pythagoras"}); err != nil {
		log.Fatal("encode error:", err)
	}

	if err := enc.Encode(P{1782, 1841, 1922, "Treehouse"}); err != nil {
		log.Fatal("encode error:", err)
	}

	// Decode (receive) and print the values.
	var q = freeQ.Get()
	defer freeQ.Put(q)

	if err := dec.Decode(q); err != nil {
		log.Fatal("decode error 1:", err)
	}
	fmt.Printf("%q: {%d, %d}\n", q.Name, *q.X, *q.Y)

	q = freeQ.Get()
	defer freeQ.Put(q)

	if err := dec.Decode(q); err != nil {
		log.Fatal("decode error 2:", err)
	}
	fmt.Printf("%q: {%d, %d}\n", q.Name, *q.X, *q.Y)

	// Output:
	// "Pythagoras": {3, 4}
	// "Treehouse": {1782, 1841}
}
