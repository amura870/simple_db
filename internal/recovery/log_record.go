package recovery

import "example.com/simpledb/internal/file"

type Op int32

const (
	Checkpoint Op = iota
	Start
	Commit
	Rollback
	SetInt
	SetString
)

type UndoTarget interface {
	RestoreInt(file.BlockId, int, int32) error
	RestoreString(file.BlockId, int, string) error
}

// 回復処理から共通の方法で扱うためのインターフェース
type LogRecord interface { // 回復マネージャーが「どの記録を取り消すか」だけ判断し、Howはこのインターフェースを通してUndoTargetに任せる
	Op() Op                // ログレコードの種類を返す
	TxNumber() int         // ログレコードが属するトランザクション番号を返す。CHECKPOINTの場合は-1を返す
	Encode() []byte        // ログレコードをバイト列に変換する.
	Undo(UndoTarget) error // ログレコードの内容を元に、UndoTargetに対して更新前の状態に戻すよう指示する
}

// START、COMMIT、ROLLBACK、CHECKPOINTを共通の型で表現する。
// ControlRecordは、トランザクションの制御操作を表すログレコード
type ControlRecord struct {
	Kind Op
	TxID int
}

func (r ControlRecord) Op() Op { return r.Kind }
func (r ControlRecord) TxNumber() int {
	if r.Kind == Checkpoint {
		return -1
	}
	return r.TxID
}
func (r ControlRecord) Undo(UndoTarget) error { return nil }
func (r ControlRecord) Encode() []byte {
	data := putInt(nil, int32(r.Kind))
	if r.Kind != Checkpoint {
		data = putInt(data, int32(r.TxID))
	}
	return data // イメージ： [Op(4バイト)][TxID(4バイト)] ただし、OpがCHECKPOINTの場合はTxIDは不要
}

// Intを更新するログレコード
type SetIntRecord struct {
	TxID     int
	Block    file.BlockId
	Offset   int
	OldValue int32 // Undo-onlyなので更新後の値は不要。このレポジトリでは、数値をint32で扱うことにしたはず
}

func (r SetIntRecord) Op() Op {
	return SetInt
}

func (r SetIntRecord) TxNumber() int {
	return r.TxID
}

func (r SetIntRecord) Encode() []byte {
	data := updateHeader(SetInt, r.TxID, r.Block, r.Offset)
	return putInt(data, r.OldValue)
}

func (r SetIntRecord) Undo(tx UndoTarget) error {
	return tx.RestoreInt(r.Block, r.Offset, r.OldValue)
}

// Stringを更新するログレコード
type SetStringRecord struct {
	TxID     int
	Block    file.BlockId
	Offset   int
	OldValue string
}

func (r SetStringRecord) Op() Op {
	return SetString
}

func (r SetStringRecord) TxNumber() int {
	return r.TxID
}

func (r SetStringRecord) Encode() []byte {
	data := updateHeader(SetString, r.TxID, r.Block, r.Offset)
	return putString(data, r.OldValue)
}

func (r SetStringRecord) Undo(tx UndoTarget) error {
	return tx.RestoreString(r.Block, r.Offset, r.OldValue)
}

var _ LogRecord = ControlRecord{}   // ControlRecordがLogRecordインターフェースを実装していることを確認するためのコンパイル時チェック
var _ LogRecord = SetIntRecord{}    // SetIntRecordがLogRecordインターフェースを実装していることを確認するためのコンパイル時チェック
var _ LogRecord = SetStringRecord{} // SetStringRecordがLogRecordインターフェースを実装していることを確認するためのコンパイル時チェック
