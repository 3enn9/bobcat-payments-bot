import { useEffect, useState } from "react";
import { fetchRogatkaRequests, type RogatkaRequest } from "../api/rogatka";

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
        {requests.map((item) => (
          <li key={item.id} className="request-item">
            <div className="request-item-meta">
              <span className="request-item-author">
                {item.maxUserName || item.maxUsername || "Без имени"}
              </span>
              <time dateTime={item.createdAt}>{formatDate(item.createdAt)}</time>
            </div>
            <p className="request-item-message">{item.message}</p>
          </li>
        ))}
      </ul>
    </div>
  );
}
