package steganalysis

// Markov is the probability of each second difference given the one
// before it, both truncated to ±3, row by row: 49 values. The second
// difference of audio is close to zero wherever the waveform is smooth,
// so ±1 changes stand out there more than in the first difference that
// Features uses (Liu, Sung & Qiao's derivative-based audio steganalysis).
func Markov(v []int32) []float64 {
	return transitions(diff(diff(v)))
}

func diff(v []int32) []int32 {
	if len(v) < 2 {
		return nil
	}
	out := make([]int32, len(v)-1)
	for i := range out {
		out[i] = v[i+1] - v[i]
	}
	return out
}
