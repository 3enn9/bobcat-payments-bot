import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { fetchEquipment, type EquipmentItem } from "../api/equipment";

type RowValues = {
  surname: string;
  company: string;
  description: string;
};

export default function KopytenkovForm() {
  const [equipment, setEquipment] = useState<EquipmentItem[]>([]);
  const [rows, setRows] = useState<Record<number, RowValues>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setError("");
      try {
        const items = await fetchEquipment();
        if (cancelled) {
          return;
        }
        setEquipment(items);
        const initial: Record<number, RowValues> = {};
        for (const item of items) {
          initial[item.id] = { surname: "", company: "", description: "" };
        }
        setRows(initial);
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
  }, []);

  function updateRow(id: number, field: keyof RowValues, value: string) {
    setRows((prev) => ({
      ...prev,
      [id]: {
        ...prev[id],
        [field]: value,
      },
    }));
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setSuccess("");
    setSaving(true);

    const payload = equipment.map((item) => ({
      equipmentId: item.id,
      number: item.number,
      surname: rows[item.id]?.surname.trim() ?? "",
      company: rows[item.id]?.company.trim() ?? "",
      description: rows[item.id]?.description.trim() ?? "",
    }));

    console.log("Kopytenkov submit (stub):", payload);
    setSuccess("Отправка пока не подключена — данные сохранены только для проверки.");
    setSaving(false);
  }

  return (
    <form className="kopytenkov-form" onSubmit={submit}>
      {loading && <p className="invoice-hint">Загрузка...</p>}
      {error && <div className="error">{error}</div>}

      {!loading && !error && (
        <ul className="kopytenkov-list">
          {equipment.map((item) => (
            <li key={item.id} className="kopytenkov-row">
              <div className="kopytenkov-row-number">№ {item.number}</div>
              <label className="kopytenkov-field">
                <span>Фамилия</span>
                <input
                  value={rows[item.id]?.surname ?? ""}
                  disabled={saving}
                  placeholder="Иванов"
                  onChange={(event) =>
                    updateRow(item.id, "surname", event.target.value)
                  }
                />
              </label>
              <label className="kopytenkov-field">
                <span>Фирма</span>
                <input
                  value={rows[item.id]?.company ?? ""}
                  disabled={saving}
                  placeholder="ООО ..."
                  onChange={(event) =>
                    updateRow(item.id, "company", event.target.value)
                  }
                />
              </label>
              <label className="kopytenkov-field">
                <span>Описание</span>
                <input
                  value={rows[item.id]?.description ?? ""}
                  disabled={saving}
                  placeholder="Комментарий"
                  onChange={(event) =>
                    updateRow(item.id, "description", event.target.value)
                  }
                />
              </label>
            </li>
          ))}
        </ul>
      )}

      {success && <div className="invoice-success">{success}</div>}
      {!loading && !error && (
        <button type="submit" disabled={saving || equipment.length === 0}>
          {saving ? "Отправляем..." : "Отправить"}
        </button>
      )}
    </form>
  );
}
