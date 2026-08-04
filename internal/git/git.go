package git

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

// SyncVault безопасно подтягивает изменения и отправляет локальный коммит
func SyncVault(repoPath string, commitMessage string) error {
	// 1. Индексируем локальные изменения
	if err := execCmd(repoPath, "git", "add", "."); err != nil {
		log.Printf("⚠️ [Git Add Error]: %v", err)
		return err
	}

	// 2. Делаем коммит локального действия бота
	_ = execCmd(repoPath, "git", "commit", "-m", commitMessage)

	// 3. Пробуем подтянуть свежие данные через pull --rebase
	if err := execCmd(repoPath, "git", "pull", "--rebase", "origin", "main"); err != nil {
		log.Printf("⚠️ [Git Rebase Conflict Detected]: %v. Отменяем rebase и сбрасываем состояние...", err)

		// Отменяем сломанный rebase
		_ = execCmd(repoPath, "git", "rebase", "--abort")

		// Принудительно забираем origin/main
		_ = execCmd(repoPath, "git", "fetch", "origin")
		_ = execCmd(repoPath, "git", "reset", "--hard", "origin/main")

		// Индексируем и коммитим файл заново поверх актуального состояния
		_ = execCmd(repoPath, "git", "add", ".")
		_ = execCmd(repoPath, "git", "commit", "-m", commitMessage)

		// Повторно пушим
		if errPush := execCmd(repoPath, "git", "push", "origin", "main"); errPush != nil {
			log.Printf("❌ [Git Push Retry Error]: %v", errPush)
			return errPush
		}

		log.Printf("✅ Git sync восстановлен и завершен: %s", commitMessage)
		return nil
	}

	// 4. Обычный push, если rebase прошёл гладко
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
		// Используем fmt.Errorf вместо fmt.Sprintf, чтобы вернуть тип error
		return fmt.Errorf("%v: %s", err, stderr.String())
	}
	return nil
}
