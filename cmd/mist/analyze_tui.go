package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/iSerganov/mist"
)

type (
	probeFunc   func(source string) (mist.Source, error)
	analyzeFunc func(ctx context.Context, source string, opts mist.AnalyzeOptions) (mist.Analysis, error)
)

type screen int

const (
	screenList screen = iota
	screenRunning
	screenReport
)

// track is one audio file in the folder, as the probe described it.
type track struct {
	path   string
	source mist.Source
	size   int64
}

func (t track) FilterValue() string { return filepath.Base(t.path) }

// folder is a subfolder in the list, or its parent when up is set.
type folder struct {
	path string
	up   bool
}

func (f folder) FilterValue() string { return filepath.Base(f.path) }

// order sorts the list: the parent first, then folders, then tracks, each
// by path.
func order(it list.Item) (int, string) {
	switch it := it.(type) {
	case folder:
		if it.up {
			return 0, ""
		}
		return 1, it.path
	case track:
		return 2, it.path
	}
	return 3, ""
}

func trackDetails(t track) string {
	return describeSource(t.source) + " · " + sourceDetails(t.source) + " · " + humanBytes(t.size)
}

type (
	probedMsg struct {
		gen   int
		track track
		audio bool
	}
	stageMsg struct {
		stage mist.Stage
		next  <-chan mist.Stage
	}
	doneMsg struct {
		analysis mist.Analysis
		err      error
	}
)

var (
	keyAnalyze   = key.NewBinding(key.WithKeys("enter", "right", "l"), key.WithHelp("enter", "open"))
	keyParent    = key.NewBinding(key.WithKeys("backspace", "left", "h"), key.WithHelp("←", "up"))
	keyReference = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reference"))
	keyBack      = key.NewBinding(key.WithKeys("b", "esc"), key.WithHelp("b/esc", "back"))
	keyCancel    = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))
	keyQuit      = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit"))
)

// browser is the analyze screen: a folder's subfolders and audio files,
// the analysis of one file while it runs, and its report.
type browser struct {
	ctx       context.Context
	dir       string
	gen       int
	notice    string
	probe     probeFunc
	analyze   analyzeFunc
	pending   int
	reference string
	screen    screen
	list      list.Model
	spin      spinner.Model
	report    viewport.Model
	current   track
	stages    []mist.Stage
	cancel    context.CancelFunc
	result    mist.Analysis
	err       error
	width     int
}

func newBrowser(ctx context.Context, dir, reference string, probe probeFunc, analyze analyzeFunc) *browser {
	b := &browser{
		ctx: ctx, dir: dir, probe: probe, analyze: analyze, reference: reference,
		report: viewport.New(viewport.WithWidth(80), viewport.WithHeight(20)),
		width:  80,
	}
	b.list = list.New(nil, trackDelegate{b: b}, 80, 20)
	b.restyle(true)
	b.retitle()
	b.list.SetStatusBarItemName("entry", "entries")
	b.list.AdditionalShortHelpKeys = func() []key.Binding { return []key.Binding{keyAnalyze, keyParent, keyReference} }
	b.list.AdditionalFullHelpKeys = b.list.AdditionalShortHelpKeys
	b.list.KeyMap.Quit = keyQuit
	b.list.KeyMap.PrevPage = key.NewBinding(key.WithKeys("pgup", "u"), key.WithHelp("pgup", "prev page"))
	b.list.KeyMap.NextPage = key.NewBinding(key.WithKeys("pgdown", "d"), key.WithHelp("pgdown", "next page"))
	return b
}

// restyle switches every style to the shades for the terminal background.
func (b *browser) restyle(dark bool) {
	styles = newPalette(dark)
	b.spin = spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(styles.title))
	b.list.Styles = list.DefaultStyles(dark)
	b.list.Styles.Title = styles.title.Padding(0, 1)
}

func (b *browser) Init() tea.Cmd {
	return tea.Batch(b.spin.Tick, tea.RequestBackgroundColor, b.open(b.dir))
}

// open lists dir: its parent and subfolders at once, and each file once
// the probe says it is audio. Probes still running for the folder left
// behind report under an older generation and are dropped.
func (b *browser) open(dir string) tea.Cmd {
	folders, files, err := listing(dir)
	if err != nil {
		b.notice = err.Error()
		return nil
	}
	b.gen++
	b.dir, b.notice, b.pending = dir, "", len(files)
	var items []list.Item
	if parent := filepath.Dir(dir); parent != dir {
		items = append(items, folder{path: parent, up: true})
	}
	for _, f := range folders {
		items = append(items, folder{path: f})
	}
	b.list.ResetFilter()
	cmds := []tea.Cmd{b.list.SetItems(items)}
	b.list.Select(0)
	b.retitle()
	sem := make(chan struct{}, runtime.NumCPU())
	for _, path := range files {
		cmds = append(cmds, b.probeCmd(b.gen, path, sem))
	}
	return tea.Batch(cmds...)
}

