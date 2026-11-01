package enum

type RoleType int8

const (
	AdminRole   RoleType = 1 // 管理员
	UserRole    RoleType = 2 // 用户
	VisitorRole RoleType = 3 // 游客
	AuditorRole RoleType = 4 // 审核员
)
