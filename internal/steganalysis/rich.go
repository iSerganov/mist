package steganalysis

import (
	"math"
	"math/cmplx"

	"github.com/iSerganov/mist/internal/dsp"
)

// RichName is the logistic model trained on Rich. It is a frozen primary
// warden: future embedder comparisons report it beside the Markov model.
const RichName = "rich"

// RichNames is the frozen feature list, in Rich's order. Summaries stand
// in for full histograms so a harness chunk stays trainable. Residue
// values are scored as the same stream: the harness flattens Vorbis
// indices, so band and Huffman conditioning is not in this vector.
func RichNames() []string {
	names := []string{}
	for k := 1; k <= 8; k++ {
		names = append(names, "pred_"+itoa(k))
	}
	for k := 1; k <= 4; k++ {
		names = append(names, "diff_abs_"+itoa(k), "diff_zero_"+itoa(k))
	}
	names = append(names, "lpc8")
	names = append(names, "scale1", "scale2", "scale2_odd", "scale4")
	names = append(names, "parity_even", "parity_odd", "lsb_mean")
	for i := range 9 {
		names = append(names, "co2_"+itoa(i))
	}
	for i := range 27 {
		names = append(names, "co3_"+itoa(i))
	}
	names = append(names, "co4_entropy", "co4_max")
	names = append(names, "sign_agree", "flatness", "phase_diff", "group_delay")
	for i := range 8 {
		names = append(names, "band_"+itoa(i))
	}
	names = append(names, "mid", "side", "side_ratio", "interchannel")
	return names
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [4]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// Rich is the frozen audio rich model for one stream. channels is 1 for a
// mono stream or for Vorbis residue indices. An interleaved PCM stream
// passes its channel count so the last four features are mid/side.
func Rich(v []int32, channels int) []float64 {
	out := make([]float64, len(RichNames()))
	if len(v) < 4 {
		return out
	}
	scale := meanAbs(v) + 1
	n := 0
	for k := 1; k <= 8; k++ {
		out[n] = meanAbs(nthDiff(v, k)) / scale
		n++
	}
	for k := 1; k <= 4; k++ {
		d := nthDiff(v, k)
		out[n] = meanAbs(d) / scale
		out[n+1] = zeroFrac(d)
		n += 2
	}
	out[n] = lpcEnergy(v, 8)
	n++
	out[n] = meanAbs(diff(v)) / scale
	out[n+1] = meanAbs(diff(take(v, 2, 0))) / scale
	out[n+2] = meanAbs(diff(take(v, 2, 1))) / scale
	out[n+3] = meanAbs(diff(take(v, 4, 0))) / scale
	n += 4
	out[n] = meanAbs(diff(take(v, 2, 0))) / scale
	out[n+1] = meanAbs(diff(take(v, 2, 1))) / scale
	out[n+2] = lsbMean(v)
	n += 3
	co2 := symmetrizedCo2(v)
	copy(out[n:], co2)
	n += len(co2)
	co := symmetrizedCo3(v)
	copy(out[n:], co)
	n += len(co)
	out[n], out[n+1] = co4Summary(v)
	n += 2
	out[n] = signAgree(v)
	n++
	flat, bands, phase, group := spectral(v)
	out[n] = flat
	out[n+1] = phase
	out[n+2] = group
	n += 3
	copy(out[n:], bands)
	n += len(bands)
	mid, side, ratio, inter := stereo(v, channels)
	out[n] = mid
	out[n+1] = side
	out[n+2] = ratio
	out[n+3] = inter
	return out
}

// RichPlanar is Rich for a stream stored as whole planes one after another,
// which is how the harness hands lossless samples to a detector. channels
// below 2, or a length that is not a multiple of the channel count, is
// scored as one stream.
func RichPlanar(v []int32, channels int) []float64 {
	if channels < 2 || len(v)%channels != 0 {
		return Rich(v, 1)
	}
	n := len(v) / channels
	acc := Rich(v[:n], 1)
	for c := 1; c < channels; c++ {
		r := Rich(v[c*n:(c+1)*n], 1)
		for i := range acc {
			acc[i] += r[i]
		}
	}
	for i := range acc {
		acc[i] /= float64(channels)
	}
	mid, side, ratio, inter := stereoPlanes(v[:n], v[n:2*n])
	base := len(acc) - 4
	acc[base] = mid
	acc[base+1] = side
	acc[base+2] = ratio
	acc[base+3] = inter
	return acc
}

func nthDiff(v []int32, order int) []int32 {
	d := v
	for range order {
		d = diff(d)
		if len(d) == 0 {
			return nil
		}
	}
	return d
}

func meanAbs(v []int32) float64 {
	if len(v) == 0 {
		return 0
	}
	var s float64
	for _, x := range v {
		s += math.Abs(float64(x))
	}
	return s / float64(len(v))
}

func zeroFrac(v []int32) float64 {
	if len(v) == 0 {
		return 0
	}
	var n int
	for _, x := range v {
		if x == 0 {
			n++
		}
	}
	return float64(n) / float64(len(v))
}

func take(v []int32, step, phase int) []int32 {
	if step < 1 {
		step = 1
	}
	out := make([]int32, 0, len(v)/step+1)
	for i := phase; i < len(v); i += step {
		out = append(out, v[i])
	}
	return out
}

func lpcEnergy(v []int32, order int) float64 {
	n := min(len(v), 2048)
	if n <= order+1 {
		return 0
	}
	r := make([]float64, order+1)
	for i := range n {
		x := float64(v[i])
		for k := 0; k <= order && i+k < n; k++ {
			r[k] += x * float64(v[i+k])
		}
	}
	if r[0] <= 0 {
		return 0
	}
	a := make([]float64, order+1)
	a[0] = 1
	e := r[0]
	for m := 1; m <= order; m++ {
		var acc float64
		for j := 1; j < m; j++ {
			acc += a[j] * r[m-j]
		}
		if e == 0 {
			return 0
		}
		k := -(r[m] + acc) / e
		next := make([]float64, order+1)
		next[0] = 1
		for j := 1; j < m; j++ {
			next[j] = a[j] + k*a[m-j]
		}
		next[m] = k
		a = next
		e *= 1 - k*k
		if e <= 0 {
			return 0
		}
	}
	return e / r[0]
}

func clip3(x int32) int {
	switch {
	case x < 0:
		return 0
	case x > 0:
		return 2
	default:
		return 1
	}
}

func co2(v []int32) []float64 {
	const side = 3
	counts := make([]float64, side*side)
	var total float64
	d := diff(v)
	for i := 0; i+1 < len(d); i++ {
		a, b := clip3(d[i]), clip3(d[i+1])
		counts[a*side+b]++
		total++
	}
	if total == 0 {
		return counts
	}
	for i := range counts {
		counts[i] /= total
	}
	return counts
}

func symmetrizedCo2(v []int32) []float64 {
	fwd, rev, neg := co2(v), co2(reversed(v)), co2(negated(v))
	out := make([]float64, len(fwd))
	for i := range out {
		out[i] = (fwd[i] + rev[i] + neg[i]) / 3
	}
	return out
}

func co4Summary(v []int32) (entropy, maxBin float64) {
	const side = 3
	counts := make([]float64, side*side*side*side)
	var total float64
	d := diff(v)
	for i := 0; i+3 < len(d); i++ {
		a, b, c, e := clip3(d[i]), clip3(d[i+1]), clip3(d[i+2]), clip3(d[i+3])
		counts[((a*side+b)*side+c)*side+e]++
		total++
	}
	if total == 0 {
		return 0, 0
	}
	for _, c := range counts {
		if c == 0 {
			continue
		}
		p := c / total
		entropy -= p * math.Log(p)
		if p > maxBin {
			maxBin = p
		}
	}
	return entropy, maxBin
}

func co3(v []int32) []float64 {
	const side = 3
	counts := make([]float64, side*side*side)
	var total float64
	d := diff(v)
	for i := 0; i+2 < len(d); i++ {
		a, b, c := clip3(d[i]), clip3(d[i+1]), clip3(d[i+2])
		counts[(a*side+b)*side+c]++
		total++
	}
	if total == 0 {
		return counts
	}
	for i := range counts {
		counts[i] /= total
	}
	return counts
}

func symmetrizedCo3(v []int32) []float64 {
	fwd := co3(v)
	rev := co3(reversed(v))
	neg := co3(negated(v))
	out := make([]float64, len(fwd))
	for i := range out {
		out[i] = (fwd[i] + rev[i] + neg[i]) / 3
	}
	return out
}

func reversed(v []int32) []int32 {
	out := make([]int32, len(v))
	for i := range v {
		out[i] = v[len(v)-1-i]
	}
	return out
}

func negated(v []int32) []int32 {
	out := make([]int32, len(v))
	for i, x := range v {
		out[i] = -x
	}
	return out
}

func signAgree(v []int32) float64 {
	d1 := diff(v)
	d2 := diff(d1)
	if len(d2) == 0 {
		return 0
	}
	var s float64
	for i := range d2 {
		s += float64(sign(d1[i]) * sign(d2[i]))
	}
	return s / float64(len(d2))
}

func sign(x int32) int32 {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	default:
		return 0
	}
}

