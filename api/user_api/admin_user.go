package user_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"

	"github.com/gin-gonic/gin"
)

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
