package core

import "github.com/ygrebnov/log/pkg/types"

type formatter interface {
	Format(record types.Record) ([]byte, error)
}
