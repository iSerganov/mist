package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/iSerganov/mist"
	"github.com/spf13/cobra"
)

type analyzeOptions struct {
	dir       string
	reference string
}

func newAnalyzeCmd() *cobra.Command {
	o := &analyzeOptions{}
	cmd := &cobra.Command{
		Use:   "analyze [folder]",
		Short: "Score audio files for a hidden message the way a warden would",
		Long: "  Browse a folder (default: the current one) and its subfolders, pick\n" +
			"  an audio file in any format, and see how a warden would read it.\n\n" +
			"  Two questions are asked. Does it carry a Mist message? The classical\n" +
			"  detectors and trained wardens are read against the calibration run\n" +
			"  built into this binary (`make calibrate`); 50% means no evidence either\n" +
			"  way. Did any other tool hide data in it? Its structure is checked for\n" +
			"  bytes no player reads, MP3 frames for the fields known to carry covert\n" +
			"  bits, and lossless silence for the trace of LSB replacement; those\n" +
			"  checks report a level and their reasons, not a probability.\n\n" +
			"  With --reference, or a file marked with r in the list, the file is also\n" +
			"  diffed against a clean encode of that original, which decides the Mist\n" +
			"  question when it really is the recording the file was made from. No key\n" +
			"  is needed or accepted.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.dir = "."
			if len(args) == 1 {
				o.dir = args[0]
			}
			return runAnalyze(cmd.Context(), o)
		},
	}
	cmd.Flags().StringVarP(&o.reference, "reference", "r", "", "original carrier to diff the file against")
	return cmd
}

func runAnalyze(ctx context.Context, o *analyzeOptions) error {
	if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
		return errors.New("analyze is interactive: run it in a terminal")
	}
	dir, err := filepath.Abs(o.dir)
	if err != nil {
		return err
	}
	if _, _, err := listing(dir); err != nil {
		return err
	}
	if o.reference != "" {
		if _, err := mist.Probe(o.reference); err != nil {
			return fmt.Errorf("reference %s: %w", o.reference, err)
		}
		if o.reference, err = filepath.Abs(o.reference); err != nil {
			return err
		}
	}
	opts := []tea.ProgramOption{tea.WithContext(ctx)}
	if !colorOn {
		opts = append(opts, tea.WithColorProfile(colorprofile.Ascii))
	}
	m := newBrowser(ctx, dir, o.reference, mist.Probe, mist.Analyze)
	_, err = tea.NewProgram(m, opts...).Run()
	if errors.Is(err, tea.ErrProgramKilled) && ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// listing is the visible folders and files directly inside dir, following
// symlinks, each sorted by name. Which files are audio is for the probe to
// say, not the extension.
func listing(dir string) (folders, files []string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, err := os.Stat(path)
		switch {
		case err != nil:
		case info.IsDir():
			folders = append(folders, path)
		case info.Mode().IsRegular():
			files = append(files, path)
		}
	}
	sort.Strings(folders)
	sort.Strings(files)
	return folders, files, nil
}
