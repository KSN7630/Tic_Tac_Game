import { useEffect, useState } from "react";

export default function MoveTimer({ moveDeadline, graceDeadline }) {
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    if (!moveDeadline) return;
    const id = setInterval(() => setNow(Date.now()), 100);
    return () => clearInterval(id);
  }, [moveDeadline, graceDeadline]);

  if (!moveDeadline) return null;

  const inGrace = now > moveDeadline && graceDeadline && now <= graceDeadline;
  const secondsLeft = Math.max(0, (moveDeadline - now) / 1000);

  return (
    <div className="move-timer">
      Time left: <span>{secondsLeft.toFixed(1)}</span>s
      {inGrace && <span className="grace-indicator">GRACE</span>}
    </div>
  );
}
