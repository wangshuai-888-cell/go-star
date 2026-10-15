package redis_sign

import (
	"fmt"
	"go-star/global"
	"go-star/models"
	"time"
)

// signKey 生成签到记录的Key
// 格式: sign:{userID}:{YYYYMM}
func signKey(userID uint, date time.Time) string {
	return fmt.Sprintf("sign:%d:%s", userID, date.Format("200601"))
}

// SignIn 用户签到（生产级：Redis + MySQL 双写）
// 返回：(是否成功签到, error)
func SignIn(userID uint, date time.Time) (bool, error) {
	key := signKey(userID, date)
	offset := date.Day() - 1
	dateStr := date.Format("2006-01-02")

	// 1. 先检查Redis（快速判断）
	if global.Redis != nil {
		signed, err := global.Redis.GetBit(key, int64(offset)).Result()
		if err == nil && signed == 1 {
			return false, nil // Redis中有记录，已签到
		}
	}

	// 2. 检查MySQL（兜底，防止Redis数据丢失）
	var count int64
	if err := global.DB.Model(&models.UserSignModel{}).
		Where("user_id = ? AND sign_date = ?", userID, dateStr).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("查询数据库失败: %w", err)
	}

	if count > 0 {
		// MySQL有记录但Redis没有，同步到Redis
		if global.Redis != nil {
			_ = global.Redis.SetBit(key, int64(offset), 1).Err()
		}
		return false, nil // 已签到
	}

	// 3. 写入MySQL（持久化）
	sign := models.UserSignModel{
		UserID:   userID,
		SignDate: date,
	}
	if err := global.DB.Create(&sign).Error; err != nil {
		// 如果是唯一索引冲突，说明已签到（并发情况）
		return false, nil
	}

	// 4. 写入Redis（缓存加速）
	if global.Redis != nil {
		_ = global.Redis.SetBit(key, int64(offset), 1).Err()
	}

	return true, nil
}

// IsSigned 查询某天是否签到（优先Redis，兜底MySQL）
func IsSigned(userID uint, date time.Time) (bool, error) {
	key := signKey(userID, date)
	offset := date.Day() - 1
	dateStr := date.Format("2006-01-02")

	// 1. 优先查Redis
	if global.Redis != nil {
		val, err := global.Redis.GetBit(key, int64(offset)).Result()
		if err == nil {
			return val == 1, nil
		}
	}

	// 2. Redis没有，查MySQL
	var count int64
	if err := global.DB.Model(&models.UserSignModel{}).
		Where("user_id = ? AND sign_date = ?", userID, dateStr).
		Count(&count).Error; err != nil {
		return false, err
	}

	signed := count > 0

	// 3. 同步到Redis（下次就快了）
	if signed && global.Redis != nil {
		_ = global.Redis.SetBit(key, int64(offset), 1).Err()
	}

	return signed, nil
}

// GetMonthSignCount 统计某月签到天数（从MySQL查）
func GetMonthSignCount(userID uint, year int, month int) (int, error) {
	// 计算月份的起止日期
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, 0)

	var count int64
	err := global.DB.Model(&models.UserSignModel{}).
		Where("user_id = ? AND sign_date >= ? AND sign_date < ?", userID, startDate, endDate).
		Count(&count).Error

	return int(count), err
}

// GetMonthSignDays 获取某月的所有签到日期（从MySQL查）
func GetMonthSignDays(userID uint, year int, month int) ([]int, error) {
	// 计算月份的起止日期
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(0, 1, 0)

	var signs []models.UserSignModel
	err := global.DB.Where("user_id = ? AND sign_date >= ? AND sign_date < ?", userID, startDate, endDate).
		Find(&signs).Error
	if err != nil {
		return nil, err
	}

	days := make([]int, 0, len(signs))
	for _, sign := range signs {
		days = append(days, sign.SignDate.Day())
	}

	return days, nil
}

// GetContinuousSignDays 获取连续签到天数（从MySQL查）
func GetContinuousSignDays(userID uint) (int, error) {
	now := time.Now()
	continuousDays := 0

	// 从今天开始往前查，最多查365天
	for i := 0; i < 365; i++ {
		checkDate := now.AddDate(0, 0, -i)
		dateStr := checkDate.Format("2006-01-02")

		var count int64
		err := global.DB.Model(&models.UserSignModel{}).
			Where("user_id = ? AND sign_date = ?", userID, dateStr).
			Count(&count).Error

		if err != nil {
			break
		}

		if count > 0 {
			continuousDays++
		} else {
			break // 遇到未签到的那天就停止
		}
	}

	return continuousDays, nil
}

// SyncHistorySignToRedis 同步历史签到记录到Redis
// 用途：Redis数据丢失后，从MySQL恢复
// 可以做成定时任务或手动触发
func SyncHistorySignToRedis(userID uint, year int, month int) error {
	if global.Redis == nil {
		return fmt.Errorf("redis未连接")
	}

	// 从MySQL查询该月的签到记录
	signs, err := GetMonthSignDays(userID, year, month)
	if err != nil {
		return err
	}

	// 写入Redis
	yearMonth := fmt.Sprintf("%d%02d", year, month)
	key := fmt.Sprintf("sign:%d:%s", userID, yearMonth)

	for _, day := range signs {
		offset := day - 1
		_ = global.Redis.SetBit(key, int64(offset), 1).Err()
	}

	return nil
}
