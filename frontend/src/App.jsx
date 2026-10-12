import { useCallback, useReducer, useRef } from "react";
import { gameReducer, initialState } from "./gameReducer.js";
import { useGameSocket } from "./hooks/useGameSocket.js";
import { useToasts } from "./hooks/useToasts.js";
import ToastArea from "./components/ToastArea.jsx";
import ConnectScreen from "./components/ConnectScreen.jsx";
import LobbyScreen from "./components/LobbyScreen.jsx";
import GameScreen from "./components/GameScreen.jsx";
import ReplayModal from "./components/ReplayModal.jsx";

export default function App() {
  const [state, dispatch] = useReducer(gameReducer, initialState);
  const { toasts, toast } = useToasts();

  // Mirrors state.userId/activeGameRoom for use inside the socket's onEvent
  // closure, which is created once and must not rely on stale render state.
  const userIdRef = useRef(null);
  const activeRoomRef = useRef(null);
  activeRoomRef.current = state.activeGameRoom;

  const handleEvent = useCallback((event, payload) => {
    switch (event) {
      case "__ws_closed__":
        toast("Connection lost. Reconnecting...", "error", 3000);
        break;

      case "__send_failed__":
        toast("Not connected to the server.");
        break;

      case "error":
        toast(`[${payload.code || "error"}] ${payload.message || "Unknown error"}`);
        break;

      case "group_chat":
        dispatch({
          type: "GROUP_CHAT",
          message: { who: payload.username || payload.from, message: payload.message, at: payload.at },
        });
        break;

      case "match_chat":
        dispatch({
          type: "MATCH_CHAT",
          message: { who: payload.from, message: payload.message, at: payload.at },
        });
        break;

      case "challenge_created":
        dispatch({ type: "CHALLENGE_RECEIVED", challenge: payload });
        break;

      case "challenge_created_ack":
        dispatch({ type: "CHALLENGE_CREATED_ACK", roomId: payload.room_id });
        toast(`Challenge created (${payload.room_id}). Waiting for an opponent...`, "info");
        break;

      case "challenge_joined":
        dispatch({ type: "CHALLENGE_JOINED", roomId: payload.room_id });
        toast(`Joined room ${payload.room_id}. Waiting for game to start...`, "info");
        break;

      case "challenge_accepted":
        toast("Opponent joined. Starting game...", "info");
        break;

      case "start_game":
        dispatch({
          type: "START_GAME",
          roomId: payload.room_id,
          playerX: payload.player_x,
          playerO: payload.player_o,
          firstTurn: payload.first_turn,
          userId: userIdRef.current,
        });
        break;

      case "move_made":
        dispatch({ type: "MOVE_MADE", board: payload.board, scores: payload.scores });
        break;

      case "game_over":
        dispatch({ type: "GAME_OVER", board: payload.board, scores: payload.scores });
        break;

      case "return_to_group_chat":
        toast(payload.message || "Returning to lobby.", "info");
        dispatch({ type: "RETURN_TO_LOBBY" });
        break;

      case "replay_data":
        dispatch({ type: "OPEN_REPLAY", moves: Array.isArray(payload) ? payload : [] });
        break;

      default:
        break;
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const { connect, send, connected, connecting } = useGameSocket(handleEvent);

  function handleConnect(addr, userId, username) {
    userIdRef.current = userId;
    dispatch({ type: "IDENTIFY", userId, username });
    connect(addr, userId, username);
    dispatch({ type: "ENTER_LOBBY" });
  }

  function handleSendGroupChat(message) {
    send("send_group_chat", { message });
  }

  function handleCreateChallenge(message) {
    send("create_challenge", { message });
  }

  function handleAcceptChallenge(roomId) {
    send("accept_challenge", { room: roomId });
    dispatch({ type: "CHALLENGE_REMOVE", roomId });
  }

  function handleCellClick(index) {
    const now = Date.now();
    const usedGrace = !!(
      state.moveDeadline &&
      now > state.moveDeadline &&
      state.graceDeadline &&
      now <= state.graceDeadline
    );
    send("make_move", { room: state.activeGameRoom, index, used_grace: usedGrace });
  }

  function handleSendMatchChat(message) {
    send("send_match_chat", { message });
  }

  function handleRequestReplay() {
    if (!state.activeGameRoom) return;
    send("get_replay", { room: state.activeGameRoom });
  }

  function handleLeaveRoom() {
    dispatch({ type: "RETURN_TO_LOBBY" });
  }

  function handleReplayStep(delta) {
    dispatch({ type: "REPLAY_STEP", delta });
  }

  function handleCloseReplay() {
    dispatch({ type: "CLOSE_REPLAY" });
  }

  return (
    <>
      <ToastArea toasts={toasts} />

      {state.screen === "connect" && (
        <ConnectScreen
          onConnect={handleConnect}
          connecting={connecting}
          status={connected ? "Connected" : connecting ? "Connecting..." : "Disconnected"}
        />
      )}

      {state.screen === "lobby" && (
        <LobbyScreen
          username={state.username}
          connected={connected}
          groupChat={state.groupChat}
          challenges={state.challenges}
          onSendGroupChat={handleSendGroupChat}
          onCreateChallenge={handleCreateChallenge}
          onAcceptChallenge={handleAcceptChallenge}
        />
      )}

      {state.screen === "game" && (
        <GameScreen
          userId={state.userId}
          connected={connected}
          activeGameRoom={state.activeGameRoom}
          playerSymbol={state.playerSymbol}
          board={state.board}
          scores={state.scores}
          gameOver={state.gameOver}
          moveDeadline={state.moveDeadline}
          graceDeadline={state.graceDeadline}
          matchChat={state.matchChat}
          onCellClick={handleCellClick}
          onSendMatchChat={handleSendMatchChat}
          onRequestReplay={handleRequestReplay}
          onLeaveRoom={handleLeaveRoom}
        />
      )}

      <ReplayModal replay={state.replay} onStep={handleReplayStep} onClose={handleCloseReplay} />
    </>
  );
}
