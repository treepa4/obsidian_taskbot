package git

import (
	"os/exec"
)

type Client struct {
	VaultPath string
}

func New(vaultPath string) *Client {
	return &Client{VaultPath: vaultPath}
}

func (g *Client) Pull() error {
	cmd := exec.Command("git", "-C", g.VaultPath, "pull")
	return cmd.Run()
}

func (g *Client) CommitAndPush(msg string) error {
	exec.Command("git", "-C", g.VaultPath, "add", ".").Run()
	exec.Command("git", "-C", g.VaultPath, "commit", "-m", msg).Run()
	cmd := exec.Command("git", "-C", g.VaultPath, "push")
	return cmd.Run()
}
