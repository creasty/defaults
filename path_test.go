package defaults

import (
	"reflect"
	"runtime"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This test is in package defaults rather than defaults_test: it takes walking and pathIndex on their
// own, which Set reaches only through a walk deeper than scanDepth. What Set makes of them — cycles
// ending, a value shared by two paths filled on each — is pinned from outside, in set_cycle_test.go
// and set_shared_test.go.

// pathEntries returns n entries linked as set and fillField link them, each naming a value of its own,
// the first at the top of a path. The values live as long as the test, since a key holds an address
// rather than a pointer, and an address a value has left can be given to another.
func pathEntries(t *testing.T, n int) []walking {
	t.Helper()

	values := make([]int, n)
	t.Cleanup(func() { runtime.KeepAlive(values) })

	p := make([]walking, n)
	for i := range p {
		p[i] = walking{ptr: unsafe.Pointer(&values[i]), typ: reflect.TypeOf(0)}
		if i > 0 {
			p[i].outer, p[i].depth, p[i].index = &p[i-1], p[i-1].depth+1, p[i-1].index
		}
	}
	return p
}

// pathBelow returns an entry naming a value of its own, below outer.
func pathBelow(t *testing.T, outer *walking) walking {
	t.Helper()

	value := new(int)
	t.Cleanup(func() { runtime.KeepAlive(value) })

	return walking{
		ptr: unsafe.Pointer(value), typ: reflect.TypeOf(0),
		outer: outer, depth: outer.depth + 1, index: outer.index,
	}
}

// pathSmallIndex returns an index with room for size entries, which grow doubles. The walk makes a
// larger one; a small one reaches grow in a handful of keys.
func pathSmallIndex(size int) *pathIndex {
	return &pathIndex{path: make([]indexEntry, size), heads: make([]int32, size)}
}

// pathKeys returns n keys, no two alike, and keeps the values they name alive for the test.
func pathKeys(t *testing.T, n int) []pathKey {
	t.Helper()

	p := pathEntries(t, n)
	keys := make([]pathKey, n)
	for i := range p {
		keys[i] = p[i].key()
	}
	return keys
}

// pathKeyInBucket returns a key other than k that an index of that many buckets puts in k's bucket,
// to test a chain of them.
func pathKeyInBucket(t *testing.T, buckets int, k pathKey) pathKey {
	t.Helper()

	mask := buckets - 1
	for n := k.len + 1; n <= k.len+1000; n++ {
		if other := (pathKey{ptr: k.ptr, typ: k.typ, len: n}); other.hash()&mask == k.hash()&mask {
			return other
		}
	}
	t.Fatalf("no second key of the first %d lengths lands in the bucket", 1000)
	return pathKey{}
}

// TestWalking_RepeatsNearFindsTheSameValueAbove covers the scan, which a value no more than scanDepth
// down is checked by: the whole path above it, for a value with its address, its type and its length.
func TestWalking_RepeatsNearFindsTheSameValueAbove(t *testing.T) {
	above := pathEntries(t, 4)
	w := pathBelow(t, &above[3])

	assert.False(t, w.repeatsNear(), "a value of its own repeats nothing")

	w.ptr, w.typ, w.len = above[1].ptr, above[1].typ, above[1].len
	assert.True(t, w.repeatsNear(), "the same value, as far up as the path goes")

	w.typ = reflect.TypeOf("")
	assert.False(t, w.repeatsNear(), "the same address under another type is another value")

	w.typ, w.len = above[1].typ, above[1].len+1
	assert.False(t, w.repeatsNear(), "a longer slice over the same array is another value")
}

// TestWalking_RepeatsNearLeavesDeeperValuesToRepeatsFar covers the split between the two: the scan
// gives up below scanDepth, where it would cost more than the index, and reports false however deep
// the value is.
func TestWalking_RepeatsNearLeavesDeeperValuesToRepeatsFar(t *testing.T) {
	above := pathEntries(t, scanDepth+2)
	w := pathBelow(t, &above[len(above)-1])
	w.ptr, w.typ, w.len = above[1].ptr, above[1].typ, above[1].len

	require.Greater(t, w.depth, scanDepth)
	assert.False(t, w.repeatsNear(), "past scanDepth the scan answers nothing")
	assert.True(t, w.repeatsFar(), "the index answers instead")
}

// TestWalking_RepeatsFarIndexesThePathAboveIt covers the first check below scanDepth: it makes the
// index, gives it to every entry on the path, and adds them all, so that a value further down is
// looked up against the whole path and not only the part below scanDepth.
func TestWalking_RepeatsFarIndexesThePathAboveIt(t *testing.T) {
	above := pathEntries(t, scanDepth+2)
	w := pathBelow(t, &above[len(above)-1])

	require.False(t, w.repeatsFar(), "a value of its own repeats nothing")

	require.NotNil(t, w.index)
	assert.Equal(t, w.depth+1, w.index.held, "every entry above it, and itself")
	for i := range above {
		assert.Same(t, w.index, above[i].index, "entry %d holds the path's index", i)
		assert.False(t, w.index.add(above[i].key()), "the index holds entry %d", i)
	}

	// A repeat adds nothing: the entry above keeps the key, and takes it out when its own walk returns.
	again := pathBelow(t, &above[len(above)-1])
	again.ptr, again.typ, again.len = above[1].ptr, above[1].typ, above[1].len
	assert.True(t, again.repeatsFar())
	assert.Equal(t, w.depth+1, w.index.held, "a repeat is not added")
}

// TestWalking_LeaveTakesOnlyItsOwnEntryOut covers what leave does as a walk returns, called as it is
// on every return, including one the index never held an entry for.
func TestWalking_LeaveTakesOnlyItsOwnEntryOut(t *testing.T) {
	above := pathEntries(t, scanDepth+2)
	w := pathBelow(t, &above[len(above)-1])
	require.False(t, w.repeatsFar())
	held := w.index.held

	w.leave()
	assert.Equal(t, held-1, w.index.held, "its own entry, and nothing above it")

	w.leave()
	assert.Equal(t, held-1, w.index.held, "leaving twice takes nothing more out")

	shallow := pathEntries(t, 1)[0]
	shallow.leave()
	assert.Nil(t, shallow.index, "a walk that made no index leaves nothing")

	// The key it took out is free again, so the next path down reaches that value.
	next := pathBelow(t, &above[len(above)-1])
	next.ptr, next.typ, next.len = w.ptr, w.typ, w.len
	assert.False(t, next.repeatsFar())
}

// TestWalking_KeyNamesTheValueItsTypeAndItsLength covers what an entry is to the index: two entries
// have one key only if they name the same value, under the same type, at the same length.
func TestWalking_KeyNamesTheValueItsTypeAndItsLength(t *testing.T) {
	value := 0
	a := walking{ptr: unsafe.Pointer(&value), typ: reflect.TypeOf(0)}

	b := a
	assert.Equal(t, a.key(), b.key(), "the same value, type and length")

	b.typ = reflect.TypeOf("")
	assert.NotEqual(t, a.key(), b.key(), "another type")

	b.typ, b.len = a.typ, a.len+1
	assert.NotEqual(t, a.key(), b.key(), "another length")

	other := 0
	c := walking{ptr: unsafe.Pointer(&other), typ: a.typ}
	assert.NotEqual(t, a.key(), c.key(), "another value")
}

// TestPathIndex_AddHoldsEachKeyOnce covers add's answer, which is the repeat: a key the index holds
// is not added again.
func TestPathIndex_AddHoldsEachKeyOnce(t *testing.T) {
	x := pathSmallIndex(4)
	keys := pathKeys(t, 3)

	for i, k := range keys {
		assert.True(t, x.add(k), "key %d", i)
	}
	assert.Equal(t, len(keys), x.held)

	for i, k := range keys {
		assert.False(t, x.add(k), "key %d again", i)
	}
	assert.Equal(t, len(keys), x.held, "nothing is held twice")
}

// TestPathIndex_PopTakesTheDeepestKeyOut covers pop, which leave calls: the keys join and leave the
// index as their values do the path, so the one taken out is the last one added.
func TestPathIndex_PopTakesTheDeepestKeyOut(t *testing.T) {
	x := pathSmallIndex(4)
	keys := pathKeys(t, 3)
	for _, k := range keys {
		require.True(t, x.add(k))
	}

	for i := len(keys) - 1; i >= 0; i-- {
		x.pop()
		require.Equal(t, i, x.held)

		for j := 0; j < i; j++ {
			assert.False(t, x.add(keys[j]), "key %d is held while key %d is out", j, i)
		}
		assert.True(t, x.add(keys[i]), "key %d is out", i)
		x.pop()
	}
}

// TestPathIndex_GrowsAndKeepsEveryKey covers growing past the room the index was made with, which a
// walk does a few hundred values down: every key is still held, and still comes out in order.
func TestPathIndex_GrowsAndKeepsEveryKey(t *testing.T) {
	x := pathSmallIndex(4)
	keys := pathKeys(t, 20)
	for i, k := range keys {
		require.True(t, x.add(k), "key %d", i)
	}

	assert.Greater(t, len(x.path), 4, "it grew")
	assert.Equal(t, len(x.path), len(x.heads), "as many buckets as room for keys")
	for i, k := range keys {
		assert.False(t, x.add(k), "key %d after growing", i)
	}

	for i := len(keys) - 1; i >= 0; i-- {
		x.pop()
		require.True(t, x.add(keys[i]), "key %d comes out last", i)
		x.pop()
	}
	assert.Equal(t, 0, x.held)
}

// TestPathIndex_HoldsKeysThatShareABucket covers the chain a bucket holds: keys that hash together
// are told apart, and taking one out leaves the others where the index can find them.
func TestPathIndex_HoldsKeysThatShareABucket(t *testing.T) {
	x := pathSmallIndex(8)
	first := pathKeys(t, 1)[0]
	second := pathKeyInBucket(t, len(x.heads), first)

	require.True(t, x.add(first))
	require.True(t, x.add(second))
	assert.False(t, x.add(first), "the first is held")
	assert.False(t, x.add(second), "the second is held")

	x.pop()
	assert.False(t, x.add(first), "the first is held after the second is out")
	assert.True(t, x.add(second), "the second is out")
}

// TestPathIndex_GrowingKeepsEachBucketDeepestFirst covers the chains grow builds again. The deepest
// key of a bucket heads its chain, so taking it out moves the head on to the next; chained the other
// way round, taking it out would lose the keys above it, which are still held.
func TestPathIndex_GrowingKeepsEachBucketDeepestFirst(t *testing.T) {
	x := pathSmallIndex(4)
	first := pathKeys(t, 1)[0]
	// The two share a bucket once x has grown to 8 of them, which the fifth key below makes it do.
	second := pathKeyInBucket(t, 8, first)
	filler := pathKeys(t, 3)

	held := append([]pathKey{first, second}, filler...)
	for i, k := range held {
		require.True(t, x.add(k), "key %d", i)
	}
	require.Equal(t, 8, len(x.heads), "it grew")
	require.Equal(t, first.hash()&7, second.hash()&7, "the two share a bucket now")

	for i := len(held) - 1; i >= 0; i-- {
		x.pop()
		for j := 0; j < i; j++ {
			assert.False(t, x.add(held[j]), "key %d is held while key %d is out", j, i)
		}
		require.True(t, x.add(held[i]), "key %d is out", i)
		x.pop()
	}
	assert.Equal(t, 0, x.held)
}
