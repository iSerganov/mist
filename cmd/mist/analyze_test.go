package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/iSerganov/mist"
	"github.com/stretchr/testify/suite"
)

type AnalyzeSuite struct {
	suite.Suite
	dir      string
	analyzed []mist.AnalyzeOptions
}

func TestAnalyzeSuite(t *testing.T) {
	suite.Run(t, &AnalyzeSuite{})
}

func (s *AnalyzeSuite) SetupTest() {
	s.dir = s.T().TempDir()
	s.analyzed = nil
	for _, name := range []string{"b.flac", "a.wav", "notes.txt", ".hidden.wav"} {
		s.Require().NoError(os.WriteFile(filepath.Join(s.dir, name), []byte(name), 0o600))
	}
	s.Require().NoError(os.Mkdir(filepath.Join(s.dir, "sub.wav"), 0o700))
}

// probe stands in for libav: a file is audio when its name says so.
func (s *AnalyzeSuite) probe(path string) (mist.Source, error) {
	if strings.HasSuffix(path, ".txt") {
		return mist.Source{}, errors.New("not audio")
	}
	return mist.Source{Codec: "flac", Container: "flac", SampleRate: 44100, Channels: 2, Lossless: true}, nil
}

func (s *AnalyzeSuite) analyze(ctx context.Context, _ string, opts mist.AnalyzeOptions) (mist.Analysis, error) {
	s.analyzed = append(s.analyzed, opts)
	if err := ctx.Err(); err != nil {
		return mist.Analysis{}, err
	}
	opts.Progress(mist.Stage{Name: "format", Status: mist.StageRan})
	return mist.Analysis{Probability: 0.99, Basis: mist.BasisKnownCover, Stages: []mist.Stage{
		{Name: "known cover", Status: mist.StageRan, Note: "12 of 1000 values (1.2%) differ"},
	}}, nil
}

// browser is a browser over the suite's folder with every probe answered.
func (s *AnalyzeSuite) browser(ctx context.Context) *browser {
	b := newBrowser(ctx, s.dir, "", s.probe, s.analyze)
	settle(b, b.open(s.dir))
	return b
}

// settle runs the probes a command started and hands their results to b.
// Anything else it would do, such as a status message, is left undone.
func settle(b *browser, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		for _, c := range msg {
			settle(b, c)
		}
	case probedMsg:
		b.Update(msg)
	}
}

// names is the list as the user sees it, by base name.
func names(b *browser) []string {
	var out []string
	for _, it := range b.list.Items() {
		switch it := it.(type) {
		case track:
			out = append(out, filepath.Base(it.path))
		case folder:
			if it.up {
				out = append(out, "..")
			} else {
				out = append(out, filepath.Base(it.path)+"/")
			}
		}
	}
	return out
}

// pick highlights the entry named name.
func (s *AnalyzeSuite) pick(b *browser, name string) {
	i := slices.Index(names(b), name)
	s.Require().GreaterOrEqual(i, 0, name)
	b.list.Select(i)
}

func press(b *browser, keys ...string) tea.Cmd {
	var cmd tea.Cmd
	for _, k := range keys {
		var msg tea.KeyPressMsg
		switch k {
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		case "esc":
			msg = tea.KeyPressMsg{Code: tea.KeyEscape}
		case "down":
			msg = tea.KeyPressMsg{Code: tea.KeyDown}
		case "right":
			msg = tea.KeyPressMsg{Code: tea.KeyRight}
		case "backspace":
			msg = tea.KeyPressMsg{Code: tea.KeyBackspace}
		default:
			msg = tea.KeyPressMsg{Code: rune(k[0]), Text: k}
		}
		_, cmd = b.Update(msg)
	}
	return cmd
}

// run executes the analysis start returned and hands its result back.
func run(b *browser, start tea.Cmd) {
	batch, ok := start().(tea.BatchMsg)
	if !ok {
		return
	}
	b.Update(batch[0]())
}

func (s *AnalyzeSuite) TestListingSkipsHiddenEntries() {
	folders, files, err := listing(s.dir)
	s.Require().NoError(err)
	s.Equal([]string{filepath.Join(s.dir, "sub.wav")}, folders)
	var got []string
	for _, f := range files {
		got = append(got, filepath.Base(f))
	}
	s.Equal([]string{"a.wav", "b.flac", "notes.txt"}, got)
}

func (s *AnalyzeSuite) TestListShowsParentThenFoldersThenAudio() {
	b := s.browser(context.Background())
	s.Equal([]string{"..", "sub.wav/", "a.wav", "b.flac"}, names(b))
	s.Zero(b.pending)
}

func (s *AnalyzeSuite) TestFoldersOpenAndCloseAgain() {
	s.Require().NoError(os.WriteFile(filepath.Join(s.dir, "sub.wav", "inner.flac"), []byte("x"), 0o600))
	b := s.browser(context.Background())
	tests := []struct {
		title string
		act   func() tea.Cmd
		dir   string
		want  []string
	}{
		{"enter opens a folder", func() tea.Cmd { s.pick(b, "sub.wav/"); return press(b, "enter") }, filepath.Join(s.dir, "sub.wav"), []string{"..", "inner.flac"}},
		{"backspace goes to the parent", func() tea.Cmd { return press(b, "backspace") }, s.dir, []string{"..", "sub.wav/", "a.wav", "b.flac"}},
		{"right opens a folder too", func() tea.Cmd { s.pick(b, "sub.wav/"); return press(b, "right") }, filepath.Join(s.dir, "sub.wav"), []string{"..", "inner.flac"}},
		{"enter on .. goes to the parent", func() tea.Cmd { s.pick(b, ".."); return press(b, "enter") }, s.dir, []string{"..", "sub.wav/", "a.wav", "b.flac"}},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			settle(b, tc.act())
			s.Equal(tc.dir, b.dir)
			s.Equal(tc.want, names(b))
		})
	}
}

