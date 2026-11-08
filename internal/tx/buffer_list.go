package tx

import (
	"fmt"

	"example.com/simpledb/internal/buffer"
	"example.com/simpledb/internal/file"
)

type bufferList struct {
	bm      *buffer.BufferMgr
	buffers map[file.BlockId]*buffer.Buffer
	pins    map[file.BlockId]int
}

func newBufferList(bm *buffer.BufferMgr) *bufferList {
	return &bufferList{bm: bm, buffers: make(map[file.BlockId]*buffer.Buffer), pins: make(map[file.BlockId]int)}
}

func (bl *bufferList) pin(blk file.BlockId) error {
	buff, err := bl.bm.Pin(blk)
	if err != nil {
		return err
	}
	bl.buffers[blk] = buff
	bl.pins[blk]++
	return nil
}

func (bl *bufferList) get(blk file.BlockId) (*buffer.Buffer, error) {
	buff, ok := bl.buffers[blk]
	if !ok {
		return nil, fmt.Errorf("block is not pinned: %s", blk.ToString())
	}
	return buff, nil
}

func (bl *bufferList) unpin(blk file.BlockId) error {
	buff, err := bl.get(blk)
	if err != nil {
		return err
	}
	bl.bm.Unpin(buff)
	bl.pins[blk]--
	if bl.pins[blk] == 0 {
		delete(bl.pins, blk)
		delete(bl.buffers, blk)
	}
	return nil
}

func (bl *bufferList) unpinAll() {
	for blk, count := range bl.pins {
		for i := 0; i < count; i++ {
			bl.bm.Unpin(bl.buffers[blk])
		}
	}
	clear(bl.pins)
	clear(bl.buffers)
}
