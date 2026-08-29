package db

import (
	"fmt"
	"strings"
	"time"
)

type GarageWorkLog struct {
	ID            int64  `json:"id"`
	WorkerName    string `json:"workerName"`
	WorkDate      string `json:"workDate"`
	TimeFrom      string `json:"timeFrom"`
	TimeTo        string `json:"timeTo"`
	WorkedMinutes int    `json:"workedMinutes"`
	Description   string `json:"description"`
}

func (d *Database) CreateGarageWorkLog(workerName, workDate, timeFrom, timeTo string, workedMinutes int, description string) (int64, error) {
	result, err := d.DB.Exec(`
		INSERT INTO garage_work_logs (worker_name, work_date, time_from, time_to, worked_minutes, description)
		VALUES (?, ?, ?, ?, ?, ?)
	`, workerName, workDate, timeFrom, timeTo, workedMinutes, description)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func FormatGarageWorkMessage(item GarageWorkLog) string {
	dateLabel := formatRuDateISO(item.WorkDate)
	timeFrom := trimSeconds(item.TimeFrom)
	timeTo := trimSeconds(item.TimeTo)

	var b strings.Builder
	b.WriteString("Гараж — отчёт о работе\n\n")
	b.WriteString(fmt.Sprintf("Работник: %s\n", item.WorkerName))
	b.WriteString(fmt.Sprintf("Дата: %s\n", dateLabel))
	b.WriteString(fmt.Sprintf("Время: %s – %s\n", timeFrom, timeTo))
	b.WriteString(fmt.Sprintf("Отработано: %s\n", FormatWorkedDuration(item.WorkedMinutes)))
	b.WriteString("\nОписание:\n")
	b.WriteString(strings.TrimSpace(item.Description))
	return b.String()
}

func FormatWorkedDuration(minutes int) string {
	if minutes <= 0 {
		return "0 мин"
	}
	hours := minutes / 60
	mins := minutes % 60
	if hours == 0 {
		return fmt.Sprintf("%d мин", mins)
	}
	if mins == 0 {
		return fmt.Sprintf("%d ч", hours)
	}
	return fmt.Sprintf("%d ч %d мин", hours, mins)
}

func trimSeconds(value string) string {
	if t, err := time.Parse("15:04:05", value); err == nil {
		return t.Format("15:04")
	}
	if t, err := time.Parse("15:04", value); err == nil {
		return t.Format("15:04")
	}
	return value
}