func lsbMean(v []int32) float64 {
	if len(v) == 0 {
		return 0
	}
	var s float64
	for _, x := range v {
		s += float64(x & 1)
	}
	return s / float64(len(v))
}

func spectral(v []int32) (flat float64, bands []float64, phase, group float64) {
	const n = 256
	bands = make([]float64, 8)
	if len(v) < n*2 {
		return 0, bands, 0, 0
	}
	off := (len(v) - n*2) / 2
	s0 := spectrum(v[off : off+n])
	s1 := spectrum(v[off+n : off+2*n])
	m0 := make([]float64, len(s0))
	var sum, logSum float64
	for i, c := range s0 {
		m0[i] = cmplx.Abs(c) + 1e-12
		sum += m0[i]
		logSum += math.Log(m0[i])
	}
	if sum > 0 {
		flat = math.Exp(logSum/float64(len(m0))) / (sum / float64(len(m0)))
	}
	edges := logEdges(len(m0), len(bands))
	for b := range bands {
		var s float64
		for i := edges[b]; i < edges[b+1]; i++ {
			s += m0[i]
		}
		bands[b] = s / (sum + 1)
	}
	phase, group = phaseShift(s0, s1)
	return flat, bands, phase, group
}

func spectrum(v []int32) []complex128 {
	a := make([]complex128, len(v))
	for i, x := range v {
		a[i] = complex(float64(x), 0)
	}
	dsp.FFT(a)
	return a[:len(a)/2]
}

