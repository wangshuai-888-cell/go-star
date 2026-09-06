package user_api

import (
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/log_service"
	"go-star/service/redis_service/redis_jwt"
	"go-star/utils/jwts"
	"go-star/utils/pwd"

	"github.com/gin-gonic/gin"
)

// 登录接口
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
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

	// 用户名和密码都正确，发token
	token, err := jwts.GetToken(jwts.Claims{
		UserID:   user.ID,
		UserName: user.Username,
		Role:     user.Role,
	})

	if err != nil {
		res.FailWithMsg("生成token失败", c)
		return
	}

	log_service.NewLoginSuccess(c, enum.UserPwdLoginType, user.ID, user.Username)
	res.OKWithData(token, c)
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

	token, err := jwts.GetToken(jwts.Claims{
		UserID:   user.ID,
		UserName: user.Username,
		Role:     user.Role,
	})
	if err != nil {
		res.FailWithMsg("生成token失败", c)
		return
	}

	res.OKWithData(token, c)
}

// 退出接口
func (UserApi) LogoutView(c *gin.Context) {
	token := c.GetHeader("token")
	if token == "" {
		token = c.Query("token")
	}
	if token == "" {
		res.FailWithMsg("token不能为空", c)
		return
	}
	redis_jwt.TokenBlack(token, redis_jwt.UserBlackType)
	res.OKWithMsg("退出成功", c)
}
