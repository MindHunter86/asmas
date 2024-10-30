package utils

type ContextKey uint8

const (
	CKeyLogger ContextKey = iota
	CKeyCliCtx
	CKeyAbortFunc
	CKeyErrorChan
	CKeyAuthService
	CKeySystem
)
