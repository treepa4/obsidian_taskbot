package notifier

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/treepa4/obsidian_taskbot/internal/kanban"
	"github.com/treepa4/obsidian_taskbot/internal/tg"
)

type Notifier struct {
	cron      *cron.Cron
	bot       *tg.Bot
	vaultPath string
	boardFile string
}

func NewNotifier(bot *tg.Bot, vaultPath, boardFile string) *Notifier {
	// Устанавливаем часовой пояс MSK (UTC+3)
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		loc = time.Local
	}

	return &Notifier{
		cron:      cron.New(cron.WithLocation(loc)),
		bot:       bot,
		vaultPath: vaultPath,
		boardFile: boardFile,
	}
}

func (n *Notifier) Start() {
	// ☀️ Утренний дайджест в 09:00 ежедневно
	_, _ = n.cron.AddFunc("0 9 * * *", func() {
		n.SendMorningDigest()
	})

	// 🌙 Вечерний дайджест в 21:00 ежедневно
	_, _ = n.cron.AddFunc("0 21 * * *", func() {
		n.SendEveningDigest()
	})

	n.cron.Start()
}

func (n *Notifier) SendMorningDigest() {
	filePath := filepath.Join(n.vaultPath, n.boardFile)
	tasks, err := kanban.ParseKanban(filePath)
	if err != nil || len(tasks) == 0 {
		n.bot.SendMessage("☀️ *Утренний дайджест:* Задач на сегодня нет! Отличного дня!")
		return
	}

	n.bot.SendMessage("☀️ *УТРЕННИЙ ДАЙДЖЕСТ*\nВот твои задачи на сегодня:")
	n.bot.SendTasksList()
}

func (n *Notifier) SendEveningDigest() {
	filePath := filepath.Join(n.vaultPath, n.boardFile)
	tasks, err := kanban.ParseKanban(filePath)
	if err != nil {
		return
	}

	var remaining int
	for _, t := range tasks {
		if !t.IsDone {
			remaining++
		}
	}

	if remaining == 0 {
		n.bot.SendMessage("🌙 *Вечерний дайджест:* Красава! Все задачи на сегодня закрыты! 🎉")
	} else {
		msg := fmt.Sprintf("🌙 *ВЕЧЕРНИЙ ДАЙДЖЕСТ*\nОсталось незавершённых задач: *%d*\nНе забудь подвести итоги дня!", remaining)
		n.bot.SendMessage(msg)
		n.bot.SendTasksList()
	}
}
