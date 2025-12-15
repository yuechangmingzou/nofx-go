# PowerShell脚本：推送到GitHub
# 使用方法：在PowerShell中运行 .\push_to_github.ps1

Write-Host "=== 推送到GitHub ===" -ForegroundColor Green

# 检查是否在正确的目录
if (-not (Test-Path ".git")) {
    Write-Host "错误: 当前目录不是git仓库" -ForegroundColor Red
    exit 1
}

# 配置Git用户信息（如果尚未配置）
$currentUser = git config user.name
if (-not $currentUser) {
    Write-Host "`n需要配置Git用户信息" -ForegroundColor Yellow
    $userName = Read-Host "请输入您的姓名"
    $userEmail = Read-Host "请输入您的邮箱"
    
    git config user.name $userName
    git config user.email $userEmail
    Write-Host "Git用户信息已配置" -ForegroundColor Green
} else {
    Write-Host "`nGit用户信息: $currentUser ($(git config user.email))" -ForegroundColor Cyan
}

# 添加远程仓库
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
Write-Host "文件已添加" -ForegroundColor Green

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
git push -u origin main

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n✅ 代码已成功推送到GitHub!" -ForegroundColor Green
    Write-Host "仓库地址: https://github.com/yuechangmingzou/nofx-go" -ForegroundColor Cyan
} else {
    Write-Host "`n❌ 推送失败" -ForegroundColor Red
    Write-Host "可能的原因:" -ForegroundColor Yellow
    Write-Host "1. 需要配置GitHub认证（Personal Access Token）" -ForegroundColor Yellow
    Write-Host "2. 网络连接问题" -ForegroundColor Yellow
    Write-Host "`n生成Token: https://github.com/settings/tokens" -ForegroundColor Cyan
}

