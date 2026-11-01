package user_api

import (
	"go-star/common"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"go-star/models/enum"
	"go-star/service/redis_service/redis_jwt"
	"go-star/utils/jwts"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
	// 请求体中的role必须是1，2，3，4中的一个，否则ShouldBind会失败
	Role enum.RoleType `json:"role" binding:"required,oneof=1 2 3 4"`
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

	oldRole := user.Role
	if oldRole == cr.Role {
		res.OKWithMsg("角色未变更", c)
		return
	}

	err = global.DB.Model(&user).Update("role", cr.Role).Error
	if err != nil {
		res.FailWithMsg("修改角色失败", c)
		return
	}

	// 旧access里是旧role的快照，必须打上变更时间戳让它立即失效
	// 这里失败不能静默：角色已改但旧token还能用，等于降级没生效
	if err := redis_jwt.SetRoleChangedAt(user.ID, time.Now()); err != nil {
		logrus.Errorf("记录角色变更时间失败 user=%d：%v", user.ID, err)
		res.FailWithMsg("角色已修改，但旧登录凭证未能失效，请强制该用户重新登录", c)
		return
	}

	// 审计留痕：谁把谁从什么角色改成了什么角色
	operator := uint(0)
	if _claims, ok := c.Get("claims"); ok {
		if claims, ok := _claims.(*jwts.MyClaims); ok {
			operator = claims.UserID
		}
	}
	logrus.Infof("角色变更 operator=%d target=%d old=%d new=%d", operator, user.ID, oldRole, cr.Role)

	res.OKWithMsg("修改角色成功", c)
}
