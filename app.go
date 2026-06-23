package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type Command struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Syntax      string   `json:"syntax"`
	Examples    []string `json:"examples"`
	Difficulty  string   `json:"difficulty"`
	Related     []string `json:"related"`
	Scenario    string   `json:"scenario"`
	Platforms  []string `json:"platforms"`
	AltFor     string   `json:"altFor"`
}

type Category struct {
	Name     string    `json:"name"`
	Icon     string    `json:"icon"`
	Role     string    `json:"role"`
	Commands []Command `json:"commands"`
}

type GuideStep struct {
	Step string `json:"step"`
}

type InstallGuide struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	URL         string      `json:"url"`
	Steps       []GuideStep `json:"steps"`
	Tips        string      `json:"tips"`
	Note        string      `json:"note"`
}

type GuideCategory struct {
	Name   string         `json:"name"`
	Icon   string         `json:"icon"`
	Guides []InstallGuide `json:"guides"`
}

type ExecResult struct {
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error"`
}

type FetchResult struct {
	Success bool   `json:"success"`
	Content string `json:"content"`
	Error   string `json:"error"`
}

type SysInfo struct {
	OS          string `json:"os"`
	Hostname    string `json:"hostname"`
	Kernel      string `json:"kernel"`
	CPU         string `json:"cpu"`
	Cores       string `json:"cores"`
	Memory      string `json:"memory"`
	Disk        string `json:"disk"`
	Uptime      string `json:"uptime"`
	GoVersion   string `json:"goVersion"`
	Shell       string `json:"shell"`
	Desktop     string `json:"desktop"`
}

type NoteData struct {
	CmdName string `json:"cmdName"`
	Note    string `json:"note"`
}

type App struct {
	ctx          context.Context
	notes        map[string]string
	notesFile    string
	cancelExec   context.CancelFunc
	execTimeout  time.Duration
	distro       string
}

func (a *App) GetCurrentDistro() string {
	return a.distro
}

func detectDistro() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		data, err = os.ReadFile("/usr/lib/os-release")
		if err != nil {
			return "linux"
		}
	}
	lines := strings.Split(string(data), "\n")
	id := ""
	idLike := ""
	for _, line := range lines {
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
		if strings.HasPrefix(line, "ID_LIKE=") {
			idLike = strings.Trim(strings.TrimPrefix(line, "ID_LIKE="), "\"")
		}
	}
	switch id {
	case "ubuntu", "debian", "kali", "linuxmint", "elementary", "pop", "zorin":
		return "debian"
	case "rhel", "centos", "fedora", "rocky", "almalinux", "ol":
		if id == "fedora" {
			return "fedora"
		}
		return "rhel"
	case "arch", "manjaro", "endeavouros", "garuda":
		return "arch"
	case "opensuse", "sles", "suse":
		return "suse"
	case "alpine":
		return "alpine"
	case "void":
		return "void"
	}
	if strings.Contains(idLike, "debian") {
		return "debian"
	}
	if strings.Contains(idLike, "rhel") || strings.Contains(idLike, "fedora") {
		return "rhel"
	}
	if strings.Contains(idLike, "arch") {
		return "arch"
	}
	if strings.Contains(idLike, "suse") {
		return "suse"
	}
	return "linux"
}

type DistroInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Pretty  string `json:"pretty"`
	Version string `json:"version"`
}

func (a *App) GetDistroInfo() DistroInfo {
	info := DistroInfo{}
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return DistroInfo{ID: a.distro, Name: "Linux"}
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			info.Pretty = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		}
		if strings.HasPrefix(line, "NAME=") {
			info.Name = strings.Trim(strings.TrimPrefix(line, "NAME="), "\"")
		}
		if strings.HasPrefix(line, "VERSION_ID=") {
			info.Version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		}
	}
	info.ID = a.distro
	return info
}

func enrichCommands(cats []Category) []Category {
	difficultyMap := map[string]string{}
	for _, s := range []string{"ls,cat,echo,pwd,mkdir,touch,cp,mv,rm,whoami,id,uname,free,df,uptime,history,alias,export,which,man,help,date,cal,clear,exit,cd,less,more,head,tail,wc,ping,curl,wget,zip,unzip,gzip,gunzip,shutdown,reboot,passwd,sudo,su,groups,chmod,chown,file,tar,ssh,scp,git,git clone,nslookup,dig,watch,sleep,kill,ps,top,hostnamectl,timedatectl,neofetch,docker,docker run"} {
		difficultyMap[s] = "beginner"
	}
	for _, s := range []string{"find,grep,sed,awk,sort,diff,cut,ln,dd,rsync,screen/tmux,nohup,bg/fg,killall,pkill,crontab,apt,apt-cache,dpkg,snap,flatpak,yum,dnf,pacman,rpm,pip,npm,xargs,tee,du,blkid,lsblk,fdisk,parted,mount,umount,swapon/swapoff,fsck,smartctl,ip,ss,netstat,nmcli,firewall-cmd,traceroute,journalctl,systemctl,useradd,usermod,userdel,groupadd,last,git init,git add,git commit,git push,git pull,git branch,git merge,git stash,git diff,git status,git log,docker ps,docker images,docker pull,docker build,docker exec,docker logs"} {
		difficultyMap[s] = "intermediate"
	}
	for _, s := range []string{"gdisk,mkfs,parted,7z,xz,bzip2,zcat/zless,dd,smartctl,nmap,tcpdump,ssh-keygen,ssh-copy-id,ssh-copy-id,sftp,telnet,nc,firewall-cmd,git revert,git rebase,git reset,docker-compose"} {
		difficultyMap[s] = "advanced"
	}

	relatedMap := map[string][]string{
		"ls":      {"cd", "pwd", "tree", "find"},
		"cat":     {"less", "more", "head", "tail", "nl"},
		"cd":      {"pwd", "ls"},
		"cp":      {"mv", "rsync", "scp"},
		"mv":      {"cp", "rename"},
		"rm":      {"rmdir", "shred"},
		"chmod":   {"chown", "ls -l"},
		"chown":   {"chmod", "chgrp"},
		"find":    {"grep", "locate", "which"},
		"grep":    {"find", "sed", "awk", "rg"},
		"ps":      {"top", "htop", "kill", "pkill"},
		"top":     {"htop", "ps", "kill"},
		"kill":    {"killall", "pkill", "xkill"},
		"free":    {"top", "vmstat", "htop"},
		"df":      {"du", "lsblk", "fdisk"},
		"du":      {"df", "ncdu"},
		"ping":    {"curl", "wget", "traceroute", "nslookup"},
		"curl":    {"wget", "httpie"},
		"wget":    {"curl", "aria2c"},
		"ssh":     {"scp", "rsync", "sftp", "ssh-keygen"},
		"scp":     {"rsync", "sftp", "ssh"},
		"rsync":   {"scp", "cp", "tar"},
		"tar":     {"gzip", "bzip2", "xz", "zip", "unzip"},
		"zip":     {"unzip", "tar", "gzip"},
		"unzip":   {"zip", "tar"},
		"gzip":    {"gunzip", "tar", "zcat/zless"},
		"sed":     {"awk", "grep", "cut", "tr"},
		"awk":     {"sed", "grep", "cut", "sort"},
		"sort":    {"uniq", "wc", "cut"},
		"diff":    {"cmp", "patch", "vimdiff"},
		"cut":     {"awk", "sed", "sort"},
		"iptables": {"firewall-cmd", "ufw", "nftables"},
		"systemctl": {"journalctl", "service"},
		"journalctl": {"systemctl", "dmesg"},
		"docker":  {"docker-compose", "podman"},
		"docker-compose": {"docker", "podman"},
		"git":     {"git add", "git commit", "git push", "git pull"},
		"git init": {"git clone", "git status"},
		"git clone": {"git init", "git pull"},
		"vim":     {"nano", "emacs", "neovim"},
		"less":    {"more", "cat", "tail"},
		"more":    {"less", "cat"},
		"head":    {"tail", "cat", "less"},
		"tail":    {"head", "cat", "less"},
		"fdisk":   {"gdisk", "parted", "lsblk", "blkid"},
		"mount":   {"umount", "lsblk", "blkid"},
		"umount":  {"mount", "lsof"},
		"dd":      {"cp", "rsync", "cat"},
		"nohup":   {"bg/fg", "screen/tmux", "disown"},
		"crontab": {"at", "systemd-timer"},
		"sudo":    {"su", "doas"},
		"su":      {"sudo", "whoami"},
		"passwd":  {"chpasswd", "usermod"},
		"ip":      {"ifconfig", "nmcli", "ss"},
		"ss":      {"netstat", "ip"},
		"nmcli":   {"ip", "nmtui"},
		"nslookup":{"dig", "host"},
		"dig":    {"nslookup", "host"},
		"traceroute": {"ping", "mtr", "tracepath"},
		"nmap":   {"netstat", "ss", "tcpdump"},
		"tcpdump":{"nmap", "tshark", "tcpflow"},
		"alias":  {"export", "history", "unalias"},
		"export": {"alias", "env", "set"},
		"xargs":  {"find", "parallel"},
		"watch":  {"sleep", "crontab"},
		"shutdown":{"reboot", "halt", "poweroff", "systemctl"},
		"useradd":{"usermod", "userdel", "passwd", "groupadd"},
		"usermod":{"useradd", "userdel", "passwd", "groups"},
		"userdel":{"useradd", "usermod"},
		"uptime": {"w", "who", "top"},
		"lscpu":  {"lspci", "lsusb", "neofetch", "uname"},
		"uname":  {"lscpu", "neofetch", "hostnamectl"},
		"dmesg":  {"journalctl", "kern.log"},
		"lsblk":  {"blkid", "fdisk", "df"},
		"blkid":  {"lsblk", "fdisk"},
		"swapon/swapoff": {"free", "fdisk"},
		"fsck":   {"smartctl", "fdisk"},
	}

	platformMap := map[string][]string{
		"apt":     {"debian"}, "apt-cache": {"debian"}, "dpkg": {"debian"},
		"yum":     {"rhel"}, "dnf": {"fedora"}, "pacman": {"arch"},
		"rpm":     {"rhel", "fedora"}, "snap": {"debian", "fedora", "arch"},
		"flatpak": {"debian", "fedora", "arch"},
		"firewall-cmd": {"rhel", "fedora"},
		"nmcli":   {"rhel", "fedora", "debian"},
		"zypper":  {"suse"},
		"apk":     {"alpine"},
	}

	altForMap := map[string]string{
		"apt":     "Debian/Ubuntu 系列使用 apt，CentOS/RHEL 用 yum/dnf，Arch 用 pacman",
		"yum":     "CentOS/RHEL 7 使用 yum，Ubuntu 用 apt，Fedora 用 dnf，Arch 用 pacman",
		"dnf":     "Fedora/RHEL 8+ 使用 dnf，Ubuntu 用 apt，Arch 用 pacman",
		"pacman":  "Arch Linux 使用 pacman，Ubuntu 用 apt，CentOS 用 yum/dnf",
		"rpm":     "RPM 是 Red Hat 系的底层包格式，Debian 系的对应是 dpkg",
		"dpkg":    "dpkg 是 Debian 系的底层包格式，RHEL 系的对应是 rpm",
		"snap":    "Snap 是 Canonical 开发的跨发行版包格式，主流发行版均支持",
		"flatpak": "Flatpak 是跨发行版应用沙箱，适用于所有主流发行版",
		"firewall-cmd": "firewall-cmd 是 RHEL/Fedora 的防火墙工具，Ubuntu 用 ufw",
		"zypper":  "openSUSE 使用 zypper，Ubuntu 用 apt，CentOS 用 yum",
	}

	for ci := range cats {
		for cj := range cats[ci].Commands {
			cmd := &cats[ci].Commands[cj]
			if d, ok := difficultyMap[cmd.Name]; ok {
				cmd.Difficulty = d
			} else if d, ok = difficultyMap[cats[ci].Name+"/"+cmd.Name]; ok {
				cmd.Difficulty = d
			} else {
				cmd.Difficulty = "intermediate"
			}
			if r, ok := relatedMap[cmd.Name]; ok {
				cmd.Related = r
			}
			cmd.Scenario = getScenario(cmd.Name, cats[ci].Name)
			if p, ok := platformMap[cmd.Name]; ok {
				cmd.Platforms = p
			} else {
				cmd.Platforms = []string{"*"}
			}
			if a, ok := altForMap[cmd.Name]; ok {
				cmd.AltFor = a
			}
		}
	}
	return cats
}

