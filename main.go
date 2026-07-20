package main

import (
	"log"
	"time"

	"github.com/treepa4/obsidian_taskbot/internal/config"
	"github.com/treepa4/obsidian_taskbot/internal/git"
	"github.com/treepa4/obsidian_taskbot/internal/kanban"
	"github.com/treepa4/obsidian_taskbot/internal/notifier"
	"github.com/treepa4/obsidian_taskbot/internal/tg"
)

func main() {
	log.Println("🚀 Запуск Obsidian Kanban Bot...")

	cfg := config.Load()
	if cfg.TelegramToken == "" {
		log.Fatal("❌ Ошибка: BOT_TOKEN не задан в .env!")
	}

	gitClient := git.New(cfg.VaultPath)

	tgBot, err := tg.New(cfg.TelegramToken, cfg.ChatID)
	if err != nil {
		log.Fatalf("❌ Ошибка инициализации Telegram бота: %v", err)
	}

	notif := notifier.New("history.json")

	gitPushFunc := func(commitMsg string) {
		if err := gitClient.CommitAndPush(commitMsg); err != nil {
			log.Printf("⚠️ Git push error: %v", err)
		}
	}

	go tgBot.StartListener(cfg.BoardFilePath, gitPushFunc)

	log.Println("✅ Все сервисы инициализированы. Демон запущен!")

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if err := gitClient.Pull(); err != nil {
			log.Printf("⚠️ Git pull warning: %v", err)
		}

		tasks, err := kanban.ParseKanban(cfg.BoardFilePath)
		if err != nil {
			log.Printf("⚠️ Ошибка парсинга Kanban: %v", err)
			continue
		}

		notif.Process(tgBot, tasks)
	}
}
