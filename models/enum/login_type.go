package enum

type LoginType int8

const (
	UserPwdLoginType LoginType = 1
	QQLoginType                = 2
	EmailLoginType   LoginType = 3
)
