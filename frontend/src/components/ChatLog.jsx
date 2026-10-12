import { useEffect, useRef } from "react";

export default function ChatLog({ messages }) {
  const listRef = useRef(null);

  useEffect(() => {
    if (listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight;
  }, [messages]);

  return (
    <ul className="chat-log" ref={listRef}>
      {messages.map((m, i) => (
        <li key={i}>
          <span className="who">{m.who || "?"}</span>
          {m.message}
          {m.at && <span className="at">{new Date(m.at).toLocaleTimeString()}</span>}
        </li>
      ))}
    </ul>
  );
}
