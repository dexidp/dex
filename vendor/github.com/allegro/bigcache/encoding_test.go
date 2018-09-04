package bigcache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEncodeDecode(t *testing.T) {
	// given
	now := uint64(time.Now().Unix())
	hash := uint64(42)
	key := "key"
	data := []byte("data")
	buffer := make([]byte, 100)

	// when
	wrapped := wrapEntry(now, hash, key, data, &buffer)

	// then
	assert.Equal(t, key, readKeyFromEntry(wrapped))
	assert.Equal(t, hash, readHashFromEntry(wrapped))
	assert.Equal(t, now, readTimestampFromEntry(wrapped))
	assert.Equal(t, data, readEntry(wrapped))
	assert.Equal(t, 100, len(buffer))
}

func TestAllocateBiggerBuffer(t *testing.T) {
	//given
	now := uint64(time.Now().Unix())
	hash := uint64(42)
	key := "1"
	data := []byte("2")
	buffer := make([]byte, 1)

	// when
	wrapped := wrapEntry(now, hash, key, data, &buffer)

	// then
	assert.Equal(t, key, readKeyFromEntry(wrapped))
	assert.Equal(t, hash, readHashFromEntry(wrapped))
	assert.Equal(t, now, readTimestampFromEntry(wrapped))
	assert.Equal(t, data, readEntry(wrapped))
	assert.Equal(t, 2+headersSizeInBytes, len(buffer))
}