func getScenario(name, catName string) string {
	scenarios := map[string]string{
		"ls":       "查看目录中有哪些文件",
		"cd":       "进入不同目录",
		"pwd":      "查看当前所在目录路径",
		"cp":       "备份文件或复制文件到其他位置",
		"mv":       "给文件重命名或移动到其他目录",
		"rm":       "删除不需要的文件或目录",
		"mkdir":    "创建新目录",
		"touch":    "创建空文件",
		"cat":      "查看小文件内容",
		"less":     "分页查看大文件内容",
		"head":     "查看文件前几行",
		"tail":     "查看文件末尾几行（常用于看日志）",
		"grep":     "在文件中搜索关键词",
		"find":     "在目录树中查找文件",
		"chmod":    "修改文件权限",
		"chown":    "修改文件所有者",
		"ps":       "查看当前运行的进程",
		"top":      "实时查看系统进程和资源占用",
		"kill":     "强制结束某个进程",
		"free":     "查看内存使用情况",
		"df":       "查看磁盘空间使用情况",
		"du":       "查看目录或文件占用多大空间",
		"ping":     "测试网络是否连通",
		"curl":     "从命令行发送 HTTP 请求",
		"wget":     "从网络下载文件",
		"ssh":      "远程登录服务器",
		"scp":      "在本地和远程之间复制文件",
		"tar":      "打包或解压文件（.tar.gz）",
		"zip":      "压缩成 ZIP 格式",
		"unzip":    "解压 ZIP 文件",
		"sudo":     "以管理员权限执行命令",
		"apt":      "安装/卸载/更新软件（Debian/Ubuntu）",
		"systemctl":"管理系统服务（启动/停止/开机自启）",
		"journalctl":"查看系统日志",
		"ifconfig": "查看和配置网络接口",
		"netstat":  "查看网络连接和监听端口",
		"crontab":  "设置定时任务",
		"history":  "查看执行过的命令历史",
		"alias":    "给常用命令设置快捷方式",
		"export":   "设置环境变量",
		"docker":   "运行和管理容器",
		"git":      "版本控制",
	}
	if s, ok := scenarios[name]; ok {
		return s
	}
	return "Linux " + catName + " 命令"
}


func NewApp() *App {
	return &App{notes: make(map[string]string), execTimeout: 60 * time.Second}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.distro = detectDistro()
	home, _ := os.UserHomeDir()
	dir := home + "/.config/linux-cmd-toolbox"
	os.MkdirAll(dir, 0755)
	a.notesFile = dir + "/notes.json"
	if data, err := os.ReadFile(a.notesFile); err == nil {
		json.Unmarshal(data, &a.notes)
	}
	if a.notes == nil {
		a.notes = make(map[string]string)
	}
}

func run(name string, args ...string) string {
	var buf bytes.Buffer
	c := exec.Command(name, args...)
	c.Stdout = &buf
	c.Stderr = &buf
	c.Run()
	return strings.TrimSpace(buf.String())
}

func (a *App) GetSystemInfo() SysInfo {
	info := SysInfo{
		OS:        runtime.GOOS,
		GoVersion: runtime.Version()[2:],
	}
	info.Hostname, _ = os.Hostname()
	info.Kernel = run("uname", "-r")
	info.CPU = run("sh", "-c", `lscpu | grep "Model name" | sed 's/.*:\s*//'`)
	if info.CPU == "" {
		info.CPU = run("sh", "-c", `cat /proc/cpuinfo | grep "model name" | head -1 | sed 's/.*:\s*//'`)
	}
	info.Cores = run("sh", "-c", `nproc`)
	info.Memory = run("sh", "-c", `free -h | grep "Mem:" | awk '{print $3 "/" $2}'`)
	info.Disk = run("sh", "-c", `df -h / | tail -1 | awk '{print $3 "/" $2 " (" $5 ")"}'`)
	info.Uptime = run("sh", "-c", `uptime -p | sed 's/up //'`)
	info.Shell = os.Getenv("SHELL")
	info.Desktop = os.Getenv("XDG_CURRENT_DESKTOP")
	return info
}

