package tui

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/i9va/tracelocal/internal/model"
	"github.com/i9va/tracelocal/internal/store"
)

const refreshInterval = 200 * time.Millisecond
const barWidth = 20

// servicePalette is the set of colors cycled through for service names.
// Chosen to be distinct and readable on dark terminals.
var servicePalette = []lipgloss.Color{
	"#3B82F6", // blue
	"#EC4899", // pink
	"#14B8A6", // teal
	"#F97316", // orange
	"#8B5CF6", // violet
	"#06B6D4", // cyan
	"#84CC16", // lime
	"#F472B6", // rose
}

// serviceColor maps a service name to a stable color from the palette.
func serviceColor(name string) lipgloss.Color {
	if name == "" {
		return clrLightGray
	}
	h := fnv.New32a()
	h.Write([]byte(name))
	return servicePalette[h.Sum32()%uint32(len(servicePalette))]
}

// serviceStyle returns a bold style using the service's stable palette color.
func serviceStyle(name string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(serviceColor(name)).Bold(true)
}

// Palette colors used across the TUI.
var (
	clrPurple    = lipgloss.Color("#7C3AED")
	clrPurpleDim = lipgloss.Color("#4C1D95")
	clrGreen     = lipgloss.Color("#10B981")
	clrRed       = lipgloss.Color("#F43F5E")
	clrYellow    = lipgloss.Color("#F59E0B")
	clrGray      = lipgloss.Color("#6B7280")
	clrLightGray = lipgloss.Color("#D1D5DB")
	clrWhite     = lipgloss.Color("#F9FAFB")
	clrRowHover  = lipgloss.Color("#1E1B4B")
)

// All style variants defined upfront — no mutations at render time.
var (
	styleHeaderLeft = lipgloss.NewStyle().
			Background(clrPurple).Foreground(clrWhite).Bold(true).Padding(0, 1)

	styleHeaderRight = lipgloss.NewStyle().
				Background(clrPurple).Foreground(clrLightGray).Padding(0, 1).Align(lipgloss.Right)

	styleFooter       = lipgloss.NewStyle().Foreground(clrGray).Padding(0, 1)
	styleSearchPrompt = lipgloss.NewStyle().Foreground(clrPurple).Bold(true).Padding(0, 1)
	styleSearchText   = lipgloss.NewStyle().Foreground(clrWhite)

	styleRowSelected = lipgloss.NewStyle().Background(clrRowHover).Foreground(clrWhite)
	styleRowNormal   = lipgloss.NewStyle().Foreground(clrLightGray)

	styleArrow = lipgloss.NewStyle().Foreground(clrGray)
	styleCursor = lipgloss.NewStyle().Foreground(clrPurple).Bold(true)

	styleDurOk  = lipgloss.NewStyle().Foreground(clrGreen)
	styleDurErr = lipgloss.NewStyle().Foreground(clrRed).Bold(true)
	styleBarOk  = lipgloss.NewStyle().Foreground(clrGreen)
	styleBarErr = lipgloss.NewStyle().Foreground(clrRed)

	styleErrBadge  = lipgloss.NewStyle().Foreground(clrRed).Bold(true)
	styleErrFilter = lipgloss.NewStyle().Foreground(clrYellow).Bold(true)

	styleMuted    = lipgloss.NewStyle().Foreground(clrGray)
	styleDivider  = lipgloss.NewStyle().Foreground(clrPurpleDim)
	styleSpanName = lipgloss.NewStyle().Foreground(clrLightGray)
	styleSpanErr  = lipgloss.NewStyle().Foreground(clrRed)
	styleBadge    = lipgloss.NewStyle().Foreground(clrGray)
	styleSpanBold = lipgloss.NewStyle().Foreground(clrLightGray).Bold(true)

	// Statistics view styles.
	styleColHeader  = lipgloss.NewStyle().Foreground(clrGray).Bold(true)
	styleStatsSel   = lipgloss.NewStyle().Background(clrRowHover).Foreground(clrWhite)
	styleStatsRow   = lipgloss.NewStyle().Foreground(clrLightGray)
	styleStatsErr0  = lipgloss.NewStyle().Foreground(clrGray)
	styleStatsErrN  = lipgloss.NewStyle().Foreground(clrRed).Bold(true)
	styleStatsDur   = lipgloss.NewStyle().Foreground(clrGreen)

	// Attributes panel styles.
	styleAttrSection = lipgloss.NewStyle().Foreground(clrPurple).Bold(true)
	styleAttrKey     = lipgloss.NewStyle().Foreground(clrLightGray)
	styleAttrVal     = lipgloss.NewStyle().Foreground(clrGray)
	styleMetaKey     = lipgloss.NewStyle().Foreground(clrGray)
	styleMetaVal     = lipgloss.NewStyle().Foreground(clrLightGray)
	styleStatusOk    = lipgloss.NewStyle().Foreground(clrGreen)
	styleStatusErr   = lipgloss.NewStyle().Foreground(clrRed).Bold(true)
	styleStatusUnset = lipgloss.NewStyle().Foreground(clrGray)
)

