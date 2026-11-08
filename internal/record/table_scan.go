package record

import (
	"errors"
	"fmt"

	"example.com/simpledb/internal/file"
	"example.com/simpledb/internal/tx"
)

// 複数のブロックに保存されたレコードを1つのテーブルとして読み書きする
type TableScan struct {
	tx       *tx.Transaction
	layout   *Layout
	filename string

	rp   *RecordPage
	slot int

	current bool // 現在位置に操作対象のレコードがあるか
	eof     bool // ファイル末尾
	closed  bool
}

func NewTableScan(t *tx.Transaction, table string, layout *Layout) (*TableScan, error) {
	if layout == nil || layout.SlotSize() <= 0 || layout.SlotSize() > t.BlockSize() {
		return nil, fmt.Errorf("invalid slot size for this block")
	}
	s := &TableScan{tx: t, layout: layout, filename: table + ".tbl", slot: -1}
	size, err := t.Size(s.filename)
	if err != nil {
		return nil, err
	}
	// ファイルのブロック数が0の場合は、新しいブロックを追加して初期化
	if size == 0 {
		err = s.moveToNewBlock()
	} else {
		err = s.moveToBlock(0)
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *TableScan) ensureOpen() error {
	if s.closed || s.rp == nil {
		return fmt.Errorf("table scan is not open")
	}
	return nil
}

func (s *TableScan) ensureCurrent() error {
	if err := s.ensureOpen(); err != nil {
		return err
	}
	if !s.current {
		return fmt.Errorf("no current record")
	}
	return nil
}

// 現在のページを解放する
func (s *TableScan) releasePage() error {
	if s.rp != nil {
		if err := s.rp.Close(); err != nil {
			return err
		}
	}

	s.rp = nil
	s.slot = -1
	s.current = false
	return nil
}

// 指定されたブロック番号のページに移動する
func (s *TableScan) moveToBlock(number int) error {
	// 先に古いページを解放する
	if err := s.releasePage(); err != nil {
		return err
	}

	// 新しいページを作成する
	block := *file.NewBlockId(s.filename, number)
	page, err := NewRecordPage(s.tx, block, s.layout)
	if err != nil {
		return err
	}

	s.rp = page
	s.eof = false
	return nil
}

func (s *TableScan) moveToNewBlock() error {
	if err := s.releasePage(); err != nil {
		return err
	}
	block, err := s.tx.Append(s.filename)
	if err != nil {
		return err
	}
	page, err := NewRecordPage(s.tx, block, s.layout)
	if err != nil {
		return err
	}
	if err := page.Format(); err != nil {
		return errors.Join(err, page.Close())
	}
	s.rp = page
	s.eof = false
	return nil
}

func (s *TableScan) blockNumber() int {
	block := s.rp.Block()
	return block.Number()
}

func (s *TableScan) atLastBlock() (bool, error) {
	size, err := s.tx.Size(s.filename)
	if err != nil {
		return false, err
	}
	return s.blockNumber() == size-1, nil
}

func (s *TableScan) BeforeFirst() error {
	if err := s.ensureOpen(); err != nil {
		return err
	}
	return s.moveToBlock(0)
}

func (s *TableScan) Next() (bool, error) {
	if err := s.ensureOpen(); err != nil {
		return false, err
	}

	s.current = false
	if s.eof {
		return false, nil
	}

	for {
		slot, err := s.rp.NextAfter(s.slot)
		if err != nil {
			return false, err
		}
		if slot >= 0 {
			s.slot = slot
			s.current = true
			return true, nil // 次の使用中レコードが見つかった
		}

		// 次のブロックに移動する. ただし、最後のブロックの場合はEOFにする
		last, err := s.atLastBlock()
		if err != nil {
			return false, err
		}
		if last {
			s.eof = true
			return false, nil
		}
		if err := s.moveToBlock(s.blockNumber() + 1); err != nil {
			return false, err
		}
	}
}

func (s *TableScan) Insert() error {
	if err := s.ensureOpen(); err != nil {
		return err
	}
	s.current = false
	s.eof = false

	for {
		slot, err := s.rp.InsertAfter(s.slot)
		if err != nil {
			return err
		}
		if slot >= 0 {
			s.slot = slot
			s.current = true
			return nil
		}
		last, err := s.atLastBlock()
		if err != nil {
			return err
		}
		if last {
			err = s.moveToNewBlock()
		} else {
			err = s.moveToBlock(s.blockNumber() + 1)
		}
		if err != nil {
			return err
		}
	}
}

func (s *TableScan) GetInt(fieldname string) (int32, error) {
	if err := s.ensureCurrent(); err != nil {
		return 0, err
	}
	return s.rp.GetInt(s.slot, fieldname)
}

func (s *TableScan) SetInt(fieldname string, value int32) error {
	if err := s.ensureCurrent(); err != nil {
		return err
	}
	return s.rp.SetInt(s.slot, fieldname, value)
}

func (s *TableScan) GetString(fieldname string) (string, error) {
	if err := s.ensureCurrent(); err != nil {
		return "", err
	}
	return s.rp.GetString(s.slot, fieldname)
}

func (s *TableScan) SetString(fieldname, value string) error {
	if err := s.ensureCurrent(); err != nil {
		return err
	}
	return s.rp.SetString(s.slot, fieldname, value)
}

func (s *TableScan) HasField(fieldname string) bool {
	_, ok := s.layout.fields[fieldname]
	return ok
}

func (s *TableScan) Delete() error {
	if err := s.ensureCurrent(); err != nil {
		return err
	}
	if err := s.rp.Delete(s.slot); err != nil {
		return err
	}
	// 探索位置は残すので、次のNextで後続レコードへ進める。
	s.current = false
	return nil
}

func (s *TableScan) GetRID() (RID, error) {
	if err := s.ensureCurrent(); err != nil {
		return RID{}, err
	}
	return RID{BlockNumber: s.blockNumber(), Slot: s.slot}, nil
}

func (s *TableScan) MoveToRID(rid RID) error {
	if err := s.ensureOpen(); err != nil {
		return err
	}
	size, err := s.tx.Size(s.filename)
	if err != nil {
		return err
	}
	capacity := s.tx.BlockSize() / s.layout.SlotSize()
	if rid.BlockNumber < 0 || rid.BlockNumber >= size || rid.Slot < 0 || rid.Slot >= capacity {
		return fmt.Errorf("invalid RID: %s", rid)
	}
	if err := s.moveToBlock(rid.BlockNumber); err != nil {
		return err
	}
	// 指定スロット自体が使用中かを調べる。
	slot, err := s.rp.NextAfter(rid.Slot - 1)
	if err != nil {
		return err
	}
	if slot != rid.Slot {
		return fmt.Errorf("RID points to an empty slot: %s", rid)
	}
	s.slot = slot
	s.current = true
	return nil
}

// Commit/Rollbackの前に呼ぶ。複数回呼んでも安全。
func (s *TableScan) Close() error {
	if s.closed {
		return nil
	}
	if err := s.releasePage(); err != nil {
		return err
	}
	s.closed = true
	s.eof = true
	return nil
}