func (a *App) SaveNote(cmdName string, note string) bool {
	if note == "" {
		delete(a.notes, cmdName)
	} else {
		a.notes[cmdName] = note
	}
	data, _ := json.MarshalIndent(a.notes, "", "  ")
	os.WriteFile(a.notesFile, data, 0644)
	return true
}

func (a *App) GetNote(cmdName string) string {
	return a.notes[cmdName]
}

func (a *App) GetAllNotes() string {
	b, _ := json.Marshal(a.notes)
	return string(b)
}

func (a *App) ImportNotes(jsonStr string) bool {
	var m map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &m); err != nil {
		return false
	}
	for k, v := range m {
		a.notes[k] = v
	}
	return true
}

func (a *App) ExportData(favsJSON string) string {
	var favs []string
	json.Unmarshal([]byte(favsJSON), &favs)
	data := map[string]interface{}{
		"exportTime": time.Now().Format("2006-01-02 15:04:05"),
		"favorites":  favs,
		"notes":      a.notes,
	}
	b, _ := json.MarshalIndent(data, "", "  ")
	return string(b)
}

func (a *App) GetCategories() []Category {
	return enrichCommands([]Category{
		{
			Name: "系统信息", Icon: "🖥", Role: "ops",
			Commands: []Command{
				{Name: "uname", Description: "显示系统信息", Syntax: "uname [选项]", Examples: []string{"uname -a  # 显示全部系统信息", "uname -r  # 显示内核版本", "uname -m  # 显示架构"}},
				{Name: "lscpu", Description: "显示 CPU 架构信息", Syntax: "lscpu", Examples: []string{"lscpu  # 显示 CPU 详细信息", "lscpu | grep 'Model name'  # 查看 CPU 型号"}},
				{Name: "free", Description: "显示内存使用情况", Syntax: "free [选项]", Examples: []string{"free -h  # 以人类可读格式显示内存", "free -m  # 以 MB 为单位显示", "free -s 2  # 每2秒刷新"}},
				{Name: "df", Description: "显示磁盘分区使用情况", Syntax: "df [选项]", Examples: []string{"df -h  # 人类可读", "df -T  # 显示文件系统类型", "df -i  # 显示 inode"}},
				{Name: "du", Description: "估算文件和目录的磁盘使用量", Syntax: "du [选项] [目录]", Examples: []string{"du -sh *  # 当前目录各项目大小", "du -h --max-depth=1  # 一层深度", "du -h /home  # 查看 /home"}},
				{Name: "lsblk", Description: "列出块设备信息", Syntax: "lsblk [选项]", Examples: []string{"lsblk  # 所有块设备", "lsblk -f  # 文件系统信息"}},
				{Name: "dmesg", Description: "显示内核环缓冲区消息", Syntax: "dmesg [选项]", Examples: []string{"dmesg | tail -20  # 最后20条", "dmesg -w  # 实时跟踪", "dmesg --level=err  # 只看错误"}},
				{Name: "lspci", Description: "列出所有 PCI 设备", Syntax: "lspci [选项]", Examples: []string{"lspci  # PCI 设备", "lspci -v  # 详细模式"}},
				{Name: "lsusb", Description: "列出 USB 设备", Syntax: "lsusb [选项]", Examples: []string{"lsusb  # USB 设备", "lsusb -t  # 树状拓扑"}},
				{Name: "uptime", Description: "显示系统运行时间", Syntax: "uptime", Examples: []string{"uptime  # 运行时间+负载"}},
				{Name: "hostnamectl", Description: "查看或修改主机名", Syntax: "hostnamectl [命令]", Examples: []string{"hostnamectl  # 主机名信息", "hostnamectl set-hostname newname  # 修改主机名"}},
				{Name: "timedatectl", Description: "查看或设置系统时间", Syntax: "timedatectl [命令]", Examples: []string{"timedatectl  # 时间信息", "timedatectl list-timezones  # 列出时区"}},
				{Name: "neofetch", Description: "显示系统信息和 Logo", Syntax: "neofetch", Examples: []string{"neofetch  # 系统信息（需安装）"}},
			},
		},
		{
			Name: "文件管理", Icon: "📁", Role: "common",
			Commands: []Command{
				{Name: "ls", Description: "列出目录内容", Syntax: "ls [选项] [路径]", Examples: []string{"ls -la  # 含隐藏文件", "ls -lh  # 人类可读大小", "ls -ltr  # 按时间排序", "ls -R  # 递归子目录"}},
				{Name: "cp", Description: "复制文件或目录", Syntax: "cp [选项] 源 目标", Examples: []string{"cp f1 f2  # 复制文件", "cp -r d1 d2  # 递归复制", "cp -a d1 d2  # 归档复制"}},
				{Name: "mv", Description: "移动或重命名文件", Syntax: "mv [选项] 源 目标", Examples: []string{"mv old.txt new.txt  # 重命名", "mv file.txt /tmp/  # 移动"}},
				{Name: "rm", Description: "删除文件或目录", Syntax: "rm [选项] 目标", Examples: []string{"rm file.txt  # 删除文件", "rm -rf dir/  # 强制删除"}},
				{Name: "chmod", Description: "修改文件权限", Syntax: "chmod [选项] 模式 文件", Examples: []string{"chmod +x script.sh  # 加执行权限", "chmod 755 file  # rwxr-xr-x"}},
				{Name: "chown", Description: "修改文件所有者", Syntax: "chown [选项] 用户[:组] 文件", Examples: []string{"chown user:group file  # 修改用户和组", "chown -R user dir  # 递归"}},
				{Name: "find", Description: "查找文件", Syntax: "find [路径] [条件]", Examples: []string{"find / -name '*.log'  # 查找 .log", "find . -type f -size +10M  # 大于10MB"}},
				{Name: "grep", Description: "搜索文件内容", Syntax: "grep [选项] 模式 [文件]", Examples: []string{"grep 'error' log.txt  # 搜索", "grep -r 'TODO' .  # 递归搜索"}},
				{Name: "ln", Description: "创建链接", Syntax: "ln [选项] 目标 链接名", Examples: []string{"ln -s /usr/bin/python3 python  # 软链接", "ln file.txt hardlink  # 硬链接"}},
				{Name: "touch", Description: "创建空文件或更新时间戳", Syntax: "touch [选项] 文件", Examples: []string{"touch newfile.txt  # 创建空文件", "touch -t 202401011200 file  # 时间戳"}},
				{Name: "mkdir", Description: "创建目录", Syntax: "mkdir [选项] 目录", Examples: []string{"mkdir newdir  # 创建", "mkdir -p a/b/c  # 递归创建"}},
				{Name: "file", Description: "查看文件类型", Syntax: "file [选项] 文件", Examples: []string{"file image.png  # 文件类型", "file -i document.pdf  # MIME"}},
			},
		},
		{
			Name: "包管理", Icon: "📦", Role: "dev",
			Commands: []Command{
				{Name: "apt", Description: "Debian/Ubuntu 包管理", Syntax: "apt [选项] 命令", Examples: []string{"apt update  # 更新索引", "apt install vim  # 安装", "apt remove vim  # 卸载", "apt upgrade  # 升级", "apt autoremove  # 清理"}},
				{Name: "apt-cache", Description: "查询 APT 缓存", Syntax: "apt-cache [命令] [包名]", Examples: []string{"apt-cache search nginx  # 搜索", "apt-cache show nginx  # 详情"}},
				{Name: "dpkg", Description: "Debian 包管理器底层", Syntax: "dpkg [选项] 操作", Examples: []string{"dpkg -i package.deb  # 安装 .deb", "dpkg -l  # 已安装列表", "dpkg -L package  # 列文件"}},
				{Name: "snap", Description: "Snap 包管理", Syntax: "snap [命令] [选项]", Examples: []string{"snap install vlc  # 安装", "snap list  # 已安装", "snap refresh  # 更新"}},
				{Name: "flatpak", Description: "Flatpak 包管理", Syntax: "flatpak [命令] [选项]", Examples: []string{"flatpak install flathub org.videolan.VLC  # 安装", "flatpak list  # 已安装"}},
				{Name: "yum", Description: "RHEL/CentOS 7 包管理", Syntax: "yum [选项] 命令", Examples: []string{"yum install nginx  # 安装", "yum update  # 更新"}},
				{Name: "dnf", Description: "Fedora/RHEL 8+ 包管理", Syntax: "dnf [选项] 命令", Examples: []string{"dnf install vim  # 安装", "dnf groupinstall 'Development Tools'  # 工具组"}},
				{Name: "pacman", Description: "Arch Linux 包管理", Syntax: "pacman [选项] 命令", Examples: []string{"pacman -S vim  # 安装", "pacman -Syu  # 更新", "pacman -Ss kw  # 搜索"}},
				{Name: "rpm", Description: "RPM 包管理", Syntax: "rpm [选项] 包", Examples: []string{"rpm -ivh pkg.rpm  # 安装", "rpm -qa  # 查询所有"}},
				{Name: "pip", Description: "Python 包管理器", Syntax: "pip [命令] [选项]", Examples: []string{"pip install torch  # 安装", "pip list  # 已安装", "pip freeze > req.txt  # 导出"}},
				{Name: "npm", Description: "Node.js 包管理器", Syntax: "npm [命令] [选项]", Examples: []string{"npm install express  # 安装", "npm init -y  # 初始化"}},
			},
		},
		{
			Name: "网络管理", Icon: "🌐", Role: "ops",
			Commands: []Command{
				{Name: "ping", Description: "测试网络连通性", Syntax: "ping [选项] 目标", Examples: []string{"ping -c 4 google.com  # 4个包", "ping -i 2 8.8.8.8  # 间隔2秒"}},
				{Name: "curl", Description: "HTTP 请求工具", Syntax: "curl [选项] URL", Examples: []string{"curl https://api.example.com  # GET", "curl -O https://ex.com/file.zip  # 下载", "curl -X POST -d 'key=val' URL  # POST"}},
				{Name: "wget", Description: "网络文件下载工具", Syntax: "wget [选项] URL", Examples: []string{"wget https://ex.com/file.zip  # 下载", "wget -c URL  # 断点续传"}},
				{Name: "ip", Description: "显示/配置网络接口", Syntax: "ip [选项] 对象 命令", Examples: []string{"ip addr  # IP 地址", "ip route  # 路由表", "ip link set eth0 up  # 启用"}},
				{Name: "ss", Description: "套接字统计工具", Syntax: "ss [选项]", Examples: []string{"ss -tuln  # 监听端口", "ss -tup  # TCP 连接+进程"}},
				{Name: "netstat", Description: "网络统计（旧版）", Syntax: "netstat [选项]", Examples: []string{"netstat -tuln  # 监听端口", "netstat -anp  # 所有连接"}},
				{Name: "nmcli", Description: "NetworkManager 工具", Syntax: "nmcli [对象] [命令]", Examples: []string{"nmcli dev status  # 设备状态", "nmcli con show  # 连接", "nmcli dev wifi list  # 扫描 WiFi"}},
				{Name: "nslookup", Description: "DNS 查询", Syntax: "nslookup 域名", Examples: []string{"nslookup google.com  # 查询 IP"}},
				{Name: "dig", Description: "DNS 查询（更强大）", Syntax: "dig [选项] 域名", Examples: []string{"dig google.com  # DNS 记录", "dig -x 8.8.8.8  # 反向查询"}},
				{Name: "traceroute", Description: "路由追踪", Syntax: "traceroute [选项] 目标", Examples: []string{"traceroute google.com  # 路由追踪"}},
				{Name: "nmap", Description: "网络扫描工具", Syntax: "nmap [选项] 目标", Examples: []string{"nmap -sP 192.168.1.0/24  # Ping 扫描", "nmap -O target  # 系统检测"}},
				{Name: "tcpdump", Description: "网络抓包工具", Syntax: "tcpdump [选项] [表达式]", Examples: []string{"tcpdump -i eth0  # 抓包", "tcpdump port 80  # HTTP 流量"}},
				{Name: "firewall-cmd", Description: "firewalld 防火墙", Syntax: "firewall-cmd [选项]", Examples: []string{"firewall-cmd --list-all  # 规则", "firewall-cmd --add-port=80/tcp --permanent  # 开放端口"}},
			},
		},
		{
			Name: "进程管理", Icon: "⚙️", Role: "ops",
			Commands: []Command{
				{Name: "ps", Description: "显示进程快照", Syntax: "ps [选项]", Examples: []string{"ps aux  # 所有进程", "ps -ef | grep nginx  # 查找"}},
				{Name: "top", Description: "实时进程信息", Syntax: "top [选项]", Examples: []string{"top  # 进程查看器", "top -u username  # 指定用户"}},
				{Name: "htop", Description: "进程查看器（增强）", Syntax: "htop", Examples: []string{"htop  # 交互式（需安装）"}},
				{Name: "kill", Description: "终止进程", Syntax: "kill [选项] PID", Examples: []string{"kill -9 1234  # 强制", "kill -15 1234  # 优雅", "kill -STOP 1234  # 暂停"}},
				{Name: "killall", Description: "按名称终止进程", Syntax: "killall [选项] 进程名", Examples: []string{"killall nginx  # 终止所有"}},
				{Name: "systemctl", Description: "Systemd 服务管理器", Syntax: "systemctl [命令] [服务]", Examples: []string{"systemctl start nginx  # 启动", "systemctl enable nginx  # 开机自启", "systemctl status nginx  # 状态", "systemctl list-units  # 列出"}},
				{Name: "journalctl", Description: "Systemd 日志查看器", Syntax: "journalctl [选项]", Examples: []string{"journalctl -u nginx  # nginx 日志", "journalctl -f  # 实时", "journalctl --since '1 hour ago'  # 最近1小时"}},
				{Name: "nohup", Description: "后台运行（忽略挂断）", Syntax: "nohup 命令 &", Examples: []string{"nohup python script.py &  # 后台运行"}},
				{Name: "bg/fg", Description: "前后台任务控制", Syntax: "bg/fg [作业号]", Examples: []string{"Ctrl+Z  # 挂起", "bg  # 后台", "fg  # 前台", "jobs  # 列出"}},
				{Name: "pkill", Description: "按模式终止进程", Syntax: "pkill [选项] 模式", Examples: []string{"pkill -f 'python.*server'  # 匹配命令行"}},
				{Name: "crontab", Description: "计划任务管理", Syntax: "crontab [选项]", Examples: []string{"crontab -e  # 编辑", "crontab -l  # 列出"}},
			},
		},
		{
			Name: "用户管理", Icon: "👤", Role: "ops",
			Commands: []Command{
				{Name: "useradd", Description: "创建新用户", Syntax: "useradd [选项] 用户名", Examples: []string{"useradd -m -s /bin/bash newuser  # 创建用户"}},
				{Name: "usermod", Description: "修改用户账户", Syntax: "usermod [选项] 用户名", Examples: []string{"usermod -aG docker user  # 加入组", "usermod -L user  # 锁定"}},
				{Name: "userdel", Description: "删除用户", Syntax: "userdel [选项] 用户名", Examples: []string{"userdel -r username  # 删除用户+家目录"}},
				{Name: "passwd", Description: "修改用户密码", Syntax: "passwd [用户名]", Examples: []string{"passwd  # 当前用户", "passwd username  # root 修改"}},
				{Name: "groupadd", Description: "创建用户组", Syntax: "groupadd [选项] 组名", Examples: []string{"groupadd developers  # 创建组"}},
				{Name: "groups", Description: "显示所属组", Syntax: "groups [用户名]", Examples: []string{"groups  # 当前用户组"}},
				{Name: "id", Description: "显示用户身份", Syntax: "id [用户名]", Examples: []string{"id  # UID/GID/组", "id -u  # 仅 UID"}},
				{Name: "whoami", Description: "显示当前用户名", Syntax: "whoami", Examples: []string{"whoami  # 当前用户名"}},
				{Name: "sudo", Description: "以超级用户执行", Syntax: "sudo [选项] 命令", Examples: []string{"sudo apt update  # root 执行", "sudo -u user command  # 指定用户"}},
				{Name: "su", Description: "切换用户", Syntax: "su [选项] [用户名]", Examples: []string{"su - username  # 切换登录"}},
				{Name: "last", Description: "登录历史", Syntax: "last [选项]", Examples: []string{"last  # 登录记录", "last -10  # 最近10条"}},
			},
		},
		{
			Name: "磁盘管理", Icon: "💾", Role: "ops",
			Commands: []Command{
				{Name: "fdisk", Description: "磁盘分区（MBR）", Syntax: "fdisk [选项] 设备", Examples: []string{"fdisk -l  # 分区表", "fdisk /dev/sda  # 操作"}},
				{Name: "gdisk", Description: "磁盘分区（GPT）", Syntax: "gdisk [选项] 设备", Examples: []string{"gdisk -l /dev/sda  # 查看 GPT"}},
				{Name: "mkfs", Description: "创建文件系统", Syntax: "mkfs -t 类型 设备", Examples: []string{"mkfs -t ext4 /dev/sdb1  # ext4", "mkfs.xfs /dev/sdb1  # XFS"}},
				{Name: "mount", Description: "挂载文件系统", Syntax: "mount [选项] 设备 挂载点", Examples: []string{"mount /dev/sdb1 /mnt/data  # 挂载", "mount -a  # fstab 全部"}},
				{Name: "umount", Description: "卸载文件系统", Syntax: "umount [选项] 设备/挂载点", Examples: []string{"umount /mnt/data  # 卸载"}},
				{Name: "dd", Description: "低级复制", Syntax: "dd [选项]", Examples: []string{"dd if=/dev/sda of=/backup.img bs=4M  # 备份", "dd if=/dev/zero of=/swapfile bs=1M count=1024  # 交换文件"}},
				{Name: "blkid", Description: "块设备属性", Syntax: "blkid [设备]", Examples: []string{"blkid  # UUID 和类型"}},
				{Name: "parted", Description: "分区工具（高级）", Syntax: "parted [选项] 设备", Examples: []string{"parted /dev/sda print  # 分区表", "parted /dev/sda mklabel gpt  # GPT"}},
				{Name: "lsblk", Description: "列出块设备", Syntax: "lsblk [选项]", Examples: []string{"lsblk  # 树状显示"}},
				{Name: "swapon/swapoff", Description: "交换分区管理", Syntax: "swapon [选项] 设备", Examples: []string{"swapon -s  # 状态", "swapon /swapfile  # 启用"}},
				{Name: "fsck", Description: "文件系统检查", Syntax: "fsck [选项] 设备", Examples: []string{"fsck /dev/sda1  # 检查"}},
				{Name: "smartctl", Description: "S.M.A.R.T. 检测", Syntax: "smartctl [选项] 设备", Examples: []string{"smartctl -H /dev/sda  # 健康状态"}},
			},
		},
		{
			Name: "压缩归档", Icon: "🗜️", Role: "common",
			Commands: []Command{
				{Name: "tar", Description: "归档工具", Syntax: "tar [选项] 归档 [文件]", Examples: []string{"tar -czf archive.tar.gz dir/  # 创建 gz", "tar -xzf archive.tar.gz  # 解压 gz", "tar -tf archive.tar  # 查看"}},
				{Name: "gzip", Description: "GNU Zip 压缩", Syntax: "gzip [选项] 文件", Examples: []string{"gzip file.txt  # 压缩", "gzip -d file.txt.gz  # 解压"}},
				{Name: "zip", Description: "创建 ZIP", Syntax: "zip [选项] 归档 文件...", Examples: []string{"zip archive.zip file1 file2  # 创建", "zip -r archive.zip dir/  # 递归"}},
				{Name: "unzip", Description: "解压 ZIP", Syntax: "unzip [选项] 文件.zip", Examples: []string{"unzip archive.zip  # 解压", "unzip archive.zip -d /target/  # 指定目录"}},
				{Name: "bzip2", Description: "BZ2 压缩", Syntax: "bzip2 [选项] 文件", Examples: []string{"bzip2 file.txt  # 压缩", "bzip2 -d file.txt.bz2  # 解压"}},
				{Name: "xz", Description: "LZMA 压缩", Syntax: "xz [选项] 文件", Examples: []string{"xz file.txt  # 压缩", "xz -d file.txt.xz  # 解压"}},
				{Name: "7z", Description: "7-Zip 压缩", Syntax: "7z [命令] [选项] 归档", Examples: []string{"7z a archive.7z file.txt  # 添加", "7z x archive.7z  # 解压"}},
				{Name: "zcat/zless", Description: "查看压缩文件", Syntax: "zcat 文件.gz", Examples: []string{"zcat access.log.gz | tail -20  # 查看压缩日志"}},
			},
		},
		{
			Name: "远程连接", Icon: "🔗", Role: "ops",
			Commands: []Command{
				{Name: "ssh", Description: "SSH 远程连接", Syntax: "ssh [选项] user@host", Examples: []string{"ssh user@192.168.1.1  # 连接", "ssh -p 2222 user@host  # 指定端口", "ssh -i key.pem user@host  # 密钥"}},
				{Name: "scp", Description: "SSH 文件复制", Syntax: "scp [选项] 源 目标", Examples: []string{"scp file.txt user@host:/tmp/  # 上传", "scp user@host:/tmp/file.txt .  # 下载"}},
				{Name: "rsync", Description: "远程同步", Syntax: "rsync [选项] 源 目标", Examples: []string{"rsync -avz dir/ user@host:/backup/  # 同步", "rsync --progress bigfile user@host:/tmp/  # 进度"}},
				{Name: "sftp", Description: "SSH 文件传输", Syntax: "sftp [选项] user@host", Examples: []string{"sftp user@host  # 交互式"}},
				{Name: "telnet", Description: "Telnet 连接", Syntax: "telnet host [端口]", Examples: []string{"telnet smtp.example.com 25  # 测试端口"}},
				{Name: "nc", Description: "Netcat 网络工具", Syntax: "nc [选项] host port", Examples: []string{"nc -zv 192.168.1.1 22  # 端口扫描", "nc -l -p 8080  # 监听"}},
				{Name: "ssh-keygen", Description: "SSH 密钥管理", Syntax: "ssh-keygen [选项]", Examples: []string{"ssh-keygen -t ed25519  # 生成密钥"}},
				{Name: "ssh-copy-id", Description: "复制公钥到远程", Syntax: "ssh-copy-id [选项] user@host", Examples: []string{"ssh-copy-id user@host  # 复制公钥"}},
				{Name: "screen/tmux", Description: "终端复用器", Syntax: "screen/tmux [命令]", Examples: []string{"tmux new -s mysession  # 创建会话", "tmux ls  # 列出"}},
			},
		},
		{
			Name: "文本处理", Icon: "📝", Role: "dev",
			Commands: []Command{
				{Name: "cat", Description: "连接文件输出", Syntax: "cat [选项] 文件", Examples: []string{"cat file.txt  # 显示", "cat -n file.txt  # 行号"}},
				{Name: "less", Description: "分页查看", Syntax: "less [选项] 文件", Examples: []string{"less large.log  # 分页 (q 退出)", "less +F file.log  # 跟踪"}},
				{Name: "head", Description: "显示开头", Syntax: "head [选项] 文件", Examples: []string{"head -n 20 file.txt  # 前20行"}},
				{Name: "tail", Description: "显示末尾", Syntax: "tail [选项] 文件", Examples: []string{"tail -f /var/log/syslog  # 实时跟踪"}},
				{Name: "vim", Description: "文本编辑器", Syntax: "vim [选项] 文件", Examples: []string{"vim file.txt  # 编辑", "vimdiff f1 f2  # 比较"}},
				{Name: "sed", Description: "流式编辑器", Syntax: "sed [选项] '命令' 文件", Examples: []string{"sed -i 's/old/new/g' file.txt  # 替换"}},
				{Name: "awk", Description: "数据处理语言", Syntax: "awk '模式{动作}' 文件", Examples: []string{"awk '{print $1}' file.txt  # 第一列", "awk -F: '{print $1}' /etc/passwd"}},
				{Name: "sort", Description: "文本排序", Syntax: "sort [选项] 文件", Examples: []string{"sort file.txt  # 字母排序", "sort -n file.txt  # 数字排序"}},
				{Name: "wc", Description: "统计", Syntax: "wc [选项] 文件", Examples: []string{"wc -l file.txt  # 行数", "wc -w file.txt  # 词数"}},
				{Name: "diff", Description: "比较差异", Syntax: "diff [选项] f1 f2", Examples: []string{"diff -u f1 f2  # 统一格式"}},
				{Name: "cut", Description: "剪切列", Syntax: "cut [选项] 文件", Examples: []string{"cut -d: -f1 /etc/passwd  # 取第一列"}},
			},
		},
		{
			Name: "Shell 技巧", Icon: "🐚", Role: "common",
			Commands: []Command{
				{Name: "alias", Description: "命令别名", Syntax: "alias [别名='命令']", Examples: []string{"alias ll='ls -la'  # 创建", "alias  # 列出"}},
				{Name: "export", Description: "环境变量", Syntax: "export 变量=值", Examples: []string{"export PATH=$PATH:/my/bin  # 添加路径"}},
				{Name: "history", Description: "命令历史", Syntax: "history [选项]", Examples: []string{"history  # 显示", "!!  # 上一条"}},
				{Name: "xargs", Description: "构建执行命令行", Syntax: "xargs [选项] 命令", Examples: []string{"find . -name '*.log' | xargs rm  # 批量删除"}},
				{Name: "tee", Description: "输出到文件和屏幕", Syntax: "tee [选项] 文件", Examples: []string{"command | tee output.log  # 保存+显示"}},
				{Name: "which", Description: "查找命令路径", Syntax: "which [命令]", Examples: []string{"which python  # 查找位置"}},
				{Name: "watch", Description: "周期性执行", Syntax: "watch [选项] 命令", Examples: []string{"watch -n 1 'free -h'  # 每秒监视"}},
				{Name: "shutdown", Description: "关机/重启", Syntax: "shutdown [选项] 时间", Examples: []string{"shutdown -h now  # 立即关机", "shutdown -r +5  # 5分钟后重启"}},
				{Name: "| (管道)", Description: "输出传给另一个命令", Syntax: "命令1 | 命令2", Examples: []string{"ps aux | grep nginx  # 查找进程"}},
				{Name: "> / >>", Description: "重定向到文件", Syntax: "命令 > 文件", Examples: []string{"echo 'hello' > file.txt  # 覆盖", "echo 'line2' >> file.txt  # 追加"}},
			},
		},
		{
			Name: "Docker 容器", Icon: "🐳", Role: "dev",
			Commands: []Command{
				{Name: "docker run", Description: "创建并运行容器", Syntax: "docker run [选项] 镜像", Examples: []string{"docker run -d -p 80:80 nginx  # 后台+端口", "docker run -it ubuntu bash  # 交互"}},
				{Name: "docker ps", Description: "列出容器", Syntax: "docker ps [选项]", Examples: []string{"docker ps  # 运行中", "docker ps -a  # 所有"}},
				{Name: "docker images", Description: "列出镜像", Syntax: "docker images", Examples: []string{"docker images  # 所有镜像"}},
				{Name: "docker pull", Description: "拉取镜像", Syntax: "docker pull 镜像[:标签]", Examples: []string{"docker pull ubuntu:22.04  # 指定版本"}},
				{Name: "docker build", Description: "构建镜像", Syntax: "docker build [选项] 路径", Examples: []string{"docker build -t myapp:1.0 .  # 构建"}},
				{Name: "docker exec", Description: "容器内执行", Syntax: "docker exec [选项] 容器 命令", Examples: []string{"docker exec -it container bash  # 进入容器"}},
				{Name: "docker logs", Description: "容器日志", Syntax: "docker logs [选项] 容器", Examples: []string{"docker logs -f container  # 实时跟踪"}},
				{Name: "docker-compose", Description: "多容器编排", Syntax: "docker-compose [命令]", Examples: []string{"docker-compose up -d  # 启动", "docker-compose down  # 停止"}},
			},
		},
		{
			Name: "Git 版本控制", Icon: "🔀", Role: "dev",
			Commands: []Command{
				{Name: "git init", Description: "初始化仓库", Syntax: "git init [目录]", Examples: []string{"git init  # 当前目录"}},
				{Name: "git clone", Description: "克隆远程仓库", Syntax: "git clone <仓库> [目录]", Examples: []string{"git clone https://github.com/user/repo.git  # HTTPS"}},
				{Name: "git add", Description: "添加到暂存区", Syntax: "git add [选项] 文件", Examples: []string{"git add .  # 所有变更", "git add -p  # 交互式"}},
				{Name: "git commit", Description: "提交", Syntax: "git commit [选项]", Examples: []string{"git commit -m 'msg'  # 提交", "git commit --amend  # 修改上次"}},
				{Name: "git push", Description: "推送到远程", Syntax: "git push [远程] [分支]", Examples: []string{"git push origin main  # 推送", "git push --tags  # 推送标签"}},
				{Name: "git pull", Description: "拉取更新", Syntax: "git pull [远程] [分支]", Examples: []string{"git pull origin main  # 拉取"}},
				{Name: "git branch", Description: "管理分支", Syntax: "git branch [选项]", Examples: []string{"git branch  # 列出", "git branch new-f  # 创建"}},
				{Name: "git merge", Description: "合并分支", Syntax: "git merge [选项] 分支", Examples: []string{"git merge feature  # 合并"}},
				{Name: "git log", Description: "提交历史", Syntax: "git log [选项]", Examples: []string{"git log --oneline --graph  # 图形化"}},
				{Name: "git status", Description: "仓库状态", Syntax: "git status", Examples: []string{"git status  # 变更状态"}},
				{Name: "git stash", Description: "暂存变更", Syntax: "git stash", Examples: []string{"git stash  # 暂存", "git stash pop  # 恢复"}},
				{Name: "git diff", Description: "查看差异", Syntax: "git diff [选项]", Examples: []string{"git diff  # 工作区差异"}},
			},
		},
	})
}

