package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPushAndPop(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(10, 0, true)
	entry := []byte("hello")

	// when
	_, err := queue.Pop()

	// then
	assert.EqualError(t, err, "Empty queue")

	// when
	queue.Push(entry)

	// then
	assert.Equal(t, entry, pop(queue))
}

func TestLen(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(100, 0, false)
	entry := []byte("hello")
	assert.Zero(t, queue.Len())

	// when
	queue.Push(entry)

	// then
	assert.Equal(t, queue.Len(), 1)
}

func TestPeek(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(100, 0, false)
	entry := []byte("hello")

	// when
	read, err := queue.Peek()

	// then
	assert.EqualError(t, err, "Empty queue")
	assert.Nil(t, read)

	// when
	queue.Push(entry)
	read, err = queue.Peek()

	// then
	assert.NoError(t, err)
	assert.Equal(t, pop(queue), read)
	assert.Equal(t, entry, read)
}

func TestReset(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(100, 0, false)
	entry := []byte("hello")

	// when
	queue.Push(entry)
	queue.Push(entry)
	queue.Push(entry)

	queue.Reset()
	read, err := queue.Peek()

	// then
	assert.EqualError(t, err, "Empty queue")
	assert.Nil(t, read)

	// when
	queue.Push(entry)
	read, err = queue.Peek()

	// then
	assert.NoError(t, err)
	assert.Equal(t, pop(queue), read)
	assert.Equal(t, entry, read)

	// when
	read, err = queue.Peek()

	// then
	assert.EqualError(t, err, "Empty queue")
	assert.Nil(t, read)
}

func TestReuseAvailableSpace(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(100, 0, false)

	// when
	queue.Push(blob('a', 70))
	queue.Push(blob('b', 20))
	queue.Pop()
	queue.Push(blob('c', 20))

	// then
	assert.Equal(t, 100, queue.Capacity())
	assert.Equal(t, blob('b', 20), pop(queue))
}

func TestAllocateAdditionalSpace(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(11, 0, false)

	// when
	queue.Push([]byte("hello1"))
	queue.Push([]byte("hello2"))

	// then
	assert.Equal(t, 22, queue.Capacity())
}

func TestAllocateAdditionalSpaceForInsufficientFreeFragmentedSpaceWhereHeadIsBeforeTail(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(25, 0, false)

	// when
	queue.Push(blob('a', 3)) // header + entry + left margin = 8 bytes
	queue.Push(blob('b', 6)) // additional 10 bytes
	queue.Pop()              // space freed, 7 bytes available at the beginning
	queue.Push(blob('c', 6)) // 10 bytes needed, 14 available but not in one segment, allocate additional memory

	// then
	assert.Equal(t, 50, queue.Capacity())
	assert.Equal(t, blob('b', 6), pop(queue))
	assert.Equal(t, blob('c', 6), pop(queue))
}

func TestUnchangedEntriesIndexesAfterAdditionalMemoryAllocationWhereHeadIsBeforeTail(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(25, 0, false)

	// when
	queue.Push(blob('a', 3))                   // header + entry + left margin = 8 bytes
	index, _ := queue.Push(blob('b', 6))       // additional 10 bytes
	queue.Pop()                                // space freed, 7 bytes available at the beginning
	newestIndex, _ := queue.Push(blob('c', 6)) // 10 bytes needed, 14 available but not in one segment, allocate additional memory

	// then
	assert.Equal(t, 50, queue.Capacity())
	assert.Equal(t, blob('b', 6), get(queue, index))
	assert.Equal(t, blob('c', 6), get(queue, newestIndex))
}

func TestAllocateAdditionalSpaceForInsufficientFreeFragmentedSpaceWhereTailIsBeforeHead(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(100, 0, false)

	// when
	queue.Push(blob('a', 70)) // header + entry + left margin = 75 bytes
	queue.Push(blob('b', 10)) // 75 + 10 + 4 = 89 bytes
	queue.Pop()               // space freed at the beginning
	queue.Push(blob('c', 30)) // 34 bytes used at the beginning, tail pointer is before head pointer
	queue.Push(blob('d', 40)) // 44 bytes needed but no available in one segment, allocate new memory

	// then
	assert.Equal(t, 200, queue.Capacity())
	assert.Equal(t, blob('c', 30), pop(queue))
	// empty blob fills space between tail and head,
	// created when additional memory was allocated,
	// it keeps current entries indexes unchanged
	assert.Equal(t, blob(0, 36), pop(queue))
	assert.Equal(t, blob('b', 10), pop(queue))
	assert.Equal(t, blob('d', 40), pop(queue))
}

