Profiling in Go is super useful for understanding where your program spends time, how many goroutines it has, memory usage, and more. Here’s a quick guide on how to use Go’s built-in profiling tools.

---

## 1. **Using the built-in `pprof` package**

### a) Add profiling endpoints in your HTTP server

If you have a web server (like one using Gorilla Mux), you can import the standard library’s `net/http/pprof` package to expose profiling data via HTTP.

```go
import (
    "net/http"
    _ "net/http/pprof"  // registers pprof handlers
)

func main() {
    // Your router setup, e.g., Gorilla Mux

    go func() {
        // pprof runs on this separate port
        http.ListenAndServe("localhost:6060", nil)
    }()

    // your app's main HTTP server...
}
```

* This sets up profiling endpoints like:

  * `http://localhost:6060/debug/pprof/`
  * `http://localhost:6060/debug/pprof/goroutine`
  * `http://localhost:6060/debug/pprof/heap`
  * `http://localhost:6060/debug/pprof/profile` (CPU profile)

---

### b) Collect a CPU profile

You can run a CPU profile for a few seconds by hitting:

```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

This collects a 30-second CPU profile and opens an interactive terminal UI.

---

### c) Analyze profiles

In the interactive `pprof` tool, you can use commands like:

* `top` — show the top functions consuming CPU time
* `list <function>` — see annotated source code for a function
* `web` — generate a graph (requires Graphviz)

---

## 2. **Using `pprof` with your binary**

If your program is not an HTTP server, you can add code to write profiles to files:

```go
import (
    "os"
    "runtime/pprof"
    "time"
)

func main() {
    f, _ := os.Create("cpu.prof")
    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()

    // Your program logic here

    time.Sleep(30 * time.Second) // simulate work
}
```

Then run:

```bash
go tool pprof cpu.prof
```

And analyze interactively.

---

## 3. **Inspect goroutine and heap profiles**

* To see currently running goroutines:

```bash
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

* To analyze heap allocations:

```bash
go tool pprof http://localhost:6060/debug/pprof/heap
```

---

## 4. **Visualizing profiles**

Inside the `pprof` interactive terminal, use:

```pprof
web
```

This opens a browser window with a flame graph visualization (Graphviz is required).

---

## 5. **Summary commands**

| Task                | Command                                                                 |
| ------------------- | ----------------------------------------------------------------------- |
| Start pprof HTTP    | Import `_ "net/http/pprof"` and run `http.ListenAndServe` on debug port |
| Collect CPU profile | `go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30`    |
| View goroutines     | `go tool pprof http://localhost:6060/debug/pprof/goroutine`             |
| View heap profile   | `go tool pprof http://localhost:6060/debug/pprof/heap`                  |

---

If you want, I can help you set up profiling in your current project or interpret a profile output. Want to try?
