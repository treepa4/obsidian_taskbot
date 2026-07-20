package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	ChatID        int64
	VaultPath     string
	BoardFilePath string
}

func Load() *Config {

	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, вычитываем переменные из системы")
	}

	chatID, _ := strconv.ParseInt(os.Getenv("CHAT_ID"), 10, 64)

	vaultPath := os.Getenv("VAULT_PATH")
	if vaultPath == "" {
		vaultPath = "/root/obsidian-vault"
	}

	boardFile := os.Getenv("BOARD_FILE")
	if boardFile == "" {
		boardFile = "test_board.md"
	}

	return &Config{
		TelegramToken: os.Getenv("BOT_TOKEN"),
		ChatID:        chatID,
		VaultPath:     vaultPath,
		BoardFilePath: vaultPath + "/" + boardFile,
	}
}
