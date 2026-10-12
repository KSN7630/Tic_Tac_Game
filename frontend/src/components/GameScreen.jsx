import ChatLog from "./ChatLog.jsx";
import ChatForm from "./ChatForm.jsx";
import Board from "./Board.jsx";
import MoveTimer from "./MoveTimer.jsx";

export default function GameScreen({
  userId,
  connected,
  activeGameRoom,
  playerSymbol,
  board,
  scores,
  gameOver,
  moveDeadline,
  graceDeadline,
  matchChat,
  onCellClick,
  onSendMatchChat,
  onRequestReplay,
  onLeaveRoom,
}) {
  const cells = board ? board.cells : Array(9).fill("");
  const myTurn = board && !board.finished && board.turn === userId;

  let turnText = "Waiting for game to start...";
  if (board) {
    turnText = board.finished ? "Game finished." : myTurn ? "Your turn" : "Opponent's turn";
  }

  return (
    <section className="screen">
      <header className="topbar">
        <span>Room {activeGameRoom}</span>
        <span className={`ws-indicator ${connected ? "open" : "closed"}`}>●</span>
      </header>

      <div className="columns">
        <div className="panel board-panel">
          <div className="turn-indicator">
            {turnText}
            {playerSymbol && ` (You are "${playerSymbol}")`}
          </div>

          <MoveTimer moveDeadline={moveDeadline} graceDeadline={graceDeadline} />

          <Board cells={cells} interactive={!!myTurn} onCellClick={onCellClick} />

          <div className="scoreboard">
            {Object.entries(scores).map(([pid, score]) => (
              <span key={pid} className="score-entry">
                {pid === userId ? "You" : "Opp."}: {score}
              </span>
            ))}
          </div>

          {gameOver && (
            <div className={`game-over-banner ${gameOver.kind}`}>
              {gameOver.kind === "win" && "You won!"}
              {gameOver.kind === "lose" && "You lost."}
              {gameOver.kind === "draw" && "Draw!"}
            </div>
          )}

          <div className="board-actions">
            <button className="secondary" onClick={onRequestReplay}>
              View Replay
            </button>
            <button className="secondary" onClick={onLeaveRoom}>
              Back to Lobby
            </button>
          </div>
        </div>

        <div className="panel chat-panel">
          <h2>Match Chat</h2>
          <ChatLog messages={matchChat} />
          <ChatForm onSend={onSendMatchChat} />
        </div>
      </div>
    </section>
  );
}
