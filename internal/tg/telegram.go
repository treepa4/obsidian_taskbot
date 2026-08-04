package tg

import (
	"fmt"
	//"log"
	"path/filepath"
	//"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/treepa4/obsidian_taskbot/internal/kanban"
)

type Bot struct {
	api       *tgbotapi.BotAPI
	chatID    int64
	vaultPath string
	boardFile string
}

func NewBot(token string, chatID int64, vaultPath, boardFile string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Bot{
		api:       api,
		chatID:    chatID,
		vaultPath: vaultPath,
		boardFile: boardFile,
	}, nil
}

func (b *Bot) SendTasksList() {
	if b.chatID == 0 {
		return
	}

	filePath := filepath.Join(b.vaultPath, b.boardFile)
	tasks, err := kanban.ParseKanban(filePath)
	if err != nil {
		b.SendMessage(fmt.Sprintf("❌ Ошибка чтения задач: %v", err))
		return
	}

	if len(tasks) == 0 {
		b.SendMessage("🎉 Все задачи выполнены! Список пуст.")
		return
	}

	var urgentTasks, todoTasks, inWorkTasks []kanban.Task

	for _, t := range tasks {
		if t.Priority {
			urgentTasks = append(urgentTasks, t)
		} else if t.InWork {
			inWorkTasks = append(inWorkTasks, t)
		} else {
			todoTasks = append(todoTasks, t)
		}
	}

	b.SendMessage("📋 *ТВОИ ЗАДАЧИ:*")

	// 1. Срочные
	if len(urgentTasks) > 0 {
		b.SendMessage("🚨 *СРОЧНО:*")
		for _, task := range urgentTasks {
			b.sendSingleTaskCard(task)
		}
	}

	// 2. Надо сделать
	if len(todoTasks) > 0 {
		b.SendMessage("📝 *НАДО СДЕЛАТЬ:*")
		for _, task := range todoTasks {
			b.sendSingleTaskCard(task)
		}
	}

	// 3. В работе
	if len(inWorkTasks) > 0 {
		b.SendMessage("⏳ *В РАБОТЕ:*")
		for _, task := range inWorkTasks {
			b.sendSingleTaskCard(task)
		}
	}
}

func (b *Bot) sendSingleTaskCard(task kanban.Task) {
	icon := "📌"
	if task.Priority {
		icon = "🚨"
	} else if task.InWork {
		icon = "⏳"
	}

	text := fmt.Sprintf("%s *%s*", icon, task.Text)
	if task.Date != "" {
		text += fmt.Sprintf("\n📅 %s", task.Date)
	}
	if task.Time != "" {
		text += fmt.Sprintf(" ⏰ %s", task.Time)
	}

	// Готовим кнопки управления
	var keyboard tgbotapi.InlineKeyboardMarkup

	// Ограничиваем длину текста задачи для callback_data
	btnText := task.Text
	if len(btnText) > 40 {
		btnText = btnText[:40]
	}

	inWorkBtn := tgbotapi.NewInlineKeyboardButtonData("⏩ В работу", "inwork:"+btnText)
	if task.InWork {
		inWorkBtn = tgbotapi.NewInlineKeyboardButtonData("↩️ В надо сделать", "todo:"+btnText)
	}

	keyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Выполнить", "done:"+btnText),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить", "del:"+btnText),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚨 Срочно", "prio:"+btnText),
			inWorkBtn,
		),
	)

	msg := tgbotapi.NewMessage(b.chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard
	b.api.Send(msg)
}

func (b *Bot) SendMessage(text string) {
	if b.chatID == 0 {
		return
	}
	msg := tgbotapi.NewMessage(b.chatID, text)
	msg.ParseMode = "Markdown"
	b.api.Send(msg)
}
