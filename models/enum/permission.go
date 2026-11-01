package enum

// Permission 权限点类型
// 用string而非iota数字，日志、审计表里要能直接读懂“谁能做什么”
type Permission string

// 权限点命名规范：资源：动作【：范围】
// 范围只用于“任意/他人”这类突破归属限制的场景
// 操作自己的资源是登录用户的天然能力，不设置权限点
const (
	// ===== 文章域 =====
	ArticleUpdateAny     Permission = "article:update:any"     // 修改任意文章（无视作者归属）
	ArticleDeleteAny     Permission = "article:delete:any"     // 删除任意文章（无视作者归属）
	ArticleSubmitAny     Permission = "article:submit:any"     // 替他人提交审核
	ArticleAudit         Permission = "article:audit"          // 审核文章：通过 / 驳回
	ArticlePublishDirect Permission = "article:publish:direct" // 发布文章免审核
	ArticleViewAll       Permission = "article:view:all"       // 查看所有状态的文章（含草稿/审核中/已驳回）

	// ===== 用户域 =====
	UserList       Permission = "user:list"        // 查看用户列表
	UserDetail     Permission = "user:detail"      // 查看任意用户详情
	UserRoleUpdate Permission = "user:role:update" // 修改他人角色

	// ===== 运营域 =====
	BannerWrite       Permission = "banner:write"       // 横幅增删改
	NotificationWrite Permission = "notification:write" // 公告增删改

	// ===== 审计与数据域 =====
	LogRead  Permission = "log:read"  // 查看操作日志
	DataRead Permission = "data:read" // 查看数据概览
	// 日志删除属于职责分离（SoD）高危权限：定义出来是为了显式禁止，任何角色都不授予
	LogDelete Permission = "log:delete"
)

// allPermissions 系统全部权限点
// 新增权限点必须同步加到这里，否则管理员会漏掉该权限（这是唯一的维护成本）
var allPermissions = []Permission{
	ArticleUpdateAny,
	ArticleDeleteAny,
	ArticleSubmitAny,
	ArticleAudit,
	ArticlePublishDirect,
	ArticleViewAll,
	UserList,
	UserDetail,
	UserRoleUpdate,
	BannerWrite,
	NotificationWrite,
	LogRead,
	DataRead,
	LogDelete,
}

// adminDenied 职责分离（SoD）黑名单：即使管理员也不授予的权限
// 登记在此的权限点不会出现在管理员的权限集合里
var adminDenied = map[Permission]bool{
	// 审计日志不允许被被审计者销毁，管理员也不例外
	LogDelete: true,
}

// adminPermissions 生成管理员权限集合：全量权限 - 职责分离黑名单
func adminPermissions() []Permission {
	perms := make([]Permission, 0, len(allPermissions))
	for _, p := range allPermissions {
		if adminDenied[p] {
			continue
		}
		perms = append(perms, p)
	}
	return perms
}

// RolePermissions 角色 → 权限映射表
var RolePermissions = map[RoleType][]Permission{
	AdminRole: adminPermissions(),
	// 审核员：能审核 + 能看待审核队列；不能改内容、不能看日志与数据
	AuditorRole: {ArticleAudit, ArticleViewAll},
	UserRole:    {},
	VisitorRole: {},
}

// HasPerm 判断角色是否持有指定权限点
func HasPerm(role RoleType, p Permission) bool {
	for _, rp := range RolePermissions[role] {
		if rp == p {
			return true
		}
	}
	return false
}
