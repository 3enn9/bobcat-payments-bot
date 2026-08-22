import { useState } from "react";
import type { FormEvent } from "react";
import { createRequest } from "../api/requests";

type Props = {
  onSuccess: (id: number) => void;
};

export default function RequestForm({ onSuccess }: Props) {
  const [name, setName] = useState("");
  const [contact, setContact] = useState("");
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");

    if (!name.trim() || !contact.trim() || !message.trim()) {
      setError("Заполните все поля.");
      return;
    }

    setLoading(true);

    try {
      const result = await createRequest({
        name,
        contact,
        message,
      });

      if (result.id) {
        onSuccess(result.id);
      }
    } catch (error) {
      setError(
        error instanceof Error
          ? error.message
          : "Произошла ошибка",
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <form className="request-form" onSubmit={submit}>
      <label>
        <span>Имя</span>
        <input
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="Иван"
          maxLength={100}
          disabled={loading}
        />
      </label>

      <label>
        <span>Контакт</span>
        <input
          value={contact}
          onChange={(event) => setContact(event.target.value)}
          placeholder="@username или телефон"
          maxLength={150}
          disabled={loading}
        />
      </label>

      <label>
        <span>Сообщение</span>
        <textarea
          value={message}
          onChange={(event) => setMessage(event.target.value)}
          placeholder="Расскажите, что вам нужно"
          maxLength={2000}
          rows={5}
          disabled={loading}
        />
      </label>

      {error && <div className="error">{error}</div>}

      <button type="submit" disabled={loading}>
        {loading ? "Отправляем..." : "Отправить заявку"}
      </button>
    </form>
  );
}