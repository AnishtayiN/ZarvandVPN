// ZarvandVPN - Windows Launcher (Go 1.20, Win7+)
package main

import (
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

//go:embed core_amd64.gz
var coreAmd64 []byte

//go:embed core_386.gz
var core386 []byte

//go:embed icon.ico
var iconData []byte

var Version = "dev"

// ---------------- settings model ----------------

type Settings struct {
	Domains    string `json:"domains"`
	Key        string `json:"key"`
	EncMethod  int    `json:"encMethod"`
	Protocol   string `json:"protocol"`
	ListenIP   string `json:"listenIP"`
	ListenPort int    `json:"listenPort"`
	SocksAuth  bool   `json:"socksAuth"`
	SocksUser  string `json:"socksUser"`
	SocksPass  string `json:"socksPass"`

	Resolvers string `json:"resolvers"`

	LocalDNS        bool   `json:"localDNS"`
	LocalDNSIP      string `json:"localDNSIP"`
	LocalDNSPort    int    `json:"localDNSPort"`
	LocalDNSCache   bool   `json:"localDNSCache"`
	LocalDNSCacheTTL int   `json:"localDNSCacheTTL"`

	PacketDup      int `json:"packetDup"`
	SetupPacketDup int `json:"setupPacketDup"`
	UpCompression  int `json:"upCompression"`
	DownCompression int `json:"downCompression"`
	CompressionMinSize int `json:"compressionMinSize"`
	RxTxWorkers    int `json:"rxTxWorkers"`
	TunnelWorkers  int `json:"tunnelWorkers"`
	ResolverStrategy int `json:"resolverStrategy"`

	MinUpMTU   int `json:"minUpMTU"`
	MaxUpMTU   int `json:"maxUpMTU"`
	MinDownMTU int `json:"minDownMTU"`
	MaxDownMTU int `json:"maxDownMTU"`

	BaseEncodeData bool   `json:"baseEncodeData"`
	LogLevel       string `json:"logLevel"`
}

func defaultSettings() Settings {
	return Settings{
		Domains:            "",
		Key:                "",
		EncMethod:          1,
		Protocol:           "SOCKS5",
		ListenIP:           "127.0.0.1",
		ListenPort:         18000,
		SocksAuth:          false,
		Resolvers:          "8.8.8.8\n1.1.1.1",
		LocalDNS:           false,
		LocalDNSIP:         "127.0.0.1",
		LocalDNSPort:       53,
		LocalDNSCache:      true,
		LocalDNSCacheTTL:   60,
		PacketDup:          3,
		SetupPacketDup:     4,
		UpCompression:      0,
		DownCompression:    0,
		CompressionMinSize: 100,
		RxTxWorkers:        4,
		TunnelWorkers:      4,
		ResolverStrategy:   0,
		MinUpMTU:           40,
		MaxUpMTU:           150,
		MinDownMTU:         40,
		MaxDownMTU:         150,
		BaseEncodeData:     false,
		LogLevel:           "INFO",
	}
}

var (
	settings      = defaultSettings()
	proxyWasOn    bool
	proxyApplied  bool
	corePath     string
	configPath   string
	settingsPath string
	resolversPath string
	cmd          *exec.Cmd
	mu           sync.Mutex
	running      bool
	mw           *walk.MainWindow
	logView      *walk.TextEdit
	startBtn     *walk.PushButton
	stopBtn      *walk.PushButton
	statusLbl    *walk.Label
)

// UI field handles
var (
	edDomains, edKey, edListenIP, edListenPort           *walk.LineEdit
	edSocksUser, edSocksPass, edResolvers                *walk.LineEdit
	edLocalDNSIP, edLocalDNSPort, edCacheTTL             *walk.LineEdit
	edDup, edSetupDup, edCompMin                         *walk.LineEdit
	edRxTx, edTunWorkers                                 *walk.LineEdit
	edMinUp, edMaxUp, edMinDown, edMaxDown               *walk.LineEdit
	cbEnc, cbProtocol, cbUpComp, cbDownComp              *walk.ComboBox
	cbLog, cbStrategy                                    *walk.ComboBox
	chkSocksAuth, chkLocalDNS, chkCache, chkBase64       *walk.CheckBox
	chkProxy                                             *walk.CheckBox
	tabWidget                                            *walk.TabWidget
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
	settingsPath = filepath.Join(dir, "settings.json")
	resolversPath = filepath.Join(dir, "client_resolvers.txt")
	return dir, nil
}

func configDirSafe() string {
	d, _ := configDir()
	return d
}

func loadSettings() {
	b, err := os.ReadFile(settingsPath)
	if err == nil {
		_ = json.Unmarshal(b, &settings)
	}
}

func persistSettings() error {
	b, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, b, 0644)
}

