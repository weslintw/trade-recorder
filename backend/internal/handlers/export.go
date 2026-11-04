package handlers

// 臨時搬家用匯出端點（Zeabur → Mac mini）。搬完即可整支刪掉。
//
//   GET /api/v1/_migrate/export?key=<KEY>          → 下載 tar.gz（env.json + trade_journal.db + minio/<object>）
//   GET /api/v1/_migrate/export?key=<KEY>&list=1   → 只回 JSON 摘要，不打包
//
// key 不存環境變數：程式碼只放 SHA-256 雜湊，避免在 Zeabur 後台多設一個變數。
// tar.gz 先寫到磁碟暫存檔再回傳，不塞記憶體（wedge 搬家時塞記憶體被 OOM 502 過）。

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"trade-journal/internal/minio"

	"github.com/gin-gonic/gin"
	miniogo "github.com/minio/minio-go/v7"
)

const migrateExportKeyHash = "876e3592ee7f17606bb19dc89db35884ccf10b43480c35b9482112884eefd0ed"

// 程式碼有讀的環境變數（帶值）；其餘只列名稱不帶值。
var migrateEnvKeys = []string{
	"PORT", "DB_PATH", "GIN_MODE", "JWT_SECRET", "ALLOW_ORIGINS",
	"MINIO_ENDPOINT", "MINIO_ACCESS_KEY", "MINIO_SECRET_KEY", "MINIO_USE_SSL",
	"CTRADER_CLIENT_ID", "CTRADER_CLIENT_SECRET", "CTRADER_REDIRECT_URI",
}

func migrateKeyOK(c *gin.Context) bool {
	k := c.Query("key")
	if k == "" {
		return false
	}
	sum := sha256.Sum256([]byte(k))
	return subtle.ConstantTimeCompare([]byte(hex.EncodeToString(sum[:])), []byte(migrateExportKeyHash)) == 1
}

func migrateEnvSnapshot() map[string]interface{} {
	values := map[string]string{}
	for _, k := range migrateEnvKeys {
		if v, ok := os.LookupEnv(k); ok {
			values[k] = v
		}
	}
	names := []string{}
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i > 0 {
			names = append(names, kv[:i])
		}
	}
	return map[string]interface{}{"values": values, "all_names": names}
}

func migrateDBPath() string {
	p := os.Getenv("DB_PATH")
	if p == "" {
		p = "./trade_journal.db"
	}
	if info, err := os.Stat(p); err == nil && info.IsDir() {
		p = filepath.Join(p, "trade_journal.db")
	}
	return p
}

type migrateObj struct {
	Key  string
	Size int64
}

func migrateListObjects(ctx context.Context, client *miniogo.Client) ([]migrateObj, int64, error) {
	var out []migrateObj
	var total int64
	for o := range client.ListObjects(ctx, minio.BucketName, miniogo.ListObjectsOptions{Recursive: true}) {
		if o.Err != nil {
			return out, total, o.Err
		}
		out = append(out, migrateObj{Key: o.Key, Size: o.Size})
		total += o.Size
	}
	return out, total, nil
}

