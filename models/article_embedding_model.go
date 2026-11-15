package models

// ArticleEmbeddingModel 文章向量表
// 单独建表而不是加字段到 article_models：1024 维向量约 10KB，
// 放进主表会让列表查询被迫带出大量无用数据
type ArticleEmbeddingModel struct {
	Model
	ArticleID  uint      `gorm:"uniqueIndex" json:"articleID"`           // 一篇文章一条
	ModelName  string    `gorm:"size:64" json:"modelName"`               // 生成向量的模型，换模型时用于识别旧数据
	Dim        int       `json:"dim"`                                    // 维度，校验用
	SourceText string    `gorm:"type:text" json:"-"`                     // 向量化用的原文，换模型时可重算
	Vector     []float32 `gorm:"type:longtext;serializer:json" json:"-"` // 向量本体，不返回给前端
}
