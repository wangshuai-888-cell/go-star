package site_api

import (
	"go-star/models/enum"
	"go-star/service/log_service"
	"time"

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
	Name string `form:"name" binding:"required"`
}

func (SiteApi) SiteUpdateView(c *gin.Context) {
	log := log_service.GetLog(c)

	log.ShowRequest()
	log.ShowRequestHeader()
	log.ShowResponseHeader()
	log.ShowResponse()
	log.SetTitle("更新站点")
	log.SetItemInfo("请求时间", time.Now())
	log.SetImage("/xx/xxx")
	log.SetLink("yarm学习地址", "https://www.baidu.com")
	c.Header("xxxx", "xxkk")
	var cr SiteUpdateRequest
	err := c.ShouldBindJSON(&cr)
	if err != nil {
		log.SetError("参数绑定失败", err)
	}
	if err != nil {
		logrus.Errorf(err.Error())
	}
	log.SetItemInfo("结构体", cr)
	log.SetItemInfo("切片", []string{"a", "b"})
	log.SetItemInfo("字符串", "你好")
	log.SetItemInfo("数字", 123)
	//id := log.Save()
	c.JSON(200, gin.H{"code": 0, "msg": "站点信息", "data": 1})
	return
}
