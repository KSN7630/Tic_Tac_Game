import { useState } from "react";

function genUserId() {
  return "u-" + Math.random().toString(36).slice(2, 10);
}

export default function ConnectScreen({ onConnect, connecting, status }) {
  const [addr, setAddr] = useState("localhost:8080");
  const [username, setUsername] = useState("");

  function handleSubmit(e) {
    e.preventDefault();
    const trimmedAddr = addr.trim();
    const trimmedName = username.trim();
    if (!trimmedAddr || !trimmedName) return;
    onConnect(trimmedAddr, genUserId(), trimmedName);
  }

  return (
    <section className="screen connect-screen">
      <form className="card" onSubmit={handleSubmit}>
        <h1>Tic Tac Game</h1>
        <label>
          Server address
          <input
            type="text"
            value={addr}
            onChange={(e) => setAddr(e.target.value)}
          />
        </label>
        <label>
          Username
          <input
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            placeholder="e.g. kartik"
            maxLength={20}
          />
        </label>
        <div className="hint">A random user ID is generated for you each session.</div>
        <button type="submit" disabled={connecting}>
          {connecting ? "Connecting..." : "Connect"}
        </button>
        <div className="status">{status}</div>
      </form>
    </section>
  );
}
