package bgen

// Choose k from n items can be done in this many ways. Originally derived from
// github.com/limix/bgen /src/util/choose.c
func Choose(n, k int) int {
	ans := 1

	if k > n-k {
		k = n - k
	}

	for j := 1; j <= k; j++ {
		if n%j == 0 {
			ans *= n / j
		} else if ans%j == 0 {
			ans = ans / j * n
		} else {
			ans = (ans * n) / j
		}

		n--
	}

	return ans
}
