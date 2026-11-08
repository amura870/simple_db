package record

import (
	"fmt"
	"unicode/utf8"

	"example.com/simpledb/internal/file"
	"example.com/simpledb/internal/tx"
)

// Simple DBでは4byteずつしか読めないので、boolではなくint32にしている
const (
	Empty int32 = 0
	Used  int32 = 1
)

// あるブロックに保存されたレコードを操作する.スロット番号とフィールド名で操作
type RecordPage struct {
	tx     *tx.Transaction
	block  file.BlockId
	layout *Layout
	closed bool // このRecordPageが使用を終了したか
}

func NewRecordPage(t *tx.Transaction, block file.BlockId, layout *Layout) (*RecordPage, error) {
	if layout.SlotSize() > t.BlockSize() {
		return nil, fmt.Errorf("slot does not fit in block")
	}
	if err := t.Pin(block); err != nil {
		return nil, err
	}
	return &RecordPage{tx: t, block: block, layout: layout}, nil
}

// トランザクションのCommit/Rollbackより前に呼ぶ
func (p *RecordPage) Close() error {
	if p.closed {
		return nil // すでに終了済み. Unpinを二重実行しないためにここで確認
	}
	if err := p.tx.Unpin(p.block); err != nil {
		return err
	}
	p.closed = true
	return nil
}

func (p *RecordPage) Block() file.BlockId {
	return p.block
}

// このブロックに格納できるレコード数
func (p *RecordPage) capacity() int {
	return p.tx.BlockSize() / p.layout.SlotSize()
}

// 入力スロットのブロック内のオフセットを計算する
func (p *RecordPage) offset(slot int) int {
	return slot * p.layout.SlotSize()
}

func (p *RecordPage) checkSlot(slot int) error {
	if p.closed {
		return fmt.Errorf("record page is closed")
	}
	if slot < 0 || slot >= p.capacity() {
		return fmt.Errorf("invalid slot: %d", slot)
	}
	return nil
}

// slot番号とフィールド名からブロック内のオフセットを計算
func (p *RecordPage) fieldPos(slot int, name string, want FieldType) (int, error) {
	if err := p.checkSlot(slot); err != nil {
		return 0, err
	}
	f, ok := p.layout.fields[name]
	if !ok {
		return 0, fmt.Errorf("unknown field: %s", name)
	}
	if f.Type != want {
		return 0, fmt.Errorf("wrong field type: %d", want)
	}
	return p.offset(slot) + f.offset, nil
}

func (p *RecordPage) GetInt(slot int, name string) (int32, error) {
	pos, err := p.fieldPos(slot, name, Integer) // 保存位置の計算
	if err != nil {
		return 0, err
	}
	return p.tx.GetInt(p.block, pos) // 実際の更新処理はTransactionを使う
}

func (p *RecordPage) SetInt(slot int, name string, value int32) error {
	pos, err := p.fieldPos(slot, name, Integer)
	if err != nil {
		return err
	}
	return p.tx.SetInt(p.block, pos, value, true)
}

func (p *RecordPage) GetString(slot int, name string) (string, error) {
	pos, err := p.fieldPos(slot, name, Varchar)
	if err != nil {
		return "", err
	}
	return p.tx.GetString(p.block, pos)
}

func (p *RecordPage) SetString(slot int, name, value string) error {
	pos, err := p.fieldPos(slot, name, Varchar)
	if err != nil {
		return err
	}
	// 内容の検査
	f := p.layout.fields[name]
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > f.Length {
		return fmt.Errorf("invalid or too long string: %s", name)
	}
	return p.tx.SetString(p.block, pos, value, true)
}

// 新しいブロックの初期化専用。既存データの消去には使わない。
func (p *RecordPage) Format() error {
	if p.closed {
		return fmt.Errorf("record page is closed")
	}
	for slot := 0; slot < p.capacity(); slot++ {
		base := p.offset(slot)
		if err := p.tx.SetInt(p.block, base, Empty, false); err != nil {
			return err
		}
		for _, name := range p.layout.order {
			f := p.layout.fields[name]
			pos := base + f.offset
			var err error
			if f.Type == Integer {
				err = p.tx.SetInt(p.block, pos, 0, false)
			} else {
				err = p.tx.SetString(p.block, pos, "", false)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Slot以降で、目的のflagを持つスロットを検索する. flagは使用中or空き
func (p *RecordPage) searchAfter(slot int, flag int32) (int, error) {
	if p.closed {
		return -1, fmt.Errorf("record page is closed")
	}
	if slot < -1 {
		return -1, fmt.Errorf("invalid starting slot: %d", slot)
	}
	for slot < p.capacity()-1 {
		slot++
		value, err := p.tx.GetInt(p.block, p.offset(slot))
		if err != nil {
			return -1, err
		}
		if value == flag {
			return slot, nil
		}
	}
	return -1, nil // 見つからないことはエラーではない。
}

// 次の使用中のレコードに移動
func (p *RecordPage) NextAfter(slot int) (int, error) {
	return p.searchAfter(slot, Used) // 使用中のスロットを探索
}

// slotのフラグを更新
func (p *RecordPage) setFlag(slot int, flag int32) error {
	if err := p.checkSlot(slot); err != nil {
		return err
	}
	return p.tx.SetInt(p.block, p.offset(slot), flag, true)
}

func (p *RecordPage) InsertAfter(slot int) (int, error) {
	next, err := p.searchAfter(slot, Empty)
	if err != nil || next < 0 {
		return next, err
	}
	if err := p.setFlag(next, Used); err != nil {
		return -1, err
	}
	return next, nil
}

func (p *RecordPage) Delete(slot int) error {
	return p.setFlag(slot, Empty)
}
