package steganalysis

import (
	"math"
	"math/rand/v2"
)

// FLD is a ridge-regularised Fisher direction. It is the linear baseline
// kept beside logistic regression. Score is a probability.
type FLD struct {
	w           []float64
	mean, scale []float64
	b           float64
}

// TrainFLD fits the direction between the two class means. An empty class
// scores everything at 0.5.
func TrainFLD(x [][]float64, y []bool) *FLD {
	dim := len(x[0])
	m := &FLD{w: make([]float64, dim), mean: make([]float64, dim), scale: make([]float64, dim)}
	m.standardise(x)
	var n0, n1 int
	mu0 := make([]float64, dim)
	mu1 := make([]float64, dim)
	for i, xi := range x {
		z := m.normalise(xi)
		dst := mu0
		if y[i] {
			dst = mu1
			n1++
		} else {
			n0++
		}
		for j, v := range z {
			dst[j] += v
		}
	}
	if n0 == 0 || n1 == 0 {
		return m
	}
	for j := range dim {
		mu0[j] /= float64(n0)
		mu1[j] /= float64(n1)
	}
	cov := make([]float64, dim*dim)
	for i, xi := range x {
		z := m.normalise(xi)
		mu := mu0
		if y[i] {
			mu = mu1
		}
		for a := range dim {
			d := z[a] - mu[a]
			for b := range dim {
				cov[a*dim+b] += d * (z[b] - mu[b])
			}
		}
	}
	nf := float64(len(x))
	var trace float64
	for a := range dim {
		trace += cov[a*dim+a]
	}
	ridge := 1e-2 * (trace/nf + 1) / float64(dim)
	for a := range dim {
		cov[a*dim+a] = cov[a*dim+a]/nf + ridge
		for b := a + 1; b < dim; b++ {
			cov[a*dim+b] /= nf
			cov[b*dim+a] = cov[a*dim+b]
		}
	}
	delta := make([]float64, dim)
	for j := range dim {
		delta[j] = mu1[j] - mu0[j]
	}
	m.w = solve(cov, delta, dim)
	var p0, p1 float64
	for j := range dim {
		p0 += m.w[j] * mu0[j]
		p1 += m.w[j] * mu1[j]
	}
	m.b = -0.5 * (p0 + p1)
	return m
}

// Score returns the positive-class probability of x.
func (m *FLD) Score(x []float64) float64 {
	if m == nil || len(m.w) == 0 {
		return 0.5
	}
	return sigmoid(m.b + dot(m.w, m.normalise(x)))
}

func (m *FLD) standardise(x [][]float64) {
	n := float64(len(x))
	for _, xi := range x {
		for j, v := range xi {
			m.mean[j] += v
		}
	}
	for j := range m.mean {
		m.mean[j] /= n
	}
	for _, xi := range x {
		for j, v := range xi {
			d := v - m.mean[j]
			m.scale[j] += d * d
		}
	}
	for j := range m.scale {
		m.scale[j] = math.Sqrt(m.scale[j] / n)
		if m.scale[j] < 1e-8 {
			m.scale[j] = 1
		}
	}
}

func (m *FLD) normalise(x []float64) []float64 {
	z := make([]float64, len(m.w))
	for j := range z {
		if j < len(x) {
			z[j] = (x[j] - m.mean[j]) / m.scale[j]
		}
	}
	return z
}

func solve(a []float64, b []float64, n int) []float64 {
	aug := make([]float64, n*(n+1))
	for i := range n {
		copy(aug[i*(n+1):i*(n+1)+n], a[i*n:(i+1)*n])
		aug[i*(n+1)+n] = b[i]
	}
	for c := range n {
		piv := c
		best := math.Abs(aug[c*(n+1)+c])
		for r := c + 1; r < n; r++ {
			if v := math.Abs(aug[r*(n+1)+c]); v > best {
				best, piv = v, r
			}
		}
		if best < 1e-12 {
			continue
		}
		if piv != c {
			for k := c; k <= n; k++ {
				aug[c*(n+1)+k], aug[piv*(n+1)+k] = aug[piv*(n+1)+k], aug[c*(n+1)+k]
			}
		}
		div := aug[c*(n+1)+c]
		for k := c; k <= n; k++ {
			aug[c*(n+1)+k] /= div
		}
		for r := range n {
			if r == c {
				continue
			}
			f := aug[r*(n+1)+c]
			for k := c; k <= n; k++ {
				aug[r*(n+1)+k] -= f * aug[c*(n+1)+k]
			}
		}
	}
	out := make([]float64, n)
	for i := range n {
		out[i] = aug[i*(n+1)+n]
	}
	return out
}

// Stumps is a small AdaBoost of threshold stumps, the frozen boosted-tree
// baseline. It is depth 1 on purpose: a deeper tree is not part of the
// frozen suite.
type Stumps struct {
	feat  []int
	thr   []float64
	ge    []bool
	alpha []float64
}

