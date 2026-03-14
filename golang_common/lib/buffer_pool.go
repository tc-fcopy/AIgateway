package lib

import (
	"bytes"
	"sync"
)

const maxPooledBufferSize = 64 * 1024

var bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

func borrowBuffer() *bytes.Buffer {
	b := bufferPool.Get().(*bytes.Buffer)
	b.Reset()
	return b
}

func releaseBuffer(b *bytes.Buffer) {
	if b == nil {
		return
	}
	if b.Cap() > maxPooledBufferSize {
		return
	}
	b.Reset()
	bufferPool.Put(b)
}
