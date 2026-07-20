package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Task struct {
	Text     string
	Priority bool
	InWork   bool
	Date     string
	Time     string
	Repeat   string
}

var taskRegex = regexp.MustCompile(`^-\s+\[\s+\]\s+(?P<priority>🔺\s+)?(?P<inwork>🏁\s+)?(?P<text>.+?)(?:\s+🔁\s+(?P<repeat>every\s+\w+))?(?:\s+📅\s+(?P<date>\d{4}-\d{2}-\d{2}))?(?:\s+⏰\s+(?P<time>\d{2}:\d{2}))?\s*$`)

func parseKanban(filePath string) ([]Task, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tasks []Task
	var currentColumn string

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "## ") {
			currentColumn = strings.TrimPrefix(line, "## ")
			continue
		}

		if !strings.HasPrefix(line, "- [ ]") || currentColumn == "Готово" {
			continue
		}

		matches := taskRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		task := Task{}
		for i, name := range taskRegex.SubexpNames() {
			if i == 0 || name == "" {
				continue
			}
			value := matches[i]

			switch name {
			case "text":
				task.Text = strings.TrimSpace(value)
			case "priority":
				if value != "" || currentColumn == "СРОЧНО!!!" {
					task.Priority = true
				}
			case "inwork":
				if value != "" || currentColumn == "В работе" {
					task.InWork = true
				}
			case "repeat":
				task.Repeat = value
			case "date":
				task.Date = value
			case "time":
				task.Time = value
			}
		}

		tasks = append(tasks, task)
	}

	return tasks, scanner.Err()
}

func main() {

	tasks, err := parseKanban("test_board.md")
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}

	for _, t := range tasks {
		fmt.Printf("Задача: %s | Срочно: %t | В работе: %t | Дата: %s | Время: %s | Повтор: %s\n",
			t.Text, t.Priority, t.InWork, t.Date, t.Time, t.Repeat)
	}
}
