export default function Board({ cells, interactive, onCellClick }) {
  return (
    <div className="board">
      {cells.map((mark, i) => (
        <button
          key={i}
          className={`cell${mark === "X" ? " x" : mark === "O" ? " o" : ""}`}
          disabled={!interactive || !!mark}
          onClick={() => onCellClick(i)}
        >
          {mark}
        </button>
      ))}
    </div>
  );
}
