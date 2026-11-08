package record

import "fmt"

// 特定のテーブル内での物理的な位置
type RID struct {
	BlockNumber int
	Slot        int
}

func (r RID) String() string {
	return fmt.Sprintf("[%d %d]", r.BlockNumber, r.Slot)
}