// TrainStumps boosts rounds stumps. It stops early when a stump cannot beat
// chance, which is the cover-like case.
func TrainStumps(x [][]float64, y []bool, rounds int) *Stumps {
	if len(x) == 0 || rounds < 1 {
		return &Stumps{}
	}
	dim := len(x[0])
	w := make([]float64, len(x))
	for i := range w {
		w[i] = 1 / float64(len(x))
	}
	s := &Stumps{}
	for range rounds {
		j, thr, ge, err := bestStump(x, y, w, dim)
		if err >= 0.5 {
			break
		}
		a := 0.5 * math.Log((1-err)/max(err, 1e-12))
		var norm float64
		for i := range x {
			vote := stumpVote(x[i], j, thr, ge)
			label := -1.0
			if y[i] {
				label = 1
			}
			w[i] *= math.Exp(-a * label * vote)
			norm += w[i]
		}
		for i := range w {
			w[i] /= norm
		}
		s.feat = append(s.feat, j)
		s.thr = append(s.thr, thr)
		s.ge = append(s.ge, ge)
		s.alpha = append(s.alpha, a)
		if err == 0 {
			break
		}
	}
	return s
}

// Score returns the positive-class probability of x.
func (s *Stumps) Score(x []float64) float64 {
	if s == nil || len(s.alpha) == 0 {
		return 0.5
	}
	var z float64
	for i := range s.alpha {
		z += s.alpha[i] * stumpVote(x, s.feat[i], s.thr[i], s.ge[i])
	}
	return sigmoid(z)
}

func bestStump(x [][]float64, y []bool, w []float64, dim int) (j int, thr float64, ge bool, err float64) {
	err = 1
	for f := range dim {
		lo, hi := x[0][f], x[0][f]
		for _, xi := range x[1:] {
			lo = math.Min(lo, xi[f])
			hi = math.Max(hi, xi[f])
		}
		if hi == lo {
			continue
		}
		for k := 1; k <= 7; k++ {
			t := lo + (hi-lo)*float64(k)/8
			for _, side := range []bool{true, false} {
				e := stumpError(x, y, w, f, t, side)
				if e < err {
					j, thr, ge, err = f, t, side, e
				}
			}
		}
	}
	return j, thr, ge, err
}

func stumpError(x [][]float64, y []bool, w []float64, j int, thr float64, ge bool) float64 {
	var e float64
	for i := range x {
		pred := stumpVote(x[i], j, thr, ge) > 0
		if pred != y[i] {
			e += w[i]
		}
	}
	return e
}

func stumpVote(x []float64, j int, thr float64, ge bool) float64 {
	high := j < len(x) && x[j] >= thr
	if high == ge {
		return 1
	}
	return -1
}

// Subspace averages Fisher directions fit on random feature subsets. It is
// the frozen random-subspace ensemble. Width is how many features each
// direction sees.
type Subspace struct {
	cols   [][]int
	models []*FLD
}

// TrainSubspace fits n models. A width larger than the feature count uses
// every feature.
func TrainSubspace(x [][]float64, y []bool, n, width int, seed uint64) *Subspace {
	if len(x) == 0 || n < 1 {
		return &Subspace{}
	}
	dim := len(x[0])
	if width < 1 || width > dim {
		width = dim
	}
	rng := rand.New(rand.NewPCG(seed, seed))
	s := &Subspace{}
	for range n {
		cols := pickCols(dim, width, rng)
		s.cols = append(s.cols, cols)
		s.models = append(s.models, TrainFLD(project(x, cols), y))
	}
	return s
}

// Score returns the mean positive-class probability across the ensemble.
func (s *Subspace) Score(x []float64) float64 {
	if s == nil || len(s.models) == 0 {
		return 0.5
	}
	var sum float64
	for i, m := range s.models {
		sum += m.Score(projectOne(x, s.cols[i]))
	}
	return sum / float64(len(s.models))
}

func pickCols(dim, width int, rng *rand.Rand) []int {
	order := rng.Perm(dim)
	cols := append([]int(nil), order[:width]...)
	// Stable order so a rerun with the same seed reads the same columns.
	for i := 1; i < len(cols); i++ {
		j := i
		for j > 0 && cols[j] < cols[j-1] {
			cols[j], cols[j-1] = cols[j-1], cols[j]
			j--
		}
	}
	return cols
}

func project(x [][]float64, cols []int) [][]float64 {
	out := make([][]float64, len(x))
	for i := range x {
		out[i] = projectOne(x[i], cols)
	}
	return out
}

func projectOne(x []float64, cols []int) []float64 {
	out := make([]float64, len(cols))
	for i, c := range cols {
		if c < len(x) {
			out[i] = x[c]
		}
	}
	return out
}
