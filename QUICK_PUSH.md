# 快速推送到GitHub

## 方法1: 使用PowerShell脚本（推荐）

```powershell
cd nofx-go
.\push_to_github.ps1
```

脚本会自动：
1. 检查并配置Git用户信息
2. 添加远程仓库
3. 提交所有更改
4. 推送到GitHub

## 方法2: 手动执行命令

### 步骤1: 配置Git用户信息（如果尚未配置）

```powershell
cd nofx-go
git config user.name "Your Name"
git config user.email "your.email@example.com"
```

### 步骤2: 添加远程仓库（如果尚未添加）

```powershell
git remote add origin https://github.com/yuechangmingzou/nofx-go.git
```

### 步骤3: 提交并推送

```powershell
git branch -M main
git add .
git commit -m "feat: Complete Go conversion to 100% - Add GetOpenOrders, enhance SL/TP guard, implement rule strategy, improve symbol filtering"
git push -u origin main
```

## 认证说明

如果推送时提示输入凭据：
- **用户名**: 您的GitHub用户名（yuechangmingzou）
- **密码**: 使用GitHub Personal Access Token（不是账户密码）

### 生成Personal Access Token

1. 访问 https://github.com/settings/tokens
2. 点击 "Generate new token" -> "Generate new token (classic)"
3. 设置权限：至少勾选 `repo` 权限
4. 生成后复制Token（只显示一次）
5. 推送时使用Token作为密码

## 验证

推送成功后，访问 https://github.com/yuechangmingzou/nofx-go 查看代码。

