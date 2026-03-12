# github-viewer

GitHub上の自分に関連するPRをターミナルで一覧表示するCLIツール。

## 前提条件

- Go 1.21+
- [GitHub CLI (`gh`)](https://cli.github.com/) がインストール・認証済みであること

## インストール

```bash
go install github.com/mrymam/github-viewer@latest
```

または直接ビルド:

```bash
go build -o github-viewer .
```

## 使い方

```bash
# 自分のPR + レビュー待ちPRを両方表示
github-viewer

# 自分が作成・アサインされたPRのみ表示
github-viewer my

# レビューリクエストされたPRのみ表示
github-viewer review
```

### Organization でフィルタ

引数または環境変数で organization を指定できます。

```bash
# 引数で指定
github-viewer -org MyOrg
github-viewer my -org MyOrg
github-viewer review -org MyOrg

# 環境変数で指定
export GV_ORG=MyOrg
github-viewer
```

引数 `-org` が優先されます。

## 環境変数

| 変数名 | 説明 |
|--------|------|
| `GV_ORG` | デフォルトの organization フィルタ |
