package frame

import (
	"testing"
	"time"

	"github.com/iSerganov/mist/internal/codec"
	"github.com/stretchr/testify/suite"
)

type FrameSuite struct {
	suite.Suite
}

func TestFrameSuite(t *testing.T) {
	suite.Run(t, &FrameSuite{})
}

func (s *FrameSuite) params(d time.Duration) Params {
	return Params{SampleRate: 100, Channels: 1, Duration: d}
}

func (s *FrameSuite) TestParamsSamples() {
	tests := []struct {
		title string
		in    Params
		want  int
	}{
		{"one second", Params{SampleRate: 44100, Duration: time.Second}, 44100},
		{"eight seconds", Params{SampleRate: 44100, Duration: 8 * time.Second}, 352800},
		{"no rate", Params{Duration: time.Second}, 0},
		{"no duration", Params{SampleRate: 44100}, 0},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Equal(tc.want, tc.in.Samples())
		})
	}
}

func pkts(pts ...int64) []codec.Packet {
	out := make([]codec.Packet, len(pts))
	for i, p := range pts {
		out[i] = codec.Packet{PTS: p, Data: []byte{byte(i)}}
	}
	return out
}

func (s *FrameSuite) TestGroupPackets() {
	tests := []struct {
		title     string
		in        []codec.Packet
		want      []int64
		wantSizes []int
	}{
		{"single window", pkts(0, 20, 40), []int64{0}, []int{3}},
		{"two windows", pkts(0, 50, 100, 150), []int64{0, 1}, []int{2, 2}},
		{"sparse windows", pkts(0, 300), []int64{0, 3}, []int{1, 1}},
		{"encoder priming folds into frame zero", pkts(-128, 0, 50), []int64{0}, []int{3}},
		{"trailing sliver is its own frame", pkts(0, 50, 100), []int64{0, 1}, []int{2, 1}},
		{"empty", nil, nil, nil},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			got := GroupPackets(tc.in, s.params(time.Second))
			s.Require().Len(got, len(tc.want))
			for i, g := range got {
				s.Equal(tc.want[i], g.Index)
				s.Len(g.Packets, tc.wantSizes[i])
			}
		})
	}
}

func (s *FrameSuite) TestGroupPacketsPartial() {
	tests := []struct {
		title string
		in    []codec.Packet
		want  []bool
	}{
		{"aligned start is complete", pkts(0, 50), []bool{false}},
		{"joined mid-window", pkts(50, 60), []bool{true}},
		{"only the joined frame is partial", pkts(50, 100, 150), []bool{true, false}},
		{"priming is not partial", pkts(-128, 10), []bool{false}},
		// Packets rarely land exactly on a boundary; a later group that
		// starts past its window start was still witnessed in full.
		{"later frames are never partial", pkts(0, 130, 260), []bool{false, false, false}},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			got := GroupPackets(tc.in, s.params(time.Second))
			s.Require().Len(got, len(tc.want))
			for i, g := range got {
				s.Equal(tc.want[i], g.Partial)
			}
		})
	}
}

func (s *FrameSuite) TestGroupPacketsRejectsZeroWindow() {
	s.Empty(GroupPackets(pkts(0, 1), Params{}))
}

func (s *FrameSuite) TestGrouperEmitsOnWindowChange() {
	g := NewGrouper(s.params(time.Second))
	_, ok := g.Push(pkts(0)[0])
	s.False(ok, "first packet cannot complete a group")
	_, ok = g.Push(pkts(50)[0])
	s.False(ok, "same window does not complete a group")

	done, ok := g.Push(pkts(100)[0])
	s.Require().True(ok, "crossing into the next window completes the previous")
	s.Equal(int64(0), done.Index)
	s.Len(done.Packets, 2)

	tail, ok := g.Flush()
	s.Require().True(ok)
	s.Equal(int64(1), tail.Index)
	s.Len(tail.Packets, 1)

	_, ok = g.Flush()
	s.False(ok, "flushing twice yields nothing")
}

func (s *FrameSuite) TestCapacity() {
	tests := []struct {
		title    string
		eligible int
		density  float64
		overhead int
		want     int
	}{
		{"room to spare", 8000, 0.10, 10, 90},
		{"exactly overhead", 800, 0.10, 10, 0},
		{"below overhead", 10, 0.10, 10, 0},
		{"no residues", 0, 0.10, 0, 0},
		{"no density", 8000, 0, 10, 0},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			s.Equal(tc.want, Capacity(tc.eligible, tc.density, tc.overhead))
		})
	}
}
