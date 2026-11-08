package buffer

import (
	"errors"
	"sync"
	"time"

	"example.com/simpledb/internal/file"
	"example.com/simpledb/internal/log"
)

const maxWait = 10 * time.Second

var ErrBufferTimeout = errors.New("buffer wait time out")

type BufferMgr struct {
	mu   sync.Mutex
	cond *sync.Cond

	bufferPool   []*Buffer
	numAvailable int
}

func NewBufferMgr(fm *file.FileMgr, lm *log.LogMgr, numBuffers int) *BufferMgr {
	bm := &BufferMgr{
		bufferPool:   make([]*Buffer, numBuffers),
		numAvailable: numBuffers,
	}
	bm.cond = sync.NewCond(&bm.mu)

	for i := range bm.bufferPool {
		bm.bufferPool[i] = NewBuffer(fm, lm)
	}
	return bm
}

func (bm *BufferMgr) Available() int {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	return bm.numAvailable
}

// 指定したトランザクションによる全ての変更をディスクに保存する
func (bm *BufferMgr) FlushAll(txnum int) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	for _, buff := range bm.bufferPool {
		if buff.ModifyingTx() == txnum {
			buff.flush()
		}
	}
}

func (bm *BufferMgr) Unpin(buff *Buffer) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	buff.unpin()
	if !buff.IsPinned() {
		bm.numAvailable++
		bm.cond.Broadcast()
	}
}

func (bm *BufferMgr) Pin(blk file.BlockId) (*Buffer, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// すぐに取得できれば、タイマーを作らず返す
	if buff := bm.tryToPin(blk); buff != nil {
		return buff, nil
	}

	timedOut := false
	timer := time.AfterFunc(maxWait, func() {
		bm.mu.Lock()
		defer bm.mu.Unlock()

		timedOut = true
		bm.cond.Broadcast()
	})
	defer timer.Stop()

	for {
		bm.cond.Wait()
		if timedOut {
			return nil, ErrBufferTimeout
		}
		if buff := bm.tryToPin(blk); buff != nil {
			return buff, nil
		}
	}
}

// 以下の内部メソッドはbm.mu.Lock()を取得済みの状態で呼ぶこと
func (bm *BufferMgr) tryToPin(blk file.BlockId) *Buffer {
	buff := bm.findExistingBuffer(blk)

	if buff == nil {
		buff = bm.chooseUnpinnedBuffer()
		if buff == nil {
			return nil
		}
		buff.assignToBlock(blk)
	}

	if !buff.IsPinned() {
		bm.numAvailable--
	}
	buff.pin()
	return buff
}

func (bm *BufferMgr) findExistingBuffer(blk file.BlockId) *Buffer {
	for _, buff := range bm.bufferPool {
		b := buff.Block()
		if b != nil && b.Equals(blk) {
			return buff
		}
	}
	return nil
}

func (bm *BufferMgr) chooseUnpinnedBuffer() *Buffer {
	for _, buff := range bm.bufferPool {
		if !buff.IsPinned() {
			return buff
		}
	}
	return nil
}
