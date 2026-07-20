package tg

import (
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/treepa4/obsidian_taskbot/internal/kanban"
)

type Bot struct {
	api    *tgbotapi.BotAPI
	ChatID int64
}

func New(token string, initialChatID int64) (*Bot, error) {
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		api:    botAPI,
		ChatID: initialChatID,
	}

	if bot.ChatID == 0 {
		log.Println("⚠️ CHAT_ID не найден! Напиши боту в Telegram сообщение или /start...")
		bot.listenForInitialChatID()
	}

	return bot, nil
}

func (b *Bot) listenForInitialChatID() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			b.ChatID = update.Message.Chat.ID
			log.Printf("✅ Chat ID успешно пойман: %d", b.ChatID)
			b.SendMessage(b.ChatID, "👋 Привет! Я запомнил этот чат и теперь буду слать уведомления сюда.")
			break
		}
	}
}

func (b *Bot) SendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("⚠️ Ошибка отправки сообщения в ТГ: %v", err)
	}
}

func BuildTaskKeyboard(taskText string, isPriority bool) tgbotapi.InlineKeyboardMarkup {
	priorityBtnText := "🚨 Сделать срочной"
	if isPriority {
		priorityBtnText = "⚪ Снять срочность"
	}

	row1 := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✅ Выполнить", fmt.Sprintf("action_done_%s", taskText)),
	)
	row2 := tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(priorityBtnText, fmt.Sprintf("action_prio_%s", taskText)),
		tgbotapi.NewInlineKeyboardButtonData("⏩ В работу", fmt.Sprintf("action_work_%s", taskText)),
	)

	return tgbotapi.NewInlineKeyboardMarkup(row1, row2)
}

func (b *Bot) SendTaskReminder(chatID int64, text string, taskText string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = BuildTaskKeyboard(taskText, false)

	if _, err := b.api.Send(msg); err != nil {
		log.Printf("⚠️ Ошибка отправки напоминания в ТГ: %v", err)
	}
}

func (b *Bot) StartListener(boardFilePath string, gitPushFunc func(msg string)) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.CallbackQuery != nil {
			data := update.CallbackQuery.Data
			parts := strings.SplitN(data, "_", 3)
			if len(parts) == 3 && parts[0] == "action" {
				actionType := parts[1]
				taskText := parts[2]

				var statusMsg string
				var err error

				switch actionType {
				case "done":
					err = kanban.MoveTaskInFile(boardFilePath, taskText, "Готово")
					statusMsg = fmt.Sprintf("🎉 Задача *\"%s\"* выполнена!", taskText)
				case "prio":
					err = kanban.TogglePriorityInFile(boardFilePath, taskText)
					statusMsg = fmt.Sprintf("🚨 Изменен приоритет задачи *\"%s\"*", taskText)
				case "work":
					err = kanban.MoveTaskInFile(boardFilePath, taskText, "В работе")
					statusMsg = fmt.Sprintf("⏩ Задача *\"%s\"* переведена в статус **В работе**", taskText)
				}

				if err != nil {
					log.Printf("⚠️ Ошибка обновления задачи: %v", err)
					continue
				}

				gitPushFunc(fmt.Sprintf("fix(kanban): %s - %s", actionType, taskText))

				callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "Обновлено!")
				if _, err := b.api.Request(callback); err != nil {
					log.Printf("⚠️ Callback error: %v", err)
				}

				editMsg := tgbotapi.NewEditMessageText(
					update.CallbackQuery.Message.Chat.ID,
					update.CallbackQuery.Message.MessageID,
					statusMsg,
				)
				editMsg.ParseMode = "Markdown"
				if _, err := b.api.Send(editMsg); err != nil {
					log.Printf("⚠️ Edit message error: %v", err)
				}
			}
			continue
		}

		if update.Message != nil && update.Message.Text != "" {
			text := update.Message.Text

			if strings.HasPrefix(text, "/") {
				continue
			}

			task, obsidianLine := kanban.ParseNaturalLanguage(text)

			targetCol := "Надо сделать"
			if task.Priority {
				targetCol = "СРОЧНО!!!"
			}

			err := kanban.AddTaskToFile(boardFilePath, obsidianLine, targetCol)
			if err != nil {
				b.SendMessage(b.ChatID, fmt.Sprintf("❌ Ошибка записи задачи: %v", err))
			} else {
				gitPushFunc(fmt.Sprintf("feat: add task '%s'", task.Text))

				response := fmt.Sprintf("✅ **Задача добавлена в %s!**\n📌 *%s*", targetCol, task.Text)
				if task.Date != "" {
					response += fmt.Sprintf("\n📅 Дата: %s", task.Date)
				}
				if task.Time != "" {
					response += fmt.Sprintf("\n⏰ Время: %s", task.Time)
				}

				b.SendMessage(b.ChatID, response)
			}
		}
	}
}