// probeCmd asks libav what path is, a few files at a time. Anything it
// cannot open as decodable audio is left out of the list.
func (b *browser) probeCmd(gen int, path string, sem chan struct{}) tea.Cmd {
	return func() tea.Msg {
		sem <- struct{}{}
		defer func() { <-sem }()
		src, err := b.probe(path)
		if err != nil {
			return probedMsg{gen: gen}
		}
		t := track{path: path, source: src}
		if info, err := os.Stat(path); err == nil {
			t.size = info.Size()
		}
		return probedMsg{gen: gen, track: t, audio: true}
	}
}

func (b *browser) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		b.restyle(msg.IsDark())
		if b.screen == screenReport {
			b.report.SetContent(b.reportContent())
		}
		return b, b.spin.Tick
	case tea.WindowSizeMsg:
		b.width = msg.Width
		b.list.SetSize(msg.Width, msg.Height-1)
		b.report.SetWidth(msg.Width)
		b.report.SetHeight(msg.Height - 2)
		if b.screen == screenReport {
			b.report.SetContent(b.reportContent())
		}
		return b, nil
	case probedMsg:
		if msg.gen != b.gen {
			return b, nil
		}
		b.pending--
		b.retitle()
		if !msg.audio {
			return b, nil
		}
		at, _ := slices.BinarySearchFunc(b.list.Items(), list.Item(msg.track), func(it, want list.Item) int {
			ik, ip := order(it)
			wk, wp := order(want)
			return cmp.Or(cmp.Compare(ik, wk), strings.Compare(ip, wp))
		})
		return b, b.list.InsertItem(at, msg.track)
	case spinner.TickMsg:
		var cmd tea.Cmd
		b.spin, cmd = b.spin.Update(msg)
		b.retitle()
		return b, cmd
	case stageMsg:
		b.stages = append(b.stages, msg.stage)
		return b, waitStage(msg.next)
	case doneMsg:
		return b, b.finish(msg)
	case tea.KeyPressMsg:
		return b.key(msg)
	}
	return b.forward(msg)
}

// retitle shows the probe's progress in the list title until it is done.
func (b *browser) retitle() {
	b.list.Title = "mist · analyze  " + home(b.dir)
	switch {
	case b.notice != "":
		b.list.Title += "  " + b.notice
	case b.pending > 0:
		b.list.Title += fmt.Sprintf("  %s probing %d files", b.spin.View(), b.pending)
	}
}

// home shortens a path under the home folder to ~.
func home(path string) string {
	if h, err := os.UserHomeDir(); err == nil && (path == h || strings.HasPrefix(path, h+string(filepath.Separator))) {
		return "~" + strings.TrimPrefix(path, h)
	}
	return path
}

func (b *browser) key(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return b, b.quit()
	}
	switch b.screen {
	case screenList:
		if b.list.FilterState() == list.Filtering {
			return b.forward(msg)
		}
		switch {
		case key.Matches(msg, keyQuit):
			return b, b.quit()
		case key.Matches(msg, keyAnalyze):
			switch it := b.list.SelectedItem().(type) {
			case track:
				return b, b.start(it)
			case folder:
				return b, b.open(it.path)
			}
			return b, nil
		case key.Matches(msg, keyParent):
			if parent := filepath.Dir(b.dir); parent != b.dir {
				return b, b.open(parent)
			}
			return b, nil
		case key.Matches(msg, keyReference):
			if t, ok := b.list.SelectedItem().(track); ok {
				b.reference = toggle(b.reference, t.path)
			}
			return b, nil
		}
	case screenRunning:
		switch {
		case key.Matches(msg, keyQuit):
			return b, b.quit()
		case key.Matches(msg, keyCancel):
			b.cancel()
		}
		return b, nil
	case screenReport:
		switch {
		case key.Matches(msg, keyQuit):
			return b, b.quit()
		case key.Matches(msg, keyBack):
			b.screen = screenList
			return b, nil
		}
	}
	return b.forward(msg)
}

func (b *browser) forward(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch b.screen {
	case screenList:
		b.list, cmd = b.list.Update(msg)
	case screenReport:
		b.report, cmd = b.report.Update(msg)
	}
	return b, cmd
}

