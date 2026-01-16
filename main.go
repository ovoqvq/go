package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// Photo 模型定义
type Photo struct {
	gorm.Model
	Filename  string `json:"filename"`
	Url       string `json:"url"`
	SortOrder int    `json:"sort_order"`
}

var db *gorm.DB

// 数据库初始化
func initDB() {
	var err error
	// 这里使用的是新导入的 glebarez/sqlite
	db, err = gorm.Open(sqlite.Open("photo.db"), &gorm.Config{})
	if err != nil {
		fmt.Printf("数据库连接依然失败: %v\n", err)
		panic(err)
	}
	fmt.Println("数据库连接成功！已创建/加载 photo.db")
	db.AutoMigrate(&Photo{})
}

func main() {
	// 【关键】必须先初始化数据库，否则下面的 db 变量都是 nil
	initDB()

	r := gin.Default()

	// 确保上传目录存在
	os.MkdirAll("./uploads", os.ModePerm)

	// 1. 静态资源映射
	r.StaticFile("/", "./static/index.html")
	r.Static("/images", "./uploads")

	// 2. 上传接口
	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("photo")
		if err != nil {
			c.JSON(400, gin.H{"error": "文件上传失败"})
			return
		}

		dst := filepath.Join("./uploads", file.Filename)
		c.SaveUploadedFile(file, dst)

		newPhoto := Photo{
			Filename:  file.Filename,
			Url:       "/images/" + file.Filename,
			SortOrder: 0,
		}
		db.Create(&newPhoto)

		c.JSON(200, newPhoto) // 返回整个对象，包含数据库分配的 ID
	})

	// 3. 获取列表
	r.GET("/api/photos", func(c *gin.Context) {
		var photos []Photo
		// 核心：必须加上 Order("sort_order asc")
		db.Order("sort_order asc").Find(&photos)
		c.JSON(200, photos)
	})

	// 4. 删除接口
	r.DELETE("/api/photos/:name", func(c *gin.Context) {
		name := filepath.Base(c.Param("name"))
		db.Where("filename = ?", name).Delete(&Photo{})
		os.Remove(filepath.Join("./uploads", name))
		c.JSON(200, gin.H{"message": "deleted"})
	})

	// 5. 排序接口
	r.POST("/api/reorder", func(c *gin.Context) {
		var order []string
		if err := c.ShouldBindJSON(&order); err != nil {
			c.JSON(400, gin.H{"error": "参数错误"})
			return
		}

		// 开启事务处理，确保数据一致性
		db.Transaction(func(tx *gorm.DB) error {
			for index, filename := range order {
				// 根据文件名，更新它对应的排序字段
				// 注意：index 就是新的顺序（0, 1, 2...）
				if err := tx.Model(&Photo{}).Where("filename = ?", filename).Update("sort_order", index).Error; err != nil {
					return err // 如果出错，事务会自动回滚
				}
			}
			return nil
		})

		c.JSON(200, gin.H{"status": "顺序保存成功"})
	})
	r.Run(":8081")
}