// tickMsg drives the periodic store poll.
type tickMsg struct{}

// ReceiverErrMsg is sent from main when the receiver fails, causing the TUI to quit.
type ReceiverErrMsg struct{ Err error }

// opStats holds aggregated metrics for a (service, operation) pair.
type opStats struct {
	service   string
	op        string
	count     int
	errCount  int
	durations []time.Duration // sorted ascending
}

func (s opStats) pct(p float64) time.Duration {
	if len(s.durations) == 0 {
		return 0
	}
	return s.durations[int(float64(len(s.durations)-1)*p)]
}

// Model is the top-level bubbletea model.
type Model struct {
	store  *store.Store
	traces []model.Trace
	width  int
	height int

	// List view
	cursor   int
	offset   int
	selected *model.Trace

	// Search / filter
	searchMode  bool
	searchQuery string
	errOnly     bool

	// Detail view
	spanCursor   int
	detailOffset int

	// Statistics view
	statsMode   bool
	statsCursor int
	statsOffset int

	// Listening addresses shown in the header
	grpcAddr string
	httpAddr string
}

// New creates a TUI model backed by the given store.
// grpcAddr and httpAddr are displayed in the header (httpAddr may be empty).
func New(s *store.Store, grpcAddr, httpAddr string) Model {
	return Model{
		store:    s,
		width:    80,
		height:   24,
		grpcAddr: grpcAddr,
		httpAddr: httpAddr,
	}
}

func (m Model) Init() tea.Cmd { return tickCmd() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		// ctrl+c always quits; q quits in every mode except when typing a search query.
		if msg.String() == "ctrl+c" || (msg.String() == "q" && !m.searchMode) {
			return m, tea.Quit
		}
		switch {
		case m.searchMode:
			m = m.handleSearchKey(msg)
		case m.statsMode:
			m = m.handleStatsKey(msg)
		case m.selected != nil:
			m = m.handleDetailKey(msg)
		default:
			m = m.handleListKey(msg)
		}

	case tickMsg:
		m.traces = m.store.GetAll()
		visible := m.visibleTraces()
		if m.cursor >= len(visible) {
			m.cursor = max(0, len(visible)-1)
		}
		if m.selected != nil {
			if _, ok := m.store.Get(m.selected.ID); !ok {
				m.selected = nil
				m.detailOffset = 0
			}
		}
		return m, tickCmd()

	case ReceiverErrMsg:
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "enter", "esc":
		m.searchMode = false
	case "ctrl+u":
		m.searchQuery = ""
		m.cursor, m.offset = 0, 0
	case "backspace":
		runes := []rune(m.searchQuery)
		if len(runes) > 0 {
			m.searchQuery = string(runes[:len(runes)-1])
			m.cursor, m.offset = 0, 0
		}
	default:
		if utf8.RuneCountInString(msg.String()) == 1 {
			m.searchQuery += msg.String()
			m.cursor, m.offset = 0, 0
		}
	}
	return m
}

func (m Model) handleStatsKey(msg tea.KeyMsg) Model {
	stats := computeStats(m.traces)
	switch msg.String() {
	case "esc", "s":
		m.statsMode = false
	case "up", "k":
		if m.statsCursor > 0 {
			m.statsCursor--
			if m.statsCursor < m.statsOffset {
				m.statsOffset = m.statsCursor
			}
		}
	case "down", "j":
		if m.statsCursor < len(stats)-1 {
			m.statsCursor++
			vr := m.statsVisibleRows()
			if m.statsCursor >= m.statsOffset+vr {
				m.statsOffset = m.statsCursor - vr + 1
			}
		}
	case "enter":
		if m.statsCursor < len(stats) {
			m.searchQuery = stats[m.statsCursor].service
			m.cursor, m.offset = 0, 0
			m.statsMode = false
		}
	}
	return m
}