func TestUnchangedEntriesIndexesAfterAdditionalMemoryAllocationWhereTailIsBeforeHead(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(100, 0, false)

	// when
	queue.Push(blob('a', 70))                   // header + entry + left margin = 75 bytes
	index, _ := queue.Push(blob('b', 10))       // 75 + 10 + 4 = 89 bytes
	queue.Pop()                                 // space freed at the beginning
	queue.Push(blob('c', 30))                   // 34 bytes used at the beginning, tail pointer is before head pointer
	newestIndex, _ := queue.Push(blob('d', 40)) // 44 bytes needed but no available in one segment, allocate new memory

	// then
	assert.Equal(t, 200, queue.Capacity())
	assert.Equal(t, blob('b', 10), get(queue, index))
	assert.Equal(t, blob('d', 40), get(queue, newestIndex))
}

func TestAllocateAdditionalSpaceForValueBiggerThanInitQueue(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(11, 0, false)

	// when
	queue.Push(blob('a', 100))

	// then
	assert.Equal(t, blob('a', 100), pop(queue))
	assert.Equal(t, 230, queue.Capacity())
}

func TestAllocateAdditionalSpaceForValueBiggerThanQueue(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(21, 0, false)

	// when
	queue.Push(make([]byte, 2))
	queue.Push(make([]byte, 2))
	queue.Push(make([]byte, 100))

	// then
	queue.Pop()
	queue.Pop()
	assert.Equal(t, make([]byte, 100), pop(queue))
	assert.Equal(t, 250, queue.Capacity())
}

func TestPopWholeQueue(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(13, 0, false)

	// when
	queue.Push([]byte("a"))
	queue.Push([]byte("b"))
	queue.Pop()
	queue.Pop()
	queue.Push([]byte("c"))

	// then
	assert.Equal(t, 13, queue.Capacity())
	assert.Equal(t, []byte("c"), pop(queue))
}

func TestGetEntryFromIndex(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(20, 0, false)

	// when
	queue.Push([]byte("a"))
	index, _ := queue.Push([]byte("b"))
	queue.Push([]byte("c"))
	result, _ := queue.Get(index)

	// then
	assert.Equal(t, []byte("b"), result)
}

func TestGetEntryFromInvalidIndex(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(1, 0, false)
	queue.Push([]byte("a"))

	// when
	result, err := queue.Get(0)

	// then
	assert.Nil(t, result)
	assert.EqualError(t, err, "Index must be grater than zero. Invalid index.")
}

func TestGetEntryFromIndexOutOfRange(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(1, 0, false)
	queue.Push([]byte("a"))

	// when
	result, err := queue.Get(42)

	// then
	assert.Nil(t, result)
	assert.EqualError(t, err, "Index out of range")
}

func TestGetEntryFromEmptyQueue(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(13, 0, false)

	// when
	result, err := queue.Get(1)

	// then
	assert.Nil(t, result)
	assert.EqualError(t, err, "Empty queue")
}

func TestMaxSizeLimit(t *testing.T) {
	t.Parallel()

	// given
	queue := NewBytesQueue(30, 50, false)

	// when
	queue.Push(blob('a', 25))
	queue.Push(blob('b', 5))
	capacity := queue.Capacity()
	_, err := queue.Push(blob('c', 15))

	// then
	assert.Equal(t, 50, capacity)
	assert.EqualError(t, err, "Full queue. Maximum size limit reached.")
	assert.Equal(t, blob('a', 25), pop(queue))
	assert.Equal(t, blob('b', 5), pop(queue))
}

func pop(queue *BytesQueue) []byte {
	entry, err := queue.Pop()
	if err != nil {
		panic(err)
	}
	return entry
}

func get(queue *BytesQueue, index int) []byte {
	entry, err := queue.Get(index)
	if err != nil {
		panic(err)
	}
	return entry
}

func blob(char byte, len int) []byte {
	b := make([]byte, len)
	for index := range b {
		b[index] = char
	}
	return b
}
