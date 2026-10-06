package forensics

import "fmt"

const (
	// silenceRun is the shortest stretch, in samples, read as silence.
	silenceRun = 2048
	// sparseShare is the most a stretch may hold of ±1 samples and still be
	// silence with bits written into it, rather than dither or a fade.
	sparseShare = 0.25
	// isolatedShare is how many of those ±1 samples must stand alone. A
	// fade's last cycles reach ±1 in runs that follow the waveform; LSB
	// replacement drops them one at a time.
	isolatedShare = 0.7
)

// Silence looks for what LSB replacement leaves in digital silence: a
// stretch of zeros with isolated ±1 samples dropped into it. An encoder or
// a recording never writes that; dither fills silence densely and a fade
// reaches ±1 in runs. Each plane is one channel of integer samples.
func Silence(planes [][]int32, rate int) Check {
	c := Check{Name: "silence", Ran: true}
	var silent, sparse, flips int
	for _, p := range planes {
		for start := 0; start < len(p); {
			end := start
			for end < len(p) && p[end] >= -1 && p[end] <= 1 {
				end++
			}
			if end-start >= silenceRun {
				silent += end - start
				if n, ok := sparseOnes(p[start:end]); ok {
					sparse += end - start
					flips += n
				}
			}
			start = end + 1
		}
	}
	if silent == 0 {
		c.Context = "no digital silence to inspect"
		return c
	}
	c.Context = fmt.Sprintf("%.1f s of digital silence across all channels", float64(silent)/float64(max(rate, 1)))
	if sparse == 0 {
		return c
	}
	l := Medium
	if rate > 0 && sparse >= rate/2 {
		l = High
	}
	c.add(l, fmt.Sprintf("%d isolated ±1 samples scattered through %.1f s of otherwise silent audio, the trace of LSB replacement",
		flips, float64(sparse)/float64(max(rate, 1))))
	return c
}

// sparseOnes reports a silent stretch with a few isolated ±1 samples in it,
// and how many.
func sparseOnes(run []int32) (int, bool) {
	var ones, isolated int
	for i, v := range run {
		if v == 0 {
			continue
		}
		ones++
		if (i == 0 || run[i-1] == 0) && (i == len(run)-1 || run[i+1] == 0) {
			isolated++
		}
	}
	if ones < 8 || float64(ones) > sparseShare*float64(len(run)) {
		return 0, false
	}
	return ones, float64(isolated) >= isolatedShare*float64(ones)
}
