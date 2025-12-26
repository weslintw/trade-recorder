package minio

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	BucketName = "trade-journal"
)

// InitMinIO 初始化MinIO客戶端
func InitMinIO() (*minio.Client, error) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	useSSL := os.Getenv("MINIO_USE_SSL") == "true"

	if endpoint == "" {
		endpoint = "localhost:9000"
	}
	if accessKey == "" {
		accessKey = "minioadmin"
	}
	if secretKey == "" {
		secretKey = "minioadmin"
	}

	// 初始化minio客戶端
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("MinIO客戶端初始化失敗: %v", err)
	}

	// 確保bucket存在
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, BucketName)
	if err != nil {
		return nil, fmt.Errorf("檢查bucket失敗: %v", err)
	}

	if !exists {
		err = client.MakeBucket(ctx, BucketName, minio.MakeBucketOptions{})
		if err != nil {
			return nil, fmt.Errorf("建立bucket失敗: %v", err)
		}
		log.Printf("已建立bucket: %s", BucketName)
	}

	// 設定bucket為公開讀取 (可選)
	policy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::` + BucketName + `/*"]
			}
		]
	}`
	err = client.SetBucketPolicy(ctx, BucketName, policy)
	if err != nil {
		log.Printf("設定bucket policy警告: %v", err)
	}

	log.Println("MinIO初始化成功")
	return client, nil
}

