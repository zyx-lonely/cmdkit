import { GetCategories, GetGuides, ExecuteCommand, CancelExecution, GetChineseSearchMap, GetSystemInfo, FetchURL, SaveNote, GetNote, GetAllNotes, ImportNotes, GetDockerContainers, DockerAction, DockerLogs, GetSysStats, TestSSH, GetCurrentDistro, GetDistroInfo } from '../wailsjs/go/main/App.js'

let allCategories = []
let allGuides = []
let notes = {}
let activeView = 'commands'
let activeCategory = -1
let activeRole = 'all'
let currentCmdName = ''
let sysMonitorTimer = null
let dockerTimer = null

function stopAllTimers() {
  if (sysMonitorTimer) { clearInterval(sysMonitorTimer); sysMonitorTimer = null }
  if (dockerTimer) { clearInterval(dockerTimer); dockerTimer = null }
}

function load(key, def) {
  try { return JSON.parse(localStorage.getItem(key)) ?? def } catch { return def }
}
function save(key, val) { localStorage.setItem(key, JSON.stringify(val)) }

// state
let execHistory = load('execHistory', []).map(e => typeof e === 'string' ? { cmd: e, time: Date.now() } : e)
let favs = load('favs', [])
let sshHosts = load('sshHosts', [])
let aliases = load('aliases', [])
let themeColor = load('themeColor', '')
let lastSearchQuery = ''

let flatCommands = []
let searchTimer = null

const DANGEROUS_CMDS = ['rm -rf', 'mkfs', 'dd if=', 'shutdown', 'halt', 'reboot', 'poweroff', 'init 0', 'init 6', ':(){ :|:& };:', 'mv /', 'chmod -R 000', '> /dev/sda', 'format', 'fdisk', 'parted']

const ZH_SEARCH = {
  '查看': ['ls','cat','less','more','head','tail','ps','top','htop','free','df','du','uptime','neofetch','lscpu','lspci','lsusb','lsblk','ip','nmcli','ss','netstat','dmesg','journalctl','systemctl','docker ps','docker logs','docker images','git status','git log','git diff','history','which','whoami','id','groups','blkid','pwd','uname','hostnamectl','timedatectl','nslookup','dig','traceroute','nmap','tcpdump'],
  '搜索': ['grep','find','locate','which','whereis'],
  '查找': ['grep','find','locate','which','whereis'],
  '删除': ['rm','rmdir','userdel','docker rm','kill','killall','pkill','fuser'],
  '复制': ['cp','scp','rsync','dd'],
  '移动': ['mv','rsync','scp'],
  '重命名': ['mv','rename'],
  '创建': ['mkdir','touch','useradd','groupadd','git init','docker run','alias','export','ln','git branch','git stash'],
  '编辑': ['vim','nano','sed','vi','emacs'],
  '安装': ['apt','dnf','yum','pacman','pip install','npm install','snap install','flatpak install','docker pull'],
  '卸载': ['apt remove','dnf remove','yum remove','pacman -R','pip uninstall','npm uninstall','snap remove'],
  '更新': ['apt update','apt upgrade','dnf upgrade','yum update','pacman -Syu','npm update','snap refresh'],
  '下载': ['wget','curl','git clone','docker pull'],
  '上传': ['scp','rsync','sftp'],
  '网络': ['ping','curl','wget','ip','ss','netstat','nslookup','dig','traceroute','nmap','tcpdump','firewall-cmd','nmcli','ssh','scp','rsync','sftp','telnet','nc'],
  '进程': ['ps','top','htop','kill','killall','pkill','nohup','crontab','systemctl','journalctl'],
  '服务': ['systemctl','service','journalctl'],
  '磁盘': ['df','du','fdisk','gdisk','lsblk','blkid','mount','umount','mkfs','parted','dd','fsck','smartctl'],
  '内存': ['free','top','htop','vmstat'],
  '日志': ['journalctl','dmesg','tail','docker logs','git log'],
  '权限': ['chmod','chown','chgrp','sudo','su','passwd'],
  '用户': ['useradd','usermod','userdel','passwd','groupadd','groups','id','whoami','sudo','su','last'],
  '打包': ['tar','zip','unzip','gzip','gunzip','bzip2','xz','7z'],
  '解压': ['tar','unzip','gunzip','bzip2 -d','xz -d','7z x'],
  '压缩': ['tar','gzip','bzip2','xz','zip'],
  '远程': ['ssh','scp','rsync','sftp','telnet','nc','ssh-keygen','ssh-copy-id'],
  '容器': ['docker','docker-compose'],
  '版本控制': ['git'],
  '帮助': ['man','help','info','whatis'],
  '定时': ['crontab','at','sleep','watch'],
  '监控': ['top','htop','watch','iotop','iftop','nmon'],
  '系统信息': ['uname','lscpu','free','df','du','lsblk','dmesg','lspci','lsusb','uptime','hostnamectl','timedatectl','neofetch'],
  '文件管理': ['ls','cp','mv','rm','chmod','chown','find','grep','ln','touch','mkdir','file','cat','less','head','tail'],
  '关机重启': ['shutdown','reboot','halt','poweroff','init 0','init 6'],
}

let currentDistro = ''
let distroInfo = {}

const PLATFORM_LABELS = {
  debian: { label: 'Debian/Ubuntu', icon: '🐧', color: '#e95420' },
  rhel: { label: 'RHEL/CentOS', icon: '🎩', color: '#ee0000' },
  fedora: { label: 'Fedora', icon: '💙', color: '#294172' },
  arch: { label: 'Arch Linux', icon: '🚀', color: '#1793d1' },
  suse: { label: 'openSUSE', icon: '🦎', color: '#73ba25' },
  alpine: { label: 'Alpine', icon: '🏔️', color: '#0d597f' },
  '*': { label: '通用', icon: '💻', color: '#666688' },
}

const DISTRO_ICONS = {
  debian: '🐧', rhel: '🎩', fedora: '💙', arch: '🚀', suse: '🦎', alpine: '🏔️',
}

const PM_COMPARE = {
  title: '跨发行版包管理速查',
  rows: [
    { op: '更新源', debian: 'apt update', rhel: 'yum check-update', fedora: 'dnf check-update', arch: 'pacman -Sy', suse: 'zypper refresh' },
    { op: '安装软件', debian: 'apt install 包名', rhel: 'yum install 包名', fedora: 'dnf install 包名', arch: 'pacman -S 包名', suse: 'zypper install 包名' },
    { op: '卸载软件', debian: 'apt remove 包名', rhel: 'yum remove 包名', fedora: 'dnf remove 包名', arch: 'pacman -R 包名', suse: 'zypper remove 包名' },
    { op: '搜索软件', debian: 'apt search 关键词', rhel: 'yum search 关键词', fedora: 'dnf search 关键词', arch: 'pacman -Ss 关键词', suse: 'zypper search 关键词' },
    { op: '升级所有', debian: 'apt upgrade', rhel: 'yum update', fedora: 'dnf upgrade', arch: 'pacman -Syu', suse: 'zypper update' },
    { op: '查看已安装', debian: 'apt list --installed', rhel: 'yum list installed', fedora: 'dnf list installed', arch: 'pacman -Q', suse: 'zypper se --installed-only' },
    { op: '清理缓存', debian: 'apt clean', rhel: 'yum clean all', fedora: 'dnf clean all', arch: 'pacman -Sc', suse: 'zypper clean' },
    { op: '查看包信息', debian: 'apt show 包名', rhel: 'yum info 包名', fedora: 'dnf info 包名', arch: 'pacman -Qi 包名', suse: 'zypper info 包名' },
    { op: '安装本地包', debian: 'dpkg -i file.deb', rhel: 'rpm -ivh file.rpm', fedora: 'rpm -ivh file.rpm', arch: 'pacman -U file.pkg.tar.zst', suse: 'rpm -ivh file.rpm' },
  ],
}

