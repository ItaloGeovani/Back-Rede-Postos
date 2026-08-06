package utils

import "time"

// SomarDiasUteis avança n dias úteis (seg–sex) a partir de base (timezone de base).
// Feriados nacionais ficam fora do escopo.
func SomarDiasUteis(base time.Time, n int) time.Time {
	if n <= 0 {
		return base
	}
	t := base
	restantes := n
	for restantes > 0 {
		t = t.Add(24 * time.Hour)
		wd := t.Weekday()
		if wd == time.Saturday || wd == time.Sunday {
			continue
		}
		restantes--
	}
	return t
}
