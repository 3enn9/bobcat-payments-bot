import { useState } from "react";
import type { FormEvent } from "react";
import { createGarageWork } from "../api/garage";

function todayISO(): string {
  const now = new Date();
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, "0");
  const d = String(now.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

export default function GarageForm() {
  const [surname, setSurname] = useState("");
  const [workDate, setWorkDate] = useState(todayISO());
  const [timeFrom, setTimeFrom] = useState("");
  const [timeTo, setTimeTo] = useState("");
  const [description, setDescription] = useState("");
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [saving, setSaving] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSuccess("");

    if (!surname.trim()) {
      setError("Укажите фамилию");
      return;
    }
    if (!workDate) {
      setError("Укажите дату");
      return;
    }
    if (!timeFrom || !timeTo) {
      setError("Укажите время начала и окончания");
      return;
    }
    if (timeTo <= timeFrom) {
      setError("Время окончания должно быть позже начала");
      return;
    }
    if (!description.trim()) {
      setError("Опишите, что сделали за время работы");
      return;
    }

    setSaving(true);
    try {
      await createGarageWork({
        workerName: surname.trim(),
        workDate,
        timeFrom,
        timeTo,
        description: description.trim(),
      });
      setSuccess("Запись сохранена");
      setTimeFrom("");
      setTimeTo("");
      setDescription("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка отправки");
    } finally {
      setSaving(false);
    }
  }

  return (
    <form className="garage-form invoice-section" onSubmit={submit}>
      <label className="invoice-field">
        <span>Фамилия</span>
        <input
          value={surname}
          disabled={saving}
          placeholder="Иванов"
          onChange={(event) => setSurname(event.target.value)}
        />
      </label>

      <label className="invoice-field">
        <span>Дата</span>
        <input
          type="date"
          className="days-off-date-input"
          value={workDate}
          disabled={saving}
          onChange={(event) => setWorkDate(event.target.value)}
        />
      </label>

      <div className="days-off-dates">
        <label className="invoice-field">
          <span>Начало работы</span>
          <input
            type="time"
            value={timeFrom}
            disabled={saving}
            onChange={(event) => setTimeFrom(event.target.value)}
          />
        </label>
        <label className="invoice-field">
          <span>Конец работы</span>
          <input
            type="time"
            value={timeTo}
            disabled={saving}
            onChange={(event) => setTimeTo(event.target.value)}
          />
        </label>
      </div>

      <label className="invoice-field">
        <span>Описание</span>
        <textarea
          value={description}
          disabled={saving}
          rows={4}
          placeholder="Что сделали за время работы"
          onChange={(event) => setDescription(event.target.value)}
        />
      </label>

      {error && <div className="error">{error}</div>}
      {success && <div className="invoice-success">{success}</div>}

      <button type="submit" disabled={saving}>
        {saving ? "Отправляем..." : "Отправить"}
      </button>
    </form>
  );
}
