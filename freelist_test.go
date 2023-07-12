package freelist_test

import (
	"sync"
	"testing"

	. "github.com/weiwenchen2022/freelist"
)

func TestList(t *testing.T) {
	type T struct {
		A    string
		next *T
	}

	var l FreeList[T]
	if l.Get() != nil {
		t.Fatal("expected empty")
	}

	l.Put(&T{A: "a"})
	l.Put(&T{A: "b"})

	if g := l.Get(); g.A != "b" {
		t.Fatalf(`got %q; want "b"`, g.A)
	}
	if g := l.Get(); g.A != "a" {
		t.Fatalf(`got %q; want "a"`, g.A)
	}

	// Put in a large number of objects so they spill into
	// stealable space.
	for i := 0; i < 100; i++ {
		l.Put(&T{A: "c"})
	}
	if g := l.Get(); g.A != "c" {
		t.Fatalf(`got %q; want "c"`, g.A)
	}
}

type T struct {
	A    int
	next *T
}

func TestListNew(t *testing.T) {
	i := 0
	l := FreeList[T]{
		New: func() *T {
			i++
			return &T{A: i}
		},
	}

	if v := l.Get(); v.A != 1 {
		t.Errorf("got %v; want 1", v.A)
	}
	if v := l.Get(); v.A != 2 {
		t.Errorf("got %v; want 2", v.A)
	}

	l.Put(&T{A: 42})
	if v := l.Get(); v.A != 42 {
		t.Errorf("got %v; want 0", v.A)
	}

	if v := l.Get(); v.A != 3 {
		t.Errorf("got %v; want 3", v.A)
	}
}

func TestListReset(t *testing.T) {
	i := 0
	var l = FreeList[T]{
		New: func() *T {
			i++
			return &T{A: i}
		},
		Reset: func(t *T) {
			*t = T{}
		},
	}

	if v := l.Get(); v.A != 1 {
		t.Fatalf("got %v; want 1", v.A)
	}
	if v := l.Get(); v.A != 2 {
		t.Fatalf("got %v; want 2", v.A)
	}

	l.Put(&T{A: 42})
	if v := l.Get(); v.A != 0 {
		t.Fatalf("got %v; want 0", v.A)
	}

	if v := l.Get(); v.A != 3 {
		t.Errorf("got %v; want 3", v.A)
	}
}

func panics(f func()) (b bool) {
	defer func() { recover() }()
	b = true
	f()
	return false
}

type NotStruct int
type StructNoNextField struct{ A int }
type StructNextFieldNotPointerToStruct struct {
	A    int
	next string
}

func TestPanics(t *testing.T) {
	tests := []struct {
		f func()
	}{
		{
			f: func() {
				var l FreeList[NotStruct]
				l.Get()
			},
		},
		{
			f: func() {
				var l FreeList[StructNoNextField]
				l.Get()
			},
		},
		{
			f: func() {
				var l FreeList[StructNextFieldNotPointerToStruct]
				l.Get()
			},
		},
	}

	for i, tt := range tests {
		if got := panics(tt.f); !got {
			t.Errorf("%d didn't panic as expected", i)
		}
	}
}

// nativeFreeList is a baseline implementation to the FreeList.
type nativeFreeList struct {
	mu   sync.Mutex
	free *T

	New   func() *T
	Reset func(*T)
}

func (l *nativeFreeList) Get() *T {
	l.mu.Lock()
	x := l.free
	if x != nil {
		l.free = x.next
		l.mu.Unlock()

		if l.Reset != nil {
			l.Reset(x)
		}
	} else {
		l.mu.Unlock()

		if l.New != nil {
			x = l.New()
		}
	}
	return x
}

func (l *nativeFreeList) Put(x *T) {
	l.mu.Lock()
	x.next = l.free
	l.free = x
	l.mu.Unlock()
}

func BenchmarkFreeList(b *testing.B) {
	b.Run("NativeFreeList", func(b *testing.B) {
		var l nativeFreeList
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				l.Put(&T{A: 1})
				l.Get()
			}
		})
	})

	b.Run("FreeList", func(b *testing.B) {
		var l FreeList[T]
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				l.Put(&T{A: 1})
				l.Get()
			}
		})
	})
}

func BenchmarkFreeListOverflow(b *testing.B) {
	b.Run("NativeFreeList", func(b *testing.B) {
		var l nativeFreeList
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for b := 0; b < 100; b++ {
					l.Put(&T{A: 1})
				}
				for b := 0; b < 100; b++ {
					l.Get()
				}
			}
		})
	})

	b.Run("FreeList", func(b *testing.B) {
		var l FreeList[T]
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				for b := 0; b < 100; b++ {
					l.Put(&T{A: 1})
				}
				for b := 0; b < 100; b++ {
					l.Get()
				}
			}
		})
	})
}
