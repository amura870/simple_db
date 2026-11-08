package log

import "example.com/simpledb/internal/file"

type LogIterator struct {
	fm              *file.FileMgr
	page            *file.Page
	block           file.BlockId
	currentPosition int
}

func NewLogIterator(
	fm *file.FileMgr, block file.BlockId,
) *LogIterator {
	page := file.NewPage(fm.BlockSize())
	fm.Read(block, page)
	return &LogIterator{
		fm:              fm,
		page:            page,
		block:           block,
		currentPosition: int(page.GetInt(0)), // ページ先頭の境界位置を取得
	}
}

func (it *LogIterator) Next() ([]byte, bool) {
	// ページの末尾なら、１つ前のブロックに戻る
	for it.currentPosition == it.fm.BlockSize() {
		if it.block.Number() == 0 {
			return nil, false
		}

		previousBlk := *file.NewBlockId(it.block.Filename(), it.block.Number()-1)
		it.fm.Read(previousBlk, it.page)
		it.block = previousBlk
		it.currentPosition = int(it.page.GetInt(0))
	}
	record := it.page.GetBytes(it.currentPosition)

	// レコードの長さを表す4バイトと、レコード本体の分進む
	it.currentPosition += intSize + len(record)
	return record, true
}
