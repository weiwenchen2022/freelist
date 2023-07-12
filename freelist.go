package freelist

import (
	"log"
	"reflect"
	"sync"
	"unsafe"
)

// A FreeList is a set of temporary objects that may be individually saved and
// retrieved.
//
// Any item stored in the FreeList must be a struct look schematically like
//
//	type T struct {
//		...
//		next *T
//		...
//	}
//
// A FreeList is safe for use by multiple goroutines simultaneously.
//
// FreeList's purpose is to cache allocated but unused items for later reuse,
// relieving pressure on the garbage collector. That is, it makes it easy to
// build efficient, thread-safe free lists.
//
// An appropriate use of a FreeList is to manage a group of temporary items
// silently shared among and potentially reused by concurrent independent
// clients of a package. FreeList provides a way to amortize allocation overhead
// across many clients.
//
// On the other hand, a free list maintained as part of a short-lived object is
// not a suitable use for a FreeList, since the overhead does not amortize well in
// that scenario. It is more efficient to have such objects implement their own
// free list.
//
// A FreeList must not be copied after first use.
//
// In the terminology of the Go memory model, a call to Put(x) “synchronizes before”
// a call to Get returning that same value x.
// Similarly, a call to New returning x “synchronizes before”
// a call to Get returning that same value x.
type FreeList[E any] struct {
	noCopy noCopy

	mu   sync.Mutex // protects free
	free *E

	initOnce sync.Once
	next     uintptr

	// New optionally specifies a function to generate
	// a value when Get would otherwise return nil.
	// It may not be changed concurrently with calls to Get.
	New func() *E

	// Reset optionally specifies a function to reset
	// a value when Get would return a cached value.
	// It may not be changed concurrently with calls to Get.
	Reset func(*E)
}

// Get selects a last put back item from the FreeList, removes it from the
// FreeList, if l.Reset is non-nil calling l.Reset with it, and returns it to the caller.
//
// If Get would otherwise return nil and l.New is non-nil, Get returns
// the result of calling l.New.
func (l *FreeList[E]) Get() *E {
	l.init()

	l.mu.Lock()
	x := l.free
	if x != nil {
		l.free = *(**E)(unsafe.Pointer(uintptr(unsafe.Pointer(x)) + l.next))
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

// Put adds x to the free list.
func (l *FreeList[E]) Put(x *E) {
	l.init()

	l.mu.Lock()
	*(**E)(unsafe.Pointer(uintptr(unsafe.Pointer(x)) + l.next)) = l.free
	l.free = x
	l.mu.Unlock()
}

// Dummy type used to generate an implicit panic. This must be defined at the
// package level; if it is defined inside a function, it prevents the inlining
// of that function.
type toPutInFreeListItemMustBeAStructThatHasFieldPointerToNext struct{}

// init ensures l is properly initialized.
func (l *FreeList[E]) init() {
	didPanic := false
	l.initOnce.Do(func() {
		didPanic = true
		typeOfE := reflect.TypeOf((*E)(nil)).Elem()
		if reflect.Struct == typeOfE.Kind() {
			nextField, ok := typeOfE.FieldByName("next")
			if ok && reflect.PointerTo(typeOfE) == nextField.Type {
				l.next = nextField.Offset
				didPanic = false
				return
			}
		}
		// We don't want to call panic here because it prevents the
		// inlining of this function.
		_ = any(nil).(toPutInFreeListItemMustBeAStructThatHasFieldPointerToNext)
	})
	if didPanic {
		log.Print(`an item stored in free list must be a struct that has field pointer to "next"`)
		_ = any(nil).(toPutInFreeListItemMustBeAStructThatHasFieldPointerToNext)
	}
}

// noCopy may be added to structs which must not be copied
// after the first use.
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
//
// Note that it must not be embedded, due to the Lock and Unlock methods.
type noCopy struct{}

// Lock is a no-op used by -copylocks checker from `go vet`.
func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
