package image_api

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"go-star/common/res"
	"go-star/global"
	"go-star/models"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var allowExt = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

const maxSize = 2 << 20 // 2MB

func (ImageApi) ImageUploadView(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		res.FailWithMsg("请选择文件", c)
		return
	}

	if fileHeader.Size > maxSize {
		res.FailWithMsg("文件大小超出限制", c)
		return
	}

	ext := strings.ToLower(path.Ext(fileHeader.Filename))
	if !allowExt[ext] {
		res.FailWithMsg("文件类型不支持", c)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		res.FailWithMsg("文件打开失败", c)
		return
	}
	defer file.Close()

	// 读入算hash（去重）
	data, err := io.ReadAll(file)
	if err != nil {
		res.FailWithMsg("文件读取失败", c)
		return
	}
	sum := md5.Sum(data)
	hash := hex.EncodeToString(sum[:])

	// 已存在则直接返回，不重复存盘
	var exists models.ImageModel
	err = global.DB.Where("hash = ?", hash).First(&exists).Error
	if err == nil {
		res.OKWithData(gin.H{
			"id":       exists.ID,
			"webPath":  exists.WebPath(),
			"filename": exists.Filename,
		}, c)
		return
	}

	day := time.Now().Format("2006-01-02")
	dir := path.Join("uploads", day)
	_ = os.MkdirAll(dir, os.ModePerm)

	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	relPath := path.Join(day, filename)       // 库里存：日期/文件名
	savePath := path.Join("uploads", relPath) // 磁盘：uploads/日期/文件名

	err = os.WriteFile(savePath, data, 0644)
	if err != nil {
		res.FailWithMsg("文件保存失败", c)
		return
	}

	image := models.ImageModel{
		Filename: fileHeader.Filename,
		Path:     relPath,
		Size:     fileHeader.Size,
		Hash:     hash,
	}
	err = global.DB.Create(&image).Error
	if err != nil {
		res.FailWithMsg("入库失败", c)
		return
	}
	res.OKWithData(gin.H{
		"id":       image.ID,
		"webPath":  image.WebPath(),
		"filename": image.Filename,
	}, c)
}
