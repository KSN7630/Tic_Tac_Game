import { useCallback, useRef, useState } from "react";

export const LOBBY_ROOM = "room-lobby";
const RECONNECT_DELAY_MS = 2000;

/**
 * Owns the single WebSocket connection to the backend and dispatches
 * incoming {event, payload} frames to the onEvent callback.
 */
export function useGameSocket(onEvent) {
  const wsRef = useRef(null);
  const paramsRef = useRef(null); // {addr, userId, username}
  const reconnectTimerRef = useRef(null);
  const [connected, setConnected] = useState(false);
  const [connecting, setConnecting] = useState(false);

  const openSocket = useCallback(() => {
    const { addr, userId, username } = paramsRef.current;
    const url = `ws://${addr}/ws?room=${encodeURIComponent(LOBBY_ROOM)}&user=${encodeURIComponent(
      userId
    )}&username=${encodeURIComponent(username)}`;

    const socket = new WebSocket(url);
    wsRef.current = socket;

    socket.onopen = () => {
      setConnected(true);
      setConnecting(false);
    };

    socket.onmessage = (ev) => {
      let data;
      try {
        data = JSON.parse(ev.data);
      } catch {
        return;
      }
      onEvent(data.event, data.payload);
    };

    socket.onclose = () => {
      setConnected(false);
      wsRef.current = null;
      onEvent("__ws_closed__", null);
      reconnectTimerRef.current = setTimeout(openSocket, RECONNECT_DELAY_MS);
    };

    socket.onerror = () => {
      // onclose fires immediately after; nothing extra to do here.
    };
  }, [onEvent]);

  const connect = useCallback(
    (addr, userId, username) => {
      paramsRef.current = { addr, userId, username };
      setConnecting(true);
      openSocket();
    },
    [openSocket]
  );

  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) clearTimeout(reconnectTimerRef.current);
    wsRef.current?.close();
    wsRef.current = null;
    setConnected(false);
  }, []);

  const send = useCallback((action, fields = {}) => {
    const socket = wsRef.current;
    if (!socket || socket.readyState !== WebSocket.OPEN) {
      onEvent("__send_failed__", { action });
      return;
    }
    socket.send(JSON.stringify({ action, ...fields }));
  }, [onEvent]);

  return { connect, disconnect, send, connected, connecting };
}
