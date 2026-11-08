package recovery

import (
	"encoding/binary"
	"fmt"
	"io"

	"example.com/simpledb/internal/file"
)

// 整数を4バイトで末尾に追加
func putInt(dst []byte, value int32) []byte {
	return binary.BigEndian.AppendUint32(dst, uint32(value))
}

// 文字列のバイト数と、その本体を末尾に追加
func putString(dst []byte, value string) []byte {
	dst = putInt(dst, int32(len(value))) // 文字列の長さを先頭に付与
	return append(dst, []byte(value)...)
}

func updateHeader(op Op, txID int, blk file.BlockId, offset int) []byte {
	data := putInt(nil, int32(op))
	data = putInt(data, int32(txID))
	data = putString(data, blk.Filename())
	data = putInt(data, int32(blk.Number()))
	return putInt(data, int32(offset))
}

type decoder struct {
	data []byte
}

func (d *decoder) integer() (int32, error) {
	if len(d.data) < 4 {
		return 0, io.ErrUnexpectedEOF
	}
	value := int32(binary.BigEndian.Uint32(d.data[:4]))
	d.data = d.data[4:]
	return value, nil
}

func (d *decoder) text() (string, error) {
	n, err := d.integer()
	if err != nil {
		return "", err
	}
	if n < 0 || int64(n) > int64(len(d.data)) {
		return "", fmt.Errorf("invalid string length: %d", n)
	}
	value := string(d.data[:int(n)])
	d.data = d.data[int(n):]
	return value, nil
}

// 整数は4バイト、文字列は「バイト数4バイト + UTF-8本体」。
// このプロジェクトのPageと同じ形式を使う。
func Decode(data []byte) (LogRecord, error) {
	d := decoder{data: data}
	rawOp, err := d.integer()
	if err != nil {
		return nil, err
	}
	op := Op(rawOp)
	var record LogRecord
	if op == Checkpoint {
		record = ControlRecord{Kind: Checkpoint}
	} else {
		if op < Start || op > SetString {
			return nil, fmt.Errorf("unknown operation: %d", op)
		}
		txID, err := d.integer()
		if err != nil {
			return nil, err
		}
		if txID <= 0 {
			return nil, fmt.Errorf("invalid transaction: %d", txID)
		}
		if op == Start || op == Commit || op == Rollback {
			record = ControlRecord{Kind: op, TxID: int(txID)}
		} else {
			filename, err := d.text()
			if err != nil {
				return nil, err
			}
			block, err := d.integer()
			if err != nil {
				return nil, err
			}
			offset, err := d.integer()
			if err != nil {
				return nil, err
			}
			if block < 0 || offset < 0 {
				return nil, fmt.Errorf("negative block or offset")
			}
			blk := *file.NewBlockId(filename, int(block))
			if op == SetInt {
				value, err := d.integer()
				if err != nil {
					return nil, err
				}
				record = SetIntRecord{int(txID), blk, int(offset), value}
			} else {
				value, err := d.text()
				if err != nil {
					return nil, err
				}
				record = SetStringRecord{int(txID), blk, int(offset), value}
			}
		}
	}
	if len(d.data) != 0 {
		return nil, fmt.Errorf("trailing bytes: %d", len(d.data))
	}
	return record, nil
}