func (m Model) handleDetailKey(msg tea.KeyMsg) Model {
	flat := flattenTree(buildTree(m.selected.Spans))
	switch msg.String() {
	case "esc":
		m.selected = nil
		m.detailOffset = 0
	case "up", "k":
		if m.spanCursor > 0 {
			m.spanCursor--
			m = m.scrollToSpan(m.spanCursor)
		}
	case "down", "j":
		if m.spanCursor < len(flat)-1 {
			m.spanCursor++
			m = m.scrollToSpan(m.spanCursor)
		}
	}
	return m
}

func (m Model) handleListKey(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "/":
		m.searchMode = true
	case "s":
		m.statsMode = true
		m.statsCursor, m.statsOffset = 0, 0
	case "e":
		m.errOnly = !m.errOnly
		m.cursor, m.offset = 0, 0
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		}
	case "down", "j":
		visible := m.visibleTraces()
		if m.cursor < len(visible)-1 {
			m.cursor++
			vr := m.visibleRows()
			if m.cursor >= m.offset+vr {
				m.offset = m.cursor - vr + 1
			}
		}
	case "enter":
		visible := m.visibleTraces()
		if len(visible) > 0 && m.cursor < len(visible) {
			t := visible[m.cursor]
			m.selected = &t
			m.spanCursor, m.detailOffset = 0, 0
		}
	}
	return m
}

// scrollToSpan adjusts detailOffset to keep the given span index visible.
// Body layout: line 0 = blank, 1 = title, 2 = divider, 3 = blank, 4+ = spans.
func (m Model) scrollToSpan(spanIdx int) Model {
	spanLine := 4 + spanIdx
	bodyH := m.height - 2
	if spanLine < m.detailOffset {
		m.detailOffset = spanLine
	}
	if spanLine >= m.detailOffset+bodyH {
		m.detailOffset = spanLine - bodyH + 1
	}
	return m
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m Model) visibleTraces() []model.Trace {
	q := strings.ToLower(m.searchQuery)
	if !m.errOnly && q == "" {
		return m.traces
	}
	out := make([]model.Trace, 0, len(m.traces))
	for _, t := range m.traces {
		if m.errOnly && !t.HasError {
			continue
		}
		if q != "" {
			root := t.RootSpan()
			svc, op := "", strings.ToLower(t.ID.String())
			if root != nil {
				svc = strings.ToLower(root.ServiceName())
				op = strings.ToLower(root.Name)
			}
			if !strings.Contains(svc, q) && !strings.Contains(op, q) &&
				!strings.Contains(strings.ToLower(t.ID.String()), q) {
				continue
			}
		}
		out = append(out, t)
	}
	return out
}

func (m Model) visibleRows() int {
	v := m.height - 4
	if v < 1 {
		v = 1
	}
	return v
}

func (m Model) statsVisibleRows() int {
	// header + blank + col-header + divider + footer = 5 fixed lines
	v := m.height - 5
	if v < 1 {
		v = 1
	}
	return v
}

func (m Model) View() string {
	switch {
	case m.statsMode:
		return m.statsView()
	case m.selected != nil:
		return m.detailView(m.selected)
	default:
		return m.listView()
	}
}

// --- Header / Divider -------------------------------------------------------

func (m Model) renderHeader(right string) string {
	lw := m.width / 2
	rw := m.width - lw
	return styleHeaderLeft.Width(lw).Render("  tracelocal") +
		styleHeaderRight.Width(rw).Render(right+"  ")
}

func renderDivider(w int) string {
	if w < 0 {
		w = 0
	}
	return styleDivider.Render(strings.Repeat("─", w))
}

// --- List view --------------------------------------------------------------

