import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import {
  assignRogatkaDriver,
  deleteRogatkaRequest,
  fetchRogatkaRequests,
  type RogatkaRequest,
} from "../api/rogatka";

type PanelMode =
  | { type: "assign"; id: number }
  | { type: "delete"; id: number }
  | null;

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
  const [panel, setPanel] = useState<PanelMode>(null);
  const [driverName, setDriverName] = useState("");
  const [actionError, setActionError] = useState("");
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

  function closePanel() {
    setPanel(null);
    setDriverName("");
    setActionError("");
  }

  function openAssign(id: number) {
    setPanel({ type: "assign", id });
    setDriverName("");
    setActionError("");
  }

  function openDelete(id: number) {
    setPanel({ type: "delete", id });
    setDriverName("");
    setActionError("");
  }

  async function confirmAssign(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (panel?.type !== "assign") {
      return;
    }

    const name = driverName.trim();
    if (!name) {
      setActionError("Введите фамилию водителя");
      return;
    }

    setSaving(true);
    setActionError("");

    try {
      await assignRogatkaDriver(panel.id, name);
      setRequests((prev) => prev.filter((item) => item.id !== panel.id));
      closePanel();
    } catch (err) {
      setActionError(
        err instanceof Error ? err.message : "Не удалось назначить водителя",
      );
    } finally {
      setSaving(false);
    }
  }

  async function confirmDelete() {
    if (panel?.type !== "delete") {
      return;
    }

    setSaving(true);
    setActionError("");

    try {
      await deleteRogatkaRequest(panel.id);
      setRequests((prev) => prev.filter((item) => item.id !== panel.id));
      closePanel();
    } catch (err) {
      setActionError(
        err instanceof Error ? err.message : "Не удалось удалить заявку",
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
          const isAssigning = panel?.type === "assign" && panel.id === item.id;
          const isDeleting = panel?.type === "delete" && panel.id === item.id;

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
                    aria-label="Удалить заявку"
                    disabled={saving}
                    onClick={() => openDelete(item.id)}
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

                  {actionError && <div className="error">{actionError}</div>}

                  <div className="assign-form-actions">
                    <button
                      type="button"
                      className="secondary-button"
                      onClick={closePanel}
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

              {isDeleting && (
                <div className="assign-form delete-confirm">
                  <p className="delete-confirm-text">Удалить заявку?</p>

                  {actionError && <div className="error">{actionError}</div>}

                  <div className="assign-form-actions">
                    <button
                      type="button"
                      className="secondary-button"
                      onClick={closePanel}
                      disabled={saving}
                    >
                      Отмена
                    </button>
                    <button
                      type="button"
                      className="danger-button"
                      onClick={confirmDelete}
                      disabled={saving}
                    >
                      {saving ? "Удаляем..." : "Удалить"}
                    </button>
                  </div>
                </div>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
