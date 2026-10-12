import { useState } from "react";
import ChatLog from "./ChatLog.jsx";
import ChatForm from "./ChatForm.jsx";

export default function LobbyScreen({
  username,
  connected,
  groupChat,
  challenges,
  onSendGroupChat,
  onCreateChallenge,
  onAcceptChallenge,
}) {
  const [challengeMessage, setChallengeMessage] = useState("");

  function handleCreate(e) {
    e.preventDefault();
    onCreateChallenge(challengeMessage.trim());
    setChallengeMessage("");
  }

  return (
    <section className="screen">
      <header className="topbar">
        <span>Lobby · {username}</span>
        <span className={`ws-indicator ${connected ? "open" : "closed"}`}>●</span>
      </header>

      <div className="columns">
        <div className="panel">
          <h2>Challenges</h2>
          <form className="challenge-form" onSubmit={handleCreate}>
            <input
              type="text"
              value={challengeMessage}
              onChange={(e) => setChallengeMessage(e.target.value)}
              placeholder="optional message"
              maxLength={60}
            />
            <button type="submit">Create</button>
          </form>
          <ul className="challenge-list">
            {challenges.length === 0 && <li style={{ color: "var(--muted)" }}>No open challenges yet.</li>}
            {challenges.map((c) => (
              <li key={c.room_id}>
                <span>
                  {c.challenger?.username || "Someone"}: {c.message || "wants to play"}
                </span>
                <button onClick={() => onAcceptChallenge(c.room_id)}>Accept</button>
              </li>
            ))}
          </ul>
        </div>

        <div className="panel chat-panel">
          <h2>Group Chat</h2>
          <ChatLog messages={groupChat} />
          <ChatForm onSend={onSendGroupChat} />
        </div>
      </div>
    </section>
  );
}