func (a *App) GetGuides() []GuideCategory {
	return []GuideCategory{
		{
			Name: "AI 工具安装", Icon: "🤖",
			Guides: []InstallGuide{
				{
					Name: "Ollama - 本地 LLM 运行", Description: "最流行的本地大模型运行工具", URL: "https://ollama.ai",
					Steps: []GuideStep{
						{Step: "curl -fsSL https://ollama.ai/install.sh | sh"},
						{Step: "ollama run qwen2:7b"},
						{Step: "ollama list"},
					},
					Tips: "Ollama 默认监听 11434 端口，可通过 REST API 调用",
					Note: "⚠️ 下载慢可设置代理: export HTTPS_PROXY=http://127.0.0.1:7890",
				},
				{
					Name: "CUDA + cuDNN 安装", Description: "NVIDIA GPU 加速计算平台", URL: "https://developer.nvidia.com/cuda-downloads",
					Steps: []GuideStep{
						{Step: "nvidia-smi  # 检查驱动"},
						{Step: "sudo apt install nvidia-driver-545"},
						{Step: "wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2404/x86_64/cuda-keyring_1.1-1_all.deb"},
						{Step: "sudo dpkg -i cuda-keyring_1.1-1_all.deb && sudo apt update"},
						{Step: "sudo apt install cuda"},
						{Step: "echo 'export PATH=/usr/local/cuda/bin:$PATH' >> ~/.bashrc"},
					},
					Tips: "nvidia-smi 显示的 CUDA Version 是驱动支持的最高版本",
				},
				{
					Name: "PyTorch", Description: "深度学习框架", URL: "https://pytorch.org",
					Steps: []GuideStep{
						{Step: "# GPU 版:"},
						{Step: "pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu121"},
						{Step: "# CPU 版:"},
						{Step: "pip install torch torchvision torchaudio"},
						{Step: "# 验证:"},
						{Step: "python -c \"import torch; print(torch.cuda.is_available())\""},
					},
				},
				{
					Name: "vLLM - 高性能 LLM 推理", Description: "高吞吐量 LLM 推理引擎", URL: "https://github.com/vllm-project/vllm",
					Steps: []GuideStep{
						{Step: "pip install vllm"},
						{Step: "python -m vllm.entrypoints.openai.api_server --model Qwen/Qwen2-7B-Instruct"},
					},
					Tips: "需要 CUDA 11.8+ 和至少 16GB 显存",
				},
				{
					Name: "Stable Diffusion WebUI", Description: "AI 绘画 Web 界面", URL: "https://github.com/AUTOMATIC1111/stable-diffusion-webui",
					Steps: []GuideStep{
						{Step: "git clone https://github.com/AUTOMATIC1111/stable-diffusion-webui.git"},
						{Step: "cd stable-diffusion-webui"},
						{Step: "./webui.sh --xformers"},
					},
					Tips: "首次启动自动下载依赖，默认 http://localhost:7860",
				},
				{
					Name: "ComfyUI", Description: "节点式 AI 绘画工具", URL: "https://github.com/comfyanonymous/ComfyUI",
					Steps: []GuideStep{
						{Step: "git clone https://github.com/comfyanonymous/ComfyUI.git"},
						{Step: "cd ComfyUI && pip install -r requirements.txt"},
						{Step: "python main.py --xformers"},
					},
				},
				{
					Name: "LangChain", Description: "LLM 应用开发框架", URL: "https://github.com/langchain-ai/langchain",
					Steps: []GuideStep{
						{Step: "pip install langchain langchain-community langchain-openai"},
						{Step: "pip install langchain-ollama"},
					},
				},
				{
					Name: "HuggingFace Transformers", Description: "预训练模型库", URL: "https://huggingface.co/docs/transformers",
					Steps: []GuideStep{
						{Step: "pip install transformers torch accelerate bitsandbytes"},
						{Step: "python -c \"from transformers import pipeline; pipe = pipeline('text-generation', model='Qwen/Qwen2-7B-Instruct')\""},
					},
					Tips: "设置 HF_ENDPOINT=https://hf-mirror.com 加速国内下载",
				},
				{
					Name: "OpenCV", Description: "计算机视觉库", URL: "https://opencv.org",
					Steps: []GuideStep{
						{Step: "pip install opencv-python opencv-python-headless"},
						{Step: "python -c \"import cv2; print(cv2.__version__)\""},
					},
				},
			},
		},
		{
			Name: "开发环境搭建", Icon: "🔧",
			Guides: []InstallGuide{
				{
					Name: "Miniconda", Description: "Python 环境管理", URL: "https://docs.conda.io/en/latest/miniconda.html",
					Steps: []GuideStep{
						{Step: "wget https://repo.anaconda.com/miniconda/Miniconda3-latest-Linux-x86_64.sh"},
						{Step: "bash Miniconda3-latest-Linux-x86_64.sh"},
						{Step: "conda create -n myenv python=3.12"},
					},
					Tips: "配置国内镜像: conda config --add channels https://mirrors.tuna.tsinghua.edu.cn/anaconda/pkgs/main/",
				},
				{
					Name: "Node.js (nvm)", Description: "Node.js 版本管理器", URL: "https://github.com/nvm-sh/nvm",
					Steps: []GuideStep{
						{Step: "curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.0/install.sh | bash"},
						{Step: "nvm install --lts"},
						{Step: "node --version && npm --version"},
					},
				},
				{
					Name: "Go 语言", Description: "Go 编译器安装", URL: "https://go.dev/dl/",
					Steps: []GuideStep{
						{Step: "wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz"},
						{Step: "sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz"},
						{Step: "echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc"},
					},
					Tips: "设置 GoProxy: go env -w GOPROXY=https://goproxy.cn,direct",
				},
				{
					Name: "Rust", Description: "Rust 编译器安装", URL: "https://rustup.rs",
					Steps: []GuideStep{
						{Step: "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh"},
						{Step: "rustc --version && cargo --version"},
					},
				},
				{
					Name: "Docker Engine", Description: "容器运行时", URL: "https://docs.docker.com/engine/install/",
					Steps: []GuideStep{
						{Step: "curl -fsSL https://get.docker.com | sh"},
						{Step: "sudo usermod -aG docker $USER"},
						{Step: "docker run hello-world"},
					},
				},
				{
					Name: "Jupyter Lab", Description: "交互式开发环境", URL: "https://jupyter.org",
					Steps: []GuideStep{
						{Step: "pip install jupyterlab"},
						{Step: "jupyter lab --ip=0.0.0.0 --port=8888"},
					},
				},
				{
					Name: "MySQL/MariaDB", Description: "关系型数据库", URL: "https://dev.mysql.com/downloads/mysql/",
					Steps: []GuideStep{
						{Step: "# Docker 方式 (推荐):"},
						{Step: "docker run -d -p 3306:3306 -e MYSQL_ROOT_PASSWORD=my-secret-pw mysql:8"},
					},
				},
				{
					Name: "Redis", Description: "内存数据库/缓存", URL: "https://redis.io/download",
					Steps: []GuideStep{
						{Step: "docker run -d -p 6379:6379 --name redis redis:7"},
						{Step: "redis-cli ping"},
					},
				},
			},
		},
		{
			Name: "常见问题解决", Icon: "🩹",
			Guides: []InstallGuide{
				{
					Name: "pip 安装慢/失败", Description: "加速 pip 安装", URL: "",
					Steps: []GuideStep{
						{Step: "# 临时使用镜像:"},
						{Step: "pip install -i https://pypi.tuna.tsinghua.edu.cn/simple 包名"},
						{Step: "# 永久配置:"},
						{Step: "pip config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple"},
					},
				},
				{
					Name: "CUDA 安装问题", Description: "解决驱动冲突/版本不匹配", URL: "",
					Steps: []GuideStep{
						{Step: "nvidia-smi  # 查看 CUDA 兼容版本"},
						{Step: "sudo apt purge nvidia-*  # 卸载旧驱动"},
						{Step: "sudo ubuntu-drivers autoinstall"},
					},
				},
				{
					Name: "Docker 权限问题", Description: "免 sudo 执行 docker", URL: "",
					Steps: []GuideStep{
						{Step: "sudo usermod -aG docker $USER"},
						{Step: "newgrp docker"},
					},
				},
				{
					Name: "端口被占用", Description: "释放被占用的端口", URL: "",
					Steps: []GuideStep{
						{Step: "sudo lsof -i :8080  # 查看占用"},
						{Step: "sudo fuser -k 8080/tcp  # 释放端口"},
					},
				},
				{
					Name: "磁盘空间不足", Description: "清理磁盘空间", URL: "",
					Steps: []GuideStep{
						{Step: "df -h  # 查看使用"},
						{Step: "sudo apt clean && sudo apt autoremove"},
						{Step: "sudo journalctl --vacuum-time=3d"},
						{Step: "docker system prune -a"},
					},
				},
				{
					Name: "Git 代理设置", Description: "加速 git 操作", URL: "",
					Steps: []GuideStep{
						{Step: "git config --global http.proxy http://127.0.0.1:7890"},
						{Step: "git config --global --unset http.proxy  # 取消"},
					},
				},
				{
					Name: "SSH 连接被拒绝", Description: "排查 SSH 问题", URL: "",
					Steps: []GuideStep{
						{Step: "sudo systemctl status ssh"},
						{Step: "sudo ufw allow ssh"},
						{Step: "ssh -vvv user@host  # 详细日志"},
					},
				},
				{
					Name: "系统负载过高", Description: "排查高负载问题", URL: "",
					Steps: []GuideStep{
						{Step: "top -o %CPU  # CPU 最高的进程"},
						{Step: "free -h  # 内存使用"},
						{Step: "ps aux --sort=-%mem | head -10"},
					},
				},
			},
		},
	}
}

