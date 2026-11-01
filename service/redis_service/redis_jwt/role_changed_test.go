package redis_jwt

import (
	"go-star/global"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis"
)

// TestRoleChangedInvalidatesOldToken 验证「改角色后旧 access 失效」这条防线
// 用 miniredis 隔离，不触碰真实 Redis，也不依赖外部服务器
func TestRoleChangedInvalidatesOldToken(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("启动 miniredis 失败：%v", err)
	}
	defer mr.Close()

	oldClient := global.Redis
	global.Redis = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() {
		_ = global.Redis.Close()
		global.Redis = oldClient
	}()

	const userID uint = 7
	const otherUserID uint = 8

	// 场景1：从没改过角色 -> 任何 token 都不判失效
	if HasRoleChangedAfter(userID, 1000) {
		t.Fatal("从未变更角色时，不应判定 token 失效")
	}

	// 场景2：1500 秒时角色被变更
	if err := SetRoleChangedAt(userID, time.Unix(1500, 0)); err != nil {
		t.Fatalf("写入变更时间失败：%v", err)
	}

	// 场景3：变更前签发的 token（iat=1000）必须失效 —— 这就是 P0 漏洞的修复点
	if !HasRoleChangedAfter(userID, 1000) {
		t.Fatal("角色变更前签发的 token 应失效，否则降级不生效")
	}

	// 场景4：变更后签发的 token（如 refresh 换来的新 token）应继续有效
	if HasRoleChangedAfter(userID, 1600) {
		t.Fatal("角色变更后新签发的 token 不应失效，否则用户永远登不进去")
	}

	// 场景5：同秒边界。iat 与变更时间相同不判失效（与改密逻辑保持一致的 < 语义）
	// 已知边界：同一秒内「先登录后改角色」存在最长 1 秒窗口，实际几乎不可能触发
	if HasRoleChangedAfter(userID, 1500) {
		t.Fatal("同一秒签发的 token 按 < 语义不应判失效")
	}

	// 场景6：用户之间互不影响
	if HasRoleChangedAfter(otherUserID, 1000) {
		t.Fatal("变更 A 的角色不应影响用户 B 的 token")
	}
}