const DIFFICULTY_ICONS = {
  beginner: { icon: '⭐', label: '入门', color: '#4caf50' },
  intermediate: { icon: '⭐⭐', label: '进阶', color: '#ff9800' },
  advanced: { icon: '⭐⭐⭐', label: '高级', color: '#f44336' },
}

// DOM refs
const $ = s => document.querySelector(s)
const $$ = s => document.querySelectorAll(s)

window.addEventListener('unhandledrejection', (e) => {
  toast('❌ ' + (e.reason?.message || '发生未知错误'))
})

window.addEventListener('DOMContentLoaded', async () => {
  allCategories = await GetCategories()
  allGuides = await GetGuides()
  flatCommands = allCategories.flatMap(c => c.commands.map(cmd => ({ ...cmd, catName: c.name })))
  const raw = await GetAllNotes()
  try { notes = JSON.parse(raw || '{}') } catch { notes = {} }
  if (!notes) notes = {}
  currentDistro = await GetCurrentDistro()
  distroInfo = await GetDistroInfo()
  applyTheme()
  applyFontSize()
  applyThemeColor()
  renderUI()
  bindEvents()
  updateFavCount()
  updateDistroBadge()
  showOnboarding()
})

function applyTheme() {
  const t = localStorage.getItem('theme') || 'dark'
  document.documentElement.setAttribute('data-theme', t)
  $('#theme-toggle').textContent = t === 'dark' ? '🌙' : '☀️'
}
function applyFontSize() {
  document.documentElement.setAttribute('data-size', localStorage.getItem('fontSize') || 'md')
  $('#font-size-select').value = localStorage.getItem('fontSize') || 'md'
}
function applyThemeColor() {
  const c = themeColor || '#7c5cfc'
  document.documentElement.style.setProperty('--accent', c)
  const h = c + '20'
  document.documentElement.style.setProperty('--accent-l', h)
}

// --- Render ---
function renderUI() {
  renderCategories()
  renderView()
}
function renderCategories() {
  const list = $('#category-list')
  const filtered = activeRole === 'all' ? allCategories : allCategories.filter(c => c.role === activeRole || !c.role)
  let html = ''
  filtered.forEach((cat, idx) => {
    const origIdx = allCategories.indexOf(cat)
    const count = cat.commands.length
    const active = activeCategory === origIdx ? 'active' : ''
    const roleLabel = cat.role === 'dev' ? '💻' : cat.role === 'ops' ? '⚙️' : cat.role === 'common' ? '📋' : ''
    html += `<li><button class="${active}" data-idx="${origIdx}">${roleLabel} ${cat.name} <span class="cat-count">${count}</span></button></li>`
  })
  list.innerHTML = html
  list.querySelectorAll('button').forEach(btn => {
    btn.addEventListener('click', () => {
      activeCategory = parseInt(btn.dataset.idx)
      activeView = 'commands'
      $('#btn-all').classList.add('active')
      $('#btn-guides').classList.remove('active')
      $('#btn-sysinfo').classList.remove('active')
      $('#btn-docker').classList.remove('active')
      $('#btn-ssh').classList.remove('active')
      renderView()
    })
  })
}
function renderView() {
  hideAllViews()
  const views = {
    commands: renderCommands,
    guides: renderGuides,
    sysinfo: renderSysInfo,
    docker: renderDocker,
    ssh: renderSSH,
    aliases: renderAliases,
  }
  if (views[activeView]) views[activeView]()
  updateButtonStates()
}
function hideAllViews() {
  ;['commands-grid','guides-container','sysinfo-container','docker-container','ssh-container','aliases-container'].forEach(id => {
    document.getElementById(id).style.display = 'none'
  })
}

function updateButtonStates() {
  $$('#category-actions .cat-btn').forEach(b => b.classList.remove('active'))
  if (activeView === 'commands') {
    const btnMap = { 'all': '#btn-all', 'guides': '#btn-guides', 'sysinfo': '#btn-sysinfo' }
    if (btnMap[activeView]) $(btnMap[activeView]).classList.add('active')
    else $('#btn-all').classList.add('active')
  } else if (activeView === 'guides') $('#btn-guides').classList.add('active')
  else if (activeView === 'sysinfo') $('#btn-sysinfo').classList.add('active')
  $('#btn-docker').classList.toggle('active', activeView === 'docker')
  $('#btn-ssh').classList.toggle('active', activeView === 'ssh')
  $$('#category-list button').forEach(b => b.classList.toggle('active', parseInt(b.dataset.idx) === activeCategory))
}

// --- Commands ---
function renderCommands() {
  const grid = $('#commands-grid')
  grid.style.display = 'grid'
  $('#view-title').textContent = activeCategory === -1 ? '全部命令' : allCategories[activeCategory].name
  let cmds = activeCategory === -1
    ? flatCommands.slice()
    : allCategories[activeCategory].commands.map(cmd => ({ ...cmd, catName: allCategories[activeCategory].name }))
  cmds = filterByRoleCommands(cmds)
  cmds = filterByDistro(cmds)
  const q = $('#search-input').value.trim().toLowerCase()
  lastSearchQuery = q
  if (q) {
    let matched = cmds.filter(c =>
      c.name.toLowerCase().includes(q) ||
      (c.desc || '').toLowerCase().includes(q) ||
      (c.syntax || '').toLowerCase().includes(q) ||
      (c.examples || []).some(e => e.toLowerCase().includes(q))
    )
    if (!matched.length) {
      const zhSet = new Set()
      for (const [zh, cmdsList] of Object.entries(ZH_SEARCH)) {
        if (zh.includes(q) || q.includes(zh)) cmdsList.forEach(c => zhSet.add(c))
      }
      if (zhSet.size) matched = cmds.filter(c => zhSet.has(c.name))
    }
    cmds = matched
  }
  $('#view-count').textContent = cmds.length + ' 条'
  renderCommandCards(grid, cmds, '🔍')
}

