// Pageクラス. メモリ上のデータの変換などの操作を実施するクラス
package file

import "encoding/binary"

const intSize = 4

type Page struct {
	data []byte
}

// 指定されたサイズの容量を持つページを作成する
func NewPage(blockSize int) *Page {
	return &Page{
		data: make([]byte, blockSize),
	}
}

func (p *Page) GetInt(offset int) int32 {
	return int32(binary.BigEndian.Uint32(p.data[offset : offset+intSize]))
}

func (p *Page) SetInt(offset int, value int32) {
	binary.BigEndian.PutUint32(p.data[offset:offset+intSize], uint32(value))
}

func (p *Page) GetBytes(offset int) []byte {
	length := int(p.GetInt(offset))
	b := make([]byte, length)
	copy(b, p.data[offset+intSize:offset+intSize+length])
	return b
}

func (p *Page) SetBytes(offset int, b []byte) {
	p.SetInt(offset, int32(len(b)))                       // offsetの位置に長さを格納
	copy(p.data[offset+intSize:offset+intSize+len(b)], b) // ４バイト先に本体を格納
}

func (p *Page) GetString(offset int) string {
	return string(p.GetBytes(offset))
}

func (p *Page) SetString(offset int, s string) {
	p.SetBytes(offset, []byte(s))
}

func MaxLength(strlen int) int {
	return intSize + strlen
}

func (p *Page) contents() []byte {
	return p.data
}
