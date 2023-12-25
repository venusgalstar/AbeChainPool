package errcode

import "errors"

var (
	ErrNilGormDB = errors.New("nil gorm db")
)