function renderCommandCards(grid, cmds, emptyIcon) {
  if (!cmds.length) {
    grid.innerHTML = `<div class="empty-state"><div class="icon">${emptyIcon || '📋'}</div><p>没有找到匹配的命令</p></div>`
    return
  }
  let html = ''
  cmds.forEach(cmd => {
    const f = favs.includes(cmd.name)
    const note = notes[cmd.name] ? notes[cmd.name].substring(0, 60) + (notes[cmd.name].length > 60 ? '...' : '') : ''
    const diff = DIFFICULTY_ICONS[cmd.difficulty] || DIFFICULTY_ICONS.intermediate
    const related = cmd.related && cmd.related.length ? cmd.related.slice(0, 4) : []
    const platBadge = getPlatformBadge(cmd)
    const isMatch = !cmd.platforms || cmd.platforms.includes('*') || cmd.platforms.includes(currentDistro)
    const hlq = lastSearchQuery
    const hlName = hlq ? highlightText(cmd.name, hlq) : cmd.name
    const hlDesc = hlq && cmd.desc ? highlightText(cmd.desc, hlq) : cmd.desc
    html += `<div class="cmd-card ${isMatch ? '' : 'cmd-foreign'}">
      <div class="cmd-header">
        <span class="cmd-name" data-cmd="${cmd.name}" data-syntax="${cmd.syntax || ''}">${hlName}</span>
        <div class="cmd-actions">
          ${cmd.syntax ? `<button class="cmd-btn run" data-cmd="${cmd.name}" data-syntax="${cmd.syntax}">▶ 执行</button>` : ''}
          <button class="cmd-btn note-btn" data-cmd="${cmd.name}">📝</button>
          <button class="cmd-btn ${f ? 'faved' : ''}" data-cmd="${cmd.name}">${f ? '★' : '☆'}</button>
        </div>
      </div>
      <div class="cmd-meta">
        <span class="diff-badge" style="background:${diff.color}22;color:${diff.color}">${diff.icon} ${diff.label}</span>
        ${platBadge}
        ${cmd.scenario ? `<span class="cmd-scenario">💡 ${cmd.scenario}</span>` : ''}
      </div>
      ${hlDesc ? `<div class="cmd-desc">${hlDesc}</div>` : ''}
      ${cmd.syntax ? `<div class="cmd-syntax">$ ${cmd.syntax}</div>` : ''}
      ${cmd.examples && cmd.examples.length ? `<div class="cmd-examples">${cmd.examples.map(e => `<div class="cmd-example" title="点击复制">$ ${e}</div>`).join('')}</div>` : ''}
      ${cmd.altFor ? `<div class="cmd-altnote">💡 ${cmd.altFor}</div>` : ''}
      ${related.length ? `<div class="cmd-related">🔗 相关: ${related.map(r => `<span class="related-tag" data-cmd="${r}">${r}</span>`).join(' ')}</div>` : ''}
      ${note ? `<div class="cmd-note-preview">📌 ${note}</div>` : ''}
    </div>`
  })
  grid.innerHTML = html
  grid.querySelectorAll('.cmd-name').forEach(el => {
    el.addEventListener('click', () => {
      const syntax = el.dataset.syntax
      if (syntax) openExec(syntax)
    })
  })
  grid.querySelectorAll('.cmd-btn.run').forEach(el => {
    el.addEventListener('click', () => openExec(el.dataset.syntax))
  })
  grid.querySelectorAll('.note-btn').forEach(el => {
    el.addEventListener('click', () => openNote(el.dataset.cmd))
  })
  grid.querySelectorAll('.cmd-btn.faved, .cmd-btn:not(.run):not(.note-btn)').forEach(el => {
    if (el.classList.contains('run') || el.classList.contains('note-btn')) return
    el.addEventListener('click', () => toggleFav(el.dataset.cmd))
  })
  grid.querySelectorAll('.cmd-example').forEach(el => {
    el.addEventListener('click', () => copyText(el.textContent.trim().replace(/^\$\s*/, '')))
  })
  grid.querySelectorAll('.related-tag').forEach(el => {
    el.addEventListener('click', () => {
      const name = el.dataset.cmd
      const found = flatCommands.find(c => c.name === name)
      if (found) {
        activeView = 'commands'
        searchFor(name)
      }
    })
  })
}

function searchFor(q) {
  const inp = $('#search-input')
  inp.value = q
  if (activeView === 'commands') renderCommands()
}

function highlightText(text, query) {
  if (!query || !text) return text || ''
  const idx = text.toLowerCase().indexOf(query.toLowerCase())
  if (idx === -1) return text
  return text.slice(0, idx) + '<mark>' + text.slice(idx, idx + query.length) + '</mark>' + text.slice(idx + query.length)
}

function filterByRoleCommands(cmds) {
  if (activeRole === 'all') return cmds
  const catsWithRole = allCategories.filter(c => c.role === activeRole)
  const cmdNames = new Set(catsWithRole.flatMap(c => c.commands.map(cmd => cmd.name)))
  return cmds.filter(c => cmdNames.has(c.name))
}

// --- Guides ---
async function renderGuides() {
  const container = $('#guides-container')
  container.style.display = 'block'
  $('#view-title').textContent = '安装指南 & 环境配置'
  const all = await GetGuides()
  let html = '<details id="fetch-section"><summary>🌐 URL 内容获取</summary><div id="fetch-body"><input id="fetch-url" placeholder="输入 URL 抓取文档..."/><div id="fetch-actions"><button id="fetch-run">获取</button></div><div id="fetch-result"></div></div></details>'
  all.forEach(g => {
    html += `<div class="guide-card">
      <div class="guide-header"><span class="guide-title">${g.name}</span></div>
      ${g.description ? `<div class="guide-subtitle">${g.description}</div>` : ''}
      <div class="guide-steps">${(g.steps || []).map(s => `<div class="guide-step">${s}</div>`).join('')}</div>
      ${g.tips ? `<div class="guide-tip">💡 ${g.tips}</div>` : ''}
      ${g.note ? `<div class="guide-note">⚠️ ${g.note}</div>` : ''}
    </div>`
  })
  container.innerHTML = html
  bindGuideEvents(container)
}
function bindGuideEvents(container) {
  const fetchRun = container.querySelector('#fetch-run')
  const fetchUrl = container.querySelector('#fetch-url')
  const fetchResult = container.querySelector('#fetch-result')
  if (fetchRun) {
    fetchRun.addEventListener('click', async () => {
      const url = fetchUrl.value.trim()
      if (!url) return toast('请输入 URL')
      fetchRun.disabled = true; fetchRun.textContent = '获取中...'
      const res = await FetchURL(url)
      fetchResult.textContent = res.success ? res.content : '❌ ' + res.error
      fetchRun.disabled = false; fetchRun.textContent = '获取'
    })
  }
  container.querySelectorAll('.guide-step').forEach(el => {
    el.addEventListener('click', () => copyText(el.textContent.trim()))
  })
}

// --- SysInfo ---
async function renderSysInfo() {
  const container = $('#sysinfo-container')
  container.style.display = 'grid'
  $('#view-title').textContent = '系统信息'
  try {
    const info = await GetSystemInfo()
    const labels = { os:'系统', hostname:'主机名', kernel:'内核', cpu:'CPU型号', cores:'核心数', memory:'内存', disk:'磁盘', uptime:'运行时间', goVersion:'Go版本', shell:'Shell', desktop:'桌面环境' }
    let html = Object.entries(info).map(([k, v]) =>
      `<div class="sys-card"><h3>${labels[k] || k}</h3><div class="val">${v || 'N/A'}</div></div>`
    ).join('')
    container.innerHTML = html + '<div class="sys-card" style="grid-column:1/-1"><h3>实时监控</h3><div id="monitor-box"><button id="start-monitor">启动实时监控</button></div></div>'
  } catch (e) {
    container.innerHTML = `<div class="empty-box"><div class="icon">🖥️</div><p>获取系统信息失败</p><p style="font-size:var(--fs-s)">${e.message || '未知错误'}</p></div>`
  }
  bindMonitorEvents()
}

