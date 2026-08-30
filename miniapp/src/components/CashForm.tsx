import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import {
  createCashEntry,
  fetchWorkerCash,
  type CashEntry,
  type CashEntryType,
} from "../api/cash";

function todayISO(): string {
  const now = new Date();
  const y = now.getFullYear();
  const m = String(now.getMonth() + 1).padStart(2, "0");
  const d = String(now.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

function formatMoney(value: number): string {
  return new Intl.NumberFormat("ru-RU", {
    style: "currency",
    currency: "RUB",
    maximumFractionDigits: 2,
  }).format(value);
}

function formatDate(value: string): string {
  const date = new Date(value.includes("T") ? value : `${value}T00:00:00`);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleDateString("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  });
}

export default function CashForm() {
  const [workerInput, setWorkerInput] = useState("");
  const [workerName, setWorkerName] = useState("");
  const [balance, setBalance] = useState(0);
  const [entries, setEntries] = useState<CashEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [entryType, setEntryType] = useState<CashEntryType>("income");
  const [amount, setAmount] = useState("");
  const [entryDate, setEntryDate] = useState(todayISO());
  const [description, setDescription] = useState("");
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState("");
  const [success, setSuccess] = useState("");

  useEffect(() => {
    if (!workerName) {
      setEntries([]);
      setBalance(0);
      return;
    }

    let cancelled = false;

    async function load() {
      setLoading(true);
      setError("");
      try {
        const data = await fetchWorkerCash(workerName);
        if (!cancelled) {
          setBalance(data.balance);
          setEntries(data.entries);
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error ? err.message : "Не удалось загрузить кассу",
          );
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
  }, [workerName]);

  function confirmWorker(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = workerInput.trim();
    if (!name) {
      setError("Введите фамилию");
      return;
    }
    setWorkerName(name);
    setError("");
    setFormError("");
    setSuccess("");
  }

  async function submitEntry(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError("");
    setSuccess("");

    const parsedAmount = Number(amount.replace(",", "."));
    if (!Number.isFinite(parsedAmount) || parsedAmount <= 0) {
      setFormError("Укажите сумму больше нуля");
      return;
    }
    if (!entryDate) {
      setFormError("Укажите дату");
      return;
    }

    setSaving(true);
    try {
      const result = await createCashEntry({
        workerName,
        entryType,
        amount: parsedAmount,
        description: description.trim(),
        entryDate,
      });
      setBalance(result.balance);
      const data = await fetchWorkerCash(workerName);
      setEntries(data.entries);
      setAmount("");
      setDescription("");
      setSuccess(entryType === "income" ? "Приход добавлен" : "Расход добавлен");
    } catch (err) {
      setFormError(
        err instanceof Error ? err.message : "Не удалось сохранить запись",
      );
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="cash-form">
      <form className="driver-lookup" onSubmit={confirmWorker}>
        <label>
          <span>Фамилия работника</span>
          <input
            value={workerInput}
            onChange={(event) => setWorkerInput(event.target.value)}
            placeholder="Иванов"
            maxLength={100}
            disabled={loading || saving}
          />
        </label>
        <button type="submit" disabled={loading || saving}>
          Подтвердить
        </button>
      </form>

      {!workerName && (
        <p className="placeholder">Введите фамилию, чтобы открыть кассу.</p>
      )}

      {workerName && loading && (
        <p className="placeholder">Загрузка кассы...</p>
      )}

      {workerName && !loading && error && <div className="error">{error}</div>}

      {workerName && !loading && !error && (
        <>
          <div className="cash-balance">
            <span>Баланс кассы</span>
            <strong>{formatMoney(balance)}</strong>
          </div>

          <form className="cash-entry-form invoice-section" onSubmit={submitEntry}>
            <div className="cash-type-toggle">
              <button
                type="button"
                className={entryType === "income" ? "cash-type active" : "cash-type"}
                disabled={saving}
                onClick={() => setEntryType("income")}
              >
                + Приход
              </button>
              <button
                type="button"
                className={entryType === "expense" ? "cash-type active" : "cash-type"}
                disabled={saving}
                onClick={() => setEntryType("expense")}
              >
                − Расход
              </button>
            </div>

            <label className="invoice-field">
              <span>Сумма, ₽</span>
              <input
                type="text"
                inputMode="decimal"
                value={amount}
                disabled={saving}
                placeholder="1500"
                onChange={(event) => setAmount(event.target.value)}
              />
            </label>

            <label className="invoice-field">
              <span>Дата</span>
              <input
                type="date"
                className="days-off-date-input"
                value={entryDate}
                disabled={saving}
                onChange={(event) => setEntryDate(event.target.value)}
              />
            </label>

            <label className="invoice-field">
              <span>Комментарий</span>
              <textarea
                value={description}
                disabled={saving}
                rows={2}
                placeholder={
                  entryType === "income"
                    ? "Частный заказ, наличные"
                    : "На что потратили"
                }
                onChange={(event) => setDescription(event.target.value)}
              />
            </label>

            {formError && <div className="error">{formError}</div>}
            {success && <div className="invoice-success">{success}</div>}

            <button type="submit" disabled={saving}>
              {saving ? "Сохраняем..." : "Добавить запись"}
            </button>
          </form>

          <section className="cash-history">
            <h2>История</h2>
            {entries.length === 0 ? (
              <p className="placeholder">Записей пока нет.</p>
            ) : (
              <ul className="cash-list">
                {entries.map((item) => {
                  const isIncome = item.entryType === "income";
                  return (
                    <li key={item.id} className="cash-list-item">
                      <div className="cash-list-main">
                        <span className={isIncome ? "cash-plus" : "cash-minus"}>
                          {isIncome ? "+" : "−"}
                          {formatMoney(item.amount)}
                        </span>
                        <time dateTime={item.entryDate}>
                          {formatDate(item.entryDate)}
                        </time>
                      </div>
                      {item.description && (
                        <p className="cash-list-desc">{item.description}</p>
                      )}
                    </li>
                  );
                })}
              </ul>
            )}
          </section>
        </>
      )}
    </div>
  );
}
