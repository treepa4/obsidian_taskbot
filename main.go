package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/treepa4/obsidian_taskbot/internal/git"
	"github.com/treepa4/obsidian_taskbot/internal/kanban"
	"github.com/treepa4/obsidian_taskbot/internal/notifier"
	"github.com/treepa4/obsidian_taskbot/internal/tg"
)

type Config struct {
	TelegramToken string
	ChatID        int64
	VaultPath     string
	BoardFile     string
}

func loadConfig() Config {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не установлен в переменных окружения")
	}

	chatIDStr := os.Getenv("TELEGRAM_CHAT_ID")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Ошибка парсинга TELEGRAM_CHAT_ID: %v", err)
	}

	vaultPath := os.Getenv("OBSIDIAN_VAULT_PATH")
	if vaultPath == "" {
		vaultPath = "./obsidian-vault"
	}

	boardFile := os.Getenv("KANBAN_BOARD_FILE")
	if boardFile == "" {
		boardFile = "Таски.md"
	}

	return Config{
		TelegramToken: token,
		ChatID:        chatID,
		VaultPath:     vaultPath,
		BoardFile:     boardFile,
	}
}

func main() {
	cfg := loadConfig()

	// 1. Инициализация Telegram бота
	bot, err := tg.NewBot(cfg.TelegramToken, cfg.ChatID, cfg.VaultPath, cfg.BoardFile)
	if err != nil {
		log.Fatalf("Ошибка запуска бота: %v", err)
	}
	log.Println("🤖 Бот успешно запущен!")

	// 2. Инициализация и запуск уведомлений (дайджестов)
	n := notifier.NewNotifier(bot, cfg.VaultPath, cfg.BoardFile)
	n.Start()
	log.Println("⏰ Уведомления и дайджесты активированы (09:00 и 21:00 MSK)")

	// 3. Обработка входящих сообщений и inline-кнопок
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	botAPI, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("Ошибка подключения к API Telegram: %v", err)
	}

	updates := botAPI.GetUpdatesChan(u)

	// Перехватываем системные сигналы для корректного завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		for update := range updates {
			// Обработка текстовых команд и добавления задач
			if update.Message != nil {
				if update.Message.Chat.ID != cfg.ChatID {
					continue
				}

				text := update.Message.Text
				filePath := filepath.Join(cfg.VaultPath, cfg.BoardFile)

				switch text {
				case "/start", "/help":
					bot.SendMessage("👋 Привет! Я твой Obsidian Kanban бот.\n\n" +
						"🔹 Отправь любой текст — я добавлю его как задачу.\n" +
						"🔹 /tasks — показать список активных задач.")
				case "/tasks", "/list":
					bot.SendTasksList()
				default:
					// Парсим задачу из текста
					task, obsidianLine := kanban.ParseNaturalLanguage(text)
					targetCol := "Надо сделать"
					if task.Priority {
						targetCol = "СРОЧНО!!!"
					}

					// Записываем в файл Obsidian
					err := kanban.AddTaskToFile(filePath, obsidianLine, targetCol)
					if err != nil {
						bot.SendMessage("❌ Ошибка при добавлении задачи: " + err.Error())
						continue
					}

					// Синхронизация с Git
					go git.SyncVault(cfg.VaultPath, "bot: add task '"+task.Text+"'")

					bot.SendMessage("✅ Добавлено в *" + targetCol + "*: " + task.Text)
				}
			}

			// Обработка нажатий на inline-кнопки (done, del, prio, inwork, todo)
			if update.CallbackQuery != nil {
				cb := update.CallbackQuery
				if cb.Message.Chat.ID != cfg.ChatID {
					continue
				}

				data := cb.Data
				filePath := filepath.Join(cfg.VaultPath, cfg.BoardFile)

				// Подтверждаем получение callback
				callbackCfg := tgbotapi.NewCallback(cb.ID, "")
				botAPI.Request(callbackCfg)

				switch {
				case len(data) > 5 && data[:5] == "done:":
					taskText := data[5:]
					_ = kanban.MoveTaskInFile(filePath, taskText, "Готово")
					bot.SendMessage("✅ Задача выполнена: *" + taskText + "*")
					go git.SyncVault(cfg.VaultPath, "bot: done task '"+taskText+"'")

				case len(data) > 4 && data[:4] == "del:":
					taskText := data[4:]
					_ = kanban.DeleteTaskFromFile(filePath, taskText)
					bot.SendMessage("🗑 Задача удалена: *" + taskText + "*")
					go git.SyncVault(cfg.VaultPath, "bot: delete task '"+taskText+"'")

				case len(data) > 5 && data[:5] == "prio:":
					taskText := data[5:]
					_ = kanban.TogglePriorityInFile(filePath, taskText)
					bot.SendMessage("⚡ Изменён приоритет задачи: *" + taskText + "*")
					go git.SyncVault(cfg.VaultPath, "bot: priority task '"+taskText+"'")

				case len(data) > 7 && data[:7] == "inwork:":
					taskText := data[7:]
					_ = kanban.MoveTaskInFile(filePath, taskText, "В работе")
					bot.SendMessage("⏳ Перенесено в *В работе*: *" + taskText + "*")
					go git.SyncVault(cfg.VaultPath, "bot: in work task '"+taskText+"'")

				case len(data) > 5 && data[:5] == "todo:":
					taskText := data[5:]
					_ = kanban.MoveTaskInFile(filePath, taskText, "Надо сделать")
					bot.SendMessage("📝 Перенесено в *Надо сделать*: *" + taskText + "*")
					go git.SyncVault(cfg.VaultPath, "bot: todo task '"+taskText+"'")
				}
			}
		}
	}()

	<-sigChan
	log.Println("🛑 Бот остановлен.")
}
