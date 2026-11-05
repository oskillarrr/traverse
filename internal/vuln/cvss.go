package vuln

import (
	"math"
	"strings"
)

// CVSS3 computes the CVSS v3.0/v3.1 base score of a vector string such as
// "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H", using the FIRST
// specification formulas. ok is false for any other vector.
func CVSS3(vector string) (score float64, ok bool) {
	parts := strings.Split(strings.TrimSpace(vector), "/")
	if len(parts) < 9 || (parts[0] != "CVSS:3.1" && parts[0] != "CVSS:3.0") {
		return 0, false
	}
	m := map[string]string{}
	for _, p := range parts[1:] {
		k, v, found := strings.Cut(p, ":")
		if !found {
			return 0, false
		}
		m[k] = v
	}
	weights := map[string]map[string]float64{
		"AV": {"N": 0.85, "A": 0.62, "L": 0.55, "P": 0.2},
		"AC": {"L": 0.77, "H": 0.44},
		"UI": {"N": 0.85, "R": 0.62},
		"C":  {"H": 0.56, "L": 0.22, "N": 0},
		"I":  {"H": 0.56, "L": 0.22, "N": 0},
		"A":  {"H": 0.56, "L": 0.22, "N": 0},
	}
	val := map[string]float64{}
	for k, w := range weights {
		x, found := w[m[k]]
		if !found {
			return 0, false
		}
		val[k] = x
	}
	changed := false
	switch m["S"] {
	case "U":
	case "C":
		changed = true
	default:
		return 0, false
	}
	var pr float64
	switch m["PR"] {
	case "N":
		pr = 0.85
	case "L":
		pr = 0.62
		if changed {
			pr = 0.68
		}
	case "H":
		pr = 0.27
		if changed {
			pr = 0.5
		}
	default:
		return 0, false
	}
	iss := 1 - (1-val["C"])*(1-val["I"])*(1-val["A"])
	var impact float64
	if changed {
		impact = 7.52*(iss-0.029) - 3.25*math.Pow(iss-0.02, 15)
	} else {
		impact = 6.42 * iss
	}
	expl := 8.22 * val["AV"] * val["AC"] * pr * val["UI"]
	if impact <= 0 {
		return 0, true
	}
	if changed {
		return roundup(math.Min(1.08*(impact+expl), 10)), true
	}
	return roundup(math.Min(impact+expl, 10)), true
}

// roundup is the CVSS v3.1 Roundup function (Appendix A).
func roundup(x float64) float64 {
	i := int64(math.Round(x * 100000))
	if i%10000 == 0 {
		return float64(i) / 100000
	}
	return float64(i/10000+1) / 10
}

// SeverityOf maps a CVSS base score to the qualitative rating scale.
func SeverityOf(score float64) string {
	switch {
	case score >= 9:
		return "critical"
	case score >= 7:
		return "high"
	case score >= 4:
		return "medium"
	case score > 0:
		return "low"
	}
	return "none"
}

// SeverityRank orders severities from most to least important.
func SeverityRank(s string) int {
	switch s {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "unknown":
		return 1
	}
	return 0
}