func (m Model) listView() string {
	visible := m.visibleTraces()

	status := fmt.Sprintf("%d trace(s)", len(visible))
	if len(visible) != len(m.traces) {
		status += fmt.Sprintf(" of %d", len(m.traces))
	}
	status += "  ·  gRPC " + m.grpcAddr
	if m.httpAddr != "" {
		status += "  ·  HTTP " + m.httpAddr
	}
	if dropped := m.store.Dropped(); dropped > 0 {
		status += fmt.Sprintf("  ·  %d dropped", dropped)
	}
	if m.errOnly {
		status = styleErrFilter.Render("⚠ errors only") + styleMuted.Render("  ·  "+status)
	}
	header := m.renderHeader(status)

	var body strings.Builder
	body.WriteString("\n")
	if len(visible) == 0 {
		if m.searchQuery != "" || m.errOnly {
			body.WriteString(styleMuted.Padding(0, 2).Render("No traces match the current filter."))
		} else {
			body.WriteString(styleMuted.Padding(0, 2).Render("Waiting for traces…"))
		}
		body.WriteString("\n")
	} else {
		vr := m.visibleRows()
		end := m.offset + vr
		if end > len(visible) {
			end = len(visible)
		}
		for i := m.offset; i < end; i++ {
			body.WriteString(m.renderTraceRow(i, &visible[i]))
			body.WriteString("\n")
		}
		if len(visible) > vr {
			fmt.Fprintf(&body, "%s\n", styleMuted.Padding(0, 2).Render(
				fmt.Sprintf("%d–%d of %d", m.offset+1, end, len(visible))))
		}
	}

	var footer string
	if m.searchMode {
		footer = styleSearchPrompt.Render("  /  ") +
			styleSearchText.Render(m.searchQuery) +
			styleSearchPrompt.Render("▌") +
			styleFooter.Render("    ctrl+u  clear    esc  close")
	} else {
		errHint := "e  errors only"
		if m.errOnly {
			errHint = "e  all"
		}
		footer = styleFooter.Render(
			"  ↑/↓  navigate    enter  select    /  search    " + errHint + "    s  stats    q  quit")
	}

	return header + "\n" + body.String() + "\n" + footer
}

func (m Model) renderTraceRow(i int, t *model.Trace) string {
	selected := i == m.cursor
	svc, op := traceLabel(t)

	durStyle := styleDurOk
	if t.HasError {
		durStyle = styleDurErr
	}

	var left strings.Builder
	if svc != "" {
		left.WriteString(serviceStyle(svc).Render(svc))
		left.WriteString(styleArrow.Render(" › "))
		left.WriteString(styleSpanName.Render(op))
	} else {
		left.WriteString(styleSpanName.Render(op))
	}
	if t.HasError {
		left.WriteString(styleErrBadge.Render("  ✖"))
	}

	rightPart := durStyle.Render(fmt.Sprintf("%6v", t.Duration())) +
		styleBadge.Render(fmt.Sprintf("  %d spans", len(t.Spans)))

	leftStr := left.String()
	gap := m.width - 4 - lipgloss.Width(leftStr) - lipgloss.Width(rightPart)
	if gap < 1 {
		gap = 1
	}

	cur := "  "
	if selected {
		cur = styleCursor.Render("▶ ")
	}
	row := cur + leftStr + strings.Repeat(" ", gap) + rightPart
	if selected {
		return styleRowSelected.Width(m.width).Render(row)
	}
	return styleRowNormal.Render(row)
}

func traceLabel(t *model.Trace) (svc, op string) {
	root := t.RootSpan()
	if root == nil {
		return "", t.ID.String()
	}
	return root.ServiceName(), root.Name
}

// --- Statistics view --------------------------------------------------------

