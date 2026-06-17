package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

var commands = []string{
	"/help",
	"/init",
	"/insert",
	"/insert-text",
	"/ask",
	"/search",
	"/search-text",
	"/size",
	"/save",
	"/load",
	"/clear",
	"/quit",
}

type model struct {
	textInput textinput.Model
	viewport  viewport.Model
	messages  []string
	index     *HNSW
	embed     *EmbedClient
	dim       int
	distType  int
	ready     bool
	width     int
	height    int
	history   []string
	histPos   int
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "/help for commands"
	ti.Prompt = CYAN + "─ " + RESET
	ti.CharLimit = 512
	ti.Width = 60
	ti.Focus()

	return model{
		textInput: ti,
		messages:  []string{},
		index:     nil,
		dim:       0,
		distType:  VDB_HNSW_L2,
		history:   []string{},
		histPos:   -1,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 2
		footerHeight := 3
		vpHeight := m.height - headerHeight - footerHeight
		if vpHeight < 1 {
			vpHeight = 1
		}
		if !m.ready {
			m.viewport = viewport.New(m.width, vpHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.KeyMap = viewport.DefaultKeyMap()
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = vpHeight
		}
		m.textInput.Width = m.width - 6
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d":
			saveIndex(m)
			return m, tea.Quit

		case "enter":
			input := strings.TrimSpace(m.textInput.Value())
			if input == "" {
				return m, nil
			}
			m.history = append(m.history, input)
			m.histPos = -1
			m.textInput.SetValue("")
			messages := m.executeCommand(input)
			for _, msg := range messages {
				m = m.addMessage(msg)
			}
			m.viewport.GotoBottom()
			return m, nil

		case "up":
			if len(m.history) > 0 {
				if m.histPos == -1 {
					m.histPos = len(m.history) - 1
				} else if m.histPos > 0 {
					m.histPos--
				}
				m.textInput.SetValue(m.history[m.histPos])
			}
			return m, nil

		case "down":
			if m.histPos >= 0 {
				m.histPos++
				if m.histPos >= len(m.history) {
					m.histPos = -1
					m.textInput.SetValue("")
				} else {
					m.textInput.SetValue(m.history[m.histPos])
				}
			}
			return m, nil

		case "tab":
			val := m.textInput.Value()
			if !strings.HasPrefix(val, "/") {
				return m, nil
			}
			parts := strings.Fields(val)
			if len(parts) == 1 || (len(parts) == 1 && !strings.HasSuffix(val, " ")) {
				prefix := strings.TrimPrefix(parts[0], "/")
				for _, cmd := range commands {
					trimmed := strings.TrimPrefix(cmd, "/")
					if strings.HasPrefix(trimmed, prefix) && cmd != parts[0] {
						m.textInput.SetValue(cmd + " ")
						break
					}
				}
			}
			return m, nil
		}
	}

	m.textInput, tiCmd = m.textInput.Update(msg)
	if m.ready {
		m.viewport, vpCmd = m.viewport.Update(msg)
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	if !m.ready {
		return "\n  " + DIM + "initializing..." + RESET
	}

	var b strings.Builder

	b.WriteString(CYAN + "╭────────────────────────────────────────────────╮" + RESET + "\n")
	b.WriteString(CYAN + "│" + RESET + BOLD + "  VectorDB · HNSW TUI" + RESET)
	pad := m.width - 32
	if pad > 0 {
		b.WriteString(strings.Repeat(" ", pad))
	}
	b.WriteString(CYAN + "│" + RESET + "\n")
	b.WriteString(CYAN + "╰────────────────────────────────────────────────╯" + RESET + "\n")

	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	b.WriteString(GRAY + strings.Repeat("─", m.width) + RESET + "\n")
	b.WriteString(m.textInput.View())

	return b.String()
}

func (m model) addMessage(msg string) model {
	m.messages = append(m.messages, "  "+msg)
	m.viewport.SetContent(strings.Join(m.messages, "\n") + "\n")
	return m
}

func (m *model) setMessages(msgs []string) {
	m.messages = msgs
	m.viewport.SetContent(strings.Join(m.messages, "\n"))
}

func (m model) executeCommand(input string) []string {
	input = strings.TrimSpace(input)

	if !strings.HasPrefix(input, "/") {
		return []string{DIM + "unknown command (type /help)" + RESET}
	}

	parts := parseArgs(input)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "/help":
		return helpText()

	case "/init":
		return m.cmdInit(args)

	case "/insert":
		return m.cmdInsert(args)

	case "/insert-text":
		return m.cmdInsertText(args)

	case "/ask":
		return m.cmdAsk(args)

	case "/search":
		return m.cmdSearch(args)

	case "/search-text":
		return m.cmdSearchText(args)

	case "/size":
		return m.cmdSize()

	case "/save":
		return m.cmdSave(args)

	case "/load":
		return m.cmdLoad(args)

	case "/clear":
		m.setMessages([]string{})
		return nil

	case "/quit", "/exit":
		saveIndex(m)
		os.Exit(0)
		return nil

	default:
		return []string{RED + "unknown command: " + input + RESET}
	}
}

