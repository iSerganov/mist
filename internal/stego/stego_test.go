package stego

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type StegoSuite struct {
	suite.Suite
}

func TestStegoSuite(t *testing.T) {
	suite.Run(t, &StegoSuite{})
}

func (s *StegoSuite) TestLSB() {
	s.Equal(uint8(0), LSB(2))
	s.Equal(uint8(1), LSB(3))
}

func (s *StegoSuite) TestMatchLeavesMatchingBit() {
	s.Equal(int32(4), Match(4, 0))
	s.Equal(int32(5), Match(5, 1))
}

func (s *StegoSuite) TestMatchFlipsByOne() {
	got := Match(4, 1)
	s.Equal(uint8(1), LSB(got))
	d := got - 4
	s.True(d == 1 || d == -1)
}

func (s *StegoSuite) TestSelectorDeterministic() {
	a := NewSelector([]byte("key-aaaaaaaaaaaaaaaaaaaaaaaaaaaa"), 100)
	b := NewSelector([]byte("key-aaaaaaaaaaaaaaaaaaaaaaaaaaaa"), 100)
	s.Equal(a.Pick(10), b.Pick(10))
	c := NewSelector([]byte("key-bbbbbbbbbbbbbbbbbbbbbbbbbbbb"), 100)
	s.NotEqual(a.Pick(10), c.Pick(10))
	s.Len(a.Pick(10), 10)
	seen := map[int]bool{}
	for _, i := range a.Pick(10) {
		s.False(seen[i])
		seen[i] = true
		s.GreaterOrEqual(i, 0)
		s.Less(i, 100)
	}
}

func (s *StegoSuite) TestFillerLength() {
	got, err := Filler(16)
	s.Require().NoError(err)
	s.Len(got, 16)
	got2, err := Filler(16)
	s.Require().NoError(err)
	s.NotEqual(got, got2)
	empty, err := Filler(0)
	s.Require().NoError(err)
	s.Empty(empty)
}

func (s *StegoSuite) TestEmbedExtractRoundTrip() {
	s.T().Skip("covered by mist EmitterSuite once libav is present")
}

func (s *StegoSuite) TestConstantDensity() {
	views := make([]ResidueView, 1000)
	for i := range views {
		views[i] = ResidueView{Index: i, Band: 9000, Value: int32(i)}
	}
	el := Eligible(views, DefaultBands)
	s.Len(el, 1000, "in-band residues are all eligible")
	s.Equal(int(1000*Density), int(float64(len(el))*Density))
}

// Embedding stays above DefaultBands.FromHz because low-frequency error is
// the audible kind; anything the codec marked unflippable is out too.
func (s *StegoSuite) TestEligibleFiltersBandsAndUnflippable() {
	tests := []struct {
		title string
		view  ResidueView
		want  bool
	}{
		{"well inside the band", ResidueView{Band: 9000}, true},
		{"at the lower bound", ResidueView{Band: DefaultBands.FromHz}, true},
		{"just below the bound", ResidueView{Band: DefaultBands.FromHz - 1}, false},
		{"bass", ResidueView{Band: 200}, false},
		{"above the upper bound", ResidueView{Band: DefaultBands.ToHz}, false},
		{"in band but unflippable", ResidueView{Band: 9000, Unflippable: true}, false},
	}
	for _, tc := range tests {
		s.Run(tc.title, func() {
			el := Eligible([]ResidueView{tc.view}, DefaultBands)
			s.Equal(tc.want, len(el) == 1)
		})
	}
}
