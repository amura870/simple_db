package recovery

import (
	"fmt"
	"math"

	"example.com/simpledb/internal/buffer"
	"example.com/simpledb/internal/log"
)

type RecoveryMgr struct {
	tx    UndoTarget
	txnum int
	lm    *log.LogMgr
	bm    *buffer.BufferMgr
}

func NewRecoveryMgr(tx UndoTarget, txnum int, lm *log.LogMgr, bm *buffer.BufferMgr) (*RecoveryMgr, error) {
	if txnum <= 0 || int64(txnum) > math.MaxInt32 {
		return nil, fmt.Errorf("invalid transaction number: %d", txnum)
	}
	rm := &RecoveryMgr{
		tx:    tx,
		txnum: txnum,
		lm:    lm,
		bm:    bm,
	}
	// 初期化時にSTARTログを追加する
	_, err := rm.appendControl(Start)
	if err != nil {
		return  nil, err
	}
	return rm, nil
}


// Intの更新を行う
func (rm *RecoveryMgr) LogSetInt(buff *buffer.Buffer, offset int) (int, error) {
	record := SetIntRecord{
		TxID:     rm.txnum,
		Block:    *buff.Block(),
		Offset:   offset,
		OldValue: buff.Contents().GetInt(offset),
	}
	return rm.lm.Append(record.Encode())
}

// Stringの更新を行う
func (rm *RecoveryMgr) LogSetString(buff *buffer.Buffer, offset int) (int, error) {
	record := SetStringRecord{
		TxID:     rm.txnum,
		Block:    *buff.Block(),
		Offset:   offset,
		OldValue: buff.Contents().GetString(offset),
	}
	return rm.lm.Append(record.Encode())
}

// コミットを行う. データを保存してからコミットログを書く。
func (rm *RecoveryMgr) Commit() error {
	return rm.finish(Commit)
}

// ロールバックを行う。トランザクションの変更を全て元に戻す。
func (rm *RecoveryMgr) Rollback() error {
	iter := rm.lm.Iterator()
	for {
		data, ok := iter.Next()
		if !ok {
			return fmt.Errorf("START not found for transaction %d", rm.txnum)
		}
		record, err := Decode(data)
		if err != nil {
			return err
		}
		if record.TxNumber() != rm.txnum {
			continue
		}
		// STARTまで遡ったらロールバック完了
		if record.Op() == Start {
			break
		}
		if err := record.Undo(rm.tx); err != nil {
			return err
		}
	}
	return rm.finish(Rollback)
}

func (rm *RecoveryMgr) Recover() error {
	finished := make(map[int]struct{})
	iter := rm.lm.Iterator()
	for {
		data, ok := iter.Next()
		if !ok {
			break
		}
		record, err := Decode(data)
		if err != nil {
			return err
		}
		// チェックポイント以前のログは回復不要
		if record.Op() == Checkpoint {
			break
		}

		switch record.Op() {
		case Commit, Rollback:
			finished[record.TxNumber()] = struct{}{}
		default:
			if _, done := finished[record.TxNumber()]; !done {
				if err := record.Undo(rm.tx); err != nil {
					return err
				}
			}
		}
	}
	return rm.finish(Checkpoint)
}

func (rm *RecoveryMgr) appendControl(op Op) (int, error) {
	return rm.lm.Append(ControlRecord{Kind: op, TxID: rm.txnum}.Encode())
}

func (rm *RecoveryMgr) finish(op Op) error {
	rm.bm.FlushAll(rm.txnum) // トランザクションの変更をディスクに書き出す

	record := ControlRecord{
		Kind: op,
		TxID: rm.txnum,
	}
	lsn, err := rm.lm.Append(record.Encode())
	if err != nil {
		return err
	}

	rm.lm.Flush(lsn) // コミットログをディスクに書き出す
	return nil
}
