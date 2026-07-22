package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/treepa4/obsidian_taskbot/internal/git"
	"github.com/treepa4/obsidian_taskbot/internal/notifier"
	"github.com/treepa4/obsidian_taskbot/internal/tg"
)

func main() {
	_ = godotenv.Load()

	telegramToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if telegramToken == "" {
		log.Fatal("❌ Ошибка: TELEGRAM_BOT_TOKEN не задан в .env")
	}

	vaultPath := os.Getenv("OBSIDIAN_VAULT_PATH")
	if vaultPath == "" {
		vaultPath = "/vault"
	}

	relBoardPath := os.Getenv("KANBAN_FILE_PATH")
	if relBoardPath == "" {
		relBoardPath = "заметки/Таски.md"
	}

	boardFilePath := filepath.Join(vaultPath, relBoardPath)

	log.Printf("🚀 Запуск Obsidian TaskBot...")

	git.PullVault(vaultPath)

	bot, err := tg.New(telegramToken, 0)
	if err != nil {
		log.Fatalf("❌ Ошибка инициализации Telegram бота: %v", err)
	}

	gitPushFunc := func(commitMsg string) {
		go func() {
			if err := git.PushVault(vaultPath, commitMsg); err != nil {
				log.Printf("⚠️ Ошибка Git Push: %v", err)
			} else {
				log.Printf("✅ Git push выполнен: %s", commitMsg)
			}
		}()
	}

	go notifier.StartScheduler(bot, boardFilePath, vaultPath)

	log.Println("🤖 Бот успешно запущен...")
	bot.StartListener(boardFilePath, gitPushFunc)
}
