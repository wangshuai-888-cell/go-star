package redis_jwt

import (
	"fmt"
	"go-star/global"
	"go-star/utils/jwts"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type BlackType int8

const (
	UserBlackType   BlackType = 1 // 用户注销登录
	AdminBlackType  BlackType = 2 // 管理员手动让你下线
	DeviceBlackType BlackType = 3 // 其他设备把自己挤下来
)

func (b BlackType) String() string {
	return fmt.Sprintf("%d", b)
}

func (b BlackType) Msg() string {
	switch b {
	case UserBlackType:
		return "已退出"
	case AdminBlackType:
		return "禁止登录"
	case DeviceBlackType:
		return "设备下线"
	}
	return "已退出"
}

func ParseBlackType(val string) BlackType {
	switch val {
	case "1":
		return UserBlackType
	case "2":
		return AdminBlackType
	case "3":
		return DeviceBlackType
	}
	return UserBlackType
}

func TokenBlack(token string, value BlackType) {
	key := fmt.Sprintf("token_black:%s", token)

	claims, err := jwts.ParseToken(token)
	if err != nil || claims == nil {
		logrus.Errorf("token解析失败 %s", err)
		return
	}
	second := claims.ExpiresAt - time.Now().Unix()

	_, err = global.Redis.Set(key, value.String(), time.Duration(second)*time.Second).Result()
	if err != nil {
		logrus.Errorf("获取token黑名单失败 %s", err)
		return
	}
}

func HasTokenBlack(token string) (blk BlackType, ok bool) {
	key := fmt.Sprintf("token_black:%s", token)
	value, err := global.Redis.Get(key).Result()
	if err != nil {
		return 0, false
	}
	return ParseBlackType(value), true
}

func HasTokenBlackByGin(c *gin.Context) (blk BlackType, ok bool) {
	token := c.GetHeader("token")
	if token == "" {
		return 0, false
	}
	return HasTokenBlack(token)
}

// SetPwdChangedAt：记录用户最近一次修改密码的时间，用于让改密前签发的access立即失效
func SetPwdChangedAt(userID uint, at time.Time) error {
	key := fmt.Sprintf("jwt:pwd_changed:%d", userID)
	// 保留7天：refresh最长也就7天，超过这个时间旧token早就过期了
	return global.Redis.Set(key, at.Unix(), 7*24*time.Hour).Err()
}

// GetPwdChangedAt：获取用户最近一次修改密码的时间戳，0 表示从没改过
func GetPwdChangedAt(userID uint) int64 {
	key := fmt.Sprintf("jwt:pwd_changed:%d", userID)
	val, err := global.Redis.Get(key).Int64()
	if err != nil {
		return 0
	}
	return val
}

// HasPwdChangedAfter：判断token签发之后密码是否被修改过（是则这个token作废）
func HasPwdChangedAfter(userID uint, issuedAt int64) bool {
	changedAt := GetPwdChangedAt(userID)
	if changedAt == 0 {
		return false
	}
	return issuedAt < changedAt
}