func (m Model) statsView() string {
	stats := computeStats(m.traces)

	status := fmt.Sprintf("%d operation(s)  ·  %d trace(s)  ·  gRPC %s",
		len(stats), len(m.traces), m.grpcAddr)
	if m.httpAddr != "" {
		status += "  ·  HTTP " + m.httpAddr
	}
	header := m.renderHeader(status)

	var body strings.Builder
	body.WriteString("\n")

	if len(stats) == 0 {
		body.WriteString(styleMuted.Padding(0, 2).Render("No data yet — waiting for traces…") + "\n")
	} else {
		colHdr := fmt.Sprintf("  %-16s %-22s %6s  %5s  %7s  %7s  %7s",
			"SERVICE", "OPERATION", "COUNT", "ERR%", "p50", "p95", "p99")
		body.WriteString(styleColHeader.Render(colHdr) + "\n")
		body.WriteString("  " + renderDivider(m.width-4) + "\n")

		vr := m.statsVisibleRows()
		statsCursor := m.statsCursor
		if statsCursor >= len(stats) {
			statsCursor = max(0, len(stats)-1)
		}
		end := m.statsOffset + vr
		if end > len(stats) {
			end = len(stats)
		}
		for i := m.statsOffset; i < end; i++ {
			body.WriteString(m.renderStatsRow(i, stats[i], statsCursor))
			body.WriteString("\n")
		}
		note := "sorted by p99 desc"
		if len(stats) > vr {
			note = fmt.Sprintf("%d–%d of %d  ·  "+note, m.statsOffset+1, end, len(stats))
		}
		body.WriteString(styleMuted.Padding(0, 2).Render(note) + "\n")
	}

	footer := styleFooter.Render(
		"  ↑/↓  navigate    enter  filter by service    esc  close    q  quit")
	return header + "\n" + body.String() + "\n" + footer
}

func (m Model) renderStatsRow(i int, st opStats, cursor int) string {
	selected := i == cursor

	errPct := 0.0
	if st.count > 0 {
		errPct = float64(st.errCount) / float64(st.count) * 100
	}
	errSty := styleStatsErr0
	if st.errCount > 0 {
		errSty = styleStatsErrN
	}

	svcLabel := truncate(st.service, 14)
	opLabel := truncate(st.op, 20)
	if svcLabel == "" {
		svcLabel = styleMuted.Render("(none)        ")
	} else {
		svcLabel = serviceStyle(st.service).Render(fmt.Sprintf("%-14s", svcLabel))
	}

	cur := "  "
	if selected {
		cur = styleCursor.Render("▶ ")
	}

	row := fmt.Sprintf("%s%s  %-22s%6d  %s  %s  %s  %s",
		cur,
		svcLabel,
		styleSpanName.Render(opLabel),
		st.count,
		errSty.Render(fmt.Sprintf("%4.0f%%", errPct)),
		styleStatsDur.Render(fmt.Sprintf("%7s", fmtDur(st.pct(0.50)))),
		styleStatsDur.Render(fmt.Sprintf("%7s", fmtDur(st.pct(0.95)))),
		styleStatsDur.Render(fmt.Sprintf("%7s", fmtDur(st.pct(0.99)))),
	)

	if selected {
		return styleStatsSel.Width(m.width).Render(row)
	}
	return styleStatsRow.Render(row)
}

// computeStats aggregates per-(service, operation) metrics across all traces,
// sorted by p99 descending.
func computeStats(traces []model.Trace) []opStats {
	type key struct{ service, op string }
	agg := make(map[key]*opStats)

	for _, t := range traces {
		for _, s := range t.Spans {
			k := key{service: s.ServiceName(), op: s.Name}
			st, ok := agg[k]
			if !ok {
				st = &opStats{service: k.service, op: k.op}
				agg[k] = st
			}
			st.count++
			if s.Status.Code == model.StatusError {
				st.errCount++
			}
			st.durations = append(st.durations, s.Duration())
		}
	}

	result := make([]opStats, 0, len(agg))
	for _, st := range agg {
		sort.Slice(st.durations, func(i, j int) bool {
			return st.durations[i] < st.durations[j]
		})
		result = append(result, *st)
	}
	sort.Slice(result, func(i, j int) bool {
		if p99i, p99j := result[i].pct(0.99), result[j].pct(0.99); p99i != p99j {
			return p99i > p99j
		}
		return result[i].service < result[j].service
	})
	return result
}

