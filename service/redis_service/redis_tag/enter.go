package redis_tag

import (
	"errors"
	"fmt"
	"go-star/global"
	"go-star/models"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 文章的标签的集合key
func articleTagsKey(articleID uint) string {
	return fmt.Sprintf("article:tags:%d", articleID)
}

// 标签下的文章集合
func tagArticlesKey(tagName string) string {
	return fmt.Sprintf("tag:articles:%s", tagName)
}

// 规范化标签：去空格、去空串、去重
func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// 判断是否是唯一索引冲突错误
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") || strings.Contains(msg, "1062")
}

// 往 Redis 写双向集合（文章→标签、标签→文章）
// Redis失败只记日志，不影响主流程
func addTagToRedis(articleID uint, tags []string) {
	if global.Redis == nil {
		return
	}
	articleKey := articleTagsKey(articleID)
	members := make([]interface{}, len(tags))
	for i, tag := range tags {
		members[i] = tag
	}
	if err := global.Redis.SAdd(articleKey, members...).Err(); err != nil {
		logrus.Errorf("redis写入文章标签失败 articleID=%d: %v", articleID, err)
		return
	}
	for _, tag := range tags {
		if err := global.Redis.SAdd(tagArticlesKey(tag), articleID).Err(); err != nil {
			logrus.Errorf("redis写入标签文章失败 tag=%s: %v", tag, err)
		}
	}
}

// 给文章添加标签（双写：先mysql，后redis）
func AddArticleTags(articleID uint, tags []string) error {
	tags = normalizeTags(tags)
	if len(tags) == 0 {
		return nil
	}

	// 先写Myusql（事务：标签和关联要么都成功，要么都回滚）
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		for _, name := range tags {
			// 标签不存在就创建
			var tag models.TagModel
			err := tx.Where("name = ?", name).Take(&tag).Error
			if err != nil {
				if !errors.Is(err, gorm.ErrRecordNotFound) {
					return err
				}
				tag = models.TagModel{Name: name}
				if err := tx.Create(&tag).Error; err != nil {
					// 并发下可能别人刚创建了同名标签，重查一下
					if isDuplicateKey(err) {
						if err := tx.Where("name = ?", name).Take(&tag).Error; err != nil {
							return err
						}
					} else {
						return err
					}
				}
			}
			// 关联已存在就跳过（幂等）
			var rel models.ArticleTagModel
			err = tx.Where("article_id = ? AND tag_id = ?", articleID, tag.ID).Take(&rel).Error
			if err == nil {
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			if err := tx.Create(&models.ArticleTagModel{
				ArticleID: articleID,
				TagID:     tag.ID,
			}).Error; err != nil {
				// 唯一索引兜底并发重复，已存在就跳过
				if isDuplicateKey(err) {
					continue
				}
				return err
			}
			if err := tx.Model(&tag).Update("article_count", gorm.Expr("article_count + ?", 1)).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Mysql 成功后再写redis
	addTagToRedis(articleID, tags)
	return nil
}

// 获取文章的所有标签
func GetArticleTags(articleID uint) ([]string, error) {
	// 优先Redis
	if global.Redis != nil {
		// SMembers:获取redis中set集合中的所有元素
		tags, err := global.Redis.SMembers(articleTagsKey(articleID)).Result()
		if err == nil && len(tags) > 0 {
			return tags, nil
		}
	}

	// Mysql兜底
	var rels []models.ArticleTagModel
	if err := global.DB.Preload("TagModel").Where("article_id = ?", articleID).Find(&rels).Error; err != nil {
		return nil, err
	}
	tags := make([]string, 0, len(rels))
	for _, rel := range rels {
		tags = append(tags, rel.TagModel.Name)
	}

	// 回填Redis（下次直接命中）
	if len(tags) > 0 {
		addTagToRedis(articleID, tags)
	}
	return tags, nil
}

// 把redis中取出的字符串集合转成uint切片
func parseIDs(members []string) []uint {
	ids := make([]uint, 0, len(members))
	for _, m := range members {
		id, err := strconv.ParseUint(m, 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, uint(id))
	}
	return ids
}

// 获取某个标签下的所有文章的ID（优先redis，未命中查Mysql并回填）
func GetTagArticles(tagName string) ([]uint, error) {
	if global.Redis != nil {
		members, err := global.Redis.SMembers(tagArticlesKey(tagName)).Result()
		if err == nil && len(members) > 0 {
			return parseIDs(members), nil
		}
	}

	var tag models.TagModel
	if err := global.DB.Where("name = ?", tagName).Take(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []uint{}, nil // 标签不存在，返回空
		}
		return nil, err
	}
	var rels []models.ArticleTagModel
	if err := global.DB.Where("tag_id = ?", tag.ID).Find(&rels).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rels))
	for _, rel := range rels {
		ids = append(ids, rel.ArticleID)
	}

	// 回填redis
	if len(ids) > 0 && global.Redis != nil {
		members := make([]interface{}, len(ids))
		for i, id := range ids {
			members[i] = id
		}
		_ = global.Redis.SAdd(tagArticlesKey(tagName), members...).Err()
	}
	return ids, nil
}

