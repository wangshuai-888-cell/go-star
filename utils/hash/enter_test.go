package hash

import "testing"

// 测试refresh token 长度
func TestSHA256Hex_length(t *testing.T) {
	got := SHA256Hex("hello")
	if len(got) != 64 {
		t.Fatalf("sha256 hex 长度应该为64，实际%d", len(got))
	}
}

func TestSHA256Hex_SameInputSameHash(t *testing.T) {
	a := SHA256Hex("refresh-token")
	b := SHA256Hex("refresh-token")
	if a != b {
		t.Fatal("相同明文应得到相同hash（刷新/退出才能对上会话）")
	}
}

func TestSHA256Hex_DifferentInputDifferentHash(t *testing.T) {
	a := SHA256Hex("device-1")
	b := SHA256Hex("device-2")
	if a == b {
		t.Fatal("不同 refresh 不应得到相同 hash")
	}
}

func TestNewRefreshToken(t *testing.T) {
	a, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	b, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	if len(a) != 64 {
		t.Fatalf("refresh 明文应为 32 字节的 hex，长度 64，实际 %d", len(a))
	}
	if a == b {
		t.Fatal("两次生成不应相同")
	}
}
