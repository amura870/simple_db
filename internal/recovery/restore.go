package recovery

import (
	"example.com/simpledb/internal/buffer"
	"example.com/simpledb/internal/file"
)

// UndoTargetを実装する構造体で、バッファマネージャーを通してディスクに更新前の値を反映する
type BufferRestorer struct {
	bm    *buffer.BufferMgr
	owner int
}

func NewBufferRestorer(bm *buffer.BufferMgr, owner int) *BufferRestorer {
	return &BufferRestorer{
		bm:    bm,
		owner: owner,
	}
}

// ログから取り出した更新前の値を、バッファマネージャーを通してディスクに反映する
func (r *BufferRestorer) RestoreInt(blk file.BlockId, offset int, value int32) error {
	buff, err := r.bm.Pin(blk)
	if err != nil {
		return err
	}
	defer r.bm.Unpin(buff)

	buff.Contents().SetInt(offset, value)
	buff.SetModified(r.owner, -1)
	return nil
}

func (r *BufferRestorer) RestoreString(blk file.BlockId, offset int, value string) error {
	buff, err := r.bm.Pin(blk)
	if err != nil {
		return err
	}
	defer r.bm.Unpin(buff)

	buff.Contents().SetString(offset, value)
	buff.SetModified(r.owner, -1)
	return nil
}

var _ UndoTarget = (*BufferRestorer)(nil) // BufferRestorerがUndoTargetを実装していることをコンパイル時に確認するためのダミー変数
