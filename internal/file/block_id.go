package file

import "fmt"

type BlockId struct {
	filename string
	number int
}

func NewBlockId(filename string, number int) *BlockId {
	return &BlockId{
		filename: filename,
		number: number,
	}
}

func (b *BlockId) Filename() string {
	return b.filename
}

func (b *BlockId) Number() int {
	return b.number
}

func (b *BlockId) Equals(b2 BlockId) bool {
	return b.filename == b2.filename && b.number == b2.number
}

func (b *BlockId) ToString() string {
	return "[file " + b.filename + ", block " + fmt.Sprint(b.number) + "]"
}
