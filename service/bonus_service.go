package service

func CalculateBonus(workedHours, monthlyNorm float64) float64 {
	if workedHours < 0 || monthlyNorm <= 0 {
		return 0
	}

	switch {
	case workedHours >= monthlyNorm:
		return 15000
	case workedHours >= 140:
		return 8000
	case workedHours >= 120:
		return 3000
	case workedHours >= 80:
		return 1500
	default:
		return 0
	}
}