// --- Monitoring ---
function bindMonitorEvents() {
  const btn = $('#start-monitor')
  if (!btn) return
  btn.addEventListener('click', async () => {
    if (sysMonitorTimer) {
      clearInterval(sysMonitorTimer); sysMonitorTimer = null
      btn.textContent = '启动实时监控'; btn.style.background = ''
      $('#monitor-stats')?.remove()
      return
    }
    btn.textContent = '⏹️ 停止监控'; btn.style.background = 'var(--accent)'; btn.style.color = '#fff'
    const box = $('#monitor-box')
    let statDiv = document.createElement('div')
    statDiv.id = 'monitor-stats'
    statDiv.style.cssText = 'display:grid;grid-template-columns:1fr 1fr 1fr;gap:10px;margin-top:10px;font-size:var(--fs-s)'
    box.appendChild(statDiv)
    const update = async () => {
      const s = await GetSysStats()
      statDiv.innerHTML = `<div>CPU: <strong>${s.cpuUsage}</strong></div><div>内存: <strong>${s.memUsed}</strong> / ${s.memTotal} (${s.memPct})</div><div>磁盘: <strong>${s.diskUsed}</strong> / ${s.diskTotal} (${s.diskPct})</div>`
    }
    await update()
    sysMonitorTimer = setInterval(update, 3000)
  })
}

// --- Docker ---
async function renderDocker() {
  const container = $('#docker-container')
  container.style.display = 'block'
  $('#view-title').textContent = '🐳 Docker 管理'
  if (dockerTimer) { clearInterval(dockerTimer); dockerTimer = null }
  container.innerHTML = '<div style="text-align:center;padding:40px;color:var(--fg2)">加载中...</div>'
  await renderDockerTable(container)
  let paused = false
  const pauseBtn = document.createElement('button')
  pauseBtn.className = 'cat-btn'
  pauseBtn.style.cssText = 'display:inline-block;width:auto;padding:4px 12px;margin-bottom:8px;background:var(--bg3);border:1px solid var(--border)'
  pauseBtn.textContent = '⏸ 暂停刷新'
  pauseBtn.addEventListener('click', () => {
    paused = !paused
    pauseBtn.textContent = paused ? '▶ 恢复刷新' : '⏸ 暂停刷新'
    if (paused) { clearInterval(dockerTimer); dockerTimer = null }
    else dockerTimer = setInterval(() => renderDockerTable(container, true), 5000)
  })
  container.insertBefore(pauseBtn, container.firstChild)
  dockerTimer = setInterval(() => renderDockerTable(container, true), 5000)
}
async function renderDockerTable(container, silent) {
  if (!silent) container = container || $('#docker-container')
  let list = []
  try {
    const raw = await GetDockerContainers()
    list = JSON.parse(raw)
  } catch (e) {
    if (!silent) container.innerHTML = `<div class="empty-box"><div class="icon">🐳</div><p>Docker 连接失败</p><p style="font-size:var(--fs-s);margin-top:4px">${e.message || '请确认 Docker 已安装并正在运行'}</p></div>`
    return
  }
  if (!list || !list.length) {
    container.innerHTML = '<div class="empty-box"><div class="icon">🐳</div><p>没有 Docker 容器</p><p style="font-size:var(--fs-s);margin-top:4px">运行 docker run hello-world 创建第一个容器</p></div>'
    return
  }
  let html = `<table class="docker-table">
    <thead><tr><th>容器名</th><th>镜像</th><th>状态</th><th>端口</th><th>操作</th><th>日志</th></tr></thead><tbody>`
  list.forEach(c => {
    const statusCls = c.status.startsWith('Up') ? 'running' : c.status.startsWith('Exited') ? 'exited' : 'paused'
    html += `<tr>
      <td><strong>${c.name}</strong></td>
      <td>${c.image}</td>
      <td><span class="docker-status ${statusCls}">${c.status}</span></td>
      <td style="font-size:11px">${c.ports || '-'}</td>
      <td style="white-space:nowrap">
        <button class="cmd-btn docker-act" data-id="${c.id}" data-act="start">▶ 启动</button>
        <button class="cmd-btn docker-act" data-id="${c.id}" data-act="stop">⏹ 停止</button>
        <button class="cmd-btn docker-act" data-id="${c.id}" data-act="restart">🔄 重启</button>
      </td>
      <td><button class="cmd-btn docker-logs" data-id="${c.id}">日志</button></td>
    </tr>`
  })
  html += '</tbody></table>'
  container.innerHTML = html
  container.querySelectorAll('.docker-act').forEach(el => {
    el.addEventListener('click', async () => {
      const res = await DockerAction(el.dataset.id, el.dataset.act)
      toast('Docker ' + el.dataset.act + ': ' + (res.success ? '成功' : '失败'))
      await renderDockerTable(container, true)
    })
  })
  container.querySelectorAll('.docker-logs').forEach(el => {
    el.addEventListener('click', async () => {
      const logs = await DockerLogs(el.dataset.id)
      openExec(logs || '无日志')
    })
  })
}

// --- SSH ---
function renderSSH() {
  const container = $('#ssh-container')
  container.style.display = 'block'
  $('#view-title').textContent = '🔗 SSH 主机管理'
  let html = '<div style="margin-bottom:12px"><button id="ssh-add" class="cmd-btn" style="background:var(--accent);color:#fff;border-color:var(--accent)">➕ 添加主机</button></div>'
  if (!sshHosts.length) {
    html += '<div class="empty-box"><div class="icon">🔗</div><p>没有 SSH 主机</p><p style="font-size:var(--fs-s);margin-top:4px">点击「添加主机」开始管理</p></div>'
  } else {
    html += `<table class="ssh-table"><thead><tr><th>名称</th><th>主机</th><th>端口</th><th>用户</th><th>操作</th></tr></thead><tbody>`
    sshHosts.forEach((h, i) => {
      html += `<tr>
        <td><strong>${h.name}</strong></td>
        <td style="font-family:monospace">${h.host}</td>
        <td>${h.port || 22}</td>
        <td>${h.user}</td>
        <td style="white-space:nowrap">
          <button class="cmd-btn ssh-connect" data-idx="${i}">🔌 测试</button>
          <button class="cmd-btn ssh-edit" data-idx="${i}">✏️</button>
        </td>
      </tr>`
    })
    html += '</tbody></table>'
  }
  container.innerHTML = html
  container.querySelector('#ssh-add').addEventListener('click', () => openSSHEdit(-1))
  container.querySelectorAll('.ssh-connect').forEach(el => {
    el.addEventListener('click', async () => {
      const h = sshHosts[parseInt(el.dataset.idx)]
      el.textContent = '测试中...'; el.disabled = true
      const res = await TestSSH(h.host, String(h.port || 22), h.user, h.keyPath || '')
      toast(res.success ? `✅ ${h.name} 连接成功` : `❌ ${h.name} ${res.error || '连接失败'}`)
      el.textContent = '🔌 测试'; el.disabled = false
    })
  })
  container.querySelectorAll('.ssh-edit').forEach(el => {
    el.addEventListener('click', () => openSSHEdit(parseInt(el.dataset.idx)))
  })
}

