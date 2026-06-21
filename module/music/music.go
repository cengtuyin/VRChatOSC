package music

import (
	"context"
	"log"
	"time"
	"vrchat_osc/module/config"

	libnp "github.com/delthas/go-libnp"
)

func Init() {
	// fmt.Println("开始监听媒体播放信息...")

	// 定时轮询
	ticker := time.NewTicker(2 * time.Second) // 每2秒查询一次
	defer ticker.Stop()

	// 第一次查询可以立即执行
	fetchAndDisplay()

	for range ticker.C {
		fetchAndDisplay()
	}
}

func fetchAndDisplay() {
	defer func() {
		if r := recover(); r != nil {
			// log.Printf("捕获到 panic (可能无媒体会话): %v", r)
		}
	}()

	// 1. 获取当前媒体信息
	info, err := libnp.GetInfo(context.Background())
	if err != nil {
		config.MusicPlaying = false
		log.Printf("获取媒体信息出错: %v", err)
		return
	}

	// 2. 检查是否有媒体在播放
	if info == nil {
		config.MusicPlaying = false
		// 没有媒体在播放，info 为 nil
		return
	}

	config.MusicPlaying = true
	config.MusicTitle = info.Title
	if len(info.Artists) > 0 {
		config.MusicArtist = info.Artists[0]
	} else {
		config.MusicArtist = ""
	}
}
