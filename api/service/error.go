package service

import "github.com/pkg/errors"

var (
	errSystem            = errors.New("system error")
	errObjectNotReadable = errors.New("object is not ready for reading")
)

var ErrorCode = map[error]int{
	errSystem:            1000,
	errObjectNotReadable: 1001,
}
