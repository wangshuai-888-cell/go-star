package bloom

import (
	"go-star/global"
	"go-star/models"
	"strconv"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/sirupsen/logrus"
)

// 布隆过滤器：把所有真实存在的文章 ID 装进一个内存位图，
// 用于在 Redis/DB 之前拦截"批量伪造的不存在 ID"（缓存穿透的终极防线）。
//
// 性质：
//   - 说"不存在" → 100% 真的不存在，可直接拦截
//   - 说"可能存在" → 有万分之一的误判，放行走正常流程兜底
//   - 不支持删除 → 已删除文章的 ID 还在布隆里，会被放行，
//     但查库后走"空值缓存"兜底，不会穿透
//
// 容量和误判率是创建时定死的：按 100 万篇文章预估，误判率 0.01%

var articleBloom *bloom.BloomFilter

// Init 服务启动时调用：从 DB 加载全部文章 ID 建布隆
func Init() {
	var ids []uint
	if err := global.DB.Model(&models.ArticleModel{}).Pluck("id", &ids).Error; err != nil {
		logrus.Errorf("布隆过滤器加载文章ID失败: %v", err)
		return
	}
	bf := bloom.NewWithEstimates(1000000, 0.0001) // 预估容量100万，误判率0.01%
	for _, id := range ids {
		bf.AddString(idKey(id))
	}
	articleBloom = bf
	logrus.Infof("布隆过滤器初始化完成，文章数=%d", len(ids))
}

// Test 文章 ID 是否可能存在
func Test(id uint) bool {
	if articleBloom == nil {
		return true // 布隆没建起来：放行降级，绝不能误杀真实文章
	}
	return articleBloom.TestString(idKey(id))
}

// Add 新文章创建后加入布隆
func Add(id uint) {
	if articleBloom == nil {
		return
	}
	articleBloom.AddString(idKey(id))
}

func idKey(id uint) string {
	return strconv.FormatUint(uint64(id), 10) // 把一个 uint64 数字转成字符串
}
