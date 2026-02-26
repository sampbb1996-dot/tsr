# TSR — A Governed Programming Language

TSR is a deterministic scripting language designed for fine-grained execution governance. It enforces **regimes** (CALM or READONLY), **strain budgets** (complexity limits), and **capability declarations** (resource restrictions).

---

## Quick Start

Requires [Go 1.21+](https://go.dev/dl/) installed on your machine.

**Step 1** — Download `tsr-lang.zip` from the website and unzip it

**Step 2** — Open a terminal inside the unzipped folder

**Step 3** — Run:
```bash
go build -o tsr ./cmd/tsr
./tsr run examples/00_hello.tsr
./tsr run --trace examples/07_commit_write_ok.tsr
```

---

## Core Governance Concepts

### 1. Regimes
- **CALM** (Default): Irreversible operations (file writes, HTTP) are allowed but **must** be inside a `commit` block.
- **READONLY**: Irreversible operations are strictly forbidden, even inside a `commit`.

### 2. Strain Budgets
Limit execution complexity to prevent resource exhaustion or infinite loops:
- `max_branch`: If/Else condition evaluations
- `max_loop`: While loop evaluations
- `max_commit`: Total commit block entries
- `max_call_depth`: Function recursion/stack depth

### 3. Capabilities
Restrict which specific resources an operation can touch using pattern matching:
```tsr
capability file_write "logs/*";   # Only allow writes to the logs folder
capability net "*.internal";      # Only allow HTTP to internal hosts
```

---

## KERNEL Compiler

KERNEL is a "collapse compiler" that translates high-level `.krn` files into auditable TSR. It automatically handles `commit` wrapping and budget calculation.

### Policy-Driven Compilation
Use a `context` block to let the policy engine determine governance automatically:
```krn
context {
  env: "prod"
  risk: "low"
}
budget auto

write_file "logs/app.log" "Starting..."
```

### Compile and Run:
```bash
./tsr compile --trace-policy examples/30_context_prod.krn > prod.tsr
./tsr run prod.tsr
```

---

## Language Reference

### TSR Syntax
```tsr
let x = 42;
fn greet(name) { return "Hello, " + name; }

regime "CALM";
capability file_write "output.txt";

commit "save" {
  write_file("output.txt", greet("User"));
}
```

### Standard Library
- `say(x)`: Print to stdout
- `read_file(path)`: Returns string content
- `write_file(path, text)`: (Governed)
- `http_get(url)`: (Governed)
- `json_parse(str)` / `json_stringify(val)`
- `now_ms()` / `sleep_ms(n)`

---

## Project Structure
- `cmd/tsr/`: CLI entry point
- `internal/tsr/`: Runtime, lexer, parser, and governance engine
- `internal/kernel/`: KERNEL compiler and policy integration
- `examples/`: Comprehensive library of `.tsr` and `.krn` files
- `tests/`: Full suite of governance and execution tests
