package git

import (
	"os/exec"
)

func PullVault(vaultPath string) error {
	cmd := exec.Command("git", "-C", vaultPath, "pull")
	return cmd.Run()
}

func PushVault(vaultPath, msg string) error {
	exec.Command("git", "-C", vaultPath, "add", ".").Run()
	exec.Command("git", "-C", vaultPath, "commit", "-m", msg).Run()
	cmd := exec.Command("git", "-C", vaultPath, "push")
	return cmd.Run()
}
