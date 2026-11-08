package tx

import (
	"fmt"
	"math"
	"sync/atomic"

	"example.com/simpledb/internal/buffer"
	"example.com/simpledb/internal/concurrency"
	"example.com/simpledb/internal/file"
	"example.com/simpledb/internal/log"
	"example.com/simpledb/internal/recovery"
)

const endOfFile = -1

var nextTxNum atomic.Int64

// 1つのTransactionは1つのgoroutineから使い、完了後は再利用しない。
type Transaction struct {
	fm      *file.FileMgr
	bm      *buffer.BufferMgr
	rm      *recovery.RecoveryMgr
	cm      *concurrency.ConcurrencyMgr
	buffers *bufferList
	txnum   int
}

func New(fm *file.FileMgr, lm *log.LogMgr, bm *buffer.BufferMgr, table *concurrency.LockTable) (*Transaction, error) {
	n := nextTxNum.Add(1)
	if n > math.MaxInt32 {
		return nil, fmt.Errorf("transaction number exhausted")
	}
	t := &Transaction{fm: fm, bm: bm, cm: concurrency.NewConcurrencyMgr(table), buffers: newBufferList(bm), txnum: int(n)}
	restorer := recovery.NewBufferRestorer(bm, t.txnum)
	rm, err := recovery.NewRecoveryMgr(restorer, t.txnum, lm, bm)
	if err != nil {
		return nil, err
	}
	t.rm = rm
	return t, nil
}

func (t *Transaction) Commit() error {
	if err := t.rm.Commit(); err != nil {
		return err
	}
	t.cm.Release()
	t.buffers.unpinAll()
	return nil
}

func (t *Transaction) Rollback() error {
	if err := t.rm.Rollback(); err != nil {
		return err
	}
	t.cm.Release()
	t.buffers.unpinAll()
	return nil
}

// 起動時、通常のトランザクションを開始する前にだけ呼ぶ。
func (t *Transaction) Recover() error {
	t.bm.FlushAll(t.txnum)
	return t.rm.Recover()
}

func (t *Transaction) Pin(blk file.BlockId) error   { return t.buffers.pin(blk) }
func (t *Transaction) Unpin(blk file.BlockId) error { return t.buffers.unpin(blk) }

func (t *Transaction) GetInt(blk file.BlockId, offset int) (int32, error) {
	buff, err := t.buffers.get(blk)
	if err != nil {
		return 0, err
	}
	if err := t.cm.SLock(blk); err != nil {
		return 0, err
	}
	return buff.Contents().GetInt(offset), nil
}

func (t *Transaction) GetString(blk file.BlockId, offset int) (string, error) {
	buff, err := t.buffers.get(blk)
	if err != nil {
		return "", err
	}
	if err := t.cm.SLock(blk); err != nil {
		return "", err
	}
	return buff.Contents().GetString(offset), nil
}

func (t *Transaction) SetInt(blk file.BlockId, offset int, value int32, okToLog bool) error {
	buff, err := t.buffers.get(blk)
	if err != nil {
		return err
	}
	if err := t.cm.XLock(blk); err != nil {
		return err
	}
	lsn := -1
	if okToLog {
		lsn, err = t.rm.LogSetInt(buff, offset)
		if err != nil {
			return err
		}
	}
	buff.Contents().SetInt(offset, value)
	buff.SetModified(t.txnum, lsn)
	return nil
}

func (t *Transaction) SetString(blk file.BlockId, offset int, value string, okToLog bool) error {
	buff, err := t.buffers.get(blk)
	if err != nil {
		return err
	}
	if err := t.cm.XLock(blk); err != nil {
		return err
	}
	lsn := -1
	if okToLog {
		lsn, err = t.rm.LogSetString(buff, offset)
		if err != nil {
			return err
		}
	}
	buff.Contents().SetString(offset, value)
	buff.SetModified(t.txnum, lsn)
	return nil
}

func (t *Transaction) Size(filename string) (int, error) {
	blk := *file.NewBlockId(filename, endOfFile)
	if err := t.cm.SLock(blk); err != nil {
		return 0, err
	}
	return t.fm.Length(filename), nil
}

func (t *Transaction) Append(filename string) (file.BlockId, error) {
	blk := *file.NewBlockId(filename, endOfFile)
	if err := t.cm.XLock(blk); err != nil {
		return file.BlockId{}, err
	}
	return t.fm.Append(filename), nil
}

func (t *Transaction) BlockSize() int      { return t.fm.BlockSize() }
func (t *Transaction) AvailableBuffs() int { return t.bm.Available() }
