package concurrency

import "example.com/simpledb/internal/file"

type lockMode uint8

const (
	shared lockMode = iota + 1
	exclusive
)

type ConcurrencyMgr struct {
	table *LockTable
	locks map[file.BlockId]lockMode
}

func NewConcurrencyMgr(table *LockTable) *ConcurrencyMgr {
	return &ConcurrencyMgr{
		table: table,
		locks: make(map[file.BlockId]lockMode),
	}
}

func (cm *ConcurrencyMgr) SLock(blk file.BlockId) error {
	if _, held := cm.locks[blk]; held {
		return nil
	}

	if err := cm.table.sLock(blk); err != nil {
		return err
	}
	cm.locks[blk] = shared
	return nil
}

func (cm *ConcurrencyMgr) XLock(blk file.BlockId) error {
	if cm.locks[blk] == exclusive {
		return nil
	}

	// まずSロックを取得
	if err := cm.SLock(blk); err != nil {
		return err
	}

	if err := cm.table.xLock(blk); err != nil {
		return err
	}

	cm.locks[blk] = exclusive
	return nil
}

func (cm *ConcurrencyMgr) Release() {
	for blk := range cm.locks {
		cm.table.unlock(blk)
		delete(cm.locks, blk)
	}
}
