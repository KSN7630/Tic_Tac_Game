function cellsAtStep(moves, step) {
  const cells = Array(9).fill("");
  const symbolForPlayer = {};
  let nextSymbol = "X";
  for (let i = 0; i < step; i++) {
    const mv = moves[i];
    if (!symbolForPlayer[mv.player_id]) {
      symbolForPlayer[mv.player_id] = nextSymbol;
      nextSymbol = nextSymbol === "X" ? "O" : "X";
    }
    cells[mv.index] = symbolForPlayer[mv.player_id];
  }
  return cells;
}

export default function ReplayModal({ replay, onStep, onClose }) {
  if (!replay) return null;
  const { moves, step } = replay;
  const cells = cellsAtStep(moves, step);

  return (
    <div className="modal">
      <div className="modal-content">
        <h2>Replay</h2>
        <div className="board">
          {cells.map((mark, i) => (
            <div key={i} className={`cell${mark === "X" ? " x" : mark === "O" ? " o" : ""}`}>
              {mark}
            </div>
          ))}
        </div>
        <div className="replay-controls">
          <button onClick={() => onStep(-1)} disabled={step <= 0}>
            &laquo; Prev
          </button>
          <span>
            {step} / {moves.length}
          </span>
          <button onClick={() => onStep(1)} disabled={step >= moves.length}>
            Next &raquo;
          </button>
        </div>
        <button className="secondary" onClick={onClose}>
          Close
        </button>
      </div>
    </div>
  );
}