func parseArgs(input string) []string {
	var parts []string
	current := strings.Builder{}
	inQuote := false
	for _, ch := range input {
		switch {
		case ch == '"':
			inQuote = !inQuote
		case ch == ' ' && !inQuote:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func helpText() []string {
	return []string{
		CYAN + BOLD + "Commands:" + RESET,
		"  " + CYAN + "/init <dim> [l2|ip] [max]" + RESET + "        " + DIM + "create a new index" + RESET,
		"  " + CYAN + "/insert <id> <val1> ..." + RESET + "            " + DIM + "insert a vector" + RESET,
		"  " + CYAN + "/insert-text <id> <text>" + RESET + "           " + DIM + "insert text (Qwen3 embed)" + RESET,
		"  " + CYAN + "/ask <question> [-k N]" + RESET + "              " + DIM + "ask a question (RAG with LLM)" + RESET,
		"  " + CYAN + "/search <val1> ... [-k N] [-ef N]" + RESET + "  " + DIM + "search k-nearest neighbors" + RESET,
		"  " + CYAN + "/search-text <text> [-k N]" + RESET + "         " + DIM + "search with text" + RESET,
		"  " + CYAN + "/size" + RESET + "                            " + DIM + "show element count" + RESET,
		"  " + CYAN + "/save [path]" + RESET + "                     " + DIM + "save index to file" + RESET,
		"  " + CYAN + "/load [path] [max]" + RESET + "               " + DIM + "load index from file" + RESET,
		"  " + CYAN + "/clear" + RESET + "                           " + DIM + "clear output" + RESET,
		"  " + CYAN + "/help" + RESET + "                            " + DIM + "show this help" + RESET,
		"  " + CYAN + "/quit" + RESET + "                            " + DIM + "exit" + RESET,
		"",
		DIM + "Tab to autocomplete commands. ↑/↓ for history." + RESET,
	}
}

func (m model) cmdInit(args []string) []string {
	if len(args) < 1 {
		return []string{RED + "usage: /init <dim> [l2|ip] [max_elements]" + RESET}
	}
	dim, err := strconv.Atoi(args[0])
	if err != nil || dim <= 0 {
		return []string{RED + "invalid dimension: " + args[0] + RESET}
	}
	distType := "l2"
	maxElements := 100000
	if len(args) > 1 {
		distType = args[1]
	}
	if len(args) > 2 {
		maxElements, err = strconv.Atoi(args[2])
		if err != nil || maxElements <= 0 {
			return []string{RED + "invalid max_elements: " + args[2] + RESET}
		}
	}

	var cDist int
	switch distType {
	case "l2":
		cDist = VDB_HNSW_L2
	case "ip":
		cDist = VDB_HNSW_IP
	default:
		return []string{RED + "unknown distance type: " + distType + RESET}
	}

	if m.index != nil {
		m.index.Destroy()
	}

	m.index = NewHNSW(dim, cDist, maxElements)
	m.dim = dim
	m.distType = cDist

	if m.index.Save(defaultIndexPath) {
		return []string{
			GREEN + BOLD + "  ✔ " + RESET + "initialized index  " + DIM + defaultIndexPath + RESET + "  dim=" + strconv.Itoa(dim) + "  dist=" + distType + "  max=" + strconv.Itoa(maxElements),
		}
	}
	return []string{RED + "failed to save index" + RESET}
}

func (m model) cmdInsert(args []string) []string {
	if m.index == nil {
		return []string{RED + "no index loaded (use /init first)" + RESET}
	}
	if len(args) < 2 {
		return []string{RED + "usage: /insert <id> <val1> <val2> ..." + RESET}
	}
	id, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return []string{RED + "invalid id: " + args[0] + RESET}
	}
	vals := make([]float64, len(args)-1)
	for i, s := range args[1:] {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return []string{RED + "invalid float at position " + strconv.Itoa(i+1) + RESET}
		}
		vals[i] = v
	}

	if m.index.Insert(id, vals) {
		m.index.Save(defaultIndexPath)
		return []string{
			GREEN + BOLD + "  ✔ " + RESET + "inserted  " + CYAN + BOLD + strconv.FormatUint(id, 10) + RESET + "  dim=" + strconv.Itoa(len(vals)),
		}
	}
	return []string{RED + "insert failed (duplicate id " + strconv.FormatUint(id, 10) + "?)" + RESET}
}

func (m model) cmdInsertText(args []string) []string {
	if len(args) < 2 {
		return []string{RED + "usage: /insert-text <id> <text>" + RESET}
	}
	id, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		return []string{RED + "invalid id: " + args[0] + RESET}
	}
	text := strings.Join(args[1:], " ")
	if text == "" {
		return []string{RED + "text cannot be empty" + RESET}
	}

	ec, err := NewEmbedClient()
	if err != nil {
		return []string{RED + "embed server not available: " + err.Error() + RESET}
	}
	defer ec.Close()

	vec, err := ec.EmbedText(text)
	if err != nil {
		return []string{RED + "embedding failed: " + err.Error() + RESET}
	}

	dim := len(vec)
	h := NewHNSW(dim, VDB_HNSW_L2, 100000)
	defer h.Destroy()

	if h.Load(defaultIndexPath, 0) {
		if h.dim != dim {
			return []string{RED + "dimension mismatch: index expects " + strconv.Itoa(h.dim) + ", embedding has " + strconv.Itoa(dim) + RESET}
		}
	}

	if h.Insert(id, vec) {
		h.Save(defaultIndexPath)
		return []string{
			GREEN + BOLD + "  ✔ " + RESET + "inserted  " + CYAN + BOLD + strconv.FormatUint(id, 10) + RESET + "  text=" + text + "  dim=" + strconv.Itoa(dim),
		}
	}
	return []string{RED + "insert failed (duplicate id " + strconv.FormatUint(id, 10) + "?)" + RESET}
}

