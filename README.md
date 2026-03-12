# github-viewer (`gv`)

GitHub上の自分に関連するPRをターミナルで表形式に一覧表示するCLIツール。

## 機能

- **自分のPR** — 作成・アサインされたPRをステータス付きで表示
  - `open` / `draft` / `approved` / `reviewed (N unresolved)`
  - レビュースレッドの未解決数をGraphQL APIで取得
- **レビューリクエスト** — レビュー待ちPRを作成者付きで表示
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

## 環境変数

| 変数名 | 説明 |
|--------|------|
| `GV_ORG` | デフォルトの organization フィルタ |
| `GV_IGNORE_REVIEWERS` | 無視するレビュアーのアカウント名（カンマ区切り） |

### `GV_IGNORE_REVIEWERS`

botなど特定アカウントのレビュースレッドを未解決数のカウントから除外します。

```bash
export GV_IGNORE_REVIEWERS="renovate[bot],dependabot[bot]"
```
