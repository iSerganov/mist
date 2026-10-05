package steganalysis

const spamT = 3

// Features is the classifier's view of a stream: every detector's score in
// Detectors order, then the share of values at each of -3…3, then the
// first-order SPAM transition probabilities between adjacent differences
// truncated to ±3. ±1 embedding adds noise that flattens both the peak of
// the histogram and those transitions.
func Features(v []int32) []float64 {
	ds := Detectors()
	out := make([]float64, 0, len(ds)+(2*spamT+1)*(2*spamT+2))
	for _, d := range ds {
		out = append(out, d.Score(v))
	}
	out = append(out, centre(v)...)
	return append(out, transitions(diff(v))...)
}

func centre(v []int32) []float64 {
	out := make([]float64, 2*spamT+1)
	for _, x := range v {
		if x >= -spamT && x <= spamT {
			out[x+spamT]++
		}
	}
	for i := range out {
		out[i] /= float64(max(len(v), 1))
	}
	return out
}

// transitions is the probability of each value of d given the one before
// it, both truncated to ±3. A row that never occurs stays zero.
func transitions(d []int32) []float64 {
	const side = 2*spamT + 1
	counts := make([]float64, side*side)
	var rows [side]float64
	for i := 0; i+1 < len(d); i++ {
		a, b := truncate(d[i]), truncate(d[i+1])
		counts[a*side+b]++
		rows[a]++
	}
	for a, n := range rows {
		if n == 0 {
			continue
		}
		for b := range side {
			counts[a*side+b] /= n
		}
	}
	return counts
}

func truncate(d int32) int {
	return int(max(-spamT, min(spamT, d))) + spamT
}