// MigrateExport 匯出整站資料（臨時）。
func MigrateExport(db *sql.DB, client *miniogo.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !migrateKeyOK(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		ctx := c.Request.Context()
		dbPath := migrateDBPath()
		var dbSize int64
		if st, err := os.Stat(dbPath); err == nil {
			dbSize = st.Size()
		}
		var objs []migrateObj
		var objBytes int64
		rawDir := ""
		if client != nil {
			var err error
			objs, objBytes, err = migrateListObjects(ctx, client)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "list objects: " + err.Error()})
				return
			}
		} else {
			// MinIO 沒起來：改把它的原始資料夾整個帶走（新機用同版 MinIO 指到這個資料夾即可讀）
			rawDir = os.Getenv("MINIO_DATA_DIR")
			if rawDir == "" {
				rawDir = "/app/data/minio"
			}
		}
		summary := gin.H{
			"db_path": dbPath, "db_bytes": dbSize,
			"minio_objects": len(objs), "minio_bytes": objBytes, "minio_raw_dir": rawDir, "minio_via_s3": client != nil,
			"env": migrateEnvSnapshot()["all_names"],
			"cwd": func() string { d, _ := os.Getwd(); return d }(),
		}
		if c.Query("list") != "" {
			c.JSON(http.StatusOK, summary)
			return
		}

		tmpDir, err := os.MkdirTemp("", "migrate-export-*")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer os.RemoveAll(tmpDir)

		// 1) DB 一致性快照（WAL 模式下直接 copy 檔案會缺 -wal 的內容）
		snapPath := filepath.Join(tmpDir, "trade_journal.db")
		if _, err := db.ExecContext(ctx, "VACUUM INTO ?", snapPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "vacuum into: " + err.Error()})
			return
		}

		// 2) 打包到磁碟
		tarPath := filepath.Join(tmpDir, "export.tar.gz")
		if err := migrateBuildTar(ctx, client, tarPath, snapPath, objs, rawDir); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "build tar: " + err.Error()})
			return
		}
		st, err := os.Stat(tarPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[migrate] export ready: %d bytes (db %d, %d objects %d bytes)", st.Size(), dbSize, len(objs), objBytes)
		name := fmt.Sprintf("trade-recorder-export-%s.tar.gz", time.Now().UTC().Format("20060102-150405"))
		c.Header("Content-Disposition", "attachment; filename="+name)
		c.Header("X-Export-Objects", fmt.Sprintf("%d", len(objs)))
		c.Header("X-Export-DB-Bytes", fmt.Sprintf("%d", dbSize))
		c.File(tarPath)
	}
}

func migrateBuildTar(ctx context.Context, client *miniogo.Client, tarPath, snapPath string, objs []migrateObj, rawDir string) error {
	f, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	now := time.Now()

	addBytes := func(name string, b []byte) error {
		h := &tar.Header{Name: name, Mode: 0600, Size: int64(len(b)), ModTime: now}
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		_, err := tw.Write(b)
		return err
	}

	envRaw, _ := json.MarshalIndent(migrateEnvSnapshot(), "", "  ")
	if err := addBytes("env.json", envRaw); err != nil {
		return err
	}

	// DB 快照
	sf, err := os.Open(snapPath)
	if err != nil {
		return err
	}
	sst, _ := sf.Stat()
	if err := tw.WriteHeader(&tar.Header{Name: "trade_journal.db", Mode: 0600, Size: sst.Size(), ModTime: now}); err != nil {
		sf.Close()
		return err
	}
	if _, err := io.Copy(tw, sf); err != nil {
		sf.Close()
		return err
	}
	sf.Close()

	// MinIO 沒起來：原始資料夾 → minio_raw/...
	if rawDir != "" {
		if err := migrateAddDir(tw, rawDir, "minio_raw"); err != nil {
			return err
		}
	}

	// MinIO 物件：以物件名稱存成 minio/<key>，新機用 S3 client 重新上傳即可，不依賴 MinIO 磁碟格式
	manifest := make([]map[string]interface{}, 0, len(objs))
	for _, o := range objs {
		obj, err := client.GetObject(ctx, minio.BucketName, o.Key, miniogo.GetObjectOptions{})
		if err != nil {
			return fmt.Errorf("get %s: %w", o.Key, err)
		}
		st, err := obj.Stat()
		if err != nil {
			obj.Close()
			return fmt.Errorf("stat %s: %w", o.Key, err)
		}
		if err := tw.WriteHeader(&tar.Header{Name: "minio/" + o.Key, Mode: 0600, Size: st.Size, ModTime: st.LastModified}); err != nil {
			obj.Close()
			return err
		}
		if _, err := io.Copy(tw, obj); err != nil {
			obj.Close()
			return fmt.Errorf("copy %s: %w", o.Key, err)
		}
		obj.Close()
		manifest = append(manifest, map[string]interface{}{
			"key": o.Key, "size": st.Size, "content_type": st.ContentType, "last_modified": st.LastModified,
		})
	}
	mRaw, _ := json.MarshalIndent(map[string]interface{}{"bucket": minio.BucketName, "objects": manifest}, "", "  ")
	if err := addBytes("minio_manifest.json", mRaw); err != nil {
		return err
	}

	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

func migrateAddDir(tw *tar.Writer, root, prefix string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." {
			return nil
		}
		name := prefix + "/" + filepath.ToSlash(rel)
		if info.IsDir() {
			return tw.WriteHeader(&tar.Header{Name: name + "/", Mode: 0700, Typeflag: tar.TypeDir, ModTime: info.ModTime()})
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: info.Size(), ModTime: info.ModTime()}); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}