func (m model) cmdSearchText(args []string) []string {
	if len(args) < 1 {
		return []string{RED + "usage: /search-text <text> [-k N]" + RESET}
	}
	k := 10
	var textParts []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-k":
			if i+1 < len(args) {
				i++
				k, _ = strconv.Atoi(args[i])
			}
		default:
			textParts = append(textParts, args[i])
		}
	}
	if k <= 0 {
		k = 10
	}
	text := strings.Join(textParts, " ")

	ec, err := NewEmbedClient()
	if err != nil {
		return []string{RED + "embed server not available: " + err.Error() + RESET}
	}
	defer ec.Close()

	vec, err := ec.EmbedText(text)
	if err != nil {
		return []string{RED + "embedding failed: " + err.Error() + RESET}
	}

	dim := len(vec)
	h := NewHNSW(dim, VDB_HNSW_L2, 100000)
	defer h.Destroy()

	if !h.Load(defaultIndexPath, 0) {
		return []string{RED + "no index found (use /init first)" + RESET}
	}

	ids, scores := h.Search(vec, k, 100)
	if len(ids) == 0 {
		return []string{CYAN + "  ∙ " + RESET + "no results found"}
	}

	lines := []string{
		CYAN + "  ∙ " + RESET + "nearest neighbors  k=" + strconv.Itoa(len(ids)) + "  text=" + text,
		"",
	}
	for i := range ids {
		rank := strconv.Itoa(i + 1)
		idStr := strconv.FormatUint(ids[i], 10)
		bar := scoreBar(scores[i], ids, scores)
		lines = append(lines, fmt.Sprintf("  %s %s  score=%.6f  %s",
			DIM+rank+"."+RESET,
			CYAN+BOLD+idStr+RESET,
			scores[i],
			bar,
		))
	}
	return lines
}

func (m model) cmdAsk(args []string) []string {
	if len(args) < 1 {
		return []string{RED + "usage: /ask <question> [-k N]" + RESET}
	}
	k := 5
	var qParts []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-k":
			if i+1 < len(args) {
				i++
				k, _ = strconv.Atoi(args[i])
			}
		default:
			qParts = append(qParts, args[i])
		}
	}
	if k <= 0 {
		k = 5
	}
	question := strings.Join(qParts, " ")

	ec, err := NewEmbedClient()
	if err != nil {
		return []string{RED + "embed server: " + err.Error() + RESET}
	}
	defer ec.Close()

	vec, err := ec.EmbedText(question)
	if err != nil {
		return []string{RED + "embedding failed: " + err.Error() + RESET}
	}

	dim := len(vec)
	h := NewHNSW(dim, VDB_HNSW_L2, 100000)
	defer h.Destroy()

	if !h.Load(defaultIndexPath, 0) {
		return []string{RED + "no index found (use /init first)" + RESET}
	}

	ids, scores := h.Search(vec, k, 200)
	if len(ids) == 0 {
		return []string{CYAN + "  ∙ " + RESET + "no relevant documents found"}
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
		return []string{RED + "no text content found for matching vectors" + RESET}
	}

	lines := []string{
		CYAN + "  ∙ " + RESET + "retrieved " + strconv.Itoa(len(contextTexts)) + " documents for: " + question,
		"",
	}
	for i := range ids {
		rank := strconv.Itoa(i + 1)
		idStr := strconv.FormatUint(ids[i], 10)
		bar := scoreBar(scores[i], ids, scores)
		text := "(no text)"
		if t, ok := texts[ids[i]]; ok {
			text = t
		}
		lines = append(lines, fmt.Sprintf("  %s %s  %s  %s",
			DIM+rank+"."+RESET,
			CYAN+BOLD+idStr+RESET,
			bar,
			text,
		))
	}
	lines = append(lines, "")

	rc := NewRAGClient("")
	answer, err := rc.Ask(question, contextTexts)
	if err != nil {
		lines = append(lines, RED+"LLM error: "+err.Error()+RESET)
	} else {
		lines = append(lines, BOLD+answer+RESET)
	}
	return lines
}

