import { useState } from "react";
import RequestForm from "./components/RequestForm";
import WorkerCabinet from "./components/WorkerCabinet";
import WorkerRequests from "./components/WorkerRequests";
import InvoiceForm from "./components/InvoiceForm";

type Screen = "home" | "client" | "worker" | "kosenko" | "invoices";

export default function App() {
  const [screen, setScreen] = useState<Screen>("home");

  if (screen === "client") {
    return (
      <div className="page">
        <div className="card">
          <button
            type="button"
            className="back-button"
            onClick={() => setScreen("home")}
          >
            ← Назад
          </button>

          <div className="eyebrow">Bobcatsar64</div>

          <h1>Оставить заявку</h1>

          <p className="subtitle">
            Заполните форму, и мы свяжемся с вами.
          </p>

          <RequestForm
            onSuccess={(id) => {
              console.log("Заявка создана:", id);
            }}
          />
        </div>
      </div>
    );
  }

  if (screen === "worker") {
    return (
      <div className="page page-worker">
        <div className="card card-worker">
          <div className="worker-header">
            <button
              type="button"
              className="back-button"
              onClick={() => setScreen("home")}
            >
              ← Назад
            </button>
            <h1>Работник</h1>
          </div>

          <WorkerCabinet />
        </div>
      </div>
    );
  }

  if (screen === "kosenko") {
    return (
      <div className="page page-worker">
        <div className="card card-worker">
          <div className="worker-header">
            <button
              type="button"
              className="back-button"
              onClick={() => setScreen("home")}
            >
              ← Назад
            </button>
            <h1>Косенко</h1>
          </div>

          <WorkerRequests />
        </div>
      </div>
    );
  }

  if (screen === "invoices") {
    return (
      <div className="page page-worker">
        <div className="card card-worker">
          <div className="worker-header">
            <button
              type="button"
              className="back-button"
              onClick={() => setScreen("home")}
            >
              ← Назад
            </button>
            <h1>Счета</h1>
          </div>

          <div className="requests-scroll">
            <InvoiceForm />
          </div>
        </div>
      </div>
    );
  }

  return (
    <main className="home-screen">
      <div className="logo">Bobcatsar64</div>

      <h1>Добро пожаловать!</h1>

      <p className="subtitle">
        Выберите, как продолжить
      </p>

      <div className="role-buttons">
        <button
          type="button"
          className="role-button"
          onClick={() => setScreen("client")}
        >
          <span className="role-icon">👤</span>

          <span>
            <strong>Клиент</strong>
            <small>Оставить заявку</small>
          </span>
        </button>

        <button
          type="button"
          className="role-button"
          onClick={() => setScreen("worker")}
        >
          <span className="role-icon">👷</span>

          <span>
            <strong>Работник</strong>
            <small>Перейти в кабинет</small>
          </span>
        </button>

        <button
          type="button"
          className="role-button"
          onClick={() => setScreen("kosenko")}
        >
          <span className="role-icon">📋</span>

          <span>
            <strong>Косенко</strong>
            <small>Заявки Рогатка</small>
          </span>
        </button>

        <button
          type="button"
          className="role-button"
          onClick={() => setScreen("invoices")}
        >
          <span className="role-icon">🧾</span>

          <span>
            <strong>Счета</strong>
            <small>Конструктор счетов</small>
          </span>
        </button>
      </div>
    </main>
  );
}