// pullSettings reads every UI field into the Settings struct (called on main thread).
func pullSettings() {
	if edDomains == nil {
		return
	}
	settings.Domains = edDomains.Text()
	settings.Key = edKey.Text()
	settings.ListenIP = strings.TrimSpace(edListenIP.Text())
	settings.ListenPort = atoiOr(edListenPort.Text(), settings.ListenPort)
	settings.SocksUser = edSocksUser.Text()
	settings.SocksPass = edSocksPass.Text()
	settings.Resolvers = edResolvers.Text()
	settings.LocalDNSIP = strings.TrimSpace(edLocalDNSIP.Text())
	settings.LocalDNSPort = atoiOr(edLocalDNSPort.Text(), settings.LocalDNSPort)
	settings.LocalDNSCacheTTL = atoiOr(edCacheTTL.Text(), settings.LocalDNSCacheTTL)
	settings.PacketDup = clamp(atoiOr(edDup.Text(), settings.PacketDup), 1, 12)
	settings.SetupPacketDup = clamp(atoiOr(edSetupDup.Text(), settings.SetupPacketDup), 1, 12)
	settings.CompressionMinSize = atoiOr(edCompMin.Text(), settings.CompressionMinSize)
	settings.RxTxWorkers = clamp(atoiOr(edRxTx.Text(), settings.RxTxWorkers), 1, 64)
	settings.TunnelWorkers = clamp(atoiOr(edTunWorkers.Text(), settings.TunnelWorkers), 1, 64)
	settings.MinUpMTU = atoiOr(edMinUp.Text(), settings.MinUpMTU)
	settings.MaxUpMTU = atoiOr(edMaxUp.Text(), settings.MaxUpMTU)
	settings.MinDownMTU = atoiOr(edMinDown.Text(), settings.MinDownMTU)
	settings.MaxDownMTU = atoiOr(edMaxDown.Text(), settings.MaxDownMTU)

	if cbEnc != nil && cbEnc.CurrentIndex() >= 0 {
		settings.EncMethod = cbEnc.CurrentIndex()
	}
	if cbProtocol != nil && cbProtocol.CurrentIndex() >= 0 {
		settings.Protocol = []string{"SOCKS5", "TCP"}[cbProtocol.CurrentIndex()]
	}
	if cbUpComp != nil && cbUpComp.CurrentIndex() >= 0 {
		settings.UpCompression = cbUpComp.CurrentIndex()
	}
	if cbDownComp != nil && cbDownComp.CurrentIndex() >= 0 {
		settings.DownCompression = cbDownComp.CurrentIndex()
	}
	if cbLog != nil && cbLog.CurrentIndex() >= 0 {
		settings.LogLevel = []string{"DEBUG", "INFO", "WARN", "ERROR"}[cbLog.CurrentIndex()]
	}
	if cbStrategy != nil && cbStrategy.CurrentIndex() >= 0 {
		settings.ResolverStrategy = cbStrategy.CurrentIndex()
	}
	if chkSocksAuth != nil {
		settings.SocksAuth = chkSocksAuth.Checked()
	}
	if chkLocalDNS != nil {
		settings.LocalDNS = chkLocalDNS.Checked()
	}
	if chkCache != nil {
		settings.LocalDNSCache = chkCache.Checked()
	}
	if chkBase64 != nil {
		settings.BaseEncodeData = chkBase64.Checked()
	}
}

