package notifier

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/treepa4/obsidian_taskbot/internal/kanban"
	"github.com/treepa4/obsidian_taskbot/internal/tg"
)

type Notifier struct {
	historyFile string
	history     map[string]bool
}

func New(historyFile string) *Notifier {
	n := &Notifier{
		historyFile: historyFile,
		history:     make(map[string]bool),
	}
	n.loadHistory()
	return n
}

func (n *Notifier) loadHistory() {
	file, err := os.ReadFile(n.historyFile)
	if err == nil {
		if err := json.Unmarshal(file, &n.history); err != nil {
			log.Printf("⚠️ Ошибка чтения %s: %v", n.historyFile, err)
		}
	}
}

func (n *Notifier) saveHistory() {
	data, err := json.MarshalIndent(n.history, "", "  ")
	if err != nil {
		log.Printf("⚠️ Ошибка сериализации истории: %v", err)
		return
	}
	_ = os.WriteFile(n.historyFile, data, 0644)
}

func (n *Notifier) Process(bot *tg.Bot, tasks []kanban.Task) {
	now := time.Now()
	todayStr := now.Format("2006-01-02")
	tomorrowStr := now.AddDate(0, 0, 1).Format("2006-01-02")
	currentTimeStr := now.Format("15:04")

	n.sendMorningDigest(bot, tasks, todayStr, currentTimeStr)

	n.sendEveningSummary(bot, tasks, todayStr, tomorrowStr, currentTimeStr)

	n.sendTaskReminders(bot, tasks, todayStr, now)
}

func (n *Notifier) sendMorningDigest(bot *tg.Bot, tasks []kanban.Task, todayStr, currentTimeStr string) {
	digestKey := fmt.Sprintf("digest_%s", todayStr)

	if currentTimeStr == "08:00" && !n.history[digestKey] {
		var todayTasks []string

		for _, t := range tasks {
			if t.Date == todayStr && !t.IsDone {
				prefix := "🔹"
				if t.Priority {
					prefix = "🚨"
				}

				timeLabel := "весь день"
				if t.Time != "" {
					timeLabel = t.Time
				}

				todayTasks = append(todayTasks, fmt.Sprintf("%s %s (%s)", prefix, t.Text, timeLabel))
			}
		}

		if len(todayTasks) > 0 {
			msgText := "🌅 **Утренний дайджест задач на сегодня:**\n\n"
			for _, item := range todayTasks {
				msgText += item + "\n"
			}
			bot.SendMessage(bot.ChatID, msgText)
		}

		n.history[digestKey] = true
		n.saveHistory()
	}
}

func (n *Notifier) sendEveningSummary(bot *tg.Bot, tasks []kanban.Task, todayStr, tomorrowStr, currentTimeStr string) {
	eveningKey := fmt.Sprintf("evening_%s", todayStr)

	if currentTimeStr == "21:00" && !n.history[eveningKey] {
		var doneToday []string
		var plannedTomorrow []string

		for _, t := range tasks {
			// Выполнено сегодня
			if t.IsDone && (t.DoneDate == todayStr || t.Date == todayStr) {
				doneToday = append(doneToday, fmt.Sprintf("✅ %s", t.Text))
			}
			// Запланировано на завтра
			if !t.IsDone && t.Date == tomorrowStr {
				timeLabel := ""
				if t.Time != "" {
					timeLabel = fmt.Sprintf(" (%s)", t.Time)
				}
				plannedTomorrow = append(plannedTomorrow, fmt.Sprintf("▫️ %s%s", t.Text, timeLabel))
			}
		}

		msgText := "🌙 **Вечерний итог дня:**\n\n"

		if len(doneToday) > 0 {
			msgText += "**Сделано за сегодня:**\n"
			for _, item := range doneToday {
				msgText += item + "\n"
			}
		} else {
			msgText += "Сегодня не было отмеченных выполненных задач.\n"
		}

		msgText += "\n**План на завтра:**\n"
		if len(plannedTomorrow) > 0 {
			for _, item := range plannedTomorrow {
				msgText += item + "\n"
			}
		} else {
			msgText += "На завтра задач пока нет. Отдыхай! ☕️\n"
		}

		bot.SendMessage(bot.ChatID, msgText)
		n.history[eveningKey] = true
		n.saveHistory()
	}
}

func (n *Notifier) sendTaskReminders(bot *tg.Bot, tasks []kanban.Task, todayStr string, now time.Time) {
	for _, t := range tasks {
		if t.Date != todayStr || t.Time == "" || t.IsDone {
			continue
		}

		taskTime, err := time.Parse("15:04", t.Time)
		if err != nil {
			log.Printf("⚠️ Ошибка парсинга времени задачи '%s': %v", t.Text, err)
			continue
		}

		taskDateTime := time.Date(now.Year(), now.Month(), now.Day(), taskTime.Hour(), taskTime.Minute(), 0, 0, now.Location())
		diffMinutes := int(taskDateTime.Sub(now).Minutes())

		var intervals []int
		if t.Priority {
			intervals = []int{90, 60, 30, 15}
		} else {
			intervals = []int{60}
		}

		for _, interval := range intervals {
			if diffMinutes >= interval-1 && diffMinutes <= interval {
				pushKey := fmt.Sprintf("push_%s_%s_%dmin", t.Text, todayStr, interval)

				if !n.history[pushKey] {
					priorityMark := ""
					if t.Priority {
						priorityMark = "🚨 [ВЫСОКИЙ ПРИОРИТЕТ]\n"
					}

					text := fmt.Sprintf("⏰ %s**Напоминание!**\n\nДо задачи *\"%s\"* осталось %d минут! (Дедлайн: %s)",
						priorityMark, t.Text, interval, t.Time)

					bot.SendTaskReminder(bot.ChatID, text, t.Text)

					n.history[pushKey] = true
					n.saveHistory()
				}
			}
		}
	}
}
