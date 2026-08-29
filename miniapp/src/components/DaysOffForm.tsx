import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import {
  createDaysOff,
  fetchDaysOff,
  fetchUpcomingDaysOff,
  formatPeriod,
  statusLabel,
  type DaysOffItem,
} from "../api/daysOff";

function todayISO(): string {
  const now = new Date();
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, "0");
  const d = String(now.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

export default function DaysOffForm() {
  const [workerInput, setWorkerInput] = useState("");
  const [workerName, setWorkerName] = useState("");
  const [dateFrom, setDateFrom] = useState(todayISO());
  const [dateTo, setDateTo] = useState(todayISO());
  const [comment, setComment] = useState("");
  const [items, setItems] = useState<DaysOffItem[]>([]);
  const [upcoming, setUpcoming] = useState<DaysOffItem[]>([]);
  const [upcomingLoading, setUpcomingLoading] = useState(true);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  useEffect(() => {
    if (workerName) {
      return;
    }

    let cancelled = false;
    async function loadUpcoming() {
      setUpcomingLoading(true);
      try {
        const list = await fetchUpcomingDaysOff();
        if (!cancelled) {
          setUpcoming(list);
        }
      } catch {
        if (!cancelled) {
          setUpcoming([]);
        }
      } finally {
        if (!cancelled) {
          setUpcomingLoading(false);
        }
      }
    }

    void loadUpcoming();
    return () => {
      cancelled = true;
    };
  }, [workerName, success]);

  useEffect(() => {
    if (!workerName) {
      setItems([]);
      return;
    }

    let cancelled = false;
    async function load() {
      setLoading(true);
      setError("");
      try {
        const list = await fetchDaysOff(workerName);
        const today = todayISO();
        if (!cancelled) {
          setItems(list.filter((item) => item.dateTo >= today));
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Ошибка загрузки");
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [workerName, success]);

  function confirmWorker(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = workerInput.trim();
    if (!name) {
      setError("Введите фамилию");
      return;
    }
    setError("");
    setWorkerName(name);
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSuccess("");

    if (!workerName) {
      setError("Сначала подтвердите фамилию");
      return;
    }
    if (dateTo < dateFrom) {
      setError("Дата окончания раньше начала");
      return;
    }
    if (dateFrom < todayISO()) {
      setError("Нельзя выбрать прошедшие даты");
      return;
    }

    setSaving(true);
    try {
      await createDaysOff({
        workerName,
        dateFrom,
        dateTo,
        comment: comment.trim(),
      });
      setSuccess("Заявка отправлена руководителю");
      setComment("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка отправки");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="days-off-form">
      {!workerName ? (
        <form className="worker-name-form" onSubmit={confirmWorker}>
          <label className="invoice-field">
            <span>Фамилия</span>
            <input
              value={workerInput}
              onChange={(event) => setWorkerInput(event.target.value)}
              placeholder="Как в кабинете работника"
            />
          </label>
          {error && <div className="error">{error}</div>}
          <button type="submit">Продолжить</button>

          <section className="invoice-section days-off-upcoming">
            <h2>Актуальные выходные</h2>
            {upcomingLoading && <p className="invoice-hint">Загрузка...</p>}
            {!upcomingLoading && upcoming.length === 0 && (
              <p className="invoice-hint">Пока нет подтверждённых выходных</p>
            )}
            <ul className="days-off-list">
              {upcoming.map((item) => (
                <li key={item.id} className="days-off-item days-off-approved">
                  <div className="days-off-item-top">
                    <strong>{item.workerName}</strong>
                    <span>{formatPeriod(item.dateFrom, item.dateTo)}</span>
                  </div>
                  {item.comment && <p>{item.comment}</p>}
                </li>
              ))}
            </ul>
          </section>
        </form>
      ) : (
        <>
          <div className="days-off-worker">
            <span>{workerName}</span>
            <button
              type="button"
              className="secondary-button days-off-change-worker"
              onClick={() => {
                setWorkerName("");
                setWorkerInput("");
                setSuccess("");
                setError("");
              }}
            >
              Сменить
            </button>
          </div>

          <form className="invoice-section" onSubmit={submit}>
            <h2>Новая заявка</h2>
            <div className="days-off-dates">
              <label className="invoice-field">
                <span>С</span>
                <input
                  type="date"
                  className="days-off-date-input"
                  min={todayISO()}
                  value={dateFrom}
                  disabled={saving}
                  onChange={(event) => {
                    setDateFrom(event.target.value);
                    if (dateTo < event.target.value) {
                      setDateTo(event.target.value);
                    }
                  }}
                />
              </label>
              <label className="invoice-field">
                <span>По</span>
                <input
                  type="date"
                  className="days-off-date-input"
                  min={dateFrom}
                  value={dateTo}
                  disabled={saving}
                  onChange={(event) => setDateTo(event.target.value)}
                />
              </label>
            </div>
            <label className="invoice-field">
              <span>Комментарий</span>
              <input
                value={comment}
                disabled={saving}
                placeholder="Необязательно"
                onChange={(event) => setComment(event.target.value)}
              />
            </label>
            {error && <div className="error">{error}</div>}
            {success && <div className="invoice-success">{success}</div>}
            <button type="submit" disabled={saving}>
              {saving ? "Отправляем..." : "Подтвердить"}
            </button>
          </form>

          <section className="invoice-section">
            <h2>Мои заявки</h2>
            {loading && <p className="invoice-hint">Загрузка...</p>}
            {!loading && items.length === 0 && (
              <p className="invoice-hint">Пока нет заявок</p>
            )}
            <ul className="days-off-list">
              {items.map((item) => (
                <li key={item.id} className={`days-off-item days-off-${item.status}`}>
                  <div className="days-off-item-top">
                    <strong>{formatPeriod(item.dateFrom, item.dateTo)}</strong>
                    <span>{statusLabel(item.status)}</span>
                  </div>
                  {item.comment && <p>{item.comment}</p>}
                </li>
              ))}
            </ul>
          </section>
        </>
      )}
    </div>
  );
}
