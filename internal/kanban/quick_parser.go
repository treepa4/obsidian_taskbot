package kanban

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

var (
	timePattern     = regexp.MustCompile(`(?i)\b(?:в\s*)?(\d{1,2})[:.-](\d{2})\b|\b(?:в\s*)?(\d{1,2})\s*(ч|часов|часа)\b`)
	urgentPattern   = regexp.MustCompile(`(?i)\b(срочно|важно|аларм|приоритет)\b|!|🔺`)
	todayPattern    = regexp.MustCompile(`(?i)\bсегодня\b`)
	tomorrowPattern = regexp.MustCompile(`(?i)\bзавтра\b`)
	afterTomPattern = regexp.MustCompile(`(?i)\bпослезавтра\b`)
)

func ParseNaturalLanguage(input string) (Task, string) {
	now := time.Now()
	taskDate := now
	hasDate := false
	hasTime := false
	timeStr := ""
	isUrgent := false

	cleanText := input

	if urgentPattern.MatchString(cleanText) {
		isUrgent = true
		cleanText = urgentPattern.ReplaceAllString(cleanText, "")
	}

	if tomorrowPattern.MatchString(cleanText) {
		taskDate = now.AddDate(0, 0, 1)
		hasDate = true
		cleanText = tomorrowPattern.ReplaceAllString(cleanText, "")
	} else if afterTomPattern.MatchString(cleanText) {
		taskDate = now.AddDate(0, 0, 2)
		hasDate = true
		cleanText = afterTomPattern.ReplaceAllString(cleanText, "")
	} else if todayPattern.MatchString(cleanText) {
		taskDate = now
		hasDate = true
		cleanText = todayPattern.ReplaceAllString(cleanText, "")
	}

	if matches := timePattern.FindStringSubmatch(cleanText); len(matches) > 0 {
		hasTime = true
		if matches[1] != "" && matches[2] != "" {
			timeStr = fmt.Sprintf("%02s:%s", matches[1], matches[2])
		} else if matches[3] != "" {
			timeStr = fmt.Sprintf("%02s:00", matches[3])
		}
		cleanText = timePattern.ReplaceAllString(cleanText, "")
	}

	if !hasDate && (hasTime || isUrgent) {
		hasDate = true
	}

	cleanText = regexp.MustCompile(`\s+`).ReplaceAllString(cleanText, " ")
	cleanText = strings.TrimSpace(cleanText)

	var obsidianLine strings.Builder
	obsidianLine.WriteString("- [ ] ")
	obsidianLine.WriteString(cleanText)

	dateStr := ""
	if hasDate {
		dateStr = taskDate.Format("2006-01-02")
		obsidianLine.WriteString(fmt.Sprintf(" 📅 %s", dateStr))
	}

	if hasTime {
		obsidianLine.WriteString(fmt.Sprintf(" ⏰ %s", timeStr))
	}

	if isUrgent {
		obsidianLine.WriteString(" 🔺")
	}

	task := Task{
		Text:     cleanText,
		Priority: isUrgent,
		Date:     dateStr,
		Time:     timeStr,
	}

	return task, obsidianLine.String()
}

func AddTaskToFile(filePath string, obsidianLine string, targetColumn string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	inserted := false
	targetHeader := "## " + targetColumn

	for _, line := range lines {
		newLines = append(newLines, line)
		if strings.TrimSpace(line) == targetHeader && !inserted {
			newLines = append(newLines, obsidianLine)
			inserted = true
		}
	}

	if !inserted {
		newLines = append(newLines, fmt.Sprintf("\n%s\n%s", targetHeader, obsidianLine))
	}

	return os.WriteFile(filePath, []byte(strings.Join(newLines, "\n")), 0644)
}

func MoveTaskInFile(filePath string, taskText string, targetColumn string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var cleanedLines []string
	var targetLine string

	for _, line := range lines {
		if strings.Contains(line, taskText) && (strings.HasPrefix(line, "- [ ]") || strings.HasPrefix(line, "- [x]")) {
			targetLine = line
			if targetColumn == "Готово" {
				// Выполняем задачу: отмечаем [x] и добавляем ✅ YYYY-MM-DD
				targetLine = strings.Replace(line, "- [ ]", "- [x]", 1)
				todayStr := time.Now().Format("2006-01-02")
				if !strings.Contains(targetLine, "✅") {
					targetLine += fmt.Sprintf(" ✅ %s", todayStr)
				}
			}
			continue
		}
		cleanedLines = append(cleanedLines, line)
	}

	if targetLine == "" {
		return fmt.Errorf("задача '%s' не найдена в файле", taskText)
	}

	var finalLines []string
	inserted := false
	targetHeader := "## " + targetColumn

	for _, line := range cleanedLines {
		finalLines = append(finalLines, line)
		if strings.TrimSpace(line) == targetHeader && !inserted {
			finalLines = append(finalLines, targetLine)
			inserted = true
		}
	}

	if !inserted {
		finalLines = append(finalLines, fmt.Sprintf("\n%s\n%s", targetHeader, targetLine))
	}

	return os.WriteFile(filePath, []byte(strings.Join(finalLines, "\n")), 0644)
}

func TogglePriorityInFile(filePath string, taskText string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var targetLine string
	var newColumn string

	for _, line := range lines {
		if strings.Contains(line, taskText) && strings.HasPrefix(line, "- [ ]") {
			if strings.Contains(line, "🔺") {
				targetLine = strings.ReplaceAll(line, " 🔺", "")
				targetLine = strings.ReplaceAll(targetLine, "🔺", "")
				newColumn = "Надо сделать"
			} else {
				targetLine = line + " 🔺"
				newColumn = "СРОЧНО!!!"
			}
			break
		}
	}

	if targetLine == "" {
		return fmt.Errorf("активная задача '%s' не найдена", taskText)
	}

	return MoveTaskInFile(filePath, taskText, newColumn)
}
