package service

import "errors"

var (
    ErrEmailInUse        = errors.New("email already verified")
    ErrInvalidCredentials = errors.New("invalid email or password")
    ErrExistsVerified   = errors.New("exists-verified")
    ErrExistsUnverified = errors.New("exists-unverified")
)
