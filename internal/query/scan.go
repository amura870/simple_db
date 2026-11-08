package query

import "example.com/simpledb/internal/record"

// 呼び出し順は `BeforeFirst → Next → 値の取得` です。`Next` がtrueを返した現在行から値を読み、次の `Next` で後続の行へ進みます。
// 問い合わせの最上位のスキャンを閉じると、その下のスキャンも閉じます。

// Nextがtrueを返したときだけ、現在行の値を読む。
// falseは終端、errorは処理の失敗を表す。
type Scan interface {
	BeforeFirst() error
	Next() (bool, error)
	GetInt(field string) (int32, error)
	GetString(field string) (string, error)
	GetVal(field string) (Constant, error)
	HasField(field string) bool
	Close() error
}

// Scanにレコードの更新機能を足したもの
type UpdateScan interface {
	Scan
	SetInt(field string, value int32) error
	SetString(field, value string) error
	SetVal(field string, value Constant) error
	Insert() error
	Delete() error
	GetRID() (record.RID, error)
	MoveToRID(rid record.RID) error
}