// 获取两篇文章的共同得的标签（优先redis交集。mysql兜底）
func GetCommonTags(articleID1, articleID2 uint) ([]string, error) {
	if global.Redis != nil {
		common, err := global.Redis.SInter(articleTagsKey(articleID1), articleTagsKey(articleID2)).Result()
		if err == nil {
			return common, nil
		}
	}

	// Musql兜底，分别查两篇文章的标签，在go里求交集
	tags1, err := GetArticleTags(articleID1)
	if err != nil {
		return nil, err
	}
	tags2, err := GetArticleTags(articleID2)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(tags1))
	for _, t := range tags1 {
		set[t] = struct{}{}
	}
	common := make([]string, 0)
	for _, t := range tags2 {
		if _, ok := set[t]; ok {
			common = append(common, t)
		}
	}
	return common, nil
}

// 查询包含任意一个标签的文章ID
func GetArticlesByTags(tags []string) ([]uint, error) {
	tags = normalizeTags(tags)
	if len(tags) == 0 {
		return nil, fmt.Errorf("标签不能为空")
	}

	if global.Redis != nil {
		keys := make([]string, len(tags))
		for i, tag := range tags {
			keys[i] = tagArticlesKey(tag)
		}
		// global.Redis.SUnion：redis中求多个set集合中的并集
		members, err := global.Redis.SUnion(keys...).Result()
		if err == nil && len(members) > 0 {
			return parseIDs(members), nil
		}
	}

	var tagIDs []uint
	// Pluck：只取符合条件的id的值，而不是整行全部数据
	if err := global.DB.Model(&models.TagModel{}).Where("name IN ?", tags).Pluck("id", &tagIDs).Error; err != nil {
		return nil, err
	}
	if len(tagIDs) == 0 {
		return []uint{}, nil
	}
	var ids []uint
	// Distinct:去重，这里针对Pluck选中的字段进行去重
	if err := global.DB.Model(&models.ArticleTagModel{}).
		Where("tag_id IN ?", tagIDs).
		Distinct().
		Pluck("article_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

// 移除文章的单个标签
func RemoveArticleTag(articleID uint, tagName string) error {
	// TrimSpace:去除字符串首尾的空格
	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		return nil
	}

	//先删Mysql
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		var tag models.TagModel
		if err := tx.Where("name = ?", tagName).Take(&tag).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		// 删关联
		result := tx.Where("article_id = ? AND tag_id = ?", articleID, tag.ID).
			Delete(&models.ArticleTagModel{})
		if result.Error != nil {
			return result.Error
		}
		// 删除成功了，计数才减一
		if result.RowsAffected > 0 {
			return tx.Model(&models.TagModel{}).
				Where("id = ? AND article_count > ?", tag.ID, 0).
				Update("article_count", gorm.Expr("article_count - ?", 1)).Error
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 再清除Redis双向索引
	if global.Redis != nil {
		// global.Redis.SRem：删除redis中set集合中的一个元素
		_ = global.Redis.SRem(tagArticlesKey(tagName), articleID).Err()
		_ = global.Redis.SRem(articleTagsKey(articleID), tagName).Err()
	}
	return nil
}

// 删除文章所有标签
func RemoveArticleTags(articleID uint) error {
	// 先拿标签名（Redis 优先，MySQL 兜底），后面清 Redis 用
	tags, _ := GetArticleTags(articleID)

	// 删除Mysql
	err := global.DB.Transaction(func(tx *gorm.DB) error {
		var tagIDs []uint
		// 从关联关系表中获取该文章所有的tag对应的id
		if err := tx.Model(&models.ArticleTagModel{}).
			Where("article_id = ?", articleID).Pluck("tag_id", &tagIDs).Error; err != nil {
			return err
		}
		if len(tagIDs) == 0 {
			return nil
		}
		// 删除关联关系表中所有article_id为articleID的数据
		if err := tx.Where("article_id = ?", articleID).
			Delete(&models.ArticleTagModel{}).Error; err != nil {
			return err
		}
		// 更新tag表中文章的数量
		// 注意这里是id IN，所以可以传数组tagIDs
		return tx.Model(&models.TagModel{}).
			Where("id IN ? AND article_count > ?", tagIDs, 0).
			Update("article_count", gorm.Expr("article_count - ?", 1)).Error
	})
	if err != nil {
		return err
	}
	if global.Redis != nil {
		for _, tag := range tags {
			_ = global.Redis.SRem(tagArticlesKey(tag), articleID).Err()
		}
		_ = global.Redis.Del(articleTagsKey(articleID)).Err()
	}
	return nil
}

// 对比新旧标签，返回需要新增和需要删除的标签
func DiffTags(oldTags, newTags []string) (addTags, removeTags []string) {
	oldSet := make(map[string]struct{}, len(oldTags))
	for _, t := range oldTags {
		oldSet[t] = struct{}{}
	}
	newSet := make(map[string]struct{}, len(newTags))
	for _, t := range newTags {
		newSet[t] = struct{}{}
	}
	for _, t := range newTags {
		if _, ok := oldSet[t]; !ok {
			addTags = append(addTags, t)
		}
	}
	for _, t := range oldTags {
		if _, ok := newSet[t]; !ok {
			removeTags = append(removeTags, t)
		}
	}
	return
}
