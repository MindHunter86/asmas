package auth

type Session struct {
	active bool
}

func (*Session) IsActive() bool { return false }