func phaseShift(a, b []complex128) (meanAbs, group float64) {
	if len(a) == 0 || len(b) < len(a) {
		return 0, 0
	}
	var s, g, prev float64
	for i := range a {
		d := cmplx.Phase(b[i]) - cmplx.Phase(a[i])
		for d > math.Pi {
			d -= 2 * math.Pi
		}
		for d < -math.Pi {
			d += 2 * math.Pi
		}
		s += math.Abs(d)
		if i > 0 {
			g += math.Abs(d - prev)
		}
		prev = d
	}
	return s / float64(len(a)), g / float64(len(a))
}

func logEdges(n, bands int) []int {
	edges := make([]int, bands+1)
	edges[0] = 1
	edges[bands] = n
	for b := 1; b < bands; b++ {
		t := math.Pow(float64(n)/2, float64(b)/float64(bands))
		edges[b] = min(n-1, max(edges[b-1]+1, int(t)))
	}
	return edges
}

func stereoPlanes(left, right []int32) (mid, side, ratio, inter float64) {
	n := min(len(left), len(right))
	if n == 0 {
		return 0, 0, 0, 0
	}
	var ms, ss, cross, ll, rr float64
	for i := range n {
		l, r := float64(left[i]), float64(right[i])
		m, s := 0.5*(l+r), 0.5*(l-r)
		ms += math.Abs(m)
		ss += math.Abs(s)
		cross += l * r
		ll += l * l
		rr += r * r
	}
	nf := float64(n)
	mid, side = ms/nf, ss/nf
	if mid > 0 {
		ratio = side / mid
	}
	den := math.Sqrt(ll * rr)
	if den > 0 {
		inter = cross / den
	}
	return mid, side, ratio, inter
}

func stereo(v []int32, channels int) (mid, side, ratio, inter float64) {
	if channels < 2 || len(v) < channels*4 {
		return 0, 0, 0, 0
	}
	n := len(v) / channels
	var ms, ss, cross, ll, rr float64
	for i := range n {
		l := float64(v[i*channels])
		r := float64(v[i*channels+1])
		m, s := 0.5*(l+r), 0.5*(l-r)
		ms += math.Abs(m)
		ss += math.Abs(s)
		cross += l * r
		ll += l * l
		rr += r * r
	}
	nf := float64(n)
	mid, side = ms/nf, ss/nf
	if mid > 0 {
		ratio = side / mid
	}
	den := math.Sqrt(ll * rr)
	if den > 0 {
		inter = cross / den
	}
	return mid, side, ratio, inter
}
