package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ANSI styling (matches opencode's CLI style)
const (
	STYLE_RESET       = "\x1b[0m"
	STYLE_BOLD        = "\x1b[1m"
	STYLE_DIM         = "\x1b[2m"
	STYLE_CYAN        = "\x1b[96m"
	STYLE_CYAN_BOLD   = "\x1b[96m\x1b[1m"
	STYLE_GRAY        = "\x1b[90m"
	STYLE_GRAY_BOLD   = "\x1b[90m\x1b[1m"
	STYLE_GREEN       = "\x1b[92m"
	STYLE_GREEN_BOLD  = "\x1b[92m\x1b[1m"
	STYLE_YELLOW      = "\x1b[93m"
	STYLE_YELLOW_BOLD = "\x1b[93m\x1b[1m"
	STYLE_RED         = "\x1b[91m"
	STYLE_RED_BOLD    = "\x1b[91m\x1b[1m"
	STYLE_BLUE        = "\x1b[94m"
	STYLE_BLUE_BOLD   = "\x1b[94m\x1b[1m"
)

// Short aliases used by the TUI
const (
	RESET = STYLE_RESET
	BOLD  = STYLE_BOLD
	DIM   = STYLE_DIM
	CYAN  = STYLE_CYAN
	GREEN = STYLE_GREEN_BOLD
	RED   = STYLE_RED_BOLD
	GRAY  = STYLE_GRAY
)

func ok(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, STYLE_GREEN_BOLD+"  ✔ "+STYLE_RESET+msg+"\n", args...)
}

func info(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, STYLE_CYAN+"  ∙ "+STYLE_RESET+msg+"\n", args...)
}

func dimLine(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, STYLE_GRAY+"  "+msg+STYLE_RESET+"\n", args...)
}

func fatal(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, STYLE_RED_BOLD+"  ✘ "+STYLE_RESET+msg+"\n", args...)
	os.Exit(1)
}

func dimInline(s string) string {
	return STYLE_GRAY + s + STYLE_RESET
}

func cyanBold(s string) string {
	return STYLE_CYAN_BOLD + s + STYLE_RESET
}

func greenBold(s string) string {
	return STYLE_GREEN_BOLD + s + STYLE_RESET
}

var wordmark = []string{
	"  ╔══════════════════════════════════════╗",
	"  ║        VectorDB  ·  HNSW CLI        ║",
	"  ╚══════════════════════════════════════╝",
}

func logo() {
	for _, line := range wordmark {
		fmt.Fprintf(os.Stderr, STYLE_CYAN+"%s"+STYLE_RESET+"\n", line)
	}
	fmt.Fprintln(os.Stderr)
}

func parseFloats(args []string) []float64 {
	vals := make([]float64, len(args))
	for i, s := range args {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			fatal("invalid float %q at position %d", s, i)
		}
		vals[i] = v
	}
	return vals
}

