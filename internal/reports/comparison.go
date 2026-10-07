package reports

import (
	"fmt"
	"sitewise/internal/delivery"
	"time"
)

// DeliveryComparison compares saved dates only. Actual takes precedence over
// forecast; an absent date is unknown, never an inferred schedule promise.
func DeliveryComparison(d delivery.Item) *Table {
	if d.Kind != "milestone" && d.Kind != "activity" {
		return nil
	}
	value := func(v *string) string {
		if v == nil || *v == "" {
			return "Unknown"
		}
		return *v
	}
	current, basis := d.ActualDate, "Actual"
	if current == nil || *current == "" {
		current, basis = d.ForecastDate, "Forecast"
	}
	variance := "Unknown"
	if current != nil && d.TargetDate != nil {
		actual, aerr := time.Parse("2006-01-02", *current)
		target, terr := time.Parse("2006-01-02", *d.TargetDate)
		if aerr == nil && terr == nil {
			days := int(actual.Sub(target).Hours() / 24)
			unit := "days"
			if days == 1 || days == -1 {
				unit = "day"
			}
			switch {
			case days > 0:
				variance = fmt.Sprintf("%d %s later", days, unit)
			case days < 0:
				variance = fmt.Sprintf("%d %s earlier", -days, unit)
			default:
				variance = "On target"
			}
		}
	}
	return &Table{Columns: []string{"Target", "Current (" + basis + ")", "Variance"}, Rows: [][]string{{value(d.TargetDate), value(current), variance}}}
}
