package defaults

import (
	"reflect"
	"unsafe"
)

// walking is a value being walked on the current path, linked to the one above it: a struct, by its
// address, or a slice or map, by the array or table it refers to. Data the caller built with a
// way back to itself leads the walk back to such a value, which Set used to follow until the stack
// overflowed. The walk goes no further there instead, and the value is finished by the walk already
// under way above. Nothing is kept once a value's walk returns, so a value that two paths reach
// without either passing through the other is walked on both.
//
// The type is part of the name, since a struct shares its address with its first field and an array
// with its first element, and so is a slice's length, since a longer slice over the same array has
// elements the shorter one lacks. The address is held as an unsafe.Pointer, which keeps the value
// alive while it is on the path, so no value made below can be given the same address.
//
// A value no more than scanDepth entries down is checked by scanning the entries above it, which
// allocates nothing. Scanning further down would make a walk n values deep take on the order of n²/2
// comparisons, so a value below scanDepth is looked up in a pathIndex instead, which the first such
// check makes.
type walking struct {
	ptr   unsafe.Pointer
	typ   reflect.Type
	len   int
	outer *walking

	// depth is the number of entries above this one.
	depth int

	// index is the path's index, or nil while the walk has made none. The check that makes it stores it
	// into every entry then on the path; an entry declared after copies it, with depth, from the entry
	// above where set or fillField declares it. A method copying it would read it off one entry and
	// store it through a pointer into another, which the compiler takes for storing any pointer the first
	// entry holds, outer among them, and every entry on the path would move to the heap.
	index *pathIndex
}

// scanDepth is the deepest a value is checked by scanning the path: a scan that deep costs about as
// much as the index does for a value, and making the index, three allocations and about 9 KB, has
// paid for itself by about 100 values down. A walk no deeper than scanDepth makes no index.
const scanDepth = 64

// repeatsNear reports whether the value w names is already being walked above it, on w.outer's path,
// if w is no more than scanDepth down. For a deeper w it reports false, and repeatsFar checks it:
// callers ask here.repeatsNear() || here.repeatsFar(), and neither half answers for the other, which
// is why neither is named as if it did.
//
// The two are kept apart so that both inline, at costs 43 and 65 against the budget of 80. One
// method holding the scan and the call to repeatsInIndex costs 101 and does not inline, and the
// call it would leave in the walk made Walk/slice/pointers/elements=1000 take 1.6 times as long in
// about half the runs.
func (w *walking) repeatsNear() bool {
	if w.depth > scanDepth {
		return false
	}
	for p := w.outer; p != nil; p = p.outer {
		if p.ptr == w.ptr && p.typ == w.typ && p.len == w.len {
			return true
		}
	}
	return false
}

// repeatsFar is repeatsNear for a w more than scanDepth down.
func (w *walking) repeatsFar() bool {
	return w.depth > scanDepth && w.repeatsInIndex()
}

// repeatsInIndex is repeatsFar's check. It makes the path's index if the walk has none yet, adds
// to it every entry above w that it does not hold, and then w, unless it already holds w's key.
func (w *walking) repeatsInIndex() bool {
	if w.index == nil {
		index := &pathIndex{path: make([]indexEntry, 256), heads: make([]int32, 256)}
		for p := w; p != nil; p = p.outer {
			p.index = index
		}
	}
	if w.index.held < w.depth {
		w.outer.addTo(w.index)
	}
	return !w.index.add(w.key())
}

// addTo adds w to x, after every entry above it that x does not hold yet. Each entry below scanDepth
// adds itself as it is checked, so the entries x lacks are within scanDepth of the top, and the calls
// to addTo stack no deeper.
func (w *walking) addTo(x *pathIndex) {
	if x.held < w.depth {
		w.outer.addTo(x)
	}
	x.add(w.key())
}

// leave takes w out of the path's index, if the index holds it, as w's walk returns. It is called on
// every return, a failed walk's too: a failure ends the whole Set, but an index still holding w would
// take a later value with w's key for a repeat.
func (w *walking) leave() {
	if w.index != nil && w.depth < w.index.held {
		w.index.pop()
	}
}

// key is w as the path's index holds it. A reflect.Type points to the one descriptor of its type, so
// the descriptor's address names the type, as == on Types does. UnsafePointer reads that address;
// Pointer would mark it as escaping.
func (w *walking) key() pathKey {
	return pathKey{ptr: uintptr(w.ptr), typ: uintptr(reflect.ValueOf(w.typ).UnsafePointer()), len: w.len}
}

// pathKey names an entry in the path's index by its address, its type and its length. They are held
// as uintptrs, not pointers: a pointer read off an entry and stored in the index would move every
// entry to the heap, as walking's index field explains. The entry keeps its value and its type alive
// while the index holds its key.
type pathKey struct {
	ptr, typ uintptr
	len      int
}

// hash mixes k into an int whose low bits pick a bucket.
func (k pathKey) hash() int {
	h := (uint64(k.ptr) ^ uint64(k.typ)*0x9e3779b97f4a7c15 ^ uint64(k.len)) * 0x9e3779b97f4a7c15
	return int(h ^ h>>32)
}

// pathIndex is the index of a path too deep to scan: the entries at the top of the path, down to some
// depth, and a hash table of their keys. Entries join and leave it as they do the path, at the bottom,
// so it holds them as a stack: path[d] is the entry d entries down, for each d below held.
//
// The table chains the entries whose keys share a bucket, deepest first. heads holds, for each
// bucket, one more than the position in path of its deepest entry, or 0 for none, and each entry's
// next does the same for the entry after it in the chain. The deepest entry of all heads its chain,
// so taking it out only moves its bucket's head on to its next.
//
// Neither slice is appended to or resliced; grow replaces both. The index is reached from an entry, so
// a slice read off it and stored back into it would be, to the compiler, a pointer read off that entry
// and stored on the heap, which would move every entry to the heap.
type pathIndex struct {
	path  []indexEntry
	heads []int32
	held  int
}

// indexEntry is an entry the path's index holds: its key, and the next entry in its bucket.
type indexEntry struct {
	key  pathKey
	next int32
}

// add adds an entry for k below the deepest x holds, and reports true, or reports false and adds
// nothing if x already holds k.
func (x *pathIndex) add(k pathKey) bool {
	if x.held == len(x.path) {
		x.grow()
	}
	b := k.hash() & (len(x.heads) - 1)
	for i := x.heads[b]; i != 0; i = x.path[i-1].next {
		if x.path[i-1].key == k {
			return false
		}
	}
	x.path[x.held] = indexEntry{key: k, next: x.heads[b]}
	x.held++
	x.heads[b] = int32(x.held)
	return true
}

// pop takes the deepest entry out of x.
func (x *pathIndex) pop() {
	x.held--
	e := x.path[x.held]
	x.heads[e.key.hash()&(len(x.heads)-1)] = e.next
}

// grow doubles x's room for entries and its buckets, as many of each and a power of two, and chains
// every entry again from the top down, so that each chain stays deepest first.
func (x *pathIndex) grow() {
	path := make([]indexEntry, 2*len(x.path))
	copy(path, x.path)
	x.path = path
	x.heads = make([]int32, len(path))
	for i := 0; i < x.held; i++ {
		b := path[i].key.hash() & (len(x.heads) - 1)
		path[i].next = x.heads[b]
		x.heads[b] = int32(i + 1)
	}
}