func usage() {
	logo()
	fmt.Fprintf(os.Stderr, STYLE_BOLD+"Usage:"+STYLE_RESET+"\n")
	fmt.Fprintf(os.Stderr, "  %s <command> [args...]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, STYLE_BOLD+"Commands:"+STYLE_RESET+"\n")
	fmt.Fprintf(os.Stderr, "  %s  %s\n", cyanBold("init   <dim> [dist] [max]"), dimInline("Create a new index"))
	fmt.Fprintf(os.Stderr, "  %s  %s\n", cyanBold("insert <id> <val1> <val2> ..."), dimInline("Insert a vector"))
	fmt.Fprintf(os.Stderr, "  %s  %s\n", cyanBold("search <val1> ... [-k N] [-ef N]"), dimInline("k-nearest neighbor search"))
	fmt.Fprintf(os.Stderr, "  %s  %s\n", cyanBold("size"), dimInline("Show element count"))
	fmt.Fprintf(os.Stderr, "  %s  %s\n", cyanBold("save   [path]"), dimInline("Save index to file"))
	fmt.Fprintf(os.Stderr, "  %s  %s\n", cyanBold("load   [path] [max]"), dimInline("Load index from file"))
	fmt.Fprintf(os.Stderr, "  %s  %s\n", cyanBold("insert-text <id> <text>"), dimInline("Insert text (embeds with Qwen3)"))
	fmt.Fprintf(os.Stderr, "  %s  %s\n", cyanBold("search-text <text> [-k N] [-ef N]"), dimInline("Search with text"))
	fmt.Fprintf(os.Stderr, "  %s  %s\n", cyanBold("ask <question> [-k N]"), dimInline("Ask a question (RAG: retrieve + LLM answer)"))
	fmt.Fprintf(os.Stderr, "  %s  %s\n", cyanBold("tui"), dimInline("Interactive TUI mode"))
	fmt.Fprintf(os.Stderr, "  %s  %s\n\n", cyanBold("help"), dimInline("Show this help"))
	fmt.Fprintf(os.Stderr, STYLE_DIM+"Distance types: l2 (default), ip     Default index: %s"+STYLE_RESET+"\n", defaultIndexPath)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "help", "--help", "-h":
		usage()
		return

	case "init":
		if len(args) < 1 {
			fatal("init requires a dimension argument")
		}
		dim, err := strconv.Atoi(args[0])
		if err != nil || dim <= 0 {
			fatal("invalid dimension %q", args[0])
		}
		distType := "l2"
		maxElements := 100000
		if len(args) > 1 {
			distType = args[1]
		}
		if len(args) > 2 {
			maxElements, err = strconv.Atoi(args[2])
			if err != nil || maxElements <= 0 {
				fatal("invalid max_elements %q", args[2])
			}
		}

		var cDist int
		switch distType {
		case "l2":
			cDist = 	VDB_HNSW_L2
		case "ip":
			cDist = VDB_HNSW_IP
		default:
			fatal("unknown distance type %q (use l2 or ip)", distType)
		}

		h := NewHNSW(dim, cDist, maxElements)
		defer h.Destroy()

		if h.Save(defaultIndexPath) {
			ok("initialized index  %s  dim=%d  dist=%s  max=%d",
				dimInline(defaultIndexPath), dim, distType, maxElements)
		} else {
			fatal("failed to save initial index")
		}

	case "insert":
		if len(args) < 2 {
			fatal("insert requires an id and at least one value")
		}
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			fatal("invalid id %q", args[0])
		}
		vals := parseFloats(args[1:])
		dim := len(vals)

		h := NewHNSW(dim, 	VDB_HNSW_L2, 100000)
		defer h.Destroy()

		if h.Load(defaultIndexPath, 0) {
			if h.dim != dim {
				fatal("dimension mismatch: index expects %d, vector has %d", h.dim, dim)
			}
		}

		if h.Insert(id, vals) {
			if h.Save(defaultIndexPath) {
				ok("inserted  id=%d  dim=%d", id, dim)
			} else {
				fatal("insert succeeded but save failed")
			}
		} else {
			fatal("insert failed (duplicate id %d?)", id)
		}

	case "search":
		if len(args) < 1 {
			fatal("search requires query values")
		}
		k := 10
		ef := 100
		var queryArgs []string
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "-k":
				if i+1 < len(args) {
					i++
					k, _ = strconv.Atoi(args[i])
				}
			case "-ef":
				if i+1 < len(args) {
					i++
					ef, _ = strconv.Atoi(args[i])
				}
			default:
				queryArgs = append(queryArgs, args[i])
			}
		}
		if k <= 0 {
			k = 10
		}
		if ef <= 0 {
			ef = 100
		}

		vals := parseFloats(queryArgs)
		dim := len(vals)

		h := NewHNSW(dim, 	VDB_HNSW_L2, 100000)
		defer h.Destroy()

		if !h.Load(defaultIndexPath, 0) {
			fatal("no index found at %s (use 'init' first)", defaultIndexPath)
		}

		ids, scores := h.Search(vals, k, ef)
		if len(ids) == 0 {
			info("no results found  ef=%d", ef)
			return
		}

		info("nearest neighbors  k=%d  ef=%d", len(ids), ef)
		fmt.Fprintln(os.Stderr)
		for i := range ids {
			rank := fmt.Sprintf("%d.", i+1)
			idStr := fmt.Sprintf("%d", ids[i])
			bar := scoreBar(scores[i], ids, scores)
			fmt.Fprintf(os.Stderr, "  %s %s  score=%s  %s\n",
				dimInline(rank),
				cyanBold(idStr),
				fmt.Sprintf("%.6f", scores[i]),
				bar,
			)
		}

	case "size":
		dim := 1
		h := NewHNSW(dim, 	VDB_HNSW_L2, 100000)
		defer h.Destroy()

		if !h.Load(defaultIndexPath, 0) {
			fmt.Println("0")
			return
		}
		n := h.Size()
		if n == 0 {
			fmt.Fprintln(os.Stderr, "0")
		} else {
			ok("index has %d vector(s)", n)
		}

	case "save":
		path := defaultIndexPath
		if len(args) > 0 {
			path = args[0]
		}
		dim := 1
		h := NewHNSW(dim, 	VDB_HNSW_L2, 100000)
		defer h.Destroy()

		if !h.Load(defaultIndexPath, 0) {
			fatal("no index to save")
		}

		if h.Save(path) {
			ok("saved index  %s  (%d vectors)", dimInline(path), h.Size())
		} else {
			fatal("save to %s failed", path)
		}

	case "load":
		path := defaultIndexPath
		maxElements := 0
		if len(args) > 0 {
			path = args[0]
		}
		if len(args) > 1 {
			maxElements, _ = strconv.Atoi(args[1])
		}
		dim := 1
		h := NewHNSW(dim, 	VDB_HNSW_L2, 100000)
		defer h.Destroy()

		if h.Load(path, maxElements) {
			ok("loaded index  %s  (%d vectors)", dimInline(path), h.Size())
		} else {
			fatal("load from %s failed", path)
		}

	case "insert-text":
		if len(args) < 2 {
			fatal("usage: insert-text <id> <text>")
		}
		id, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			fatal("invalid id %q", args[0])
		}
		text := strings.Join(args[1:], " ")
		if text == "" {
			fatal("text cannot be empty")
		}

		ec, err := NewEmbedClient()
		if err != nil {
			fatal("embed server: %v", err)
		}
		defer ec.Close()

		vec, err := ec.EmbedText(text)
		if err != nil {
			fatal("embedding failed: %v", err)
		}
		dim := len(vec)

		h := NewHNSW(dim, VDB_HNSW_L2, 100000)
		defer h.Destroy()

		if h.Load(defaultIndexPath, 0) {
			if h.dim != dim {
				fatal("dimension mismatch: index expects %d, embedding has %d", h.dim, dim)
			}
		}

		if h.Insert(id, vec) {
			if h.Save(defaultIndexPath) {
				ts := NewTextStore("data/texts.json")
				ts.Set(id, text)
				ok("inserted  id=%d  dim=%d  text=%q", id, dim, text)
			} else {
				fatal("insert succeeded but save failed")
			}
		} else {
			fatal("insert failed (duplicate id %d?)", id)
		}

	case "ask":
		if len(args) < 1 {
			fatal("usage: ask <question> [-k N]")
		}
		k := 5
		var questionParts []string
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "-k":
				if i+1 < len(args) {
					i++
					k, _ = strconv.Atoi(args[i])
				}
			default:
				questionParts = append(questionParts, args[i])
			}
		}
		if k <= 0 {
			k = 5
		}
		question := strings.Join(questionParts, " ")

		ec, err := NewEmbedClient()
		if err != nil {
			fatal("embed server: %v", err)
		}
		defer ec.Close()

		vec, err := ec.EmbedText(question)
		if err != nil {
			fatal("embedding failed: %v", err)
		}

		dim := len(vec)
		h := NewHNSW(dim, VDB_HNSW_L2, 100000)
		defer h.Destroy()

		if !h.Load(defaultIndexPath, 0) {
			fatal("no index found at %s (use 'init' first)", defaultIndexPath)
		}

		ids, scores := h.Search(vec, k, 200)
		if len(ids) == 0 {
			fatal("no relevant documents found")
		}

		ts := NewTextStore("data/texts.json")
		texts := ts.GetBatch(ids)

		var contextTexts []string
		for _, id := range ids {
			if t, ok := texts[id]; ok {
				contextTexts = append(contextTexts, t)
			}
		}

		if len(contextTexts) == 0 {
			fatal("no text content found for the matching vectors")
		}

		info("retrieved %d documents for: %q", len(contextTexts), question)
		fmt.Fprintln(os.Stderr)
		for i := range ids {
			idStr := fmt.Sprintf("%d", ids[i])
			bar := scoreBar(scores[i], ids, scores)
			text := dimInline("(no text)")
			if t, ok := texts[ids[i]]; ok {
				text = t
			}
			fmt.Fprintf(os.Stderr, "  %s %s  %s  %s\n",
				dimInline(fmt.Sprintf("%d.", i+1)),
				cyanBold(idStr),
				bar,
				text,
			)
		}
		fmt.Fprintln(os.Stderr)

		rc := NewRAGClient("")
		answer, err := rc.Ask(question, contextTexts)
		if err != nil {
			fatal("LLM request failed: %v", err)
		}

		fmt.Fprintf(os.Stderr, "\n%s%s%s\n", STYLE_BOLD, answer, STYLE_RESET)

	case "search-text":
		if len(args) < 1 {
			fatal("usage: search-text <text> [-k N] [-ef N]")
		}
		k := 10
		ef := 100
		var textParts []string
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "-k":
				if i+1 < len(args) {
					i++
					k, _ = strconv.Atoi(args[i])
				}
			case "-ef":
				if i+1 < len(args) {
					i++
					ef, _ = strconv.Atoi(args[i])
				}
			default:
				textParts = append(textParts, args[i])
			}
		}
		if k <= 0 {
			k = 10
		}
		if ef <= 0 {
			ef = 100
		}
		text := strings.Join(textParts, " ")

		ec, err := NewEmbedClient()
		if err != nil {
			fatal("embed server: %v", err)
		}
		defer ec.Close()

		vec, err := ec.EmbedText(text)
		if err != nil {
			fatal("embedding failed: %v", err)
		}

		dim := len(vec)
		h := NewHNSW(dim, VDB_HNSW_L2, 100000)
		defer h.Destroy()

		if !h.Load(defaultIndexPath, 0) {
			fatal("no index found at %s (use 'init' first)", defaultIndexPath)
		}

		ids, scores := h.Search(vec, k, ef)
		if len(ids) == 0 {
			info("no results found  ef=%d", ef)
			return
		}

		info("nearest neighbors  k=%d  ef=%d  text=%q", len(ids), ef, text)
		fmt.Fprintln(os.Stderr)
		for i := range ids {
			rank := fmt.Sprintf("%d.", i+1)
			idStr := fmt.Sprintf("%d", ids[i])
			bar := scoreBar(scores[i], ids, scores)
			fmt.Fprintf(os.Stderr, "  %s %s  score=%s  %s\n",
				dimInline(rank),
				cyanBold(idStr),
				fmt.Sprintf("%.6f", scores[i]),
				bar,
			)
		}

	case "tui":
		startTUI()

	default:
		fatal("unknown command %q", cmd)
	}
}


