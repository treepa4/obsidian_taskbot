package git

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

// SyncVault делает pull --rebase, фиксирует локальные изменения и делает push
func SyncVault(repoPath string, commitMessage string) error {
	// 1. Индексируем изменения
	if err := execCmd(repoPath, "git", "add", "."); err != nil {
		log.Printf("⚠️ [Git Add Error]: %v", err)
		return err
	}

	// 2. Делаем коммит (если есть изменения)
	_ = execCmd(repoPath, "git", "commit", "-m", commitMessage)

	// 3. Подтягиваем свежие данные с GitHub (pull --rebase)
	if err := execCmd(repoPath, "git", "pull", "--rebase", "origin", "main"); err != nil {
		log.Printf("⚠️ [Git Pull Rebase Error]: %v, отмена rebase...", err)
		_ = execCmd(repoPath, "git", "rebase", "--abort")
		return err
	}

	// 4. Пушим в репозиторий
	if err := execCmd(repoPath, "git", "push", "origin", "main"); err != nil {
		log.Printf("❌ [Git Push Error]: %v", err)
		return err
	}

	log.Printf("✅ Git sync успешен: %s", commitMessage)
	return nil
}

func execCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, stderr.String())
	}
	return nil
}