// --- Aliases ---
function renderAliases() {
  const container = $('#aliases-container')
  container.style.display = 'block'
  $('#view-title').textContent = '📌 自定义别名'
  let html = '<div style="margin-bottom:12px"><button id="alias-add" class="cmd-btn" style="background:var(--accent);color:#fff;border-color:var(--accent)">➕ 添加别名</button></div>'
  if (!aliases.length) {
    html += '<div class="empty-box"><div class="icon">📌</div><p>没有别名</p><p style="font-size:var(--fs-s);margin-top:4px">添加常用命令的快捷别名</p></div>'
  } else {
    html += `<table class="ssh-table"><thead><tr><th>别名</th><th>命令</th><th>操作</th></tr></thead><tbody>`
    aliases.forEach((a, i) => {
      html += `<tr>
        <td><strong style="color:var(--accent);font-family:monospace">${a.name}</strong></td>
        <td style="font-family:monospace">${a.cmd}</td>
        <td style="white-space:nowrap">
          <button class="cmd-btn alias-run" data-idx="${i}">▶ 执行</button>
          <button class="cmd-btn alias-edit" data-idx="${i}">✏️</button>
        </td>
      </tr>`
    })
    html += '</tbody></table>'
  }
  container.innerHTML = html
  container.querySelector('#alias-add').addEventListener('click', () => openAliasEdit(-1))
  container.querySelectorAll('.alias-run').forEach(el => {
    el.addEventListener('click', () => {
      const a = aliases[parseInt(el.dataset.idx)]
      openExec(a.cmd)
    })
  })
  container.querySelectorAll('.alias-edit').forEach(el => {
    el.addEventListener('click', () => openAliasEdit(parseInt(el.dataset.idx)))
  })
}

// --- Exec ---
function openExec(content) {
  $('#exec-overlay').classList.add('show')
  const inp = $('#exec-input')
  inp.value = content
  inp.focus()
  inp.setSelectionRange(content.length, content.length)
  $('#exec-output').textContent = '点击「执行」或按 Enter'
  renderExecHistory()
}
function renderExecHistory() {
  const drop = $('#exec-history-drop')
  if (!execHistory.length || !$('#exec-overlay').classList.contains('show')) {
    drop.style.display = 'none'
    return
  }
  drop.style.display = 'block'
  let html = `<div style="padding:6px 10px;border-bottom:1px solid var(--border);display:flex;justify-content:space-between;font-size:10px;color:var(--fg2)"><span>历史记录</span><button id="exec-history-clear" style="background:none;border:none;color:var(--fg2);cursor:pointer;font-size:10px;text-decoration:underline">清空</button></div>`
  html += execHistory.map((item, i) => {
    const timeStr = item.time ? formatTime(item.time) : ''
    return `<div class="sug-item" data-idx="${i}"><span><span class="sug-name">$ ${item.cmd}</span> <span class="sug-time">${timeStr}</span></span></div>`
  }).join('')
  drop.innerHTML = html
  drop.querySelectorAll('.sug-item').forEach(el => {
    el.addEventListener('click', () => {
      const item = execHistory[parseInt(el.dataset.idx)]
      $('#exec-input').value = item.cmd
      drop.style.display = 'none'
      $('#exec-run').focus()
    })
  })
  const clearBtn = drop.querySelector('#exec-history-clear')
  if (clearBtn) clearBtn.addEventListener('click', (e) => { e.stopPropagation(); clearExecHistory() })
}
function addExecHistory(cmd) {
  execHistory = execHistory.filter(h => h.cmd !== cmd)
  execHistory.unshift({ cmd, time: Date.now() })
  if (execHistory.length > 50) execHistory = execHistory.slice(0, 50)
  save('execHistory', execHistory)
}

function clearExecHistory() {
  execHistory = []
  save('execHistory', execHistory)
  renderExecHistory()
  toast('执行历史已清空')
}

