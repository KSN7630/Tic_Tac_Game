import { useCallback, useRef, useState } from "react";

export function useToasts() {
  const [toasts, setToasts] = useState([]);
  const seqRef = useRef(0);

  const toast = useCallback((message, kind = "error", ttl = 4000) => {
    const id = ++seqRef.current;
    setToasts((prev) => [...prev, { id, message, kind }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, ttl);
  }, []);

  return { toasts, toast };
}
