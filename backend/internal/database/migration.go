package database

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
	"trade-journal/internal/minio"

	"github.com/google/uuid"
	miniogo "github.com/minio/minio-go/v7"
)

var base64Regex = regexp.MustCompile(`data:image/([a-zA-Z0-9]+);base64,([a-zA-Z0-9+/=]+)`)

// MigrateBase64ToMinIO 將資料庫中所有 Base64 圖片遷移到 MinIO
func MigrateBase64ToMinIO(db *sql.DB, minioClient *miniogo.Client) {
	log.Println("[Migration] 開始執行 Base64 -> MinIO 遷移作業...")
	start := time.Now()

	// 1. 遷移每日規劃 (daily_plans)
	migrateDailyPlans(db, minioClient)

	// 2. 遷移交易紀錄 (trades)
	migrateTrades(db, minioClient)

	log.Printf("[Migration] 遷移作業完成，耗時: %v", time.Since(start))
}

func migrateDailyPlans(db *sql.DB, client *miniogo.Client) {
	rows, err := db.Query("SELECT id, notes, trend_analysis FROM daily_plans WHERE notes LIKE '%data:image/%' OR trend_analysis LIKE '%data:image/%'")
	if err != nil {
		log.Printf("[Migration] 查詢 daily_plans 失敗: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var notes, trendAnalysis string
		if err := rows.Scan(&id, &notes, &trendAnalysis); err != nil {
			continue
		}

		newNotes, nChanged := processContent(notes, client, "plan-note")
		newTrend, tChanged := processContent(trendAnalysis, client, "plan-trend")

		if nChanged || tChanged {
			_, err := db.Exec("UPDATE daily_plans SET notes = ?, trend_analysis = ?, updated_at = updated_at WHERE id = ?", newNotes, newTrend, id)
			if err != nil {
				log.Printf("[Migration] 更新 daily_plans ID:%d 失敗: %v", id, err)
			} else {
				log.Printf("[Migration] 已遷移 daily_plans ID:%d 的圖片 (%d 欄位有變動)", id, countChanges(nChanged, tChanged))
			}
		}
	}
}

func migrateTrades(db *sql.DB, client *miniogo.Client) {
	query := `
		SELECT id, notes, entry_reason, exit_reason, entry_signals, entry_pattern, 
		       entry_strategy_image, entry_strategy_image_original,
		       legend_king_image, legend_king_image_original,
		       legend_htf_image, legend_htf_image_original
		FROM trades 
		WHERE notes LIKE '%data:image/%' OR entry_reason LIKE '%data:image/%' 
		   OR exit_reason LIKE '%data:image/%' OR entry_signals LIKE '%data:image/%'
		   OR entry_pattern LIKE '%data:image/%' OR entry_strategy_image LIKE 'data:image/%'
		   OR legend_king_image LIKE 'data:image/%' OR legend_htf_image LIKE 'data:image/%'
	`
	rows, err := db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var n, er, exr, s, p, si, sio, ki, kio, hi, hio sql.NullString

		if err := rows.Scan(&id, &n, &er, &exr, &s, &p, &si, &sio, &ki, &kio, &hi, &hio); err != nil {
			continue
		}

		nN, c1 := processContent(n.String, client, "trade-note")
		nER, c2 := processContent(er.String, client, "trade-entry-reason")
		nEXR, c3 := processContent(exr.String, client, "trade-exit-reason")
		nS, c4 := processContent(s.String, client, "trade-signal")
		nP, c5 := processContent(p.String, client, "trade-pattern")
		nSI, c6 := processContent(si.String, client, "trade-strat")
		nSIO, c7 := processContent(sio.String, client, "trade-strat-ori")
		nKI, c8 := processContent(ki.String, client, "trade-king")
		nKIO, c9 := processContent(kio.String, client, "trade-king-ori")
		nHI, c10 := processContent(hi.String, client, "trade-htf")
		nHIO, c11 := processContent(hio.String, client, "trade-htf-ori")

		if c1 || c2 || c3 || c4 || c5 || c6 || c7 || c8 || c9 || c10 || c11 {
			_, err := db.Exec(`
				UPDATE trades SET 
					notes=?, entry_reason=?, exit_reason=?, entry_signals=?, entry_pattern=?,
					entry_strategy_image=?, entry_strategy_image_original=?,
					legend_king_image=?, legend_king_image_original=?,
					legend_htf_image=?, legend_htf_image_original=?,
					updated_at = updated_at
				WHERE id = ?`,
				nN, nER, nEXR, nS, nP, nSI, nSIO, nKI, nKIO, nHI, nHIO, id)
			if err != nil {
				log.Printf("[Migration] 更新 trades ID:%d 失敗: %v", id, err)
			} else {
				log.Printf("[Migration] 已遷移 trades ID:%d 的圖片", id)
			}
		}
	}
}

func countChanges(bools ...bool) int {
	count := 0
	for _, b := range bools {
		if b {
			count++
		}
	}
	return count
}

func processContent(content string, client *miniogo.Client, prefix string) (string, bool) {
	if content == "" || !strings.Contains(content, "data:image/") {
		return content, false
	}

	changed := false
	matches := base64Regex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		fullMatch := match[0]
		ext := match[1]
		b64Data := match[2]

		// 解碼 Base64
		data, err := base64.StdEncoding.DecodeString(b64Data)
		if err != nil {
			log.Printf("[Migration] 解碼失敗: %v", err)
			continue
		}

		// 上傳到 MinIO
		now := time.Now()
		fileName := fmt.Sprintf("%s-%s-%s.%s",
			now.Format("20060102"),
			prefix,
			uuid.New().String()[:8],
			ext,
		)
		objectPath := fmt.Sprintf("%s/%s", now.Format("2006-01"), fileName)

		_, err = client.PutObject(context.Background(), minio.BucketName, objectPath, bytes.NewReader(data), int64(len(data)), miniogo.PutObjectOptions{
			ContentType: "image/" + ext,
		})

		if err == nil {
			content = strings.Replace(content, fullMatch, objectPath, 1)
			changed = true
		} else {
			log.Printf("[Migration] 上傳失敗: %v", err)
		}
	}

	return content, changed
}
