package buffer

import (
	"example.com/simpledb/internal/file"
	"example.com/simpledb/internal/log"
)

type Buffer struct {
	fm       *file.FileMgr
	lm       *log.LogMgr
	contents *file.Page
	blk      *file.BlockId

	pins  int // 現在のpin回数. 同時に複数のクライアントから接続されると、2以上になる.
	txnum int // 変更したトランザクション番号. -1なら未変更
	lsn   int // 対応するログのLSN.
}

func NewBuffer(fm *file.FileMgr, lm *log.LogMgr) *Buffer {
	return &Buffer{
		fm:       fm,
		lm:       lm,
		contents: file.NewPage(fm.BlockSize()),
		txnum:    -1,
		lsn:      -1,
	}
}

func (b *Buffer) Contents() *file.Page {
	return b.contents
}

func (b *Buffer) Block() *file.BlockId {
	return b.blk
}

func (b *Buffer) SetModified(txnum int, lsn int) {
	b.txnum = txnum
	if lsn >= 0 {
		b.lsn = lsn
	}
}

func (b *Buffer) IsPinned() bool {
	return b.pins > 0 // １つ以上のクライアントからpinされている
}

// ページを変更したトランザクションの番号を返すメソッド
func (b *Buffer) ModifyingTx() int {
	return b.txnum
}

// 現在のブロックをメモリから追い出して、新しいブロックの情報を保持する
func (b *Buffer) assignToBlock(blk file.BlockId) {
	// 今まで保持していたブロックの変更を保存する
	b.flush()

	b.blk = &blk
	b.fm.Read(blk, b.contents)
	b.pins = 0
}

// 変更内容をディスクに書き出す
func (b *Buffer) flush() {
	// 変更がない場合は、ディスクへの書き出しは不要
	if b.txnum < 0 {
		return
	}

	// ログを保存してから、データ本体を保存
	b.lm.Flush(b.lsn)
	b.fm.Write(*b.blk, b.contents)

	// 書き出しが完了したので、未変更の状態に戻す
	b.txnum = -1
}

func (b *Buffer) pin() {
	b.pins++
}

func (b *Buffer) unpin() {
	b.pins--
}