// pushSettings writes the struct back into UI fields (called on main thread).
func pushSettings() {
	if edDomains == nil {
		return
	}
	edDomains.SetText(settings.Domains)
	edKey.SetText(settings.Key)
	edListenIP.SetText(settings.ListenIP)
	edListenPort.SetText(strconv.Itoa(settings.ListenPort))
	edSocksUser.SetText(settings.SocksUser)
	edSocksPass.SetText(settings.SocksPass)
	edResolvers.SetText(settings.Resolvers)
	edLocalDNSIP.SetText(settings.LocalDNSIP)
	edLocalDNSPort.SetText(strconv.Itoa(settings.LocalDNSPort))
	edCacheTTL.SetText(strconv.Itoa(settings.LocalDNSCacheTTL))
	edDup.SetText(strconv.Itoa(settings.PacketDup))
	edSetupDup.SetText(strconv.Itoa(settings.SetupPacketDup))
	edCompMin.SetText(strconv.Itoa(settings.CompressionMinSize))
	edRxTx.SetText(strconv.Itoa(settings.RxTxWorkers))
	edTunWorkers.SetText(strconv.Itoa(settings.TunnelWorkers))
	edMinUp.SetText(strconv.Itoa(settings.MinUpMTU))
	edMaxUp.SetText(strconv.Itoa(settings.MaxUpMTU))
	edMinDown.SetText(strconv.Itoa(settings.MinDownMTU))
	edMaxDown.SetText(strconv.Itoa(settings.MaxDownMTU))

	cbEnc.SetCurrentIndex(settings.EncMethod)
	cbProtocol.SetCurrentIndex(idxOf([]string{"SOCKS5", "TCP"}, settings.Protocol))
	cbUpComp.SetCurrentIndex(settings.UpCompression)
	cbDownComp.SetCurrentIndex(settings.DownCompression)
	cbLog.SetCurrentIndex(idxOf([]string{"DEBUG", "INFO", "WARN", "ERROR"}, settings.LogLevel))
	cbStrategy.SetCurrentIndex(settings.ResolverStrategy)

	chkSocksAuth.SetChecked(settings.SocksAuth)
	chkLocalDNS.SetChecked(settings.LocalDNS)
	chkCache.SetChecked(settings.LocalDNSCache)
	chkBase64.SetChecked(settings.BaseEncodeData)
}

func atoiOr(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func idxOf(list []string, v string) int {
	for i, s := range list {
		if strings.EqualFold(s, v) {
			return i
		}
	}
	return 0
}

func splitList(s string) []string {
	f := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ';' || r == ' '
	})
	out := make([]string, 0, len(f))
	for _, x := range f {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}

