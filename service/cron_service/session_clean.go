package cron_service

import (
	"go-star/global"
	"go-star/models"
	"time"

	"github.com/sirupsen/logrus"
)

const sessionCleanLimit = 500 // 每次最多删除多少条

func CleanExpiredSessions() {
	if global.DB == nil {
		return
	}

	now := time.Now()
	result := global.DB.
		Where("is_revoked = ? OR expires_at < ?", true, now).
		Limit(sessionCleanLimit).
		Delete(&models.UserSessionModel{})

	if result.Error != nil {
		logrus.Errorf("清理会话失败：%v", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		logrus.Infof("清理会话完成，删除%d条", result.RowsAffected)
	}
}
