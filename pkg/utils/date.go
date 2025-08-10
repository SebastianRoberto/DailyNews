package utils

import (
	"fmt"
	"time"
)

// FormatDate convierte una fecha a formato legible en español
func FormatDate(date time.Time) string {
	if date.IsZero() {
		return "Reciente"
	}

	now := time.Now()
	diff := now.Sub(date)

	// Si es hoy
	if date.Year() == now.Year() && date.YearDay() == now.YearDay() {
		return "Hoy"
	}

	// Si es ayer
	yesterday := now.AddDate(0, 0, -1)
	if date.Year() == yesterday.Year() && date.YearDay() == yesterday.YearDay() {
		return "Ayer"
	}

	// Si es esta semana (2-7 días)
	daysDiff := int(diff.Hours() / 24)
	if daysDiff >= 2 && daysDiff <= 7 {
		return fmt.Sprintf("%d días", daysDiff)
	}

	// Si es más antiguo, mostrar día y mes
	months := []string{"Ene", "Feb", "Mar", "Abr", "May", "Jun", "Jul", "Ago", "Sep", "Oct", "Nov", "Dic"}
	month := months[date.Month()-1]
	day := date.Day()

	return fmt.Sprintf("%d %s", day, month)
}

// FormatDateFromString convierte una fecha ISO string a formato legible
func FormatDateFromString(dateString string) string {
	if dateString == "" {
		return "Reciente"
	}

	// Parsear la fecha ISO
	date, err := time.Parse(time.RFC3339, dateString)
	if err != nil {
		// Intentar otros formatos comunes
		formats := []string{
			"2006-01-02T15:04:05.000Z",
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05.000",
			"2006-01-02 15:04:05",
		}

		for _, format := range formats {
			if parsed, err := time.Parse(format, dateString); err == nil {
				date = parsed
				break
			}
		}

		if date.IsZero() {
			return "Reciente"
		}
	}

	return FormatDate(date)
}

// GetDateRange devuelve las fechas de inicio y fin para rangos predefinidos
func GetDateRange(rangeType string) (time.Time, time.Time) {
	now := time.Now()

	switch rangeType {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		end := start.Add(24*time.Hour - time.Second)
		return start, end
	case "yesterday":
		yesterday := now.AddDate(0, 0, -1)
		start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
		end := start.Add(24*time.Hour - time.Second)
		return start, end
	case "this_week":
		// Lunes de esta semana
		weekday := now.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		} else {
			weekday--
		}
		start := now.AddDate(0, 0, -int(weekday))
		start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
		end := start.AddDate(0, 0, 7).Add(-time.Second)
		return start, end
	case "this_month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0).Add(-time.Second)
		return start, end
	case "last_month":
		start := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, 0).Add(-time.Second)
		return start, end
	default:
		// Últimos 7 días por defecto
		end := now
		start := now.AddDate(0, 0, -7)
		return start, end
	}
}

// FormatDateRange devuelve un string legible para un rango de fechas
func FormatDateRange(start, end time.Time) string {
	if start.Year() == end.Year() && start.Month() == end.Month() && start.Day() == end.Day() {
		// Mismo día
		return FormatDate(start)
	}

	months := []string{"Ene", "Feb", "Mar", "Abr", "May", "Jun", "Jul", "Ago", "Sep", "Oct", "Nov", "Dic"}

	if start.Year() == end.Year() {
		if start.Month() == end.Month() {
			// Mismo mes
			return fmt.Sprintf("%d-%d %s", start.Day(), end.Day(), months[start.Month()-1])
		}
		// Diferentes meses, mismo año
		return fmt.Sprintf("%d %s - %d %s", start.Day(), months[start.Month()-1], end.Day(), months[end.Month()-1])
	}

	// Diferentes años
	return fmt.Sprintf("%d %s %d - %d %s %d", start.Day(), months[start.Month()-1], start.Year(), end.Day(), months[end.Month()-1], end.Year())
}
