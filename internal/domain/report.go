package domain

import "time"

type TrendPoint struct {
	Bucket          time.Time `json:"bucket"`
	AreaID          string    `json:"area_id"`
	Submitted       int       `json:"submitted"`
	Closed          int       `json:"closed"`
	Rejected        int       `json:"rejected"`
	Overdue         int       `json:"overdue"`
	AverageHours    float64   `json:"average_hours"`
	SatisfactionAvg float64   `json:"satisfaction_average"`
}

type QualitySummary struct {
	AreaID             string  `json:"area_id"`
	OpenCount          int     `json:"open_count"`
	OverdueCount       int     `json:"overdue_count"`
	ClosureRate        float64 `json:"closure_rate"`
	AverageClosureHour float64 `json:"average_closure_hours"`
	Satisfaction       float64 `json:"satisfaction"`
}

func Percentage(part, total int) float64 {
	if total <= 0 {
		return 0
	}
	return float64(part) / float64(total) * 100
}
