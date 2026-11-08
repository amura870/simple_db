package concurrency

import (
	"errors"
	"sync"
	"time"

	"example.com/simpledb/internal/file"
)

const maxWait = 10 * time.Second

var ErrLockTimeout = errors.New("lock wait time out")

// ブロックのロック状況を管理する. 全トランザクションで共有する
type LockTable struct {
	mu    sync.Mutex
	cond  *sync.Cond
	locks map[file.BlockId]int // 0: ロックなし, 正の値: 共有ロック数, 負の値: 排他ロック
}

func NewLockTable() *LockTable {
	lt := &LockTable{
		locks: make(map[file.BlockId]int),
	}
	lt.cond = sync.NewCond(&lt.mu)
	return lt
}

// Shared Lock（共有ロック）を取得する. Read操作のためのロックで、複数のトランザクションが同時に取得可能
func (lt *LockTable) sLock(blk file.BlockId) error {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if err := lt.waitUntil(func() bool {
		return lt.locks[blk] >= 0 // XロックがなければOK
	}); err != nil {
		return err
	}
	lt.locks[blk]++
	return nil
}

// Exclusive Lock（排他ロック）を取得する. Write操作のためのロックで、同時に取得できるトランザクションは1つだけ
func (lt *LockTable) xLock(blk file.BlockId) error {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	if err := lt.waitUntil(func() bool {
		return lt.locks[blk] == 1 // 自分自身のSロックしかなければOK
	}); err != nil {
		return err
	}
	lt.locks[blk] = -1
	return nil
}

func (lt *LockTable) unlock(blk file.BlockId) {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	n := lt.locks[blk]
	if n == 0 {
		return
	}
	if n > 1 {
		lt.locks[blk] = n - 1
	} else {
		delete(lt.locks, blk)
	}
	// Sロックが2個から1個に減った場合も、
	// Xロックへの昇格を待つトランザクションを起こす。
	lt.cond.Broadcast()
}

// lt.muを取得済みで呼ぶ.
func (lt *LockTable) waitUntil(ready func() bool) error {
	if ready() {
		return nil
	}

	timedOut := false
	timer := time.AfterFunc(maxWait, func() {
		lt.mu.Lock()
		defer lt.mu.Unlock()

		timedOut = true
		lt.cond.Broadcast()
	})
	defer timer.Stop()

	for {
		lt.cond.Wait()
		if timedOut {
			return ErrLockTimeout
		}
		if ready() {
			return nil
		}
	}
}
