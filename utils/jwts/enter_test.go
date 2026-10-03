package jwts

import (
	"go-star/conf"
	"go-star/global"
	"go-star/models/enum"
	"strings"
	"testing"
)

func setupJwt(t *testing.T) {
	t.Helper() // 失败时行号会指到真正的测试，而不是 setup 函数。
	global.Config = &conf.Config{
		Jwt: conf.Jwt{
			Expire: 1,
			Secret: "test-secret",
			Issuer: "go-star-test",
		},
	}
}

func TestGetTokenAndParseToken(t *testing.T) {
	setupJwt(t)

	token, err := GetToken(Claims{
		UserID:   1,
		UserName: "xiaozhang3",
		Role:     enum.UserRole,
	})
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}
	if token == "" {
		t.Fatal("token 不能为空")
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if claims.UserID != 1 {
		t.Errorf("UserID = %d, want 1", claims.UserID)
	}
	if claims.UserName != "xiaozhang3" {
		t.Errorf("UserName = %s, want xiaozhang3", claims.UserName)
	}
	if claims.Role != enum.UserRole {
		t.Errorf("Role = %v, want UserRole", claims.Role)
	}
}

func TestParseToken_Empty(t *testing.T) {
	setupJwt(t)

	_, err := ParseToken("")
	if err == nil || err.Error() != "请登录" {
		t.Fatalf("空 token 应变请登录，实际 %v", err)
	}
}

func TestParseToken_Tampered(t *testing.T) {
	setupJwt(t)

	token, err := GetToken(Claims{UserID: 1, UserName: "a", Role: enum.UserRole})
	if err != nil {
		t.Fatal(err)
	}
	// 改最后一个字符，破坏签名
	bad := token[:len(token)-1] + "x"
	if bad == token {
		bad = token + "x"
	}

	_, err = ParseToken(bad)
	if err == nil {
		t.Fatal("篡改后的 token 应解析失败")
	}
	if !strings.Contains(err.Error(), "token") && !strings.Contains(err.Error(), "无效") && !strings.Contains(err.Error(), "不合法") {
		t.Logf("失败原因（可接受）: %v", err)
	}
}
