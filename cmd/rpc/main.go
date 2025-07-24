package main

import (
	"context"
	"database/sql"
	"dex-ingest-sol/cmd/rpc/common"
	"dex-ingest-sol/cmd/rpc/db"
	_ "github.com/go-sql-driver/mysql" // 替换为你用的 driver
	_ "github.com/lib/pq"
	"log"
	"time"
)

const lindormSourceDSN = ""
const lindormDSN = ""

func main() {

	lindormSource, err := sql.Open("mysql", lindormSourceDSN)
	if err != nil {
		log.Fatalf("连接lindorm失败: %v", err)
	}
	defer lindormSource.Close()

	lindorm, err := sql.Open("mysql", lindormDSN)
	if err != nil {
		log.Fatalf("连接lindorm失败: %v", err)
	}
	defer lindorm.Close()

	//cl := client.NewClient(endpoint)

	lastAddr := ""
	total := 0

	for {
		var pools []*common.Pool
		lastAddr, pools = db.LoadPools(lindormSource, lastAddr)
		if len(pools) == 0 {
			log.Printf("✅ 查询到空池，迁移完成，总计迁移 %d 个 pools", total)
			return
		}

		log.Printf("准备写入 %d 个 pools，lastAddr=%s", len(pools), lastAddr)

		err := db.InsertPools(context.Background(), lindorm, pools)
		if err != nil {
			log.Printf("❌ 写入失败 at %s: %v", lastAddr, err)
			time.Sleep(200 * time.Millisecond)
			continue
		}

		total += len(pools)
		log.Printf("✅ 当前已迁移 %d 个 pools，最新 lastAddr=%s", total, lastAddr)

		time.Sleep(20 * time.Millisecond) // 防止压垮 DB
	}
}
