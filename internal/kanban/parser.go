package kanban

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

type Task struct {
	Text     string
	Priority bool
	InWork   bool
	IsDone   bool
	DoneDate string
	Date     string
	Time     string
	Repeat   string
}

var (
	dateRegex   = regexp.MustCompile(`📅\s*(\d{4}-\d{2}-\d{2})`)
	timeRegex   = regexp.MustCompile(`⏰\s*(\d{2}:\d{2})`)
	repeatRegex = regexp.MustCompile(`🔁\s*(every\s+\w+)`)
	doneRegex   = regexp.MustCompile(`✅\s*\d{4}-\d{2}-\d{2}`)
)

func ParseTaskLine(line string, column string) Task {
	task := Task{
		Priority: column == "СРОЧНО!!!",
		InWork:   column == "В работе",
		IsDone:   column == "Готово" || strings.HasPrefix(line, "- [x]"),
	}

	if strings.Contains(line, "🔺") {
		task.Priority = true
	}
	if strings.Contains(line, "🏁") {
		task.InWork = true
	}

	if match := dateRegex.FindStringSubmatch(line); len(match) > 1 {
		task.Date = match[1]
	}
	if match := timeRegex.FindStringSubmatch(line); len(match) > 1 {
		task.Time = match[1]
	}
	if match := repeatRegex.FindStringSubmatch(line); len(match) > 1 {
		task.Repeat = match[1]
	}
	if match := doneRegex.FindStringSubmatch(line); len(match) > 1 {
		task.DoneDate = match[1]
	}

	cleanText := strings.TrimPrefix(line, "- [ ]")
	cleanText = strings.TrimPrefix(cleanText, "- [x]")
	cleanText = dateRegex.ReplaceAllString(cleanText, "")
	cleanText = timeRegex.ReplaceAllString(cleanText, "")
	cleanText = repeatRegex.ReplaceAllString(cleanText, "")
	cleanText = doneRegex.ReplaceAllString(cleanText, "")
	cleanText = strings.ReplaceAll(cleanText, "🔺", "")
	cleanText = strings.ReplaceAll(cleanText, "🏁", "")

	task.Text = strings.TrimSpace(cleanText)
	return task
}

func ParseKanban(filePath string) ([]Task, error) {
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

		task := ParseTaskLine(line, currentColumn)
		tasks = append(tasks, task)
	}

	return tasks, scanner.Err()
}