func (a *App) ExecuteCommand(cmd string) ExecResult {
	if a.cancelExec != nil {
		a.cancelExec()
	}
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return ExecResult{Success: false, Error: "空命令"}
	}
	name := parts[0]
	args := parts[1:]
	var stdout, stderr bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), a.execTimeout)
	a.cancelExec = cancel
	defer func() {
		cancel()
		a.cancelExec = nil
	}()
	c := exec.CommandContext(ctx, name, args...)
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	out := strings.TrimSpace(stdout.String())
	errStr := strings.TrimSpace(stderr.String())
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return ExecResult{Success: false, Output: out, Error: "命令执行超时（" + a.execTimeout.String() + "）"}
		}
		if ctx.Err() == context.Canceled {
			return ExecResult{Success: false, Output: out, Error: "命令已被用户取消"}
		}
		if errStr == "" {
			errStr = err.Error()
		}
		return ExecResult{Success: false, Output: out, Error: errStr}
	}
	return ExecResult{Success: true, Output: out}
}

func (a *App) CancelExecution() bool {
	if a.cancelExec != nil {
		a.cancelExec()
		a.cancelExec = nil
		return true
	}
	return false
}

func (a *App) GetChineseSearchMap() string {
	m := map[string][]string{
		"查看": {"ls","cat","less","more","head","tail","ps","top","htop","free","df","du","uptime","neofetch","lscpu","lspci","lsusb","lsblk","ip","nmcli","ss","netstat","dmesg","journalctl","systemctl status","docker ps","docker logs","docker images","git status","git log","git diff","history","which","whoami","id","groups","blkid","pwd","uname","hostnamectl","timedatectl","nslookup","dig","traceroute","nmap","tcpdump"},
		"搜索": {"grep","find","locate","which","whereis","apt-cache search","dnf search","pacman -Ss"},
		"查找": {"grep","find","locate","which","whereis"},
		"删除": {"rm","rmdir","userdel","docker rm","git branch -d","kill","killall","pkill","fuser"},
		"复制": {"cp","scp","rsync","dd","cpio"},
		"移动": {"mv","rsync","scp"},
		"重命名": {"mv","rename"},
		"创建": {"mkdir","touch","useradd","groupadd","git init","docker run","docker create","alias","export","ln","git branch","git stash"},
		"编辑": {"vim","nano","sed","vi","emacs"},
		"安装": {"apt install","dnf install","yum install","pacman -S","pip install","npm install","snap install","flatpak install","docker pull","go install","cargo install"},
		"卸载": {"apt remove","dnf remove","yum remove","pacman -R","pip uninstall","npm uninstall","snap remove"},
		"更新": {"apt update","apt upgrade","dnf upgrade","yum update","pacman -Syu","pip install --upgrade","npm update","snap refresh"},
		"下载": {"wget","curl","apt download","git clone","docker pull"},
		"上传": {"scp","rsync","sftp","curl -T"},
		"网络": {"ping","curl","wget","ip","ss","netstat","nslookup","dig","traceroute","nmap","tcpdump","firewall-cmd","nmcli","ssh","scp","rsync","sftp","telnet","nc"},
		"进程": {"ps","top","htop","kill","killall","pkill","nohup","bg/fg","crontab","systemctl","journalctl"},
		"服务": {"systemctl","service","journalctl","systemctl start","systemctl stop","systemctl restart","systemctl enable","systemctl status"},
		"磁盘": {"df","du","fdisk","gdisk","lsblk","blkid","mount","umount","mkfs","parted","dd","fsck","smartctl","swapon/swapoff"},
		"内存": {"free","top","htop","vmstat"},
		"日志": {"journalctl","dmesg","tail -f","less","cat","docker logs","git log"},
		"权限": {"chmod","chown","chgrp","umask","sudo","su","passwd"},
		"用户": {"useradd","usermod","userdel","passwd","groupadd","groups","id","whoami","sudo","su","last"},
		"打包": {"tar","zip","unzip","gzip","gunzip","bzip2","xz","7z"},
		"解压": {"tar -xzf","unzip","gunzip","bzip2 -d","xz -d","7z x"},
		"压缩": {"tar -czf","gzip","bzip2","xz","zip","7z a"},
		"远程": {"ssh","scp","rsync","sftp","telnet","nc","ssh-keygen","ssh-copy-id","screen/tmux"},
		"容器": {"docker run","docker ps","docker images","docker pull","docker build","docker exec","docker logs","docker-compose","docker stop","docker start","docker restart"},
		"版本控制": {"git init","git clone","git add","git commit","git push","git pull","git branch","git merge","git log","git status","git stash","git diff"},
		"帮助": {"man","--help","-h","help","info","whatis"},
		"定时": {"crontab","at","sleep","watch","systemd-timer"},
		"监控": {"top","htop","watch","iotop","iftop","nmon","glances","htop"},
		"系统信息": {"uname","lscpu","free","df -h","du -sh","lsblk","dmesg","lspci","lsusb","uptime","hostnamectl","timedatectl","neofetch"},
		"文件管理": {"ls","cp","mv","rm","chmod","chown","find","grep","ln","touch","mkdir","file","cat","less","more","head","tail","vim","sed","awk","sort","wc","diff","cut"},
		"关机": {"shutdown","reboot","halt","poweroff","init 0","init 6","systemctl poweroff","systemctl reboot"},
	}
	b, _ := json.Marshal(m)
	return string(b)
}