func fmtDur(d time.Duration) string {
	if d == 0 {
		return "—"
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// --- Detail view ------------------------------------------------------------

func (m Model) detailView(t *model.Trace) string {
	svc, op := traceLabel(t)

	var titleParts strings.Builder
	if svc != "" {
		titleParts.WriteString(serviceStyle(svc).Render(svc))
		titleParts.WriteString(styleArrow.Render(" › "))
	}
	titleParts.WriteString(styleSpanBold.Render(op))
	titleParts.WriteString(styleMuted.Render(fmt.Sprintf("  %v  ·  %d spans", t.Duration(), len(t.Spans))))

	tree := buildTree(t.Spans)
	flat := flattenTree(tree)

	spanCursor := m.spanCursor
	if len(flat) > 0 && spanCursor >= len(flat) {
		spanCursor = len(flat) - 1
	}

	status := fmt.Sprintf("%d trace(s)  ·  gRPC %s", len(m.traces), m.grpcAddr)
	if m.httpAddr != "" {
		status += "  ·  HTTP " + m.httpAddr
	}
	header := m.renderHeader(status)

	// Build the full scrollable body then apply the viewport window.
	var rawBody strings.Builder
	rawBody.WriteString("\n")
	rawBody.WriteString("  " + titleParts.String() + "\n")
	rawBody.WriteString("  " + renderDivider(m.width-2) + "\n")
	rawBody.WriteString("\n")

	if len(t.Spans) == 0 {
		rawBody.WriteString(styleMuted.Padding(0, 2).Render("No spans.") + "\n")
	} else {
		traceStart := t.Spans[0].StartTime
		for _, s := range t.Spans[1:] {
			if s.StartTime.Before(traceStart) {
				traceStart = s.StartTime
			}
		}
		idx := 0
		renderTreeStyled(&rawBody, tree, 0, traceStart, t.Duration(), spanCursor, &idx)

		if len(flat) > 0 {
			rawBody.WriteString("\n")
			renderAttributesPanel(&rawBody, flat[spanCursor].span, m.width)
		}
	}

	// Apply scroll: clip to viewport.
	lines := strings.Split(rawBody.String(), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	bodyH := m.height - 2 // one line each for header and footer
	start := m.detailOffset
	if start >= len(lines) {
		start = max(0, len(lines)-1)
	}
	end := start + bodyH
	if end > len(lines) {
		end = len(lines)
	}

	visible := strings.Join(lines[start:end], "\n")
	if end-start < bodyH {
		visible += strings.Repeat("\n", bodyH-(end-start))
	}

	scrollHint := ""
	if len(lines) > bodyH {
		pct := 100 * end / len(lines)
		scrollHint = styleMuted.Render(fmt.Sprintf("  %d%%", pct))
	}
	footer := styleFooter.Render("  ↑/↓  navigate span    esc  back    q  quit") + scrollHint

	return header + "\n" + visible + "\n" + footer
}

// --- Span tree --------------------------------------------------------------

type spanNode struct {
	span     model.Span
	children []*spanNode
}

func buildTree(spans []model.Span) []*spanNode {
	nodes := make(map[model.SpanID]*spanNode, len(spans))
	for i := range spans {
		nodes[spans[i].SpanID] = &spanNode{span: spans[i]}
	}
	roots := make([]*spanNode, 0)
	for _, n := range nodes {
		if !n.span.ParentID.IsZero() {
			if parent, ok := nodes[n.span.ParentID]; ok {
				parent.children = append(parent.children, n)
				continue
			}
		}
		roots = append(roots, n)
	}
	var sortNodes func([]*spanNode)
	sortNodes = func(ns []*spanNode) {
		sort.Slice(ns, func(i, j int) bool {
			return ns[i].span.StartTime.Before(ns[j].span.StartTime)
		})
		for _, n := range ns {
			sortNodes(n.children)
		}
	}
	sortNodes(roots)
	return roots
}

// flattenTree returns nodes in DFS / tree-display order.
func flattenTree(nodes []*spanNode) []*spanNode {
	var out []*spanNode
	var walk func([]*spanNode)
	walk = func(ns []*spanNode) {
		for _, n := range ns {
			out = append(out, n)
			walk(n.children)
		}
	}
	walk(nodes)
	return out
}

func renderTreeStyled(b *strings.Builder, nodes []*spanNode, depth int, traceStart time.Time, traceDur time.Duration, spanCursor int, idx *int) {
	indent := strings.Repeat("  ", depth)
	for _, n := range nodes {
		s := n.span
		isErr := s.Status.Code == model.StatusError
		isSelected := *idx == spanCursor
		*idx++

		svc := s.ServiceName()
		var label strings.Builder
		if svc != "" {
			label.WriteString(serviceStyle(svc).Render(svc))
			label.WriteString(styleArrow.Render(" › "))
		}
		if isErr {
			label.WriteString(styleSpanErr.Render(s.Name))
		} else {
			label.WriteString(styleSpanName.Render(s.Name))
		}

		bar := styledTimingBar(traceStart, traceDur, s, isErr)
		durSty := styleDurOk
		if isErr {
			durSty = styleDurErr
		}
		errMark := ""
		if isErr {
			errMark = styleErrBadge.Render("  ✖")
		}
		cur := "  "
		if isSelected {
			cur = styleCursor.Render("▶ ")
		}

		fmt.Fprintf(b, "  %s%s%s  %s  %s%s\n",
			cur, indent, label.String(), bar,
			durSty.Render(fmt.Sprintf("%6v", s.Duration())),
			errMark)

		renderTreeStyled(b, n.children, depth+1, traceStart, traceDur, spanCursor, idx)
	}
}

func styledTimingBar(traceStart time.Time, traceDur time.Duration, s model.Span, isErr bool) string {
	const filled = "█"
	const empty = "░"

	barSty := styleBarOk
	if isErr {
		barSty = styleBarErr
	}
	if traceDur == 0 {
		return styleMuted.Render("[") + barSty.Render(strings.Repeat(filled, barWidth)) + styleMuted.Render("]")
	}

	off := int(float64(s.StartTime.Sub(traceStart)) / float64(traceDur) * barWidth)
	wid := int(float64(s.Duration()) / float64(traceDur) * barWidth)
	if off < 0 {
		off = 0
	}
	if wid < 1 {
		wid = 1
	}
	if off+wid > barWidth {
		wid = barWidth - off
	}

	return styleMuted.Render("[") +
		styleMuted.Render(strings.Repeat(empty, off)) +
		barSty.Render(strings.Repeat(filled, wid)) +
		styleMuted.Render(strings.Repeat(empty, barWidth-off-wid)) +
		styleMuted.Render("]")
}

// --- Attributes panel -------------------------------------------------------

func renderAttributesPanel(b *strings.Builder, s model.Span, width int) {
	svc := s.ServiceName()
	var spanLabel strings.Builder
	if svc != "" {
		spanLabel.WriteString(serviceStyle(svc).Render(svc))
		spanLabel.WriteString(styleArrow.Render(" › "))
	}
	spanLabel.WriteString(styleSpanBold.Render(s.Name))

	b.WriteString("  " + renderDivider(width-2) + "\n")
	b.WriteString("  " + spanLabel.String() + "\n\n")

	statusStr, statusSty := renderStatus(s.Status)
	fmt.Fprintf(b, "  %s %s    %s %s    %s %s\n\n",
		styleMetaKey.Render("kind"), styleMetaVal.Render(s.Kind.String()),
		styleMetaKey.Render("status"), statusSty.Render(statusStr),
		styleMetaKey.Render("duration"), styleMetaVal.Render(s.Duration().String()),
	)

	if len(s.Attributes) > 0 {
		b.WriteString(styleAttrSection.Render("  attributes") + "\n")
		for _, a := range s.Attributes {
			fmt.Fprintf(b, "  %-30s  %s\n",
				styleAttrKey.Render(truncate(a.Key, 28)),
				styleAttrVal.Render(truncate(a.Value.String(), 60)),
			)
		}
		b.WriteString("\n")
	}

	if len(s.Events) > 0 {
		b.WriteString(styleAttrSection.Render(fmt.Sprintf("  events (%d)", len(s.Events))) + "\n")
		for _, e := range s.Events {
			fmt.Fprintf(b, "  %s  %s\n",
				styleMuted.Render(e.Timestamp.Format("15:04:05.000")),
				styleSpanName.Render(e.Name),
			)
			for _, a := range e.Attributes {
				fmt.Fprintf(b, "    %-28s  %s\n",
					styleAttrKey.Render(truncate(a.Key, 26)),
					styleAttrVal.Render(truncate(a.Value.String(), 56)),
				)
			}
		}
	}
}

func renderStatus(st model.Status) (string, lipgloss.Style) {
	switch st.Code {
	case model.StatusOK:
		return "ok", styleStatusOk
	case model.StatusError:
		msg := "error"
		if st.Message != "" {
			msg += ": " + truncate(st.Message, 40)
		}
		return msg, styleStatusErr
	default:
		return "unset", styleStatusUnset
	}
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
