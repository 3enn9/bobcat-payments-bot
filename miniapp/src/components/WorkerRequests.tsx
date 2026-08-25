import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import {
  assignRogatkaDriver,
  fetchRogatkaRequests,
  type RogatkaRequest,
} from "../api/rogatka";

function formatDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default function WorkerRequests() {
  const [requests, setRequests] = useState<RogatkaRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [assigningId, setAssigningId] = useState<number | null>(null);
  const [driverName, setDriverName] = useState("");
  const [assignError, setAssignError] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError("");

      try {
        const items = await fetchRogatkaRequests();
        if (!cancelled) {
          setRequests(items);
        }
      } catch (err) {
        if (!cancelled) {
          setError(
            err instanceof Error ? err.message : "Не удалось загрузить заявки",
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
  }, []);

  function openAssign(id: number) {
    setAssigningId(id);
    setDriverName("");
    setAssignError("");
  }

  function closeAssign() {
    setAssigningId(null);
    setDriverName("");
    setAssignError("");
  }

  async function confirmAssign(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (assigningId === null) {
      return;
    }

    const name = driverName.trim();
    if (!name) {
      setAssignError("Введите фамилию водителя");
      return;
    }

    setSaving(true);
    setAssignError("");

    try {
      await assignRogatkaDriver(assigningId, name);
      setRequests((prev) => prev.filter((item) => item.id !== assigningId));
      closeAssign();
    } catch (err) {
      setAssignError(
        err instanceof Error ? err.message : "Не удалось назначить водителя",
      );
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return <p className="placeholder">Загрузка заявок...</p>;
  }

  if (error) {
    return <div className="error">{error}</div>;
  }

  if (requests.length === 0) {
    return <p className="placeholder">Заявок пока нет.</p>;
  }

  return (
    <div className="requests-scroll">
      <ul className="requests-list">
        {requests.map((item) => {
          const isAssigning = assigningId === item.id;

          return (
            <li key={item.id} className="request-item">
              <div className="request-item-top">
                <div className="request-item-meta">
                  <span className="request-item-author">
                    {item.maxUserName || item.maxUsername || "Без имени"}
                  </span>
                  <time dateTime={item.createdAt}>
                    {formatDate(item.createdAt)}
                  </time>
                </div>

                <div className="request-item-actions">
                  <button
                    type="button"
                    className="icon-action icon-action-plus"
                    aria-label="Назначить водителя"
                    disabled={saving}
                    onClick={() => openAssign(item.id)}
                  >
                    +
                  </button>
                  <button
                    type="button"
                    className="icon-action icon-action-minus"
                    aria-label="Дополнительное действие"
                    disabled
                    title="Скоро"
                  >
                    −
                  </button>
                </div>
              </div>

              <p className="request-item-message">{item.message}</p>

              {isAssigning && (
                <form className="assign-form" onSubmit={confirmAssign}>
                  <label>
                    <span>Фамилия водителя</span>
                    <input
                      value={driverName}
                      onChange={(event) => setDriverName(event.target.value)}
                      placeholder="Иванов"
                      maxLength={100}
                      autoFocus
                      disabled={saving}
                    />
                  </label>

                  {assignError && <div className="error">{assignError}</div>}

                  <div className="assign-form-actions">
                    <button
                      type="button"
                      className="secondary-button"
                      onClick={closeAssign}
                      disabled={saving}
                    >
                      Отмена
                    </button>
                    <button type="submit" disabled={saving}>
                      {saving ? "Сохраняем..." : "Подтвердить"}
                    </button>
                  </div>
                </form>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