func quoteList(items []string) string {
	parts := make([]string, 0, len(items))
	for _, it := range items {
		parts = append(parts, "\""+strings.ReplaceAll(it, "\"", "")+"\"")
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// writeConfig renders client_config.toml + client_resolvers.txt from settings.
func writeConfig() error {
	if _, err := configDir(); err != nil {
		return err
	}
	domains := splitList(settings.Domains)
	resolvers := splitList(settings.Resolvers)
	if len(resolvers) == 0 {
		resolvers = []string{"8.8.8.8", "1.1.1.1"}
	}
	if err := os.WriteFile(resolversPath, []byte(strings.Join(resolvers, "\n")+"\n"), 0644); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# ZarvandVPN - auto-generated. Edit via the app Settings tab.\n")
	w := func(k, v string) { fmt.Fprintf(&b, "%-42s = %s\n", k, v) }

	w("DOMAINS", quoteList(domains))
	w("DATA_ENCRYPTION_METHOD", strconv.Itoa(settings.EncMethod))
	w("ENCRYPTION_KEY", "\""+strings.ReplaceAll(settings.Key, "\"", "")+"\"")
	w("PROTOCOL_TYPE", "\""+settings.Protocol+"\"")
	w("LISTEN_IP", "\""+settings.ListenIP+"\"")
	w("LISTEN_PORT", strconv.Itoa(settings.ListenPort))
	w("SOCKS5_AUTH", strconv.FormatBool(settings.SocksAuth))
	w("SOCKS5_USER", "\""+settings.SocksUser+"\"")
	w("SOCKS5_PASS", "\""+settings.SocksPass+"\"")
	b.WriteString("\n")
	w("LOCAL_DNS_ENABLED", strconv.FormatBool(settings.LocalDNS))
	w("LOCAL_DNS_IP", "\""+settings.LocalDNSIP+"\"")
	w("LOCAL_DNS_PORT", strconv.Itoa(settings.LocalDNSPort))
	w("LOCAL_DNS_CACHE_PERSIST_TO_FILE", strconv.FormatBool(settings.LocalDNSCache))
	w("LOCAL_DNS_CACHE_TTL_SECONDS", strconv.Itoa(settings.LocalDNSCacheTTL))
	b.WriteString("\n")
	w("RESOLVER_BALANCING_STRATEGY", strconv.Itoa(settings.ResolverStrategy))
	w("PACKET_DUPLICATION_COUNT", strconv.Itoa(clamp(settings.PacketDup, 1, 12)))
	w("SETUP_PACKET_DUPLICATION_COUNT", strconv.Itoa(clamp(settings.SetupPacketDup, 1, 12)))
	w("UPLOAD_COMPRESSION_TYPE", strconv.Itoa(settings.UpCompression))
	w("DOWNLOAD_COMPRESSION_TYPE", strconv.Itoa(settings.DownCompression))
	w("COMPRESSION_MIN_SIZE", strconv.Itoa(settings.CompressionMinSize))
	w("RX_TX_WORKERS", strconv.Itoa(clamp(settings.RxTxWorkers, 1, 64)))
	w("TUNNEL_PROCESS_WORKERS", strconv.Itoa(clamp(settings.TunnelWorkers, 1, 64)))
	b.WriteString("\n")
	w("MIN_UPLOAD_MTU", strconv.Itoa(settings.MinUpMTU))
	w("MAX_UPLOAD_MTU", strconv.Itoa(settings.MaxUpMTU))
	w("MIN_DOWNLOAD_MTU", strconv.Itoa(settings.MinDownMTU))
	w("MAX_DOWNLOAD_MTU", strconv.Itoa(settings.MaxDownMTU))
	b.WriteString("\n")
	w("BASE_ENCODE_DATA", strconv.FormatBool(settings.BaseEncodeData))
	w("LOG_LEVEL", "\""+settings.LogLevel+"\"")
	return os.WriteFile(configPath, []byte(b.String()), 0644)
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

const proxyAddr = "127.0.0.1:18000"

func applySystemProxy(on bool) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.SET_VALUE)
	if err != nil {
		appendLog("system proxy error: "+err.Error())
		return
	}
	defer k.Close()
	if on {
		_ = k.SetDWordValue("ProxyEnable", 1)
		_ = k.SetStringValue("ProxyServer", proxyAddr)
		_ = k.SetStringValue("ProxyOverride", "localhost;127.*;192.168.*;10.*;<local>")
		proxyApplied = true
		appendLog("System proxy ON -> " + proxyAddr)
	} else {
		_ = k.SetDWordValue("ProxyEnable", 0)
		proxyApplied = false
		appendLog("System proxy OFF")
	}
}

func refreshInternet() {
	exec.Command("cmd", "/C", "taskkill /F /IM explorer.exe >nul & start explorer.exe").Start()
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
	mu.Unlock()

	pullSettings()
	if len(splitList(settings.Domains)) == 0 {
		walk.MsgBox(mw, "ZarvandVPN", "Add at least one tunnel domain in the Settings tab.", walk.MsgBoxIconWarning)
		if tabWidget != nil {
			tabWidget.SetCurrentIndex(1)
		}
		return
	}
	if strings.TrimSpace(settings.Key) == "" {
		walk.MsgBox(mw, "ZarvandVPN", "Encryption key is empty. Set it in the Settings tab.", walk.MsgBoxIconWarning)
		if tabWidget != nil {
			tabWidget.SetCurrentIndex(1)
		}
		return
	}
	if err := writeConfig(); err != nil {
		walk.MsgBox(mw, "ZarvandVPN", "Config error: "+err.Error(), walk.MsgBoxIconError)
		return
	}
	_ = persistSettings()

	mu.Lock()
	running = true
	mu.Unlock()

	go func() {
		appendLog("Starting ZarvandVPN core (" + Version + ")...")
		if mw != nil {
			mw.Synchronize(func() {
				applySystemProxy(chkProxy != nil && chkProxy.Checked())
			})
		}
		logFile := filepath.Join(configDirSafe(), "core.log")
		c := exec.Command(corePath, "-config", configPath, "-resolvers", resolversPath, "-log", logFile)
		c.Dir = configDirSafe()
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
		setRunning(true)
		err := c.Wait()
		if mw != nil {
			mw.Synchronize(func() {
				if proxyApplied {
					applySystemProxy(false)
				}
			})
		}
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

func settingsRow(label string, w Widget) Widget {
	return Composite{
		Layout: HBox{MarginsZero: true},
		Children: []Widget{
			Label{Text: label, MinSize: Size{Width: 190}},
			w,
		},
	}
}

func lineEdit(assignTo **walk.LineEdit) Widget {
	return LineEdit{AssignTo: assignTo, MinSize: Size{Width: 240}}
}

func numberEdit(assignTo **walk.LineEdit, width int) Widget {
	return LineEdit{AssignTo: assignTo, MinSize: Size{Width: width}}
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
	loadSettings()

	err := (MainWindow{
		AssignTo: &mw,
		Title:    "ZarvandVPN - " + Version,
		MinSize:  Size{Width: 620, Height: 620},
		Size:     Size{Width: 700, Height: 700},
		Layout:   VBox{},
		Children: []Widget{
			Label{Text: "ZarvandVPN  |  DNS Tunnel Client  |  " + arch(), Font: Font{PointSize: 14, Bold: true}},
			TabWidget{
				AssignTo: &tabWidget,
				Pages: []TabPage{
					{
						Title:  "Connect",
						Layout: VBox{},
						Children: []Widget{
							HSplitter{
								Children: []Widget{
									PushButton{AssignTo: &startBtn, Text: "Connect", MinSize: Size{Width: 140, Height: 40}, OnClicked: func() { startCore() }},
									PushButton{AssignTo: &stopBtn, Text: "Disconnect", Enabled: false, MinSize: Size{Width: 140, Height: 40}, OnClicked: func() { stopCore() }},
								},
							},
							Label{AssignTo: &statusLbl, Text: "Status: Disconnected", TextColor: walk.RGB(200, 0, 0), Font: Font{PointSize: 11, Bold: true}},
							Label{Text: "Local proxy (point your app at this):"},
							Label{Text: "    SOCKS5  127.0.0.1:18000", Font: Font{Family: "Consolas", PointSize: 10}},
							CheckBox{AssignTo: &chkProxy, Text: "Use as Windows system proxy automatically (browsers/apps)  — port " + proxyAddr},
							Label{Text: "Log:"},
							TextEdit{AssignTo: &logView, VScroll: true, ReadOnly: true},
							Label{Text: "Config: %APPDATA%\\ZarvandVPN\\client_config.toml", TextColor: walk.RGB(120, 120, 120)},
						},
					},
					{
						Title:  "Settings",
						Layout: VBox{},
						Children: []Widget{
							ScrollView{
								Layout: VBox{},
								Children: []Widget{
									Label{Text: "Tunnel & Security", Font: Font{Bold: true}},
									settingsRow("Tunnel domains (comma separated)", lineEdit(&edDomains)),
									settingsRow("Encryption key", LineEdit{AssignTo: &edKey, PasswordMode: true, MinSize: Size{Width: 240}}),
									settingsRow("Encryption method", ComboBox{AssignTo: &cbEnc, Model: []string{"None", "XOR", "ChaCha20", "AES-128-GCM", "AES-192-GCM", "AES-256-GCM"}, MinSize: Size{Width: 240}}),
									settingsRow("Protocol", ComboBox{AssignTo: &cbProtocol, Model: []string{"SOCKS5", "TCP"}, MinSize: Size{Width: 240}}),

									Label{Text: "Resolvers (one IP per line)", Font: Font{Bold: true}},
									LineEdit{AssignTo: &edResolvers, MinSize: Size{Height: 60}},

									Label{Text: "Local Proxy", Font: Font{Bold: true}},
									settingsRow("Listen IP", lineEdit(&edListenIP)),
									settingsRow("Listen port", numberEdit(&edListenPort, 120)),
									CheckBox{AssignTo: &chkSocksAuth, Text: "Require SOCKS5 username/password"},
									settingsRow("SOCKS5 username", lineEdit(&edSocksUser)),
									settingsRow("SOCKS5 password", lineEdit(&edSocksPass)),

									Label{Text: "Local DNS", Font: Font{Bold: true}},
									CheckBox{AssignTo: &chkLocalDNS, Text: "Enable local DNS server"},
									settingsRow("DNS listen IP", lineEdit(&edLocalDNSIP)),
									settingsRow("DNS listen port", numberEdit(&edLocalDNSPort, 120)),
									CheckBox{AssignTo: &chkCache, Text: "Persist DNS cache to file"},
									settingsRow("DNS cache TTL (seconds)", numberEdit(&edCacheTTL, 120)),

									Label{Text: "Performance", Font: Font{Bold: true}},
									settingsRow("Packet duplication (1-12)", numberEdit(&edDup, 120)),
									settingsRow("Setup duplication (1-12)", numberEdit(&edSetupDup, 120)),
									settingsRow("Upload compression", ComboBox{AssignTo: &cbUpComp, Model: []string{"Off", "Zstd", "LZ4", "Zlib"}, MinSize: Size{Width: 240}}),
									settingsRow("Download compression", ComboBox{AssignTo: &cbDownComp, Model: []string{"Off", "Zstd", "LZ4", "Zlib"}, MinSize: Size{Width: 240}}),
									settingsRow("Compression min size (bytes)", numberEdit(&edCompMin, 120)),
									settingsRow("RX/TX workers", numberEdit(&edRxTx, 120)),
									settingsRow("Tunnel process workers", numberEdit(&edTunWorkers, 120)),
									settingsRow("Resolver balancing", ComboBox{AssignTo: &cbStrategy, Model: []string{"Round-robin", "Least-latency", "Random", "Sticky"}, MinSize: Size{Width: 240}}),

									Label{Text: "MTU", Font: Font{Bold: true}},
									settingsRow("Min upload MTU", numberEdit(&edMinUp, 120)),
									settingsRow("Max upload MTU", numberEdit(&edMaxUp, 120)),
									settingsRow("Min download MTU", numberEdit(&edMinDown, 120)),
									settingsRow("Max download MTU", numberEdit(&edMaxDown, 120)),

									Label{Text: "Advanced", Font: Font{Bold: true}},
									CheckBox{AssignTo: &chkBase64, Text: "Base-encode data (extra obfuscation)"},
									settingsRow("Log level", ComboBox{AssignTo: &cbLog, Model: []string{"DEBUG", "INFO", "WARN", "ERROR"}, MinSize: Size{Width: 240}}),

									HSplitter{
										Children: []Widget{
											PushButton{Text: "Save Settings", MinSize: Size{Width: 150, Height: 34}, OnClicked: func() {
												pullSettings()
												if err := writeConfig(); err != nil {
													walk.MsgBox(mw, "ZarvandVPN", "Save failed: "+err.Error(), walk.MsgBoxIconError)
													return
												}
												if err := persistSettings(); err != nil {
													walk.MsgBox(mw, "ZarvandVPN", "Save failed: "+err.Error(), walk.MsgBoxIconError)
													return
												}
												walk.MsgBox(mw, "ZarvandVPN", "Settings saved.", walk.MsgBoxIconInformation)
											}},
											PushButton{Text: "Reset Defaults", MinSize: Size{Width: 150, Height: 34}, OnClicked: func() {
												settings = defaultSettings()
												pushSettings()
											}},
										},
									},
									VSpacer{},
								},
							},
						},
					},
				},
			},
		},
	}).Create()
	if err != nil {
		walk.MsgBox(nil, "ZarvandVPN", "UI error: "+err.Error(), walk.MsgBoxIconError)
		return
	}
	pushSettings()
	setRunning(false)
	_ = writeConfig()
	if icoPath := filepath.Join(configDirSafe(), "zarvand.ico"); true {
		_ = os.WriteFile(icoPath, iconData, 0644)
		if ic, err := walk.NewIconFromFile(icoPath); err == nil {
			mw.SetIcon(ic)
		}
	}
	mw.Run()
}
