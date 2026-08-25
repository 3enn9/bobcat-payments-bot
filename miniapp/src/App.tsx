import { useState } from "react";
import RequestForm from "./components/RequestForm";
import WorkerRequests from "./components/WorkerRequests";

type Screen = "home" | "client" | "worker";

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
          <button
            type="button"
            className="back-button"
            onClick={() => setScreen("home")}
          >
            ← Назад
          </button>

          <div className="eyebrow">GOPAYGO</div>

          <h1>Работник</h1>

          <p className="subtitle">Заявки из группы «Рогатка»</p>

          <WorkerRequests />
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
      </div>
    </main>
  );
}
