package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"trade-journal/internal/minio"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	miniogo "github.com/minio/minio-go/v7"
)

// UploadImage 上傳圖片到MinIO
func UploadImage(client *miniogo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, header, err := c.Request.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "請選擇圖片檔案"})
			return
		}
		defer file.Close()

		// 取得交易品種和類型（可選）
		symbol := c.PostForm("symbol")
		if symbol == "" {
			symbol = "UNKNOWN"
		}

		// 生成檔案名稱: YYYY-MM/YYYYMMDD-SYMBOL-UUID.ext
		now := time.Now()
		ext := filepath.Ext(header.Filename)
		fileName := fmt.Sprintf("%s-%s-%s%s",
			now.Format("20060102"),
			symbol,
			uuid.New().String()[:8],
			ext,
		)

		// 按月份組織路徑
		objectPath := fmt.Sprintf("%s/%s", now.Format("2006-01"), fileName)

		// 上傳到MinIO
		ctx := context.Background()
		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "image/jpeg"
		}

	_, err = client.PutObject(ctx, minio.BucketName, objectPath, file, header.Size, miniogo.PutObjectOptions{
		ContentType: contentType,
	})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "圖片上傳失敗: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"path":    objectPath,
			"message": "圖片上傳成功",
		})
	}
}

// GetImage 從MinIO取得圖片
func GetImage(client *miniogo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		filename := c.Param("filename")
		if filename == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "檔案名稱不能為空"})
			return
		}

		// 從query參數取得完整路徑（包含月份資料夾）
		objectPath := c.Query("path")
		if objectPath == "" {
			objectPath = filename
		}

	ctx := context.Background()
	object, err := client.GetObject(ctx, minio.BucketName, objectPath, miniogo.GetObjectOptions{})
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "圖片不存在"})
			return
		}
		defer object.Close()

		// 取得物件資訊
		stat, err := object.Stat()
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "圖片不存在"})
			return
		}

		// 設定回應標頭
		c.Header("Content-Type", stat.ContentType)
		c.Header("Content-Length", fmt.Sprintf("%d", stat.Size))
		c.Header("Cache-Control", "public, max-age=604800, immutable") // 7 days

		// 串流傳送圖片
		io.Copy(c.Writer, object)
	}
}

