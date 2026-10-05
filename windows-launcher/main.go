// ZarvandVPN - Windows Launcher (Go 1.20, Win7+)
package main

import (
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/sys/windows"
)

//go:embed core_amd64.gz
var coreAmd64 []byte

//go:embed core_386.gz
var core386 []byte

//go:embed client_config.toml.simple
var defaultConfig string

//go:embed icon.ico
var iconData []byte

var (
	Version    = "dev"
	corePath   string
	configPath string
	cmd        *exec.Cmd
	mu         sync.Mutex
	running    bool
	mw         *walk.MainWindow
	logView    *walk.TextEdit
	startBtn   *walk.PushButton
	stopBtn    *walk.PushButton
	statusLbl  *walk.Label
)

func arch() string {
	if runtime.GOARCH == "386" {
		return "x86"
	}
	return "x64"
}

func configDir() (string, error) {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		appdata = "."
	}
	dir := filepath.Join(appdata, "ZarvandVPN")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	configPath = filepath.Join(dir, "client_config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return "", err
		}
	}
	return dir, nil
}

func configDirSafe() string {
	d, _ := configDir()
	return d
}

func extractCore() error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	corePath = filepath.Join(dir, "zarvand-core-"+arch()+".exe")
	if _, err := os.Stat(corePath); err == nil {
		return nil
	}
	var data []byte
	if arch() == "x64" {
		data = coreAmd64
	} else {
		data = core386
	}
	gz, err := gzip.NewReader(strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	out, err := os.OpenFile(corePath, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, gz)
	return err
}

func appendLog(s string) {
	if logView == nil || mw == nil {
		return
	}
	mw.Synchronize(func() {
		logView.AppendText(s + "\r\n")
	})
}

func scanOutput(r io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			appendLog(strings.TrimSpace(string(buf[:n])))
		}
		if err != nil {
			return
		}
	}
}

func startCore() {
	mu.Lock()
	if running {
		mu.Unlock()
		return
	}
	running = true
	mu.Unlock()

	go func() {
		appendLog("Starting ZarvandVPN core (" + Version + ")...")
		logFile := filepath.Join(configDirSafe(), "core.log")
		c := exec.Command(corePath, "-config", configPath, "-log", logFile)
		c.SysProcAttr = &windows.SysProcAttr{HideWindow: true}
		stdout, _ := c.StdoutPipe()
		stderr, _ := c.StderrPipe()
		if err := c.Start(); err != nil {
			appendLog("ERROR: " + err.Error())
			setRunning(false)
			return
		}
		mu.Lock()
		cmd = c
		mu.Unlock()
		go scanOutput(stdout)
		go scanOutput(stderr)
		err := c.Wait()
		appendLog("Core exited: " + fmt.Sprint(err))
		setRunning(false)
	}()
}

func stopCore() {
	mu.Lock()
	c := cmd
	mu.Unlock()
	if c != nil && c.Process != nil {
		appendLog("Stopping...")
		_ = c.Process.Kill()
	}
}

func setRunning(v bool) {
	if mw == nil {
		return
	}
	mw.Synchronize(func() {
		if v {
			startBtn.SetEnabled(false)
			stopBtn.SetEnabled(true)
			statusLbl.SetText("Status: Connected")
			statusLbl.SetTextColor(walk.RGB(0, 160, 0))
		} else {
			startBtn.SetEnabled(true)
			stopBtn.SetEnabled(false)
			statusLbl.SetText("Status: Disconnected")
			statusLbl.SetTextColor(walk.RGB(200, 0, 0))
		}
	})
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-version" {
		fmt.Println("ZarvandVPN " + Version)
		return
	}
	if err := extractCore(); err != nil {
		walk.MsgBox(nil, "ZarvandVPN", "Extract error: "+err.Error(), walk.MsgBoxIconError)
		return
	}

	var err error
	err = (MainWindow{
		AssignTo: &mw,
		Title:    "ZarvandVPN - " + Version,
		MinSize:  Size{Width: 480, Height: 420},
		Size:     Size{Width: 520, Height: 470},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "ZarvandVPN  |  DNS Tunnel Client  |  " + arch(), Font: Font{PointSize: 14, Bold: true}},
			HSplitter{
				Children: []Widget{
					PushButton{AssignTo: &startBtn, Text: "Connect", MinSize: Size{Width: 120, Height: 36}, OnClicked: func() { startCore() }},
					PushButton{AssignTo: &stopBtn, Text: "Disconnect", Enabled: false, MinSize: Size{Width: 120, Height: 36}, OnClicked: func() { stopCore() }},
				},
			},
			Label{AssignTo: &statusLbl, Text: "Status: Disconnected", TextColor: walk.RGB(200, 0, 0), Font: Font{PointSize: 11, Bold: true}},
			Label{Text: "Log:"},
			TextEdit{AssignTo: &logView, VScroll: true, ReadOnly: true},
			Label{Text: "Config: %APPDATA%\\ZarvandVPN\\client_config.toml   |   SOCKS5: 127.0.0.1:18000", TextColor: walk.RGB(120, 120, 120)},
		},
	}).Create()
	if err != nil {
		walk.MsgBox(nil, "ZarvandVPN", "UI error: "+err.Error(), walk.MsgBoxIconError)
		return
	}
	setRunning(false)
	if icoPath := filepath.Join(configDirSafe(), "zarvand.ico"); true {
		_ = os.WriteFile(icoPath, iconData, 0644)
		if ic, err := walk.NewIconFromFile(icoPath); err == nil {
			mw.SetIcon(ic)
		}
	}
	mw.Run()
}
