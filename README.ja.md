# 🌸 prpr

[![CI](https://github.com/kanywst/prpr/actions/workflows/ci.yml/badge.svg)](https://github.com/kanywst/prpr/actions/workflows/ci.yml) [![Go Reference](https://pkg.go.dev/badge/github.com/kanywst/prpr.svg)](https://pkg.go.dev/github.com/kanywst/prpr) [![Go Report Card](https://goreportcard.com/badge/github.com/kanywst/prpr)](https://goreportcard.com/report/github.com/kanywst/prpr) [![Release](https://img.shields.io/github/v/release/kanywst/prpr?sort=semver)](https://github.com/kanywst/prpr/releases/latest) [![Go version](https://img.shields.io/github/go-mod/go-version/kanywst/prpr)](go.mod) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

[English](README.md) | **日本語**

自分が見られる GitHub の **open PR を全部まとめて眺める TUI**。`enter` でブラウザが開いて、マージされたら 🎉 と一緒に消えていく。

対象の owner は自動で決まる: **ログインユーザー自身 + 所属している全 org**。設定ファイルはいらない。

UI は既定で英語。`--lang ja` で日本語になる。

![PR 一覧を移動し、詳細ペインを開き、絞り込み、タブを切り替え、マージのバナーが出るまでの録画](docs/demo.gif)

## できること

- **owner 自動検出** — ログインユーザーと所属 org を GitHub に聞いて、そこの open PR を全部並べる
- **マージ検知** — 一覧から消えた PR の最終状態を引いて、🎉 バナーを数秒出してからフェードアウト
- **4 つのタブ** — すべて / 自分の / レビュー待ち / 下書き。それぞれ件数バッジ付き
- **絞り込み** — `/` でタイトル・リポ・作者・`#番号`・ラベルを横断検索。スペース区切りは AND
- **詳細ペイン** — `d` でブランチ・差分・レビュアー・本文。幅 100 桁以上なら左右分割、狭ければ全面
- **自動更新** — 既定 60 秒ごと。ヘッダーに次の更新までのカウントダウンが出る
- **バックグラウンドでは止まる** — ターミナルがフォーカスを失うとポーリングを停止 (`⏸ 休憩中`)
- **マウス対応** — クリックで選択、ホイールでスクロール
- **ライト / ダーク自動切り替え** — 端末の背景色を検出してパレットを組み替える
- **英語 / 日本語** — 既定は英語。`--lang ja` で UI 全体が日本語になる
- **`gh` の認証を使い回す** — トークンの設定は不要

## インストール

`gh` CLI がログイン済みであることが前提 (`gh auth login`)。prpr はその認証情報をそのまま使う。

```bash
go install github.com/kanywst/prpr@latest
```

[Releases](https://github.com/kanywst/prpr/releases/latest) からビルド済みバイナリを落としてもいい。

## 使い方

```bash
prpr
```

オプション:

| フラグ | 既定値 | 説明 |
| --- | --- | --- |
| `--owner` | 自動検出 | 監視する owner。繰り返し・カンマ区切り可 |
| `--interval` | `1m` | 自動更新の間隔 (最小 `5s`) |
| `--timeout` | `20s` | 1 回の更新のタイムアウト |
| `--lang` | `en` | UI の言語: `en` か `ja` |
| `--demo` | off | GitHub ではなく固定のサンプルデータで動かす |
| `--version` | | バージョンを表示して終了 |

例:

```bash
# org をひとつだけ見る
prpr --owner 0-draft

# 30 秒ごとに更新
prpr --interval 30s

# 日本語 UI
prpr --lang ja
```

### キーバインド

| キー | 動作 |
| --- | --- |
| `↑` / `k`, `↓` / `j` | カーソル移動 |
| `g` / `G` | 先頭 / 末尾 |
| `pgup` / `pgdn` | ページ送り |
| `tab` / `shift+tab` | タブ切り替え |
| `enter` / `o` | ブラウザで開く |
| `y` | URL をクリップボードにコピー |
| `d` | 詳細ペインの開閉 |
| `ctrl+u` / `ctrl+d` | 詳細ペインをスクロール |
| `r` | 今すぐ更新 |
| `/` | 絞り込み |
| `esc` | 絞り込み解除 |
| `?` | ヘルプの詳細表示 |
| `ctrl+z` | 一時停止 |
| `q` / `ctrl+c` | 終了 |

### アイコン

| アイコン | 意味 |
| --- | --- |
| 🟢 / 🟡 / 🔴 / ⚪ | CI 通過 / 実行中 / 失敗 / なし |
| 📝 | 下書き |
| ✅ / 🔁 / 👀 | 承認済み / 変更依頼 / レビュー待ち |

## 構成

| パッケージ | 役割 |
| --- | --- |
| `internal/gh` | ドメイン型 (`PR`) と GitHub GraphQL アダプタ |
| `internal/browser` | URL を開く OS 依存部分 |
| `internal/demo` | `--demo` が使う固定データ |
| `internal/ui` | Bubble Tea の MVU。純粋関数と状態を分離 |
| `main.go` | フラグ解析と配線 |

インターフェース (`ui.Fetcher`) は利用側で定義してあるので、テストは GitHub に触らずにモデル全体を駆動できる。

[Bubble Tea v2](https://github.com/charmbracelet/bubbletea) / [Bubbles v2](https://github.com/charmbracelet/bubbles) / [Lip Gloss v2](https://github.com/charmbracelet/lipgloss) / [go-gh](https://github.com/cli/go-gh) の上に乗っている。

## 開発

```bash
make check   # fmt + vet + lint + test
make test    # race 付きテスト
make build   # ./bin/prpr
make demo    # VHS で docs/demo.gif を録り直す
make help    # ターゲット一覧
```

レンダリング結果を目で見たいときはこれ:

```bash
go test ./internal/ui -run TestDumpRender -v
```

## ライセンス

[MIT](LICENSE)