function formatTime(ts) {
  const d = new Date(ts)
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

// --- Favorites ---
function toggleFav(name) {
  const idx = favs.indexOf(name)
  if (idx >= 0) { favs.splice(idx, 1) } else { favs.push(name) }
  save('favs', favs)
  updateFavCount()
  renderCommands()
}
function clearAllFavs() {
  if (!favs.length) return
  if (!confirm('确定清空所有收藏？')) return
  favs = []
  save('favs', favs)
  updateFavCount()
  toast('已清空所有收藏')
  if (activeView === 'commands') renderCommands()
}
function updateDistroBadge() {
  const el = $('#distro-badge')
  if (!el) return
  const icon = DISTRO_ICONS[currentDistro] || '🐧'
  const name = distroInfo.pretty || distroInfo.name || currentDistro
  el.innerHTML = `${icon} ${name}`
}
function getPlatformBadge(cmd) {
  if (!cmd.platforms || cmd.platforms.length === 0 || (cmd.platforms.length === 1 && cmd.platforms[0] === '*')) return ''
  const icons = cmd.platforms.map(p => {
    const info = PLATFORM_LABELS[p]
    return info ? info.icon : '💻'
  })
  return `<span class="platform-badge" title="适用于 ${cmd.platforms.join(', ')}">${icons.join(' ')}</span>`
}
function filterByDistro(cmds) {
  const filter = $('#distro-filter')?.value || currentDistro
  if (filter === 'all') return cmds
  const distroKey = filter
  return cmds.filter(c => {
    if (!c.platforms || c.platforms.length === 0) return true
    return c.platforms.includes(distroKey) || c.platforms.includes('*')
  })
}

function updateFavCount() {
  $('#fav-count').textContent = favs.length
}

// --- Notes ---
async function openNote(name) {
  currentCmdName = name
  $('#note-overlay').classList.add('show')
  $('#note-cmd-name').textContent = name
  let content = notes[name] || ''
  const note = await GetNote(name)
  if (note) { notes[name] = note; content = note }
  $('#note-textarea').value = content
}
async function saveNoteHandler() {
  const name = currentCmdName
  const content = $('#note-textarea').value.trim()
  if (!name) return
  if (!content) {
    delete notes[name]
    await SaveNote(name, '')
  } else {
    notes[name] = content
    await SaveNote(name, content)
  }
  $('#note-overlay').classList.remove('show')
  renderCommands()
  toast('笔记已保存')
}

// --- SSH Edit ---
function openSSHEdit(idx) {
  const overlay = $('#sshedit-overlay')
  overlay.classList.add('show')
  if (idx < 0) {
    $('#sshedit-title').textContent = '添加 SSH 主机'
    $('#sshedit-name').value = ''
    $('#sshedit-host').value = ''
    $('#sshedit-port').value = '22'
    $('#sshedit-user').value = 'root'
    $('#sshedit-key').value = ''
    $('#sshedit-delete').style.display = 'none'
    overlay.dataset.idx = '-1'
  } else {
    const h = sshHosts[idx]
    $('#sshedit-title').textContent = '编辑 SSH 主机'
    $('#sshedit-name').value = h.name
    $('#sshedit-host').value = h.host
    $('#sshedit-port').value = String(h.port || 22)
    $('#sshedit-user').value = h.user
    $('#sshedit-key').value = h.keyPath || ''
    $('#sshedit-delete').style.display = 'inline-block'
    overlay.dataset.idx = String(idx)
  }
}
function saveSSH() {
  const idx = parseInt($('#sshedit-overlay').dataset.idx)
  const obj = {
    name: $('#sshedit-name').value.trim(),
    host: $('#sshedit-host').value.trim(),
    port: parseInt($('#sshedit-port').value) || 22,
    user: $('#sshedit-user').value.trim() || 'root',
    keyPath: $('#sshedit-key').value.trim()
  }
  if (!obj.name || !obj.host) return toast('请填写名称和主机地址')
  if (idx < 0) sshHosts.push(obj)
  else sshHosts[idx] = obj
  save('sshHosts', sshHosts)
  $('#sshedit-overlay').classList.remove('show')
  renderSSH()
  toast('SSH 主机已保存')
}
function deleteSSH() {
  const idx = parseInt($('#sshedit-overlay').dataset.idx)
  if (idx >= 0) sshHosts.splice(idx, 1)
  save('sshHosts', sshHosts)
  $('#sshedit-overlay').classList.remove('show')
  renderSSH()
  toast('SSH 主机已删除')
}

// --- Alias Edit ---
function openAliasEdit(idx) {
  const overlay = $('#alias-overlay')
  overlay.classList.add('show')
  if (idx < 0) {
    $('#alias-name').value = ''
    $('#alias-cmd').value = ''
    $('#alias-delete').style.display = 'none'
    overlay.dataset.idx = '-1'
  } else {
    const a = aliases[idx]
    $('#alias-name').value = a.name
    $('#alias-cmd').value = a.cmd
    $('#alias-delete').style.display = 'inline-block'
    overlay.dataset.idx = String(idx)
  }
}
function saveAlias() {
  const idx = parseInt($('#alias-overlay').dataset.idx)
  const name = $('#alias-name').value.trim()
  const cmd = $('#alias-cmd').value.trim()
  if (!name || !cmd) return toast('请填写别名和命令')
  if (idx < 0) aliases.push({ name, cmd })
  else aliases[idx] = { name, cmd }
  save('aliases', aliases)
  $('#alias-overlay').classList.remove('show')
  renderAliases()
  toast('别名已保存')
}
function deleteAlias() {
  const idx = parseInt($('#alias-overlay').dataset.idx)
  if (idx >= 0) aliases.splice(idx, 1)
  save('aliases', aliases)
  $('#alias-overlay').classList.remove('show')
  renderAliases()
  toast('别名已删除')
}

// --- Color Theme ---
function openColorPicker() {
  const overlay = $('#color-overlay')
  overlay.classList.add('show')
  const grid = $('#color-presets')
  const presets = ['#7c5cfc','#5c3cfc','#2196f3','#00bcd4','#4caf50','#8bc34a','#ff9800','#ff5722','#e91e63','#9c27b0','#607d8b','#f44336']
  grid.innerHTML = `<div class="color-grid">${presets.map(c => `<div class="color-swatch ${c === (themeColor || '#7c5cfc') ? 'active' : ''}" style="background:${c}" data-color="${c}"></div>`).join('')}</div>`
  grid.querySelectorAll('.color-swatch').forEach(el => {
    el.addEventListener('click', () => {
      grid.querySelectorAll('.color-swatch').forEach(s => s.classList.remove('active'))
      el.classList.add('active')
      themeColor = el.dataset.color
      save('themeColor', themeColor)
      applyThemeColor()
      toast('主题色已更新')
    })
  })
}

// --- Export ---
function exportData() {
  const data = { favs, notes, sshHosts, aliases, execHistory }
  const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = 'linux-toolbox-export.json'
  a.click()
  URL.revokeObjectURL(a.href)
  toast('数据已导出')
}

// --- Copy ---
function copyText(t) {
  navigator.clipboard.writeText(t).then(() => toast('已复制: ' + t.substring(0, 50)))
}

// --- Toast ---
function toast(msg) {
  const el = $('#toast')
  el.textContent = msg
  el.classList.add('show')
  setTimeout(() => el.classList.remove('show'), 2500)
}

// --- Onboarding ---
function showOnboarding() {
  if (localStorage.getItem('onboarded')) return
  const overlay = document.createElement('div')
  overlay.id = 'onboard-overlay'
  overlay.innerHTML = `<div id="onboard-dialog">
    <div id="onboard-header">
      <div style="font-size:48px;text-align:center;margin-bottom:8px">🐧</div>
      <h2>欢迎使用 Linux 命令工具箱</h2>
      <p style="color:var(--fg2);margin-top:4px">专为 Linux 新手打造的随身命令手册</p>
    </div>
    <div id="onboard-body">
      <div class="onboard-step"><div class="onboard-icon">📖</div><div><strong>130+ 常用命令</strong><br><span class="onboard-desc">按分类浏览，每个命令都有语法说明和示例</span></div></div>
      <div class="onboard-step"><div class="onboard-icon">⭐</div><div><strong>难度分级</strong><br><span class="onboard-desc">⭐入门 ⭐⭐进阶 ⭐⭐⭐高级，从易到难循序渐进</span></div></div>
      <div class="onboard-step"><div class="onboard-icon">🔍</div><div><strong>中文搜索</strong><br><span class="onboard-desc">输入"查看进程"、"安装软件"也能搜到对应命令</span></div></div>
      <div class="onboard-step"><div class="onboard-icon">⚡</div><div><strong>一键执行</strong><br><span class="onboard-desc">点击命令即可执行，支持超时取消，带高危命令警告</span></div></div>
      <div class="onboard-step"><div class="onboard-icon">🖥️</div><div><strong>系统诊断</strong><br><span class="onboard-desc">实时监控 CPU/内存/磁盘，管理 Docker 和 SSH</span></div></div>
      <div class="onboard-step"><div class="onboard-icon">📝</div><div><strong>记录学习</strong><br><span class="onboard-desc">收藏常用命令，为每个命令添加个人笔记</span></div></div>
    </div>
    <div id="onboard-footer">
      <button id="onboard-start" class="primary">开始使用 🚀</button>
    </div>
  </div>`
  document.body.appendChild(overlay)
  overlay.querySelector('#onboard-start').addEventListener('click', () => {
    localStorage.setItem('onboarded', '1')
    overlay.remove()
  })
  overlay.addEventListener('click', (e) => {
    if (e.target === overlay) {
      localStorage.setItem('onboarded', '1')
      overlay.remove()
    }
  })
}

// --- Bind Events ---
function bindEvents() {
  // Theme
  $('#theme-toggle').addEventListener('click', () => {
    const cur = document.documentElement.getAttribute('data-theme')
    const next = cur === 'dark' ? 'light' : 'dark'
    document.documentElement.setAttribute('data-theme', next)
    localStorage.setItem('theme', next)
    $('#theme-toggle').textContent = next === 'dark' ? '🌙' : '☀️'
  })

  // Font size
  $('#font-size-select').addEventListener('change', () => {
    const v = $('#font-size-select').value
    localStorage.setItem('fontSize', v)
    applyFontSize()
  })

  // Distro filter
  $('#distro-filter').addEventListener('change', () => {
    if (activeView === 'commands') renderCommands()
  })

  // Role buttons
  $$('.role-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      $$('.role-btn').forEach(b => b.classList.remove('active'))
      btn.classList.add('active')
      activeRole = btn.dataset.role
      activeCategory = -1
      renderUI()
    })
  })

  // Nav buttons
  $('#btn-all').addEventListener('click', () => {
    stopAllTimers()
    activeView = 'commands'; activeCategory = -1
    renderView(); renderCategories()
  })
  $('#btn-guides').addEventListener('click', () => {
    stopAllTimers()
    activeView = 'guides'
    renderView()
  })
  $('#btn-sysinfo').addEventListener('click', () => {
    stopAllTimers()
    activeView = 'sysinfo'
    renderView()
  })
  $('#btn-docker').addEventListener('click', () => {
    activeView = 'docker'
    renderView()
  })
  $('#btn-ssh').addEventListener('click', () => {
    stopAllTimers()
    activeView = 'ssh'
    renderView()
  })
  $('#btn-favs').addEventListener('click', showFavorites)
  $('#btn-more').addEventListener('click', () => {
    const m = $('#more-menu')
    m.style.display = m.style.display === 'none' ? 'flex' : 'none'
  })
  document.addEventListener('click', (e) => {
    if (!e.target.closest('#btn-more') && !e.target.closest('#more-menu')) {
      $('#more-menu').style.display = 'none'
    }
  })
  $('#menu-aliases').addEventListener('click', () => {
    stopAllTimers()
    activeView = 'aliases'
    $('#more-menu').style.display = 'none'
    renderView()
  })
  $('#menu-pm').addEventListener('click', () => {
    stopAllTimers()
    $('#more-menu').style.display = 'none'
    showPMCompare()
  })
  $('#menu-color').addEventListener('click', () => {
    $('#more-menu').style.display = 'none'
    openColorPicker()
  })
  $('#menu-export').addEventListener('click', () => {
    $('#more-menu').style.display = 'none'
    exportData()
  })

  // Search (debounced)
  $('#search-input').addEventListener('input', () => {
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(() => {
      if (activeView === 'commands') renderCommands()
      renderSuggestions()
    }, 150)
  })
  $('#search-input').addEventListener('focus', renderSuggestions)
  $('#search-input').addEventListener('blur', () => setTimeout(() => { $('#search-suggest').style.display = 'none' }, 200))

  // Exec dialog
  $('#exec-close').addEventListener('click', () => $('#exec-overlay').classList.remove('show'))
  $('#exec-overlay').addEventListener('click', (e) => {
    if (e.target === e.currentTarget) $('#exec-overlay').classList.remove('show')
  })
  $('#exec-run').addEventListener('click', doExec)
  $('#exec-input').addEventListener('keydown', (e) => {
    if (e.key === 'Enter') doExec()
    if (e.key === 'ArrowDown') { e.preventDefault(); moveExecHistorySel(1) }
    if (e.key === 'ArrowUp') { e.preventDefault(); moveExecHistorySel(-1) }
  })
  $('#exec-input').addEventListener('input', () => { $('#exec-history-drop').style.display = 'none' })
  $('#exec-cancel').addEventListener('click', async () => {
    $('#exec-output').innerHTML = '<span class="exec-info">⏹️ 正在取消...</span>'
    await CancelExecution()
    $('#exec-output').innerHTML = '<span class="exec-warn">⏹️ 命令已取消</span>'
    $('#exec-run').disabled = false; $('#exec-run').textContent = '▶ 执行 (Enter)'
    $('#exec-cancel').style.display = 'none'
  })
  $('#exec-clear').addEventListener('click', () => {
    $('#exec-output').innerHTML = ''
    $('#exec-copy').style.display = 'none'
  })
  $('#exec-copy').addEventListener('click', copyExecOutput)
  $('#exec-show-history').addEventListener('click', () => {
    const drop = $('#exec-history-drop')
    if (drop.style.display === 'block') drop.style.display = 'none'
    else renderExecHistory()
  })
  document.addEventListener('click', (e) => {
    if (!e.target.closest('#exec-body')) {
      $('#exec-history-drop').style.display = 'none'
    }
  })

  // Note dialog
  $('#note-close').addEventListener('click', () => $('#note-overlay').classList.remove('show'))
  $('#note-overlay').addEventListener('click', (e) => {
    if (e.target === e.currentTarget) $('#note-overlay').classList.remove('show')
  })
  $('#note-save').addEventListener('click', saveNoteHandler)
  $('#note-delete').addEventListener('click', async () => {
    const name = currentCmdName
    if (!name) return
    $('#note-textarea').value = ''
    delete notes[name]
    await SaveNote(name, '')
    $('#note-overlay').classList.remove('show')
    renderCommands()
    toast('笔记已删除')
  })

  // Color dialog
  $('#color-close').addEventListener('click', () => $('#color-overlay').classList.remove('show'))
  $('#color-overlay').addEventListener('click', (e) => {
    if (e.target === e.currentTarget) $('#color-overlay').classList.remove('show')
  })

  // SSH edit dialog
  $('#sshedit-close').addEventListener('click', () => $('#sshedit-overlay').classList.remove('show'))
  $('#sshedit-overlay').addEventListener('click', (e) => {
    if (e.target === e.currentTarget) $('#sshedit-overlay').classList.remove('show')
  })
  $('#sshedit-save').addEventListener('click', saveSSH)
  $('#sshedit-cancel').addEventListener('click', () => $('#sshedit-overlay').classList.remove('show'))
  $('#sshedit-delete').addEventListener('click', deleteSSH)

  // Alias dialog
  $('#alias-close').addEventListener('click', () => $('#alias-overlay').classList.remove('show'))
  $('#alias-overlay').addEventListener('click', (e) => {
    if (e.target === e.currentTarget) $('#alias-overlay').classList.remove('show')
  })
  $('#alias-save').addEventListener('click', saveAlias)
  $('#alias-cancel').addEventListener('click', () => $('#alias-overlay').classList.remove('show'))
  $('#alias-delete').addEventListener('click', deleteAlias)

  // Keyboard shortcuts
  document.addEventListener('keydown', (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'f') {
      e.preventDefault(); $('#search-input').focus()
    }
    if (e.key === 'Escape') {
      if ($('#exec-overlay').classList.contains('show')) $('#exec-overlay').classList.remove('show')
      if ($('#note-overlay').classList.contains('show')) $('#note-overlay').classList.remove('show')
      if ($('#color-overlay').classList.contains('show')) $('#color-overlay').classList.remove('show')
      if ($('#sshedit-overlay').classList.contains('show')) $('#sshedit-overlay').classList.remove('show')
      if ($('#alias-overlay').classList.contains('show')) $('#alias-overlay').classList.remove('show')
      $('#more-menu').style.display = 'none'
    }
  })
}

