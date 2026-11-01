package enum

import "testing"

// TestPermissionMapping 权限映射表回归测试
// 价值：新增权限点时若忘记同步 allPermissions，管理员会自动漏权限，
// 本测试会立刻失败（实际已抓到过一次）。改动 permission.go 后务必跑一遍。
func TestPermissionMapping(t *testing.T) {
	if got, want := len(allPermissions), 14; got != want {
		t.Fatalf("权限点总数 = %d, want %d：新增/删除权限点后请同步更新本断言", got, want)
	}

	t.Logf("Admin   权限 = %v", RolePermissions[AdminRole])
	t.Logf("Auditor 权限 = %v", RolePermissions[AuditorRole])

	cases := []struct {
		role RoleType
		perm Permission
		want bool
		desc string
	}{
		// 管理员：全量，但职责分离例外
		{AdminRole, ArticleAudit, true, "管理员能审核"},
		{AdminRole, ArticlePublishDirect, true, "管理员免审核直接发布"},
		{AdminRole, LogRead, true, "管理员能看日志"},
		{AdminRole, LogDelete, false, "SoD：管理员不能删审计日志"},
		// 审核员：审核 + 看待审核队列
		{AuditorRole, ArticleAudit, true, "审核员能审核"},
		{AuditorRole, ArticleViewAll, true, "审核员能看审核队列"},
		{AuditorRole, ArticleUpdateAny, false, "审核员不能改别人文章"},
		{AuditorRole, ArticlePublishDirect, false, "审核员不能免审核发布"},
		{AuditorRole, LogRead, false, "审核员看不到日志"},
		{AuditorRole, LogDelete, false, "审核员不能删日志"},
		{AuditorRole, DataRead, false, "审核员看不到数据概览"},
		// 普通用户：无额外权限
		{UserRole, ArticleViewAll, false, "普通用户不能看全部状态文章"},
		{UserRole, ArticlePublishDirect, false, "普通用户不能免审核发布"},
		// 未知角色：失败即拒绝
		{RoleType(99), LogRead, false, "脏数据角色失败即拒绝"},
	}

	for _, c := range cases {
		if got := HasPerm(c.role, c.perm); got != c.want {
			t.Errorf("[FAIL] %s: HasPerm(role=%d, perm=%s) = %v, want %v", c.desc, c.role, c.perm, got, c.want)
		}
	}
}