func (a *App) FetchURL(url string) FetchResult {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return FetchResult{Success: false, Error: err.Error()}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FetchResult{Success: false, Error: err.Error()}
	}
	content := string(body)
	if len(content) > 10000 {
		content = content[:10000]
	}
	return FetchResult{Success: true, Content: content}
}

type DockerContainer struct {
	ID     string `json:"id"`
	Image  string `json:"image"`
	Status string `json:"status"`
	Name   string `json:"name"`
	Ports  string `json:"ports"`
}

func (a *App) GetDockerContainers() string {
	out := run("docker", "ps", "-a", "--format", `{"id":"{{.ID}}","image":"{{.Image}}","status":"{{.Status}}","name":"{{.Names}}","ports":"{{.Ports}}"}`)
	if out == "" {
		return "[]"
	}
	lines := strings.Split(out, "\n")
	var containers []DockerContainer
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var c DockerContainer
		if err := json.Unmarshal([]byte(line), &c); err == nil {
			containers = append(containers, c)
		}
	}
	if containers == nil {
		return "[]"
	}
	b, _ := json.Marshal(containers)
	return string(b)
}

func (a *App) DockerAction(id, action string) ExecResult {
	return runExec("docker", action, id)
}

func (a *App) DockerLogs(id string) string {
	return run("docker", "logs", "--tail", "50", id)
}

