package site_api

import (
	"go-star/models/enum"
	"go-star/service/log_service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type SiteApi struct {
}

func (SiteApi) SiteInfoView(c *gin.Context) {
	log_service.NewLoginSuccess(c, enum.UserPwdLoginType)
	log_service.NewLoginFail(c, enum.UserPwdLoginType, "用户不存在", "xiaowang", "1234")

	c.JSON(200, gin.H{"code": 0, "msg": "站点信息"})
	return
}

type SiteUpdateRequest struct {
	Name string `form:"name"`
}

func (SiteApi) SiteUpdateView(c *gin.Context) {
	log := log_service.GetLog(c)

	//log.ShowRequest()
	log.ShowResponse()

	var cr SiteUpdateRequest
	err := c.ShouldBindJSON(&cr)
	if err != nil {
		logrus.Errorf(err.Error())
	}
	c.JSON(200, gin.H{"code": 0, "msg": "站点信息"})
	return
}
