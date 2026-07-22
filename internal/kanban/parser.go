package kanban

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
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
	dateRegex        = regexp.MustCompile(`📅\s*(\d{4}-\d{2}-\d{2})`)
	timeRegex        = regexp.MustCompile(`⏰\s*(\d{2}:\d{2})`)
	repeatRegex      = regexp.MustCompile(`🔁\s*(every\s+\w+)`)
	doneRegex        = regexp.MustCompile(`✅\s*\d{4}-\d{2}-\d{2}`)
	timePattern      = regexp.MustCompile(`(?i)\b(?:в\s+)?([0-1]?[0-9]|2[0-3])[:.]([0-5][0-9])\b`)
	shortTimePattern = regexp.MustCompile(`(?i)\bв\s+([0-1]?[0-9]|2[0-3])(?:\s*ч(?:аса|асов)?)?\b`)
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

		if (!strings.HasPrefix(line, "- [ ]") && !strings.HasPrefix(line, "- [x]")) || currentColumn == "Готово" {
			continue
		}

		task := ParseTaskLine(line, currentColumn)
		tasks = append(tasks, task)
	}

	return tasks, scanner.Err()
}

// matchTask проверяет, относится ли строка файла к переданному поисковому запросу (устойчиво к обрезке UTF-8)
func matchTask(line string, taskText string) bool {
	trimmedLine := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmedLine, "- [ ]") && !strings.HasPrefix(trimmedLine, "- [x]") {
		return false
	}

	cleanSearch := strings.TrimRight(taskText, " ")
	return strings.Contains(line, cleanSearch)
}

func DeleteTaskFromFile(filePath string, taskText string) error {
	input, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	var newLines []string
	found := false

	for _, line := range lines {
		if matchTask(line, taskText) {
			found = true
			continue
		}
		newLines = append(newLines, line)
	}

	if !found {
		return fmt.Errorf("задача '%s' не найдена", taskText)
	}

	output := strings.Join(newLines, "\n")
	return os.WriteFile(filePath, []byte(output), 0644)
}

func AddTaskToFile(filePath string, taskLine string, targetColumn string) error {
	input, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	var newLines []string
	inserted := false

	colHeader := "## " + targetColumn

	for _, line := range lines {
		newLines = append(newLines, line)
		if strings.TrimSpace(line) == colHeader && !inserted {
			newLines = append(newLines, taskLine)
			inserted = true
		}
	}

	if !inserted {
		var finalLines []string
		added := false
		for _, line := range newLines {
			if strings.HasPrefix(line, "%% kanban:settings") && !added {
				finalLines = append(finalLines, colHeader, taskLine, "")
				added = true
			}
			finalLines = append(finalLines, line)
		}
		if !added {
			finalLines = append(finalLines, colHeader, taskLine)
		}
		newLines = finalLines
	}

	output := strings.Join(newLines, "\n")
	return os.WriteFile(filePath, []byte(output), 0644)
}

func MoveTaskInFile(filePath string, taskText string, targetColumn string) error {
	input, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	var taskLine string
	var newLines []string

	for _, line := range lines {
		if matchTask(line, taskText) && taskLine == "" {
			taskLine = line
			continue
		}
		newLines = append(newLines, line)
	}

	if taskLine == "" {
		return fmt.Errorf("задача '%s' не найдена", taskText)
	}

	if targetColumn == "Готово" {
		if !strings.HasPrefix(strings.TrimSpace(taskLine), "- [x]") {
			taskLine = strings.Replace(taskLine, "- [ ]", "- [x]", 1)
		}
		if !strings.Contains(taskLine, "✅") {
			taskLine += fmt.Sprintf(" ✅ %s", time.Now().Format("2006-01-02"))
		}
	} else {
		if strings.HasPrefix(strings.TrimSpace(taskLine), "- [x]") {
			taskLine = strings.Replace(taskLine, "- [x]", "- [ ]", 1)
		}
	}

	colHeader := "## " + targetColumn
	var finalLines []string
	inserted := false

	for _, line := range newLines {
		finalLines = append(finalLines, line)
		if strings.TrimSpace(line) == colHeader && !inserted {
			finalLines = append(finalLines, taskLine)
			inserted = true
		}
	}

	output := strings.Join(finalLines, "\n")
	return os.WriteFile(filePath, []byte(output), 0644)
}

