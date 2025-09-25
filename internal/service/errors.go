package service

import "errors"

var (
    ErrExistsVerified   = errors.New("exists-verified")
    ErrExistsUnverified = errors.New("exists-unverified")
)
