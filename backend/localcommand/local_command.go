package localcommand

import (
	"bytes"
	"os"
	"os/exec"
	"syscall"
	"text/template"
	"unsafe"

	"github.com/kr/pty"
	"github.com/pkg/errors"
)

const (
	DefaultCloseSignal       = syscall.SIGINT
	DefaultWindowTitleFormat = "GoTTY - {{ .Command }} {{ .Hostname }}"
)

var (
	defaultWindowTitleTemplate = template.New(DefaultWindowTitleFormat)
)

type LocalCommand struct {
	cmd           *exec.Cmd
	pty           *os.File
	closeSignal   syscall.Signal
	titleTemplate *template.Template
}

func New(argv []string, options ...Option) (*LocalCommand, error) {
	cmd := exec.Command(argv[0], argv[1:]...)

	pty, err := pty.Start(cmd)
	if err != nil {
		// todo close cmd?
		return nil, errors.Wrapf(err, "failed to start command `%s`")
	}

	lcmd := &LocalCommand{
		cmd:           cmd,
		pty:           pty,
		closeSignal:   DefaultCloseSignal,
		titleTemplate: defaultWindowTitleTemplate,
	}

	for _, option := range options {
		option(lcmd)
	}

	return lcmd, nil
}

func (lcmd *LocalCommand) Read(p []byte) (n int, err error) {
	return lcmd.pty.Read(p)
}

func (lcmd *LocalCommand) Write(p []byte) (n int, err error) {
	return lcmd.pty.Write(p)
}

func (lcmd *LocalCommand) Close() error {
	lcmd.pty.Close()

	// Even if the PTY has been closed,
	// Read(0 in processSend() keeps blocking and the process doen't exit
	if lcmd.cmd != nil && lcmd.cmd.Process != nil {
		lcmd.cmd.Process.Signal(lcmd.closeSignal)
		lcmd.cmd.Wait()
	}
	return nil
}

func (lcmd *LocalCommand) ResizeTerminal(width, height uint16) error {
	window := struct {
		row uint16
		col uint16
		x   uint16
		y   uint16
	}{
		height,
		width,
		0,
		0,
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		lcmd.pty.Fd(),
		syscall.TIOCSWINSZ,
		uintptr(unsafe.Pointer(&window)),
	)
	if errno != 0 {
		return errno
	} else {
		return nil
	}
}

func (lcmd *LocalCommand) WindowTitle() (title string, err error) {
	hostname, _ := os.Hostname()

	titleVars := struct {
		Command  string
		Pid      int
		Hostname string
	}{
		Command:  lcmd.cmd.Path,
		Pid:      lcmd.cmd.Process.Pid,
		Hostname: hostname,
	}

	titleBuffer := new(bytes.Buffer)
	if err := lcmd.titleTemplate.Execute(titleBuffer, titleVars); err != nil {
		return "", err
	}
	return titleBuffer.String(), nil
}
