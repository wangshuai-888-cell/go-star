# ============================================================
# go-star 本地交叉编译 → Linux amd64
#
# 用法：在项目根目录执行
#   .\scripts\build.ps1
#
# 产物：go-star-linux.new（项目根目录）
# 环境变量只在本脚本作用域内生效，不会污染终端
# ============================================================

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
if (-not $Root) { $Root = $PSScriptRoot + "\.." }
Set-Location $Root

$OutFile = "go-star-linux.new"

# 先删旧产物：万一编译失败，不会留下一个"看起来是新的"的旧包，
# 避免把它误传上服务器（这类事故排查起来极痛苦）
if (Test-Path $OutFile) {
    Remove-Item $OutFile -Force
    Write-Host "已清理上一次的 $OutFile" -ForegroundColor DarkGray
}

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host " go-star 交叉编译 → linux/amd64" -ForegroundColor Cyan
Write-Host " 工作目录: $Root" -ForegroundColor DarkGray
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

# --- 设置交叉编译环境 ---
$env:CGO_ENABLED = "0"
$env:GOOS        = "linux"
$env:GOARCH      = "amd64"

$Code = 1
$Start = Get-Date
try {
    go build -o $OutFile .
    $Code = $LASTEXITCODE
}
finally {
    # 必须清理：否则本窗口后续所有 go 命令都会当成 Linux 目标
    Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
}

if ($Code -ne 0) {
    Write-Host ""
    Write-Host "❌ 编译失败，退出码 = $Code" -ForegroundColor Red
    exit $Code
}

$Elapsed = (Get-Date) - $Start

# --- 校验产物 ---
$Item  = Get-Item $OutFile
$Mb    = [math]::Round($Item.Length / 1MB, 2)
$Hash  = (Get-FileHash -Algorithm MD5 $OutFile).Hash

$fs  = [System.IO.File]::OpenRead($Item.FullName)
$buf = New-Object byte[] 4
$null = $fs.Read($buf, 0, 4)
$fs.Close()
$Header = ($buf | ForEach-Object { $_.ToString('X2') }) -join ' '

if ($Header -ne "7F 45 4C 46") {
    Write-Host ""
    Write-Host "❌ 文件头 = $Header，不是 ELF！环境变量可能没生效" -ForegroundColor Red
    Write-Host "   若为 4D 5A (MZ)，说明编出来的是 Windows 程序" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "✅ 编译成功" -ForegroundColor Green
Write-Host "   文件   : $OutFile"
Write-Host "   大小   : $Mb MB  ($($Item.Length) 字节)"
Write-Host "   文件头 : $Header  (Linux ELF)"
Write-Host "   耗时   : $([math]::Round($Elapsed.TotalSeconds, 1)) 秒"
Write-Host ""
Write-Host "   MD5    : $Hash" -ForegroundColor Yellow
Write-Host ""
Write-Host "-------------------- 下一步 --------------------" -ForegroundColor Cyan
Write-Host " 1) 上传："
Write-Host "      scp $OutFile root@your-server-ip:/home/admin/go-star/"
Write-Host "      ↑ 把 your-server-ip 换成你自己的服务器地址" -ForegroundColor DarkGray
Write-Host ""
Write-Host " 2) SSH 登录服务器后部署："
Write-Host "      cd /home/admin/go-star"
Write-Host "      ./deploy.sh"
Write-Host ""
Write-Host " 3) deploy.sh 会打印新二进制 MD5，"
Write-Host "    与上面的 $Hash 核对（大小写不敏感）" -ForegroundColor DarkGray
Write-Host "------------------------------------------------" -ForegroundColor Cyan
