# 非交互式GitHub推送脚本
# 请先修改下面的用户信息，然后运行此脚本

# ========== 配置区域 ==========
# 请修改为您的GitHub信息
$GIT_USER_NAME = "yuechangmingzou"
$GIT_USER_EMAIL = "your-email@example.com"  # 请修改为您的邮箱
# =============================

Write-Host "=== 配置Git并推送到GitHub ===" -ForegroundColor Green

# 检查是否在正确的目录
if (-not (Test-Path ".git")) {
    Write-Host "错误: 当前目录不是git仓库" -ForegroundColor Red
    exit 1
}

# 配置Git用户信息
Write-Host "`n配置Git用户信息..." -ForegroundColor Yellow
git config user.name $GIT_USER_NAME
git config user.email $GIT_USER_EMAIL
Write-Host "Git用户信息已配置: $GIT_USER_NAME <$GIT_USER_EMAIL>" -ForegroundColor Green

# 检查远程仓库
Write-Host "`n检查远程仓库..." -ForegroundColor Yellow
$remote = git remote get-url origin 2>$null
if (-not $remote) {
    git remote add origin https://github.com/yuechangmingzou/nofx-go.git
    Write-Host "已添加远程仓库" -ForegroundColor Green
} else {
    Write-Host "远程仓库已存在: $remote" -ForegroundColor Cyan
}

# 设置主分支
git branch -M main

# 添加所有文件
Write-Host "`n添加文件到暂存区..." -ForegroundColor Yellow
git add .
$fileCount = (git status --short | Measure-Object -Line).Lines
Write-Host "已添加 $fileCount 个文件" -ForegroundColor Green

# 提交
Write-Host "`n提交更改..." -ForegroundColor Yellow
$commitMessage = "feat: Complete Go conversion to 100% - Add GetOpenOrders, enhance SL/TP guard, implement rule strategy, improve symbol filtering"
git commit -m $commitMessage
if ($LASTEXITCODE -eq 0) {
    Write-Host "提交成功" -ForegroundColor Green
} else {
    Write-Host "提交失败，请检查错误信息" -ForegroundColor Red
    exit 1
}

# 推送到GitHub
Write-Host "`n推送到GitHub..." -ForegroundColor Yellow
Write-Host "注意: 如果提示输入凭据，请使用GitHub Personal Access Token作为密码" -ForegroundColor Cyan
Write-Host "生成Token: https://github.com/settings/tokens" -ForegroundColor Cyan
Write-Host "`n正在推送..." -ForegroundColor Yellow

git push -u origin main

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n✅ 代码已成功推送到GitHub!" -ForegroundColor Green
    Write-Host "仓库地址: https://github.com/yuechangmingzou/nofx-go" -ForegroundColor Cyan
} else {
    Write-Host "`n❌ 推送失败" -ForegroundColor Red
    Write-Host "`n可能的原因和解决方案:" -ForegroundColor Yellow
    Write-Host "1. 需要GitHub Personal Access Token" -ForegroundColor Yellow
    Write-Host "   - 访问: https://github.com/settings/tokens" -ForegroundColor White
    Write-Host "   - 生成新Token，勾选 'repo' 权限" -ForegroundColor White
    Write-Host "   - 推送时使用Token作为密码" -ForegroundColor White
    Write-Host "`n2. 或者使用SSH方式（需要配置SSH密钥）" -ForegroundColor Yellow
    Write-Host "   git remote set-url origin git@github.com:yuechangmingzou/nofx-go.git" -ForegroundColor White
}

