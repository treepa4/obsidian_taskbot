package notifier

import (
	"log"
	"time"

	"github.com/treepa4/obsidian_taskbot/internal/kanban"
	"github.com/treepa4/obsidian_taskbot/internal/tg"
)

func StartScheduler(bot *tg.Bot, boardFilePath string, vaultPath string) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		// Пример проверки утреннего дайджеста (например, в 09:00)
		if now.Hour() == 9 && now.Minute() == 0 {
			tasks, err := kanban.ParseKanban(boardFilePath)
			if err != nil {
				log.Printf("⚠️ Ошибка чтения доски для дайджеста: %v", err)
				continue
			}

			if bot.ChatID != 0 {
				bot.SendMessage(bot.ChatID, "🌅 **Доброе утро! Вот ваши активные задачи на сегодня:**")
				for _, t := range tasks {
					if t.IsDone {
						continue
					}
					msgText := "📌 " + t.Text
					if t.Date != "" {
						msgText += "\n📅 " + t.Date
					}
					bot.SendTaskReminder(bot.ChatID, msgText, t.Text)
				}
			}
		}
	}
}
