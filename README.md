# ghv

GitHub上の自分に関連するPRをターミナルで表形式に一覧表示するCLIツール。

## 機能

- **自分のPR** — 作成・アサインされたPRをステータス付きで表示
  - `open` / `draft` / `approved (N)` / `changes requested (N)` / `reviewed (N unresolved)`
  - レビュー状態をGraphQL APIで取得
- **レビューリクエスト** — レビュー待ちPRを作成者付きで表示
- **Bot PR** — 指定リポジトリのBot作成PRを一覧表示（`user.type` で自動判定）
- **TUIモード** — インタラクティブなターミナルUI（bubbletea + lipgloss）
  - タブ切替、カーソル移動、Enterでブラウザ表示
  - `--poll` オプションで自動ポーリング対応
- **Notifyモード** — 新しいレビューリクエストをmacOS通知で自動通知
  - `--polling` でポーリング間隔を指定（デフォルト5分）
- **クリップボードコピー** — `-copy` オプションでPRタイトルをリッチテキストリンクとしてクリップボードにコピー（Slack/Teamsに貼り付け可能）
- タイトルはOSC 8ハイパーリンク対応（クリックでPRページを開く）

## 前提条件

- Go 1.21+
- [GitHub CLI (`gh`)](https://cli.github.com/) がインストール・認証済みであること

## インストール

```bash
go install github.com/mrymam/ghv@latest
```

または手動ビルド:
```bash
go build -o ghv .
```

## 使い方

```bash
# 自分のPR + レビュー待ちPRを両方表示
ghv

# 自分が作成・アサインされたPRのみ表示
ghv my

# レビューリクエストされたPRのみ表示
ghv review

# Bot PRを表示
ghv bot

# TUIモードで表示
ghv tui

# TUIモード + 5分ごとに自動リロード
ghv tui --poll 5m

# 新しいレビューリクエストをmacOS通知で監視
ghv notify

# 3分間隔でポーリング
ghv notify --polling 3m

# PR一覧をクリップボードにコピー
ghv -copy
ghv my -copy
```

### Organization でフィルタ

```bash
# 引数で指定
ghv -org MyOrg
ghv my -org MyOrg

# 環境変数で指定
export GHV_ORG=MyOrg
ghv
```

引数 `-org` が優先されます。

### クリップボードにコピー

```bash
ghv -copy
```

表示されたPRのタイトルをリッチテキストリンクとしてクリップボードにコピーします。Slack、Teams等にそのまま貼り付けるとクリック可能なリンクになります（macOS専用、`textutil` + `pbcopy` を使用）。

`GHV_DEFAULT_COPY_ON=true` を設定すると、`-copy` を指定しなくても常にコピーされます。`-copy` フラグを明示的に指定した場合はフラグが優先されます。

### TUIモード

```bash
# 基本起動
ghv tui

# 自動リロード（--poll で間隔指定）
ghv tui --poll 5m    # 5分ごと
ghv tui --poll 30s   # 30秒ごと
```

| キー | 操作 |
|------|------|
| `↑↓` / `jk` | カーソル移動 |
| `←→` / `hl` / `tab` | タブ切替 |
| `enter` | ブラウザでPRを開く |
| `r` | 手動リロード |
| `q` / `esc` | 終了 |

`--poll` を指定しない場合は自動リロードは無効です。`r` キーで手動リロードできます。

### Notifyモード

```bash
# デフォルト5分間隔で監視
ghv notify

# 3分間隔
ghv notify --polling 3m

# 1時間間隔
ghv notify --polling 1h
```

新しいレビューリクエストが来ると、macOS通知センターにサウンド付きで通知されます。`Ctrl+C` で停止します。

> **注意:** Notifyモードは **macOS専用** です。
>
> - [`terminal-notifier`](https://github.com/julienXX/terminal-notifier) がインストールされている場合、通知クリックでPRをブラウザで開けます
>   ```bash
>   brew install terminal-notifier
>   ```
> - 未インストールの場合は `osascript` にフォールバックします（通知クリックでのURL遷移は不可）

## 環境変数

| 変数名 | 説明 |
|--------|------|
| `GHV_ORG` | デフォルトの organization フィルタ |
| `GHV_IGNORE_REVIEWERS` | 無視するレビュアーのアカウント名（カンマ区切り） |
| `GHV_BOT_REPOS` | Bot PR をwatchするリポジトリ名一覧（カンマ区切り、org名不要） |
| `GHV_DEFAULT_COPY_ON` | `true` で `-copy` をデフォルトON、`false` でOFF（未指定時はOFF） |

### `GHV_BOT_REPOS`

`ghv bot` で監視するリポジトリ名を指定します（org名不要、`GHV_ORG` から自動補完）。各リポジトリのopen PRのうち、GitHubの `user.type == "Bot"` で自動判定されたものだけを表示します。

```bash
export GHV_ORG=myorg
export GHV_BOT_REPOS="frontend,backend"
ghv bot
```

### `GHV_IGNORE_REVIEWERS`

botなど特定アカウントのレビュースレッドを未解決数のカウントから除外します。

```bash
export GHV_IGNORE_REVIEWERS="renovate[bot],dependabot[bot]"
```

## ライブラリとして使う

`pkg/github` パッケージを外部プロジェクトから Go ライブラリとして利用できます。

```bash
go get github.com/mrymam/ghv@latest
```

```go
import gh "github.com/mrymam/ghv/pkg/github"

// PR検索
prs, err := gh.SearchPRs("review-requested:octocat org:myorg")

// ユーザー名取得
username, err := gh.GetUsername()

// レビューステータス取得
ignore := gh.ParseIgnoredReviewers("renovate[bot],dependabot[bot]")
statusMap := gh.FetchAllReviewStatuses(prs, ignore)

// ステータスラベル生成
label, color := gh.CombinedStatus(pr, statusMap[pr.HTMLURL])
```

### 公開 API

| パッケージ | 関数/型 | 説明 |
|-----------|---------|------|
| `pkg/github` | `PR` | GitHub PR 構造体（`FullRepoName()`, `RepoName()` メソッド付き） |
| | `SearchResult` | 検索 API レスポンス型 |
| | `ReviewStatus` | レビュースレッド集計（Unresolved, ApproveCount 等） |
| | `FetchResult` | PR取得結果（`PRs []PR`, `Err error`） |
| | `SearchPRs(qualifier) ([]PR, error)` | `gh api` で PR 検索 |
| | `GetUsername() (string, error)` | 認証済みユーザー名を取得 |
| | `MergePRs(a, b []PR) []PR` | 2つの PR スライスを重複排除してマージ |
| | `SortPRsByUpdated(prs []PR)` | 更新日時の降順でソート |
| | `FetchReviewStatus(pr, ignore) ReviewStatus` | 個別 PR のレビューステータスを GraphQL で取得 |
| | `FetchAllReviewStatuses(prs, ignore) map[string]ReviewStatus` | 複数 PR のレビューステータスを並列取得 |
| | `ParseIgnoredReviewers(csv) map[string]bool` | カンマ区切り文字列を無視リストにパース |
| | `CombinedStatus(pr, rs) (label, color)` | PR 状態とレビュー状態を統合したラベルを返す |

> **前提:** `gh` CLI がインストール・認証済みである必要があります。

