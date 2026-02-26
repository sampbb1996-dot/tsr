package main

import (
        "fmt"
        "net/http"
        "os"
        "path/filepath"
        "strings"
)

func installScript(host string) string {
        return `#!/bin/sh
set -e

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)          ARCH="amd64" ;;
  aarch64|arm64)   ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

URL="https://` + host + `/releases/tsr-${OS}-${ARCH}"
DEST="$HOME/.local/bin/tsr"
mkdir -p "$HOME/.local/bin"

echo "Downloading TSR for ${OS}/${ARCH}..."
curl -fsSL "$URL" -o "$DEST"
chmod +x "$DEST"

echo ""
echo "Installed to $DEST"
echo "Make sure $HOME/.local/bin is in your PATH:"
echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
echo ""
echo "Then run: tsr run <yourfile.tsr>"
`
}

func landingPage(host string) string {
        installCmd := "curl -fsSL https://" + host + "/install.sh | sh"
        return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>TSR — Governed Programming Language</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: #0d1117; color: #e6edf3; line-height: 1.7; }
  .hero { max-width: 760px; margin: 0 auto; padding: 64px 24px 48px; }
  h1 { font-size: 2.4rem; font-weight: 700; letter-spacing: -0.5px; margin-bottom: 12px; }
  h1 span { color: #58a6ff; }
  .subtitle { font-size: 1.1rem; color: #8b949e; margin-bottom: 40px; max-width: 560px; }
  .badges { display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 40px; }
  .badge { background: #161b22; border: 1px solid #30363d; border-radius: 6px; padding: 5px 12px; font-size: 0.82rem; color: #8b949e; }
  .badge strong { color: #e6edf3; }
  .install-box { background: #161b22; border: 1px solid #388bfd55; border-radius: 10px; padding: 24px 28px; margin-bottom: 48px; }
  .install-box .label { font-size: 0.8rem; color: #8b949e; text-transform: uppercase; letter-spacing: 0.08em; margin-bottom: 10px; }
  .install-box pre { background: #0d1117; border: 1px solid #30363d; border-radius: 6px; padding: 14px 18px; font-size: 0.95rem; color: #79c0ff; margin: 0; user-select: all; }
  .install-box .note { font-size: 0.82rem; color: #484f58; margin-top: 10px; }
  .section { max-width: 760px; margin: 0 auto; padding: 0 24px 48px; }
  h2 { font-size: 1.3rem; font-weight: 600; margin-bottom: 16px; color: #e6edf3; border-bottom: 1px solid #21262d; padding-bottom: 8px; }
  pre { background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 20px; overflow-x: auto; font-size: 0.87rem; color: #e6edf3; margin-bottom: 24px; line-height: 1.6; }
  .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 16px; margin-bottom: 24px; }
  .card { background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 20px; }
  .card h3 { font-size: 0.95rem; font-weight: 600; color: #58a6ff; margin-bottom: 8px; }
  .card p { font-size: 0.87rem; color: #8b949e; }
  .platforms { display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 24px; }
  .platform { background: #161b22; border: 1px solid #30363d; border-radius: 6px; padding: 6px 14px; font-size: 0.83rem; color: #8b949e; text-decoration: none; }
  .platform:hover { border-color: #58a6ff; color: #e6edf3; }
  .source-link { color: #58a6ff; text-decoration: none; font-size: 0.9rem; }
  footer { text-align: center; padding: 40px 24px; color: #484f58; font-size: 0.85rem; border-top: 1px solid #21262d; }
</style>
</head>
<body>
<div class="hero">
  <h1>TSR — <span>Governed</span> Programming</h1>
  <p class="subtitle">A scripting language with runtime enforcement of regimes, strain budgets, and capability declarations. Every irreversible operation is explicit, auditable, and bounded.</p>
  <div class="badges">
    <span class="badge"><strong>macOS</strong> &amp; Linux</span>
    <span class="badge"><strong>Single binary</strong> · no runtime</span>
    <span class="badge"><strong>KERNEL</strong> collapse compiler</span>
  </div>

  <div class="install-box">
    <div class="label">Install (macOS &amp; Linux)</div>
    <pre>` + installCmd + `</pre>
    <div class="note">Installs to ~/.local/bin/tsr &mdash; works on macOS (Intel &amp; Apple Silicon) and Linux</div>
  </div>
</div>

<div class="section">
  <h2>After Installing</h2>
<pre>tsr run examples/00_hello.tsr
tsr run --trace examples/07_commit_write_ok.tsr
tsr compile examples/30_context_prod.krn | tsr run /dev/stdin</pre>

  <h2>Direct Downloads</h2>
  <div class="platforms">
    <a class="platform" href="/releases/tsr-linux-amd64">Linux x86_64</a>
    <a class="platform" href="/releases/tsr-linux-arm64">Linux ARM64</a>
    <a class="platform" href="/releases/tsr-darwin-amd64">macOS Intel</a>
    <a class="platform" href="/releases/tsr-darwin-arm64">macOS Apple Silicon</a>
    <a class="platform" href="/releases/tsr-windows-amd64.exe">Windows x64</a>
  </div>
  <p style="margin-bottom:32px"><a class="source-link" href="/download">Download source (ZIP)</a></p>

  <h2>What is TSR?</h2>
  <div class="grid">
    <div class="card">
      <h3>Regimes</h3>
      <p>CALM requires all writes and HTTP calls inside a <code>commit</code> block. READONLY forbids them entirely.</p>
    </div>
    <div class="card">
      <h3>Strain Budgets</h3>
      <p>Bound complexity at runtime — max branches, loops, call depth, and commit entries.</p>
    </div>
    <div class="card">
      <h3>Capabilities</h3>
      <p>Restrict which file paths or hostnames an operation may touch, enforced at runtime.</p>
    </div>
    <div class="card">
      <h3>KERNEL Compiler</h3>
      <p>Write high-level <code>.krn</code> intent files. KERNEL collapses them into fully explicit, auditable TSR.</p>
    </div>
  </div>

  <h2>TSR Example</h2>
<pre>regime "CALM";
capability file_write "logs/*";

commit "write-log" {
  write_file("logs/app.txt", "operation recorded");
}
say("Done.");</pre>
</div>

<footer>TSR Language &mdash; curl installer for macOS &amp; Linux &mdash; Windows: download binary above</footer>
</body>
</html>`
}

func main() {
        http.HandleFunc("/install.sh", func(w http.ResponseWriter, r *http.Request) {
                host := r.Host
                w.Header().Set("Content-Type", "text/plain; charset=utf-8")
                fmt.Fprint(w, installScript(host))
        })

        http.HandleFunc("/releases/", func(w http.ResponseWriter, r *http.Request) {
                name := strings.TrimPrefix(r.URL.Path, "/releases/")
                path := filepath.Join("releases", filepath.Base(name))
                data, err := os.ReadFile(path)
                if err != nil {
                        http.Error(w, "binary not found", 404)
                        return
                }
                w.Header().Set("Content-Type", "application/octet-stream")
                w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(name)))
                w.Write(data)
        })

        http.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
                data, err := os.ReadFile("tsr-lang.zip")
                if err != nil {
                        http.Error(w, "zip not found", 500)
                        return
                }
                w.Header().Set("Content-Type", "application/zip")
                w.Header().Set("Content-Disposition", `attachment; filename="tsr-lang.zip"`)
                w.Write(data)
        })

        http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
                host := r.Host
                w.Header().Set("Content-Type", "text/html; charset=utf-8")
                fmt.Fprint(w, landingPage(host))
        })

        port := os.Getenv("PORT")
        if port == "" {
                port = "5000"
        }
        fmt.Println("TSR server running on :" + port)
        http.ListenAndServe("0.0.0.0:"+port, nil)
}
