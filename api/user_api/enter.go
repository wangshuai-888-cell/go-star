package user_api

import (
	"go-star/common"
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

type UserApi struct{}

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

// 获取用户信息接口
func (UserApi) UserInfoView(c *gin.Context) {
	// 从中间件中获取claims
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	// 按userID查库
	var user models.UserModel
	err := global.DB.Take(&user, "id = ?", claims.UserID).Error
	if err != nil {
		res.FailWithMsg("用户不存在", c)
		return
	}

	res.OKWithData(user, c)
}

// 修改密码
type ChangePwdRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func (UserApi) ChangePwdView(c *gin.Context) {
	var cr ChangePwdRequest
	err := c.ShouldBindJSON(&cr)

	if err != nil {
		res.FailWithError(err, c)
		return
	}

	// 从token中拿当前用户
	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var user models.UserModel
	err = global.DB.Take(&user, claims.UserID).Error
	if err != nil {
		res.FailWithMsg("用户不存在", c)
		return
	}

	// 校验旧密码
	if !pwd.CheckPwd(user.Password, cr.OldPassword) {
		res.FailWithMsg("旧密码错误", c)
		return
	}

	if cr.OldPassword == cr.NewPassword {
		res.FailWithMsg("新密码不能与旧密码相同", c)
		return
	}

	// 新密码加密
	hashPwd, err := pwd.HashPwd(cr.NewPassword)
	if err != nil {
		res.FailWithMsg("密码加密失败", c)
		return
	}

	// 更新密码
	err = global.DB.Model(&user).Update("password", hashPwd).Error
	if err != nil {
		res.FailWithMsg("更新密码失败", c)
		return
	}

	// 更新后拉黑当前token，重新登录
	token := c.GetHeader("token")
	if token == "" {
		token = c.Query("token")
	}
	if token == "" {
		res.FailWithMsg("token不能为空", c)
		return
	}
	redis_jwt.TokenBlack(token, redis_jwt.UserBlackType)

	res.OKWithMsg("修改密码成功", c)
}

// 修改用户信息
type UpdateUserRequest struct {
	Nickname string `json:"nickname"`
	Abstract string `json:"abstract"` // 简介
	Avatar   string `json:"avatar"`   // 头像
}

func (UserApi) UpdateUserView(c *gin.Context) {
	var cr UpdateUserRequest
	err := c.ShouldBindJSON(&cr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	_claims, _ := c.Get("claims")
	claims := _claims.(*jwts.MyClaims)

	var user models.UserModel
	err = global.DB.Take(&user, claims.UserID).Error
	if err != nil {
		res.FailWithMsg("用户不存在", c)
		return
	}

	err = global.DB.Model(&user).Updates(models.UserModel{
		Nickname: cr.Nickname,
		Abstract: cr.Abstract,
		Avatar:   cr.Avatar,
	}).Error
	if err != nil {
		res.FailWithMsg("更新用户信息失败", c)
		return
	}

	res.OKWithMsg("更新用户信息成功", c)
}

// 获取用户列表
type UserListRequest struct {
	common.PageInfo
	Role enum.RoleType `form:"role"`
}

func (UserApi) UserListView(c *gin.Context) {
	var cr UserListRequest
	err := c.ShouldBindQuery(&cr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	list, count, err := common.ListQuery(models.UserModel{
		Role: cr.Role,
	}, common.Options{
		PageInfo: cr.PageInfo,
		Likes:    []string{"username", "nickname"},
	})
	if err != nil {
		res.FailWithMsg("获取用户列表失败", c)
		return
	}

	res.OKWithList(list, count, c)
}

// 用户详情
func (UserApi) UserDetailView(c *gin.Context) {
	var cr models.IDRequest
	err := c.ShouldBindUri(&cr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	var user models.UserModel
	err = global.DB.Take(&user, cr.ID).Error
	if err != nil {
		res.FailWithMsg("用户不存在", c)
		return
	}

	res.OKWithData(user, c)
}

// 修改用户角色
type UpdateRoleRequest struct {
	// 请求体中的role必须是1，2，3中的一个，否则ShouldBind会失败
	Role enum.RoleType `json:"role" binding:"required,oneof=1 2 3"`
}

func (UserApi) UpdateRoleView(c *gin.Context) {
	var idCr models.IDRequest
	err := c.ShouldBindUri(&idCr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	var cr UpdateRoleRequest
	err = c.ShouldBindJSON(&cr)
	if err != nil {
		res.FailWithError(err, c)
		return
	}

	var user models.UserModel
	err = global.DB.Take(&user, idCr.ID).Error
	if err != nil {
		res.FailWithMsg("用户不存在", c)
		return
	}

	err = global.DB.Model(&user).Update("role", cr.Role).Error
	if err != nil {
		res.FailWithMsg("修改角色失败", c)
		return
	}

	res.OKWithMsg("修改角色成功", c)
}