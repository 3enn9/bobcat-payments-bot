import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import {
  CASH_HISTORY_LIMIT,
  createCashEntry,
  fetchCashWorkers,
  fetchWorkerCash,
  updateCashEntry,
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

function toDateInputValue(value: string): string {
  const trimmed = value.trim();
  if (/^\d{4}-\d{2}-\d{2}$/.test(trimmed)) {
    return trimmed;
  }

  const isoPrefix = trimmed.match(/^(\d{4}-\d{2}-\d{2})/);
  if (isoPrefix) {
    return isoPrefix[1];
  }

  const date = new Date(trimmed.includes("T") ? trimmed : `${trimmed}T12:00:00`);
  if (Number.isNaN(date.getTime())) {
    return trimmed;
  }

  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

function formatAmountInput(value: number): string {
  if (Number.isInteger(value)) {
    return String(value);
  }
  return String(value).replace(".", ",");
}

type EditDraft = {
  entryType: CashEntryType;
  amount: string;
  entryDate: string;
  description: string;
};

async function reloadCash(workerName: string) {
  const data = await fetchWorkerCash(workerName);
  return data;
}

export default function CashForm() {
  const [workerInput, setWorkerInput] = useState("");
  const [workerName, setWorkerName] = useState("");
  const [balance, setBalance] = useState(0);
  const [entries, setEntries] = useState<CashEntry[]>([]);
  const [totalEntries, setTotalEntries] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [entryType, setEntryType] = useState<CashEntryType>("income");
  const [amount, setAmount] = useState("");
  const [entryDate, setEntryDate] = useState(todayISO());
  const [description, setDescription] = useState("");
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState("");
  const [success, setSuccess] = useState("");
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editDraft, setEditDraft] = useState<EditDraft | null>(null);
  const [editError, setEditError] = useState("");
  const [editSaving, setEditSaving] = useState(false);
  const [workers, setWorkers] = useState<string[]>([]);
  const [workersLoading, setWorkersLoading] = useState(false);
  const [workersError, setWorkersError] = useState("");

  useEffect(() => {
    if (workerName) {
      return;
    }

    let cancelled = false;

    async function loadWorkers() {
      setWorkersLoading(true);
      setWorkersError("");
      try {
        const list = await fetchCashWorkers();
        if (!cancelled) {
          setWorkers(list);
        }
      } catch (err) {
        if (!cancelled) {
          setWorkers([]);
          setWorkersError(
            err instanceof Error
              ? err.message
              : "Не удалось загрузить список работников",
          );
        }
      } finally {
        if (!cancelled) {
          setWorkersLoading(false);
        }
      }
    }

    void loadWorkers();
    return () => {
      cancelled = true;
    };
  }, [workerName]);

  useEffect(() => {
    if (!workerName) {
      setEntries([]);
      setBalance(0);
      setTotalEntries(0);
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
          setTotalEntries(data.total);
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

  function selectWorker(name: string) {
    setWorkerName(name);
    setWorkerInput(name);
    setEditingId(null);
    setEditDraft(null);
    setError("");
    setFormError("");
    setSuccess("");
  }

  function confirmWorker(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = workerInput.trim();
    if (!name) {
      setError("Введите фамилию");
      return;
    }
    selectWorker(name);
  }

  const searchQuery = workerInput.trim().toLowerCase();
  const filteredWorkers = workers.filter((name) =>
    searchQuery === "" ? true : name.toLowerCase().includes(searchQuery),
  );

  function parseAmount(raw: string): number | null {
    const parsed = Number(raw.replace(",", "."));
    if (!Number.isFinite(parsed) || parsed <= 0) {
      return null;
    }
    return parsed;
  }

  async function submitEntry(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError("");
    setSuccess("");

    const parsedAmount = parseAmount(amount);
    if (parsedAmount === null) {
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
      const data = await reloadCash(workerName);
      setEntries(data.entries);
      setTotalEntries(data.total);
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

  function startEdit(item: CashEntry) {
    setEditingId(item.id);
    setEditDraft({
      entryType: item.entryType,
      amount: formatAmountInput(item.amount),
      entryDate: toDateInputValue(item.entryDate),
      description: item.description,
    });
    setEditError("");
    setSuccess("");
  }

  function cancelEdit() {
    setEditingId(null);
    setEditDraft(null);
    setEditError("");
  }

  async function saveEdit(id: number) {
    if (!editDraft) {
      return;
    }

    const parsedAmount = parseAmount(editDraft.amount);
    if (parsedAmount === null) {
      setEditError("Укажите сумму больше нуля");
      return;
    }
    if (!editDraft.entryDate) {
      setEditError("Укажите дату");
      return;
    }

    setEditSaving(true);
    setEditError("");
    try {
      const result = await updateCashEntry(id, {
        workerName,
        entryType: editDraft.entryType,
        amount: parsedAmount,
        description: editDraft.description.trim(),
        entryDate: editDraft.entryDate,
      });
      setBalance(result.balance);
      const data = await reloadCash(workerName);
      setEntries(data.entries);
      setTotalEntries(data.total);
      cancelEdit();
      setSuccess("Запись обновлена");
    } catch (err) {
      setEditError(
        err instanceof Error ? err.message : "Не удалось обновить запись",
      );
    } finally {
      setEditSaving(false);
    }
  }

  const busy = saving || editSaving;

  return (
    <div className="cash-form">
      {!workerName ? (
        <form className="driver-lookup" onSubmit={confirmWorker}>
          <label>
            <span>Фамилия работника</span>
            <input
              value={workerInput}
              onChange={(event) => setWorkerInput(event.target.value)}
              placeholder="Иванов"
              maxLength={100}
            />
          </label>
          {error && <div className="error">{error}</div>}
          <button type="submit">Подтвердить</button>

          <section className="cash-workers-section">
            <h2>Работники</h2>
            {workersError && <div className="error">{workersError}</div>}
            {workersLoading && <p className="placeholder">Загрузка списка...</p>}
            {!workersLoading && !workersError && filteredWorkers.length === 0 && (
              <p className="placeholder">
                {workers.length === 0
                  ? "Список пока пуст"
                  : "Никого не найдено по запросу"}
              </p>
            )}
            {!workersLoading && !workersError && filteredWorkers.length > 0 && (
              <ul className="cash-workers-list">
                {filteredWorkers.map((name) => (
                  <li key={name}>
                    <button
                      type="button"
                      className="cash-worker-button"
                      onClick={() => selectWorker(name)}
                    >
                      {name}
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </form>
      ) : (
        <>
          <div className="days-off-worker">
            <span>{workerName}</span>
            <button
              type="button"
              className="secondary-button days-off-change-worker"
              disabled={busy}
              onClick={() => {
                setWorkerName("");
                setWorkerInput("");
                setEditingId(null);
                setEditDraft(null);
                setError("");
                setFormError("");
                setSuccess("");
              }}
            >
              Сменить
            </button>
          </div>

          {loading && <p className="placeholder">Загрузка кассы...</p>}

          {!loading && error && <div className="error">{error}</div>}

          {!loading && !error && (
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
                disabled={busy}
                onClick={() => setEntryType("income")}
              >
                + Приход
              </button>
              <button
                type="button"
                className={entryType === "expense" ? "cash-type active" : "cash-type"}
                disabled={busy}
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
                disabled={busy}
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
                disabled={busy}
                onChange={(event) => setEntryDate(event.target.value)}
              />
            </label>

            <label className="invoice-field">
              <span>Комментарий</span>
              <textarea
                value={description}
                disabled={busy}
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

            <button type="submit" disabled={busy}>
              {saving ? "Сохраняем..." : "Добавить запись"}
            </button>
          </form>

          <section className="cash-history">
            <div className="cash-history-head">
              <h2>Последние операции</h2>
              {totalEntries > 0 && (
                <span className="cash-history-note">
                  {totalEntries > CASH_HISTORY_LIMIT
                    ? `Показаны ${CASH_HISTORY_LIMIT} из ${totalEntries}`
                    : `${totalEntries} ${totalEntries === 1 ? "запись" : totalEntries < 5 ? "записи" : "записей"}`}
                </span>
              )}
            </div>

            {entries.length === 0 ? (
              <p className="placeholder">Записей пока нет.</p>
            ) : (
              <ul className="cash-list">
                {entries.map((item) => {
                  const isIncome = item.entryType === "income";
                  const isEditing = editingId === item.id;

                  if (isEditing && editDraft) {
                    return (
                      <li key={item.id} className="cash-list-item cash-list-item-editing">
                        <div className="cash-type-toggle">
                          <button
                            type="button"
                            className={
                              editDraft.entryType === "income"
                                ? "cash-type active"
                                : "cash-type"
                            }
                            disabled={editSaving}
                            onClick={() =>
                              setEditDraft({ ...editDraft, entryType: "income" })
                            }
                          >
                            + Приход
                          </button>
                          <button
                            type="button"
                            className={
                              editDraft.entryType === "expense"
                                ? "cash-type active"
                                : "cash-type"
                            }
                            disabled={editSaving}
                            onClick={() =>
                              setEditDraft({ ...editDraft, entryType: "expense" })
                            }
                          >
                            − Расход
                          </button>
                        </div>

                        <label className="invoice-field">
                          <span>Сумма, ₽</span>
                          <input
                            type="text"
                            inputMode="decimal"
                            value={editDraft.amount}
                            disabled={editSaving}
                            onChange={(event) =>
                              setEditDraft({
                                ...editDraft,
                                amount: event.target.value,
                              })
                            }
                          />
                        </label>

                        <label className="invoice-field">
                          <span>Дата</span>
                          <input
                            type="date"
                            className="days-off-date-input"
                            value={editDraft.entryDate}
                            disabled={editSaving}
                            onChange={(event) =>
                              setEditDraft({
                                ...editDraft,
                                entryDate: event.target.value,
                              })
                            }
                          />
                        </label>

                        <label className="invoice-field">
                          <span>Комментарий</span>
                          <textarea
                            value={editDraft.description}
                            disabled={editSaving}
                            rows={2}
                            onChange={(event) =>
                              setEditDraft({
                                ...editDraft,
                                description: event.target.value,
                              })
                            }
                          />
                        </label>

                        {editError && <div className="error">{editError}</div>}

                        <div className="cash-edit-actions">
                          <button
                            type="button"
                            className="secondary-button"
                            disabled={editSaving}
                            onClick={cancelEdit}
                          >
                            Отмена
                          </button>
                          <button
                            type="button"
                            disabled={editSaving}
                            onClick={() => void saveEdit(item.id)}
                          >
                            {editSaving ? "Сохраняем..." : "Сохранить"}
                          </button>
                        </div>
                      </li>
                    );
                  }

                  return (
                    <li key={item.id} className="cash-list-item">
                      <div className="cash-list-row">
                        <span
                          className={
                            isIncome ? "cash-sign cash-sign-plus" : "cash-sign cash-sign-minus"
                          }
                          aria-hidden="true"
                        >
                          {isIncome ? "+" : "−"}
                        </span>

                        <div className="cash-list-body">
                          <div className="cash-list-main">
                            <span className={isIncome ? "cash-plus" : "cash-minus"}>
                              {formatMoney(item.amount)}
                            </span>
                            <time dateTime={item.entryDate}>
                              {formatDate(item.entryDate)}
                            </time>
                          </div>
                          {item.description && (
                            <p className="cash-list-desc">{item.description}</p>
                          )}
                        </div>

                        <button
                          type="button"
                          className="cash-edit-button"
                          disabled={busy}
                          onClick={() => startEdit(item)}
                        >
                          Изменить
                        </button>
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}
          </section>
            </>
          )}
        </>
      )}
    </div>
  );
}
