package domain

import "errors"

var (
	ErrTxAlreadyConfirmed = errors.New("tx already confirmed")
	ErrTxIsContract       = errors.New("tx is a contract deploy")
	ErrTxDoesntApply      = errors.New("tx doesn't apply for a snipe, it probably isn't from the target router or operation wasn't an add liquidity one")
)
