# 安装与认证（github-explorer 参考资料）

仅当 SKILL.md 第二步检测到 `NEED_SETUP`（`gh` 未安装或未认证）时执行本文件。安装/认证多为交互式或需 sudo，**让用户在自己的终端用 `! <命令>` 执行**，完成后回来继续。

## 安装

```bash
# macOS
brew install gh

# Windows (PowerShell)
winget install --id GitHub.cli

# Linux (Debian/Ubuntu) —— 其它发行版见 https://github.com/cli/cli#installation
(type -p wget >/dev/null || (sudo apt update && sudo apt-get install wget -y)) \
  && sudo mkdir -p -m 755 /etc/apt/keyrings \
  && out=$(mktemp) && wget -nv -O$out https://cli.github.com/packages/githubcli-archive-keyring.gpg \
  && cat $out | sudo tee /etc/apt/keyrings/githubcli-archive-keyring.gpg > /dev/null \
  && sudo chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg \
  && sudo mkdir -p -m 755 /etc/apt/sources.list.d \
  && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
  && sudo apt update && sudo apt install gh -y
```

## 认证

```bash
gh auth login        # 选 GitHub.com → HTTPS → 浏览器登录
gh auth status       # 验证
```
