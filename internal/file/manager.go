// FileMgrクラス. ディスク上のデータをメモリに読み込む
package file

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type FileMgr struct {
	dbDirectory string
	blockSize   int
	isNew       bool
	openFiles   map[string]*os.File
	mu          sync.Mutex
}

func NewFileMgr(dbDirectory string, blockSize int) *FileMgr {
	_, err := os.Stat(dbDirectory)
	isNew := os.IsNotExist(err)
	if err != nil && !isNew {
		panic(err)
	}

	fm := &FileMgr{
		dbDirectory: dbDirectory,
		blockSize:   blockSize,
		isNew:       isNew,
		openFiles:   make(map[string]*os.File),
	}

	// 新しいDBの場合はディレクトリを作成する
	if isNew {
		if err := os.MkdirAll(dbDirectory, 0755); err != nil {
			panic(err)
		}
	}

	// 一時ファイルの削除
	entries, err := os.ReadDir(dbDirectory)
	if err != nil {
		panic(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "temp") {
			err := os.Remove(filepath.Join(dbDirectory, entry.Name()))
			if err != nil {
				panic(err)
			}
		}
	}

	return fm
}

// ディスク上のファイルの中身をメモリ上のPageに読み込む
func (fm *FileMgr) Read(blk BlockId, p *Page) {
	fm.mu.Lock() // 同時に複数のgoroutineがReadを呼び出す可能性があるため、排他制御を行う
	defer fm.mu.Unlock()

	f := fm.getFile(blk.filename)
	offset := int64(blk.number * fm.blockSize)
	// ファイルの先頭からoffsetバイト目にシークし、そこから読み込む
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		panic(err)
	}
	if _, err := f.Read(p.contents()); err != nil && err != io.EOF {
		panic(err)
	}
}

// メモリ上のPageのデータをディスク上のファイルに書き込む
func (fm *FileMgr) Write(blk BlockId, p *Page) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	f := fm.getFile(blk.filename)
	offset := int64(blk.number * fm.blockSize)
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		panic(err)
	}
	if _, err := f.Write(p.contents()); err != nil {
		panic(err)
	}
}

// ファイルの末尾に新しい空ブロックを追加する
func (fm *FileMgr) Append(filename string) BlockId {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	blk := BlockId{
		filename: filename,
		number:   fm.Length(filename), // 追加するブロックの番号を取得
	}
	f := fm.getFile(filename)
	offset := int64(blk.number * fm.blockSize)
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		panic(err)
	}

	// 全て0のブロックを末尾に追加する
	if _, err := f.Write(make([]byte, fm.blockSize)); err != nil {
		panic(err)
	}
	return blk
}

// ファイルのブロック数を取得する
func (fm *FileMgr) Length(filename string) int {
	info, err := fm.getFile(filename).Stat()
	if err != nil {
		panic(err)
	}
	return int(info.Size() / int64(fm.blockSize))
}

func (fm *FileMgr) getFile(filename string) *os.File {
	if f, ok := fm.openFiles[filename]; ok {
		return f
	}

	f, err := os.OpenFile(filepath.Join(fm.dbDirectory, filename), os.O_RDWR|os.O_CREATE|os.O_SYNC, 0644)
	if err != nil {
		panic(err)
	}
	fm.openFiles[filename] = f
	return f
}

func (fm *FileMgr) BlockSize() int {
	return fm.blockSize
}
