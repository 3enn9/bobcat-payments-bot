import { useEffect, useState } from "react";
import { listBunkerHaulMessages, type BunkerHaulMessage } from "../api/bunkerHauls";

function todayISO(): string {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export default function WorkerMessages() {
  const [surname, setSurname] = useState("");
  const [date, setDate] = useState(todayISO);
  const [messages, setMessages] = useState<BunkerHaulMessage[]>([]);
  const [comments, setComments] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const normalizedSurname = surname.trim();
    if (!normalizedSurname) {
      setMessages([]);
      setLoading(false);
      setError("");
      return;
    }

    let cancelled = false;
	setMessages([]);
	setLoading(true);
	setError("");
    const timer = window.setTimeout(() => {
      void listBunkerHaulMessages(normalizedSurname, date)
        .then((items) => {
          if (!cancelled) {
            setMessages(items);
          }
        })
        .catch((err: unknown) => {
          if (!cancelled) {
            setMessages([]);
            setError(err instanceof Error ? err.message : "Ошибка загрузки");
          }
        })
        .finally(() => {
          if (!cancelled) {
            setLoading(false);
          }
        });
    }, 250);

    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [surname, date]);

  return (
    <div className="worker-requests bunker-worker">
      <label className="invoice-field">
        <span>Фамилия работника</span>
        <input
          value={surname}
          onChange={(event) => setSurname(event.target.value)}
          placeholder="Введите фамилию"
          autoComplete="off"
          maxLength={128}
        />
      </label>

      <label className="invoice-field">
        <span>Дата</span>
        <input
          type="date"
          value={date}
          onChange={(event) => setDate(event.target.value)}
        />
      </label>

      {loading && <p className="placeholder">Загружаю сообщения…</p>}
      {!loading && error && <div className="error">{error}</div>}
      {!loading && !error && surname.trim() !== "" && messages.length === 0 && (
        <p className="placeholder">Сообщений за эту дату нет.</p>
      )}

      {messages.length > 0 && (
        <ul className="requests-list bunker-message-list">
          {messages.map((message) => (
            <li key={message.id} className="request-item">
              <div className="request-item-meta">
                <time>{message.sentAt}</time>
              </div>
              <p className="request-item-message bunker-message-text">
                {message.text}
              </p>
            </li>
          ))}
        </ul>
      )}

      <label className="worker-comment">
        <span>Комментарии</span>
        <textarea
          value={comments}
          onChange={(event) => setComments(event.target.value)}
          placeholder="Введите комментарий"
          maxLength={2000}
          rows={3}
        />
      </label>

      <button type="button" className="fuels-search-btn" disabled>
        Отправить
      </button>
      <small className="invoice-hint">Отправка пока не настроена.</small>
    </div>
  );
}
