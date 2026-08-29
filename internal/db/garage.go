package db

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