function moveExecHistorySel(dir) {
  const items = $('#exec-history-drop').querySelectorAll('.sug-item')
  if (!items.length) return
  let sel = -1
  items.forEach((it, i) => { if (it.classList.contains('sel')) sel = i })
  items.forEach(it => it.classList.remove('sel'))
  sel = Math.max(0, Math.min(items.length - 1, (sel < 0 ? (dir > 0 ? 0 : items.length - 1) : sel + dir)))
  items[sel].classList.add('sel')
  items[sel].scrollIntoView({ block: 'nearest' })
  $('#exec-input').value = execHistory[parseInt(items[sel].dataset.idx)].cmd
}

async function doExec() {
  const cmd = $('#exec-input').value.trim()
  if (!cmd) return
  if (DANGEROUS_CMDS.some(d => cmd.includes(d))) {
    if (!confirm('⚠️ 这条命令看起来有破坏性，确认执行？\n\n' + cmd)) return
  }
  addExecHistory(cmd)
  $('#exec-history-drop').style.display = 'none'
  $('#exec-output').innerHTML = '<span class="exec-info">⏳ 执行中...</span>'
  $('#exec-run').disabled = true; $('#exec-run').textContent = '执行中...'
  $('#exec-cancel').style.display = 'inline-block'
  $('#exec-copy').style.display = 'none'
  const expanded = expandAlias(cmd)
  const startTime = performance.now()
  try {
    const res = await ExecuteCommand(expanded)
    const elapsed = ((performance.now() - startTime) / 1000).toFixed(2)
    renderExecOutput(expanded, res, elapsed)
  } catch (e) {
    $('#exec-output').innerHTML = `<span class="exec-error">❌ 执行出错: ${e.message || e}</span>`
  } finally {
    $('#exec-run').disabled = false; $('#exec-run').textContent = '▶ 执行 (Enter)'
    $('#exec-cancel').style.display = 'none'
  }
}