func TogglePriorityInFile(filePath string, taskText string) error {
	input, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(input), "\n")
	var taskLine string
	var currentCol string
	var newLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			currentCol = strings.TrimPrefix(trimmed, "## ")
		}

		if matchTask(line, taskText) && taskLine == "" {
			taskLine = line
			continue
		}
		newLines = append(newLines, line)
	}

	if taskLine == "" {
		return fmt.Errorf("задача '%s' не найдена", taskText)
	}

	targetCol := "СРОЧНО!!!"
	if currentCol == "СРОЧНО!!!" {
		targetCol = "Надо сделать"
		taskLine = strings.ReplaceAll(taskLine, "🔺", "")
	} else {
		if !strings.Contains(taskLine, "🔺") {
			taskLine += " 🔺"
		}
	}

	colHeader := "## " + targetCol
	var finalLines []string
	inserted := false

	for _, line := range newLines {
		finalLines = append(finalLines, line)
		if strings.TrimSpace(line) == colHeader && !inserted {
			finalLines = append(finalLines, taskLine)
			inserted = true
		}
	}

	output := strings.Join(finalLines, "\n")
	return os.WriteFile(filePath, []byte(output), 0644)
}

func ParseNaturalLanguage(input string) (Task, string) {
	now := time.Now()
	taskDate := ""
	taskTime := ""
	isPriority := false

	lowerInput := strings.ToLower(input)

	if strings.Contains(lowerInput, "срочно") || strings.Contains(lowerInput, "важно") ||
		strings.Contains(input, "🔺") || strings.Contains(input, "🚨") {
		isPriority = true
	}

	if match := timePattern.FindStringSubmatch(input); len(match) == 3 {
		taskTime = fmt.Sprintf("%02s:%s", match[1], match[2])
	} else if match := shortTimePattern.FindStringSubmatch(input); len(match) == 2 {
		taskTime = fmt.Sprintf("%02s:00", match[1])
	}

	cleanInput := input

	if strings.Contains(lowerInput, "сегодня") {
		taskDate = now.Format("2006-01-02")
	} else if strings.Contains(lowerInput, "послезавтра") {
		taskDate = now.AddDate(0, 0, 2).Format("2006-01-02")
	} else if strings.Contains(lowerInput, "завтра") {
		taskDate = now.AddDate(0, 0, 1).Format("2006-01-02")
	} else {
		targetWeekday := parseWeekday(lowerInput)
		if targetWeekday != -1 {
			daysAhead := (int(targetWeekday) - int(now.Weekday()) + 7) % 7
			if daysAhead == 0 {
				daysAhead = 7
			}
			taskDate = now.AddDate(0, 0, daysAhead).Format("2006-01-02")
		}
	}

	wordsToRemove := []string{
		"сегодня", "завтра", "послезавтра", "срочно", "важно",
		"в понедельник", "во вторник", "в среду", "в четверг", "в пятницу", "в субботу", "в воскресенье",
		"в пн", "во вт", "в ср", "в чт", "в пт", "в сб", "в вс",
	}

	for _, word := range wordsToRemove {
		re := regexp.MustCompile("(?i)\\b" + regexp.QuoteMeta(word) + "\\b")
		cleanInput = re.ReplaceAllString(cleanInput, "")
	}

	cleanInput = timePattern.ReplaceAllString(cleanInput, "")
	cleanInput = shortTimePattern.ReplaceAllString(cleanInput, "")
	cleanInput = strings.ReplaceAll(cleanInput, "🔺", "")
	cleanInput = strings.ReplaceAll(cleanInput, "🚨", "")
	cleanInput = strings.Join(strings.Fields(cleanInput), " ")

	obsidianLine := fmt.Sprintf("- [ ] %s", cleanInput)
	if isPriority {
		obsidianLine += " 🔺"
	}
	if taskDate != "" {
		obsidianLine += fmt.Sprintf(" 📅 %s", taskDate)
	}
	if taskTime != "" {
		obsidianLine += fmt.Sprintf(" ⏰ %s", taskTime)
	}

	task := Task{
		Text:     cleanInput,
		Priority: isPriority,
		Date:     taskDate,
		Time:     taskTime,
	}

	return task, obsidianLine
}

func parseWeekday(input string) time.Weekday {
	switch {
	case strings.Contains(input, "понедельник") || strings.Contains(input, " пн"):
		return time.Monday
	case strings.Contains(input, "вторник") || strings.Contains(input, " вт"):
		return time.Tuesday
	case strings.Contains(input, "среду") || strings.Contains(input, " ср"):
		return time.Wednesday
	case strings.Contains(input, "четверг") || strings.Contains(input, " чт"):
		return time.Thursday
	case strings.Contains(input, "пятницу") || strings.Contains(input, " пт"):
		return time.Friday
	case strings.Contains(input, "субботу") || strings.Contains(input, " сб"):
		return time.Saturday
	case strings.Contains(input, "воскресенье") || strings.Contains(input, " вс"):
		return time.Sunday
	default:
		return -1
	}
}
