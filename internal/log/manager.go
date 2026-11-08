package log

import (
	"fmt"
	"sync"

	"example.com/simpledb/internal/file"
)

const intSize = 4

type LogMgr struct {
	mu           sync.Mutex
	fm           *file.FileMgr
	logFile      string
	logPage      *file.Page
	currentBlock file.BlockId
	// LSN (Log Sequence Number) はログレコードの順序を一意に識別するための番号
	latestLSN    int
	lastSavedLSN int
}

func NewLogMgr(fm *file.FileMgr, logFile string) *LogMgr {
	lm := &LogMgr{
		fm:      fm,
		logFile: logFile,
		logPage: file.NewPage(fm.BlockSize()),
	}

	logSize := fm.Length(logFile)
	if logSize == 0 {
		lm.currentBlock = lm.appendNewBlock()
	} else {
		lm.currentBlock = *file.NewBlockId(logFile, logSize-1)
		fm.Read(lm.currentBlock, lm.logPage)
	}
	return lm
}

// ページの先頭                                     ページの末尾
// [境界位置 (4 Byte)][    空き領域    ][新しいレコード][古いレコード]
//                                                 ↑ boundary
// 各レコード = [長さ（4 Byte）][データ]

// レコードをログに追加する. 追加されたレコードのLSNを返す
func (lm *LogMgr) Append(record []byte) (int, error) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// ページ先頭の境界位置と、レコード長の保存に各4バイト必要（intSize = 4）
	maxRecordSize := lm.fm.BlockSize() - 2*intSize
	if len(record) > maxRecordSize {
		return 0, fmt.Errorf(
			"record size %d exceeds maximum record size %d", len(record), maxRecordSize,
		)
	}

	boundary := int(lm.logPage.GetInt(0)) // 既存レコードが始まる位置. 新しいレコードはこの位置の前に追加される
	bytesNeeded := intSize + len(record)

	// ページにレコードを追加できるか確認
	if boundary-bytesNeeded < intSize {
		lm.flush() // Flushを呼ぶとデッドロックになる
		lm.currentBlock = lm.appendNewBlock()
		boundary = int(lm.logPage.GetInt(0))
	}

	recordPosition := boundary - bytesNeeded // 空き領域との境界
	lm.logPage.SetBytes(recordPosition, record)
	lm.logPage.SetInt(0, int32(recordPosition))

	lm.latestLSN++
	return lm.latestLSN, nil
}

func (lm *LogMgr) Flush(lsn int) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if lsn <= lm.lastSavedLSN {
		return
	}
	lm.flush()
}

// 指定されたLSNまでのログをディスクに書き出して永続化する. flushはロック済みのメソッドから呼ぶ
func (lm *LogMgr) flush() {
	lm.fm.Write(lm.currentBlock, lm.logPage)
	lm.lastSavedLSN = lm.latestLSN
}

// appendNewBlockは初期化中、またはロック取得済みの状態で呼ぶ
func (lm *LogMgr) appendNewBlock() file.BlockId {
	blk := lm.fm.Append(lm.logFile)

	// 境界位置を末尾に戻し、論理的に空のページにする
	lm.logPage.SetInt(0, int32(lm.fm.BlockSize()))
	lm.fm.Write(blk, lm.logPage)
	return blk
}

func (lm *LogMgr) Iterator() *LogIterator {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.flush() // 最新のログをディスクに書き出す. LogIteratorはディスクから読むので、メモリに未反映のログがないようにする
	return NewLogIterator(lm.fm, lm.currentBlock)
}