func (m model) cmdSearch(args []string) []string {
	if m.index == nil {
		return []string{RED + "no index loaded (use /init first)" + RESET}
	}
	if len(args) < 1 {
		return []string{RED + "usage: /search <val1> ... [-k N] [-ef N]" + RESET}
	}
	k := 10
	ef := 100
	var queryStrs []string
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
			queryStrs = append(queryStrs, args[i])
		}
	}
	if k <= 0 {
		k = 10
	}
	if ef <= 0 {
		ef = 100
	}

	vals := make([]float64, len(queryStrs))
	for i, s := range queryStrs {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return []string{RED + "invalid float at position " + strconv.Itoa(i) + RESET}
		}
		vals[i] = v
	}

	ids, scores := m.index.Search(vals, k, ef)
	if len(ids) == 0 {
		return []string{CYAN + "  ∙ " + RESET + "no results found  " + DIM + "ef=" + strconv.Itoa(ef) + RESET}
	}

	lines := []string{
		CYAN + "  ∙ " + RESET + "nearest neighbors  k=" + strconv.Itoa(len(ids)) + "  ef=" + strconv.Itoa(ef),
		"",
	}
	for i := range ids {
		rank := strconv.Itoa(i + 1)
		idStr := strconv.FormatUint(ids[i], 10)
		bar := scoreBar(scores[i], ids, scores)
		lines = append(lines, fmt.Sprintf("  %s %s  score=%.6f  %s",
			DIM+rank+"."+RESET,
			CYAN+BOLD+idStr+RESET,
			scores[i],
			bar,
		))
	}
	return lines
}

func (m model) cmdSize() []string {
	if m.index == nil {
		return []string{DIM + "no index loaded" + RESET}
	}
	n := m.index.Size()
	if n == 0 {
		return []string{DIM + "index is empty" + RESET}
	}
	return []string{
		GREEN + BOLD + "  ✔ " + RESET + "index has " + strconv.Itoa(n) + " vector(s)",
	}
}

func (m model) cmdSave(args []string) []string {
	if m.index == nil {
		return []string{RED + "no index to save" + RESET}
	}
	path := defaultIndexPath
	if len(args) > 0 {
		path = args[0]
	}
	if m.index.Save(path) {
		return []string{
			GREEN + BOLD + "  ✔ " + RESET + "saved index  " + DIM + path + RESET + "  (" + strconv.Itoa(m.index.Size()) + " vectors)",
		}
	}
	return []string{RED + "save to " + path + " failed" + RESET}
}

func (m model) cmdLoad(args []string) []string {
	path := defaultIndexPath
	maxElements := 0
	if len(args) > 0 {
		path = args[0]
	}
	if len(args) > 1 {
		maxElements, _ = strconv.Atoi(args[1])
	}

	// We need to know the dimension to load. Try to load with the current
	// dim, or default to reading the saved meta file.
	dim := m.dim
	if dim == 0 {
		dim = 1
	}

	if m.index != nil {
		m.index.Destroy()
	}
	m.index = NewHNSW(dim, VDB_HNSW_L2, 100000)

	if m.index.Load(path, maxElements) {
		m.dim = m.index.dim
		return []string{
			GREEN + BOLD + "  ✔ " + RESET + "loaded index  " + DIM + path + RESET + "  (" + strconv.Itoa(m.index.Size()) + " vectors)",
		}
	}
	m.index.Destroy()
	m.index = nil
	return []string{RED + "load from " + path + " failed (try /init first)" + RESET}
}

func saveIndex(m model) {
	if m.index != nil && m.index.Size() > 0 {
		m.index.Save(defaultIndexPath)
	}
}

func startTUI() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, RED+"tui error: %v"+RESET+"\n", err)
		os.Exit(1)
	}
}
