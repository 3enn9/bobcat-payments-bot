import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";
import {
  completeRogatkaRequest,
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

type Draft = {
  photos: File[];
  comment: string;
  showComment: boolean;
};

export default function WorkerCabinet() {
  const [driverInput, setDriverInput] = useState("");
  const [driverName, setDriverName] = useState("");
  const [requests, setRequests] = useState<RogatkaRequest[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [drafts, setDrafts] = useState<Record<number, Draft>>({});
  const [sendingId, setSendingId] = useState<number | null>(null);
  const [actionError, setActionError] = useState<Record<number, string>>({});
  const fileInputs = useRef<Record<number, HTMLInputElement | null>>({});

  useEffect(() => {
    if (!driverName) {
      setRequests([]);
      return;
    }

    let cancelled = false;

    async function load() {
      setLoading(true);
      setError("");

      try {
        const items = await fetchRogatkaRequests("assigned", driverName);
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
  }, [driverName]);

  function confirmDriver(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = driverInput.trim();
    if (!name) {
      setError("Введите фамилию");
      return;
    }

    setDriverName(name);
    setDrafts({});
    setActionError({});
    setError("");
  }

  function getDraft(id: number): Draft {
    return drafts[id] ?? { photos: [], comment: "", showComment: false };
  }

  function updateDraft(id: number, patch: Partial<Draft>) {
    setDrafts((prev) => ({
      ...prev,
      [id]: {
        ...getDraft(id),
        ...patch,
      },
    }));
  }

  async function sendRequest(id: number) {
    const draft = getDraft(id);
    setSendingId(id);
    setActionError((prev) => ({ ...prev, [id]: "" }));

    try {
      await completeRogatkaRequest(
        id,
        driverName,
        draft.comment.trim(),
        draft.photos,
      );
      setRequests((prev) => prev.filter((item) => item.id !== id));
      setDrafts((prev) => {
        const next = { ...prev };
        delete next[id];
        return next;
      });
    } catch (err) {
      setActionError((prev) => ({
        ...prev,
        [id]:
          err instanceof Error ? err.message : "Не удалось отправить заявку",
      }));
    } finally {
      setSendingId(null);
    }
  }

  return (
    <div className="worker-requests">
      <form className="driver-lookup" onSubmit={confirmDriver}>
        <label>
          <span>Фамилия работника</span>
          <input
            value={driverInput}
            onChange={(event) => setDriverInput(event.target.value)}
            placeholder="Иванов"
            maxLength={100}
            disabled={loading || sendingId !== null}
          />
        </label>
        <button type="submit" disabled={loading || sendingId !== null}>
          Подтвердить
        </button>
      </form>

      {!driverName && (
        <p className="placeholder">Введите фамилию, чтобы увидеть заявки.</p>
      )}

      {driverName && loading && (
        <p className="placeholder">Загрузка заявок...</p>
      )}

      {driverName && !loading && error && <div className="error">{error}</div>}

      {driverName && !loading && !error && requests.length === 0 && (
        <p className="placeholder">Назначенных заявок нет.</p>
      )}

      {driverName && !loading && !error && requests.length > 0 && (
        <div className="requests-scroll">
          <ul className="requests-list">
            {requests.map((item) => {
              const draft = getDraft(item.id);
              const busy = sendingId === item.id;

              return (
                <li key={item.id} className="request-item">
                  <div className="request-item-meta">
                    <time dateTime={item.createdAt}>
                      {formatDate(item.createdAt)}
                    </time>
                  </div>

                  <p className="request-item-message">{item.message}</p>

                  <div className="worker-badges">
                    {draft.photos.length > 0 && (
                      <span className="badge">
                        📷 {draft.photos.length} фото
                      </span>
                    )}
                    {draft.comment.trim() && (
                      <span className="badge">💬 комментарий</span>
                    )}
                  </div>

                  {draft.showComment && (
                    <label className="worker-comment">
                      <span>Описание</span>
                      <textarea
                        value={draft.comment}
                        onChange={(event) =>
                          updateDraft(item.id, {
                            comment: event.target.value,
                          })
                        }
                        placeholder="Комментарий по заявке"
                        maxLength={2000}
                        rows={3}
                        disabled={busy}
                      />
                    </label>
                  )}

                  {draft.photos.length > 0 && (
                    <ul className="photo-names">
                      {draft.photos.map((file) => (
                        <li key={`${file.name}-${file.size}-${file.lastModified}`}>
                          {file.name}
                        </li>
                      ))}
                    </ul>
                  )}

                  {actionError[item.id] && (
                    <div className="error">{actionError[item.id]}</div>
                  )}

                  <div className="worker-actions">
                    <input
                      ref={(node) => {
                        fileInputs.current[item.id] = node;
                      }}
                      type="file"
                      accept="image/*"
                      multiple
                      hidden
                      onChange={(event) => {
                        const files = Array.from(event.target.files ?? []);
                        updateDraft(item.id, {
                          photos: [...getDraft(item.id).photos, ...files].slice(
                            0,
                            5,
                          ),
                        });
                        event.target.value = "";
                      }}
                    />

                    <button
                      type="button"
                      className="secondary-button"
                      disabled={busy}
                      onClick={() => fileInputs.current[item.id]?.click()}
                    >
                      Фото
                    </button>

                    <button
                      type="button"
                      className="secondary-button"
                      disabled={busy}
                      onClick={() =>
                        updateDraft(item.id, {
                          showComment: !draft.showComment,
                        })
                      }
                    >
                      Описание
                    </button>

                    <button
                      type="button"
                      disabled={busy}
                      onClick={() => void sendRequest(item.id)}
                    >
                      {busy ? "Отправка..." : "Отправить"}
                    </button>
                  </div>
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </div>
  );
}
