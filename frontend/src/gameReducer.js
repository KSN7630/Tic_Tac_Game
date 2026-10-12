export const MAX_MOVE_TIME_MS = 5000; // mirrors backend ScoringEngine.MaxMoveTime
export const GRACE_PERIOD_MS = 3000; // mirrors backend ScoringEngine.GracePeriod

export const initialState = {
  screen: "connect", // connect | lobby | game
  username: null,
  userId: null,
  groupChat: [],
  matchChat: [],
  challenges: [], // { room_id, challenger, message }
  activeGameRoom: null,
  playerSymbol: null, // "X" | "O"
  board: null,
  scores: {},
  gameOver: null, // { kind: "win" | "lose" | "draw" }
  moveDeadline: null,
  graceDeadline: null,
  replay: null, // { moves, step }
};

let toastSeq = 0;
export function nextToastId() {
  return ++toastSeq;
}

export function gameReducer(state, action) {
  switch (action.type) {
    case "IDENTIFY":
      return { ...state, username: action.username, userId: action.userId };

    case "ENTER_LOBBY":
      return { ...state, screen: "lobby" };

    case "GROUP_CHAT":
      return { ...state, groupChat: [...state.groupChat, action.message] };

    case "MATCH_CHAT":
      return { ...state, matchChat: [...state.matchChat, action.message] };

    case "CHALLENGE_RECEIVED":
      if (action.challenge.challenger?.id === state.userId) return state;
      return { ...state, challenges: [...state.challenges, action.challenge] };

    case "CHALLENGE_REMOVE":
      return {
        ...state,
        challenges: state.challenges.filter((c) => c.room_id !== action.roomId),
      };

    case "CHALLENGE_CREATED_ACK":
      return { ...state, activeGameRoom: action.roomId };

    case "CHALLENGE_JOINED":
      // The backend sends this to the accepter AFTER start_game (order:
      // challenge_accepted -> start_game -> challenge_joined), so this must
      // NOT touch board/scores/gameOver - doing so wiped the board START_GAME
      // had just seeded, leaving every cell disabled for the whole match.
      return {
        ...state,
        activeGameRoom: action.roomId,
        screen: "game",
      };

    case "START_GAME": {
      const { roomId, playerX, playerO, firstTurn, userId } = action;
      const symbol = playerX === userId ? "X" : playerO === userId ? "O" : null;
      const now = Date.now();
      return {
        ...state,
        screen: "game",
        activeGameRoom: roomId,
        playerSymbol: symbol,
        // Server doesn't send a move_made until the first move is played, but
        // someone has to be able to click first — seed an empty board here so
        // the turn check in GameScreen has something to compare against.
        board: {
          cells: Array(9).fill(""),
          turn: firstTurn,
          finished: false,
          winner: "",
          player_x: playerX,
          player_o: playerO,
        },
        scores: {},
        gameOver: null,
        moveDeadline: now + MAX_MOVE_TIME_MS,
        graceDeadline: now + MAX_MOVE_TIME_MS + GRACE_PERIOD_MS,
      };
    }

    case "MOVE_MADE": {
      const now = Date.now();
      const finished = action.board.finished;
      return {
        ...state,
        board: action.board,
        scores: action.scores,
        moveDeadline: finished ? null : now + MAX_MOVE_TIME_MS,
        graceDeadline: finished ? null : now + MAX_MOVE_TIME_MS + GRACE_PERIOD_MS,
      };
    }

    case "GAME_OVER": {
      let kind = "draw";
      if (action.board.winner) {
        kind = action.board.winner === state.userId ? "win" : "lose";
      }
      return {
        ...state,
        board: action.board,
        scores: action.scores,
        gameOver: { kind },
        moveDeadline: null,
        graceDeadline: null,
      };
    }

    case "RETURN_TO_LOBBY":
      return {
        ...state,
        screen: "lobby",
        activeGameRoom: null,
        playerSymbol: null,
        board: null,
        gameOver: null,
        moveDeadline: null,
        graceDeadline: null,
      };

    case "OPEN_REPLAY":
      return { ...state, replay: { moves: action.moves, step: 0 } };

    case "REPLAY_STEP": {
      if (!state.replay) return state;
      const next = Math.max(0, Math.min(state.replay.moves.length, state.replay.step + action.delta));
      return { ...state, replay: { ...state.replay, step: next } };
    }

    case "CLOSE_REPLAY":
      return { ...state, replay: null };

    default:
      return state;
  }
}
