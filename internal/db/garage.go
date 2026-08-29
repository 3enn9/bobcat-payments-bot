package db

type GarageWorkLog struct {
	ID          int64  `json:"id"`
	WorkerName  string `json:"workerName"`
	WorkDate    string `json:"workDate"`
	TimeFrom    string `json:"timeFrom"`
	TimeTo      string `json:"timeTo"`
	Description string `json:"description"`
}

func (d *Database) CreateGarageWorkLog(workerName, workDate, timeFrom, timeTo, description string) (int64, error) {
	result, err := d.DB.Exec(`
		INSERT INTO garage_work_logs (worker_name, work_date, time_from, time_to, description)
		VALUES (?, ?, ?, ?, ?)
	`, workerName, workDate, timeFrom, timeTo, description)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