// start runs the analysis of t off the UI goroutine. Each stage arrives
// as a stageMsg as it finishes, and the result as a doneMsg.
func (b *browser) start(t track) tea.Cmd {
	ctx, cancel := context.WithCancel(b.ctx)
	b.screen, b.current, b.stages, b.cancel = screenRunning, t, nil, cancel
	opts := mist.AnalyzeOptions{}
	if b.reference != t.path {
		opts.Reference = b.reference
	}
	stages := make(chan mist.Stage, 64)
	opts.Progress = func(s mist.Stage) { stages <- s }
	run := func() tea.Msg {
		a, err := b.analyze(ctx, t.path, opts)
		close(stages)
		return doneMsg{analysis: a, err: err}
	}
	return tea.Batch(run, waitStage(stages), b.spin.Tick)
}

func waitStage(stages <-chan mist.Stage) tea.Cmd {
	return func() tea.Msg {
		s, ok := <-stages
		if !ok {
			return nil
		}
		return stageMsg{stage: s, next: stages}
	}
}

func (b *browser) finish(msg doneMsg) tea.Cmd {
	b.cancel()
	if errors.Is(msg.err, context.Canceled) {
		if b.ctx.Err() != nil {
			return tea.Quit
		}
		b.screen = screenList
		return nil
	}
	b.result, b.err, b.screen = msg.analysis, msg.err, screenReport
	b.report.SetContent(b.reportContent())
	b.report.GotoTop()
	return nil
}

func (b *browser) quit() tea.Cmd {
	if b.cancel != nil {
		b.cancel()
	}
	return tea.Quit
}

func (b *browser) reportContent() string {
	if b.err != nil {
		return fmt.Sprintf("\n  %s %s\n\n  %s\n", styles.alarm.Render("✗"), styles.bold.Render(filepath.Base(b.current.path)), b.err)
	}
	ref := ""
	if b.reference != b.current.path {
		ref = b.reference
	}
	return renderReport(b.result, b.current, ref, b.width)
}

func (b *browser) View() tea.View {
	v := tea.NewView(b.content())
	v.AltScreen = true
	return v
}

func (b *browser) content() string {
	switch b.screen {
	case screenRunning:
		return b.runningView()
	case screenReport:
		return b.report.View() + "\n" + help(keyBack, keyQuit, key.NewBinding(key.WithHelp("↑/↓", "scroll")))
	}
	return b.list.View()
}

func (b *browser) runningView() string {
	var s strings.Builder
	fmt.Fprintf(&s, "\n  %s %s %s\n", styles.title.Render("mist"), styles.faint.Render("· analyzing ·"), styles.bold.Render(filepath.Base(b.current.path)))
	fmt.Fprintf(&s, "  %s\n\n", styles.faint.Render(trackDetails(b.current)))
	for _, st := range b.stages {
		mark, note := styles.good.Render("✓"), ""
		switch st.Status {
		case mist.StageSkipped:
			mark, note = styles.warn.Render("–"), "skipped"
		case mist.StageNotRun:
			mark, note = styles.faint.Render("·"), "not run"
		}
		fmt.Fprintf(&s, "  %s %-12s %s\n", mark, st.Name, styles.faint.Render(note))
	}
	fmt.Fprintf(&s, "  %s %s\n\n", b.spin.View(), styles.faint.Render("running the next stage…"))
	return s.String() + help(keyCancel, keyQuit)
}

func help(keys ...key.Binding) string {
	parts := make([]string, len(keys))
	for i, k := range keys {
		h := k.Help()
		parts[i] = styles.bold.Render(h.Key) + " " + styles.faint.Render(h.Desc)
	}
	return "  " + strings.Join(parts, styles.faint.Render(" · "))
}

func toggle(current, path string) string {
	if current == path {
		return ""
	}
	return path
}

// trackDelegate draws a track as its name over what the probe said about
// it, with a badge on the reference, and a folder as its name.
type trackDelegate struct {
	b *browser
}

func (d trackDelegate) Height() int                         { return 2 }
func (d trackDelegate) Spacing() int                        { return 1 }
func (d trackDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d trackDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var name, details string
	switch it := item.(type) {
	case track:
		name = filepath.Base(it.path)
		if it.path == d.b.reference {
			name += " " + styles.badge.Render("REF")
		}
		details = styles.faint.Render(trackDetails(it))
	case folder:
		name, details = "▸ "+filepath.Base(it.path)+"/", styles.faint.Render("folder")
		if it.up {
			name, details = "◂ ..", styles.faint.Render("parent folder · "+home(it.path))
		}
	default:
		return
	}
	bar := "  "
	if index == m.Index() {
		bar = styles.title.Render("▌ ")
		name = styles.title.Render(name)
	}
	_, _ = fmt.Fprintf(w, "%s%s\n%s%s", bar, name, bar, details)
}
