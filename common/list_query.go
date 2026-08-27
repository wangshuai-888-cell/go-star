package common

import (
	"fmt"
	"go-star/global"

	"gorm.io/gorm"
)

type PageInfo struct {
	Limit int    `form:"limit"` // 一页显示多少条
	Page  int    `form:"page"`  // 第几页
	Key   string `form:"key"`   // 模糊匹配的参数
	Order string `form:"order"` // 排序 前端可以覆盖
}

func (p PageInfo) GetPage() int {
	if p.Page > 20 || p.Page <= 0 {
		return 1
	}
	return p.Page
}

func (p PageInfo) GetLimit() int {
	if p.Limit <= 0 || p.Limit > 100 {
		return 10
	}
	return p.Limit
}

func (p PageInfo) GetOffset() int {
	return (p.GetPage() - 1) * p.GetLimit()
}

type Options struct {
	PageInfo     PageInfo
	Likes        []string
	Preloads     []string
	Where        *gorm.DB
	Debug        bool
	DefaultOrder string
}

// 第一个参数：要查哪张表
func ListQuery[T any](model T, option Options) (list []T, count int, err error) {
	// 自己的基础查询
	query := global.DB.Model(model).Where(model)

	// 日志
	if option.Debug {
		query = query.Debug()
	}

	// 模糊匹配
	if len(option.Likes) > 0 && option.PageInfo.Key != "" {
		likes := global.DB.Where("")
		for _, column := range option.Likes {
			likes.Or(fmt.Sprintf("%s like ?", column), fmt.Sprintf("%%%s%%", option.PageInfo.Key))
		}
		query = query.Where(likes)
	}

	// 高级查询
	if option.Where != nil {
		query = query.Where(option.Where)
	}

	// 预加载
	for _, preload := range option.Preloads {
		query = query.Preload(preload)
	}

	// 查总数
	var _c int64
	query.Count(&_c)
	count = int(_c)

	// 分页
	limit := option.PageInfo.GetLimit()
	offset := option.PageInfo.GetOffset()

	// 排序
	if option.PageInfo.Order != "" {
		// 在外层配置了
		query = query.Order(option.PageInfo.Order)
	} else {
		if option.DefaultOrder != "" {
			query = query.Order(option.DefaultOrder)
		}
	}

	err = query.Offset(offset).Limit(limit).Find(&list).Error
	return
}
