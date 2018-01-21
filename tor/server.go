package tor

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

type Tor struct {
	cmd *exec.Cmd
}

func (t *Tor) Start() {
	t.cmd = exec.Command("tor", "-f", "/run/tor/torfile")
	t.cmd.Stdout = os.Stdout
	t.cmd.Stderr = os.Stderr

	err := t.cmd.Start()
	if err != nil {
		fmt.Print(err)
		return
	}
}

func (t *Tor) Reload() {
	t.cmd.Process.Signal(syscall.SIGHUP)
	t.cmd.Process.Signal(os.Kill)

	t.Start()
}