func (s *AnalyzeSuite) TestProbesFromAFolderLeftBehindAreDropped() {
	b := s.browser(context.Background())
	stale := probedMsg{gen: b.gen - 1, audio: true, track: track{path: filepath.Join(s.dir, "old.flac")}}
	b.Update(stale)
	s.NotContains(names(b), "old.flac")
}

func (s *AnalyzeSuite) TestAnalyzeThenBackThenQuit() {
	b := s.browser(context.Background())
	s.pick(b, "a.wav")
	press(b, "r")
	s.Equal(filepath.Join(s.dir, "a.wav"), b.reference)
	start := press(b, "down", "enter")
	s.Equal(screenRunning, b.screen)
	run(b, start)
	s.Equal(screenReport, b.screen)
	s.Require().Len(s.analyzed, 1)
	s.Equal(filepath.Join(s.dir, "a.wav"), s.analyzed[0].Reference)
	s.Contains(ansi.Strip(b.content()), "99%")
	press(b, "b")
	s.Equal(screenList, b.screen)
	s.Equal(tea.QuitMsg{}, press(b, "q")())
}

func (s *AnalyzeSuite) TestReferenceIsNotUsedOnItself() {
	b := s.browser(context.Background())
	s.pick(b, "a.wav")
	press(b, "r")
	run(b, press(b, "enter"))
	s.Require().Len(s.analyzed, 1)
	s.Empty(s.analyzed[0].Reference)
}

func (s *AnalyzeSuite) TestReferenceToggles() {
	b := s.browser(context.Background())
	s.pick(b, "a.wav")
	press(b, "r", "r")
	s.Empty(b.reference)
}

func (s *AnalyzeSuite) TestEscapeCancelsARunningAnalysis() {
	b := s.browser(context.Background())
	s.pick(b, "a.wav")
	start := press(b, "enter")
	press(b, "esc")
	run(b, start)
	s.Equal(screenList, b.screen)
}

func (s *AnalyzeSuite) TestRenderReport() {
	t := track{path: "song.flac", source: mist.Source{Codec: "flac", Container: "flac", SampleRate: 44100, Channels: 2, Lossless: true}}
	tests := []struct {
		title string
		a     mist.Analysis
		want  []string
	}{
		{
			"a format mist cannot write is ruled out",
			mist.Analysis{Basis: mist.BasisFormat, Stages: []mist.Stage{{Name: "format", Status: mist.StageRan, Note: "mp3 is lossy"}}},
			[]string{"0%", "Mist cannot write this format", "Not found", "mp3 is lossy", "Other steganography: NONE"},
		},
		{
			"data hidden by another tool is found",
			mist.Analysis{Basis: mist.BasisFormat, Suspicion: mist.SuspicionHigh, Stages: []mist.Stage{
				{Name: "format", Status: mist.StageRan, Note: "mp3 is lossy"},
				{Name: "container", Scope: mist.ScopeAny, Status: mist.StageRan, Suspicion: mist.SuspicionHigh,
					Findings: []string{"high: 500 bytes after the last audio frame", "low: the copyright bit changes in 1 of 40 frames"}},
				{Name: "mp3 ancillary", Scope: mist.ScopeAny, Status: mist.StageRan, Note: "1044 bytes of ancillary data"},
			}},
			[]string{
				"Other steganography: HIGH", "● 500 bytes after the last audio frame",
				"unusual but common in honest files: the copyright bit changes",
				"no data hidden outside the audio: mp3 ancillary", "Any steganography tool",
			},
		},
		{
			"a known-cover hit is found",
			mist.Analysis{Basis: mist.BasisKnownCover, Probability: 0.99, Stages: []mist.Stage{
				{Name: "known cover", Status: mist.StageRan, Note: "12 of 1000 values differ"},
			}},
			[]string{"99%", "a message is likely", "Found", "12 of 1000 values differ, the way Mist changes a file"},
		},
		{
			"blind stages at chance say so",
			mist.Analysis{Basis: mist.BasisBlind, Probability: 0.5, Calibration: "flac (corpus, 84 carriers, abc)", Stages: []mist.Stage{
				{Name: "spa", Status: mist.StageRan, Score: 0.2, LR: 1.1, AUC: 0.51},
				{Name: "rs", Status: mist.StageRan, Score: 0.9, LR: 5, AUC: 0.6},
				{Name: "markov", Status: mist.StageRan, Score: 0.1, LR: 0.2, AUC: 0.6},
				{Name: "selection", Status: mist.StageNotRun, Note: "needs the recipient's public key"},
			}},
			[]string{
				"50%", "no evidence either way", "spa cannot tell it", "×5.0 stego", "×5.0 clean",
				"rs scores it like the stego files", "markov scores it like clean audio",
				"Not run: selection", "84 carriers",
			},
		},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			got := ansi.Strip(renderReport(tc.a, t, "", 100))
			for _, w := range tc.want {
				s.Contains(strings.Join(strings.Fields(got), " "), w)
			}
		})
	}
}
