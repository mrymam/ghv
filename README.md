# github-viewer (`gv`)

GitHub上の自分に関連するPRをターミナルで表形式に一覧表示するCLIツール。

## 機能

- **自分のPR** — 作成・アサインされたPRをステータス付きで表示
  - `open` / `draft` / `approved` / `reviewed (N unresolved)`
  - レビュースレッドの未解決数をGraphQL APIで取得
- **レビューリクエスト** — レビュー待ちPRを作成者付きで表示
- **Bot PR** — 指定リポジトリのBot作成PRを一覧表示（`user.type` で自動判定）
- **クリップボードコピー** — `-copy` オプションでPRタイトルをリッチテキストリンクとしてクリップボードにコピー（Slack/Teamsに貼り付け可能）
- タイトルはOSC 8ハイパーリンク対応（クリックでPRページを開く）

## 前提条件

- Go 1.21+
- [GitHub CLI (`gh`)](https://cli.github.com/) がインストール・認証済みであること

## インストール

```bash
./install.sh
```

`/usr/local/bin/gv` にインストールされます。

または直接ビルド:

```bash
go build -o gv .
```

## 使い方

```bash
# 自分のPR + レビュー待ちPRを両方表示
gv

# 自分が作成・アサインされたPRのみ表示
gv my

# レビューリクエストされたPRのみ表示
gv review

# Bot PRを表示
gv bot

# PR一覧をクリップボードにコピー
gv -copy
gv my -copy
```

### Organization でフィルタ

```bash
# 引数で指定
gv -org MyOrg
gv my -org MyOrg

# 環境変数で指定
export GV_ORG=MyOrg
gv
```

引数 `-org` が優先されます。

### クリップボードにコピー

```bash
gv -copy
```

表示されたPRのタイトルをリッチテキストリンクとしてクリップボードにコピーします。Slack、Teams等にそのまま貼り付けるとクリック可能なリンクになります（macOS専用、`textutil` + `pbcopy` を使用）。

`GV_DEFAULT_COPY_ON=true` を設定すると、`-copy` を指定しなくても常にコピーされます。`-copy` フラグを明示的に指定した場合はフラグが優先されます。

## 環境変数

| 変数名 | 説明 |
|--------|------|
| `GV_ORG` | デフォルトの organization フィルタ |
| `GV_IGNORE_REVIEWERS` | 無視するレビュアーのアカウント名（カンマ区切り） |
| `GV_BOT_REPOS` | Bot PR をwatchするリポジトリ名一覧（カンマ区切り、org名不要） |
| `GV_DEFAULT_COPY_ON` | `true` で `-copy` をデフォルトON、`false` でOFF（未指定時はOFF） |

### `GV_BOT_REPOS`

`gv bot` で監視するリポジトリ名を指定します（org名不要、`GV_ORG` から自動補完）。各リポジトリのopen PRのうち、GitHubの `user.type == "Bot"` で自動判定されたものだけを表示します。

```bash
export GV_ORG=myorg
export GV_BOT_REPOS="frontend,backend"
gv bot
```

### `GV_IGNORE_REVIEWERS`

botなど特定アカウントのレビュースレッドを未解決数のカウントから除外します。

```bash
export GV_IGNORE_REVIEWERS="renovate[bot],dependabot[bot]"
```

## ファイル構成

| ファイル | 内容 |
|----------|------|
| `main.go` | エントリポイント、サブコマンドルーティング |
| `my.go` | 自分のPR表示（`cmdMy`, `printMySection`） |
| `review.go` | レビューリクエストPR表示（`cmdReview`, `printSection`） |
| `bot.go` | Bot PR表示（`cmdBot`） |
| `util.go` | PR型定義、API呼び出し、ヘルパー関数 |
| `clipboard.go` | クリップボードコピー機能（HTML→RTF変換） |
