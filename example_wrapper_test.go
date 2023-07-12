// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package freelist_test

import (
	"bytes"
	"io"
	"os"
	"time"

	"github.com/weiwenchen2022/freelist"
)

type Buf struct {
	*bytes.Buffer
	next *Buf // for free list
}

var freeBuf = freelist.FreeList[Buf]{
	New: func() *Buf {
		return &Buf{Buffer: new(bytes.Buffer)}
	},

	Reset: (*Buf).Reset,
}

// timeNow is a fake version of time.Now for tests.
func timeNow() time.Time {
	return time.Unix(1136214245, 0)
}

func Log(w io.Writer, key, val string) {
	b := freeBuf.Get()
	defer freeBuf.Put(b)

	// Replace this with time.Now() in a real logger.
	b.WriteString(timeNow().UTC().Format(time.RFC3339))
	b.WriteByte(' ')
	b.WriteString(key)
	b.WriteByte('=')
	b.WriteString(val)
	_, _ = io.Copy(w, b)
}

func Example_wrapper() {
	Log(os.Stdout, "path", "/search?q=flowers")
	// Output: 2006-01-02T15:04:05Z path=/search?q=flowers
}