type SysStats struct {
	CPUUsage  string `json:"cpuUsage"`
	MemTotal  string `json:"memTotal"`
	MemUsed   string `json:"memUsed"`
	MemPct    string `json:"memPct"`
	DiskTotal string `json:"diskTotal"`
	DiskUsed  string `json:"diskUsed"`
	DiskPct   string `json:"diskPct"`
	Time      string `json:"time"`
}

func (a *App) GetSysStats() SysStats {
	mem := run("sh", "-c", `free -h | grep "Mem:" | awk '{print $2, $3, int($3/$2*100)"%"}'`)
	parts := strings.Fields(mem)
	disk := run("sh", "-c", `df -h / | tail -1 | awk '{print $2, $3, $5}'`)
	dparts := strings.Fields(disk)
	cpu := run("sh", "-c", `top -bn1 | grep "Cpu(s)" | awk '{print $2+$4"%"}'`)

	s := SysStats{Time: time.Now().Format("15:04:05")}
	if len(parts) >= 3 {
		s.MemTotal = parts[0]
		s.MemUsed = parts[1]
		s.MemPct = parts[2]
	}
	if len(dparts) >= 3 {
		s.DiskTotal = dparts[0]
		s.DiskUsed = dparts[1]
		s.DiskPct = dparts[2]
	}
	s.CPUUsage = strings.TrimSpace(cpu)
	if s.CPUUsage == "" {
		s.CPUUsage = "N/A"
	}
	return s
}

func (a *App) TestSSH(host, port, user, keyPath string) ExecResult {
	args := []string{"-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=5", "-p", port}
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	args = append(args, user+"@"+host, "echo", "OK")
	return runExec("ssh", args...)
}

func runExec(name string, args ...string) ExecResult {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := strings.TrimSpace(stdout.String())
	errStr := strings.TrimSpace(stderr.String())
	if err != nil {
		if errStr == "" {
			errStr = err.Error()
		}
	}
	return ExecResult{Success: err == nil, Output: output, Error: errStr}
}
