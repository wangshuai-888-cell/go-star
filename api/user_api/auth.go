package user_api

import (
	"go-star/common/res"
	"go-star/core"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/log_service"
	"go-star/service/redis_service/redis_jwt"
	"go-star/utils/hash"
	"go-star/utils/jwts"
	"go-star/utils/pwd"
	"time"

	"github.com/gin-gonic/gin"
)

// 登录注册成功后返回给前端的双token
type TokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// 登录接口
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// 签发access + refresh，并写入会话表
func issueTokens(c *gin.Context, user models.UserModel) (TokenResponse, error) {
	access, err := jwts.GetToken(jwts.Claims{
		UserID:   user.ID,
		UserName: user.Username,
		Role:     user.Role,
	})
	if err != nil {
		return TokenResponse{}, err
	}

	refresh, err := hash.NewRefreshToken()
	if err != nil {
		return TokenResponse{}, err
	}

	ip := c.ClientIP()
	now := time.Now()
	expireHours := global.Config.Jwt.RefreshExpire
	if expireHours <= 0 {
		expireHours = 168
	}

	session := models.UserSessionModel{
		UserID:           user.ID,
		RefreshTokenHash: hash.SHA256Hex(refresh), // 只存哈希
		UserAgent:        c.GetHeader("User-Agent"),
		IP:               ip,
		Addr:             core.GetIpAddr(ip),
		ExpiresAt:        now.Add(time.Duration(expireHours) * time.Hour),
		LastUsedAt:       now,
		IsRevoked:        false,
	}
	if err := global.DB.Create(&session).Error; err != nil {
		return TokenResponse{}, err
	}

	return TokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}

func (UserApi) LoginView(c *gin.Context) {
	var cr LoginRequest
	err := c.ShouldBindJSON(&cr) // 把前端传过来的JSON绑定到cr中
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	// 去数据库中查找用户名是否存在
	var user models.UserModel
	err = global.DB.Take(&user, "username = ?", cr.Username).Error
	if err != nil {
		log_service.NewLoginFail(c, enum.UserPwdLoginType, "用户名或密码错误", cr.Username, cr.Password)
		res.FailWithMsg("用户名或密码错误", c)
		return
	}

	// 对比密码是否正确
	if !pwd.CheckPwd(user.Password, cr.Password) {
		log_service.NewLoginFail(c, enum.UserPwdLoginType, "用户名或密码错误", cr.Username, cr.Password)
		res.FailWithMsg("用户名或密码错误", c)
		return
	}

	tokens, err := issueTokens(c, user)
	if err != nil {
		res.FailWithMsg("生成token失败", c)
		return
	}

	log_service.NewLoginSuccess(c, enum.UserPwdLoginType, user.ID, user.Username)
	res.OKWithData(tokens, c)
}

// 注册接口
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (UserApi) RegisterView(c *gin.Context) {
	var cr RegisterRequest
	err := c.ShouldBindJSON(&cr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	// 用户名是否存在
	var user models.UserModel
	err = global.DB.Take(&user, "username = ?", cr.Username).Error
	if err == nil {
		res.FailWithMsg("用户名已存在", c)
		return
	}

	// 密码加密
	hashPwd, err := pwd.HashPwd(cr.Password)
	if err != nil {
		res.FailWithMsg("密码加密失败", c)
		return
	}

	user = models.UserModel{
		Username: cr.Username,
		Nickname: cr.Username,
		Password: hashPwd,
		Role:     enum.UserRole,
	}
	err = global.DB.Create(&user).Error
	if err != nil {
		res.FailWithMsg("注册失败", c)
		return
	}

	_ = global.DB.Create(&models.UserConfModel{
		UserID:      user.ID,
		OpenCollect: false,
	}).Error

	tokens, err := issueTokens(c, user)
	if err != nil {
		res.FailWithMsg("生成token失败", c)
		return
	}
	res.OKWithData(tokens, c)
}

// 刷新token
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

func (UserApi) RefreshView(c *gin.Context) {
	var cr RefreshRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	tokenHash := hash.SHA256Hex(cr.RefreshToken)

	var session models.UserSessionModel
	err := global.DB.Where(
		"refresh_token_hash = ? AND is_revoked = ? AND expires_at > ?",
		tokenHash, false, time.Now(),
	).Take(&session).Error
	if err != nil {
		res.FailWithMsg("refresh token已失效", c)
		return
	}

	// 查出用户（角色可能变过，重新签发）
	var user models.UserModel
	if err := global.DB.Take(&user, session.UserID).Error; err != nil {
		res.FailWithMsg("用户不存在", c)
		return
	}

	// 先作废当前refresh（轮换）， 只更新is_revoked字段
	if err := global.DB.Model(&session).Updates(map[string]any{
		"is_revoked": true,
	}).Error; err != nil {
		res.FailWithMsg("刷新失败", c)
		return
	}

	tokens, err := issueTokens(c, user)
	if err != nil {
		res.FailWithMsg("生成token失败", c)
		return
	}

	res.OKWithData(tokens, c)
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// 退出接口
func (UserApi) LogoutView(c *gin.Context) {
	var cr LogoutRequest
	if err := c.ShouldBindJSON(&cr); err != nil {
		res.FailWithError(err, c)
		return
	}

	// 1. 拉黑access（立刻不能调需要登录的接口）
	token := c.GetHeader("token")
	if token == "" {
		token = c.Query("token")
	}
	if token == "" {
		res.FailWithMsg("token不能为空", c)
		return
	}
	redis_jwt.TokenBlack(token, redis_jwt.UserBlackType)

	// 2.作废当前 refresh 对应会话
	tokenHash := hash.SHA256Hex(cr.RefreshToken)
	result := global.DB.Model(&models.UserSessionModel{}).
		Where("refresh_token_hash = ? AND is_revoked = ?", tokenHash, false).
		Update("is_revoked", true)
	if result.Error != nil {
		res.FailWithMsg("退出失败", c)
		return
	}
	//  RowsAffected不是UserSessionModel表结构中的字段，是gorm执行完Update后返回的接口中带的一个数字，表示这次sql实际改了几行
	if result.RowsAffected == 0 {
		res.FailWithMsg("refreshToken无效或已失效", c)
		return
	}
	res.OKWithMsg("退出成功", c)
}