function copyExecOutput() {
  const el = $('#exec-output')
  const text = el.textContent
  if (!text || text === '点击「执行」或按 Enter') return toast('没有可复制的内容')
  navigator.clipboard.writeText(text).then(() => toast('输出已复制'))
}

function renderExecOutput(cmd, res, elapsed) {
  const el = $('#exec-output')
  let html = `<div class="exec-cmdline">$ ${cmd}</div>`
  if (res.output) {
    const highlighted = highlightOutput(res.output)
    html += `<div class="exec-output">${highlighted}</div>`
  }
  if (res.error) {
    html += `<div class="exec-stderr">${escapeHtml(res.error)}</div>`
  }
  if (!res.output && !res.error) {
    html += '<div class="exec-info">(空输出)</div>'
  }
  if (!res.success && res.error) {
    html += '<div class="exec-fail">❌ 执行失败</div>'
  }
  html += `<div class="exec-footer-info">⏱️ ${elapsed || '?'}s${res.success ? ' ✅ 成功' : ''}</div>`
  el.innerHTML = html
  el.scrollTop = el.scrollHeight
  $('#exec-copy').style.display = 'inline-block'
}

function highlightOutput(text) {
  const lines = text.split('\n')
  return lines.map(line => {
    const l = line.toLowerCase()
    if (l.includes('error') || l.includes('fail') || l.includes('failed') || l.includes('denied') ||
        l.includes('not found') || l.includes('cannot') || l.includes('无法') || l.includes('错误') ||
        l.includes('权限') || l.includes('拒绝') || l.includes('失败')) {
      return `<span class="hl-error">${escapeHtml(line)}</span>`
    }
    if (l.includes('warn') || l.includes('warning') || l.includes('注意') || l.includes('deprecated')) {
      return `<span class="hl-warn">${escapeHtml(line)}</span>`
    }
    if (l.includes('success') || l.includes('ok') || l.includes('done') || l.includes('完成') ||
        l.includes('成功') || l.startsWith('+')) {
      return `<span class="hl-success">${escapeHtml(line)}</span>`
    }
    return escapeHtml(line)
  }).join('\n')
}

function escapeHtml(text) {
  const d = document.createElement('div')
  d.textContent = text
  return d.innerHTML
}
function expandAlias(cmd) {
  const parts = cmd.split(/\s+/)
  if (parts.length) {
    const a = aliases.find(x => x.name === parts[0])
    if (a) return a.cmd + cmd.substring(parts[0].length)
  }
  return cmd
}

function renderSuggestions() {
  const drop = $('#search-suggest')
  const q = $('#search-input').value.trim().toLowerCase()
  if (!q || activeView !== 'commands') { drop.style.display = 'none'; return }
  let all = filterByRoleCommands(flatCommands)
  let results = all.filter(c =>
    c.name.toLowerCase().includes(q) || (c.desc || '').toLowerCase().includes(q) ||
    (c.syntax || '').toLowerCase().includes(q) || (c.examples || []).some(e => e.toLowerCase().includes(q))
  )
  if (results.length < 1) {
    const zhCmds = new Set()
    for (const [zh, cmds] of Object.entries(ZH_SEARCH)) {
      if (zh.includes(q) || q.includes(zh)) cmds.forEach(c => zhCmds.add(c))
    }
    if (zhCmds.size) {
      results = all.filter(c => zhCmds.has(c.name))
    }
  }
  results = results.slice(0, 8)
  if (!results.length) { drop.style.display = 'none'; return }
  drop.style.display = 'block'
  drop.innerHTML = results.map(c => {
    const diff = DIFFICULTY_ICONS[c.difficulty] || DIFFICULTY_ICONS.intermediate
    return `<div class="sug-item" data-cmd="${c.name}" data-syntax="${c.syntax || ''}">
      <span><span class="sug-name">${c.name}</span> <span class="sug-desc">${(c.desc || '').substring(0, 40)}</span></span>
      <span style="display:flex;gap:6px;align-items:center">
        <span class="diff-badge-sm" style="color:${diff.color};font-size:10px">${diff.icon}</span>
        <span style="color:var(--fg2);font-size:10px">${c.catName}</span>
      </span>
    </div>`
  }).join('')
  drop.querySelectorAll('.sug-item').forEach(el => {
    el.addEventListener('mousedown', (e) => {
      e.preventDefault()
      const syntax = el.dataset.syntax
      if (syntax) openExec(syntax)
      else {
        $('#search-input').value = el.dataset.cmd
        renderCommands()
      }
      drop.style.display = 'none'
    })
  })
}

function showPMCompare() {
  stopAllTimers()
  $('#view-title').textContent = '📦 跨发行版包管理速查'
  hideAllViews()
  const grid = $('#commands-grid')
  grid.style.display = 'grid'
  grid.style.gridTemplateColumns = '1fr'
  const curDistro = currentDistro
  const distroOrder = ['debian', 'rhel', 'fedora', 'arch', 'suse']
  const distroNames = { debian: '🐧 Debian/Ubuntu', rhel: '🎩 RHEL/CentOS', fedora: '💙 Fedora', arch: '🚀 Arch Linux', suse: '🦎 openSUSE' }
  const curIdx = distroOrder.indexOf(curDistro)
  let html = `<div class="pm-intro">不同 Linux 发行版的包管理命令对照表。当前系统: <strong>${distroNames[curDistro] || '💻 ' + curDistro}</strong></div>`
  html += '<table class="pm-table"><thead><tr><th>操作</th>'
  distroOrder.forEach(d => {
    html += `<th class="${d === curDistro ? 'pm-cur' : ''}">${distroNames[d] || d}</th>`
  })
  html += '</tr></thead><tbody>'
  PM_COMPARE.rows.forEach(r => {
    html += '<tr>'
    html += `<td class="pm-op">${r.op}</td>`
    distroOrder.forEach(d => {
      const isCur = d === curDistro
      html += `<td class="${isCur ? 'pm-cur' : ''}"><code>${r[d] || '-'}</code></td>`
    })
    html += '</tr>'
  })
  html += '</tbody></table>'
  html += '<div style="margin-top:12px;font-size:var(--fs-s);color:var(--fg2);text-align:center">💡 当前发行版的命令已高亮标记</div>'
  grid.innerHTML = html
  grid.querySelectorAll('code').forEach(el => {
    el.addEventListener('click', () => {
      const txt = el.textContent.trim()
      if (txt && txt !== '-') openExec(txt)
    })
  })
  $('#view-count').textContent = ''
}

function showFavorites() {
  stopAllTimers()
  activeView = 'commands'
  activeCategory = -1
  $('#view-title').textContent = '⭐ 收藏的命令'
  $('#btn-all').classList.add('active')
  $('#btn-guides').classList.remove('active')
  $('#btn-sysinfo').classList.remove('active')
  hideAllViews()
  const grid = $('#commands-grid')
  grid.style.display = 'grid'
  let cmds = flatCommands.filter(c => favs.includes(c.name))
  cmds = filterByRoleCommands(cmds)
  $('#view-count').textContent = cmds.length + ' 条'
  renderCommandCards(grid, cmds, '⭐')
}
