package backendbasics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"gitlab.com/efronlicht/blog/articles/backendbasics/middleware"
)

// GetJSON makes a GET request to the given URL, and decodes the response body as JSON into t.
func GetJSON[T any](ctx context.Context, c *http.Client, url string) (t T, err error) {
	if ctx == nil {
		return t, errors.New("nil context")
	}
	if c == nil {
		return t, errors.New("nil client")
	}
	if err := ctx.Err(); err != nil {
		return t, ctx.Err()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return t, fmt.Errorf("GET %v: invalid request: %w", url, err)
	}
	req.Header.Add("Accept", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return t, fmt.Errorf("GET %v: %w", url, err)
	}
	defer resp.Body.Close()
	if status := resp.StatusCode; status < 200 || status >= 300 {
		return t, fmt.Errorf("GET %v: expected 2xx status code, got %v", url, status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return t, fmt.Errorf("GET %v: decoding JSON as %T: %w", url, t, err)
	}
	return t, nil
}

// WriteJSON sets the Content-Type header to application/json, then writes t as JSON to w's body.
// If no pre-existing status code has been set, it sets the status code to 200.
func WriteJSON(w http.ResponseWriter, t any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(t)
}

type piece byte

const (
	empty piece = iota
	whitePawn
	whiteKnight
	whiteBishop
	whiteRook
	whiteQueen
	whiteKing
	_
	_
	blackPawn
	blackKnight
	blackBishop
	blackRook
	blackQueen
	blackKing
)

type Move struct {
	From, To struct{ X, Y int8 }
	White    bool
	Previous [8][8]piece
}

func NextMove(board [8][8]piece, m Move) ([8][8]piece, error) {
	panic("TODO")
}

func FromJSON[T any](r io.Reader) (t T, err error) {
	if r == nil {
		return t, errors.New("nil reader")
	}
	if err := json.NewDecoder(r).Decode(&t); err != nil {
		return t, fmt.Errorf("decoding JSON as %T: %w", t, err)
	}
	if rc, ok := r.(io.ReadCloser); ok {
		return t, rc.Close()
	}
	return t, nil
}

type Sessions struct {
	mux sync.RWMutex
	m   map[string]*Game
}
type Game struct {
	ID    string
	Board [8][8]piece
	mux   sync.Mutex
}

func handle(r *http.Request, s *Sessions) ([8][8]piece, error, int) {
	if r.Method != http.MethodPost {
		return [8][8]piece{}, fmt.Errorf("invalid method %q", r.Method), http.StatusMethodNotAllowed

	}
	gameID := r.URL.Query().Get("gameID")
	if gameID == "" {
		return [8][8]piece{}, fmt.Errorf("missing gameID query parameter"), http.StatusBadRequest
	}

	move, err := FromJSON[Move](r.Body)
	if err != nil {
		return [8][8]piece{}, fmt.Errorf("decoding JSON: %w", err), http.StatusBadRequest
	}
	if move.From.X < 0 || move.From.X >= 8 || move.From.Y < 0 || move.From.Y >= 8 {
		return [8][8]piece{}, fmt.Errorf("invalid move.From: %+v", move.From), http.StatusBadRequest
	}
	s.mux.RLock() // read lock; check for this gameID
	game, ok := s.m[gameID]
	s.mux.RUnlock() // regardless of whether we found it, we need to unlock the map
	if !ok {
		return [8][8]piece{}, fmt.Errorf("invalid gameID %q", gameID), http.StatusBadRequest
	}
	game.mux.Lock()         // obtain a write lock on THIS game
	defer game.mux.Unlock() // unlock THIS game when we're done

	if game.Board != move.Previous {
		return [8][8]piece{}, fmt.Errorf("board does not match previous move"), http.StatusBadRequest
	}
	// now no one else can modify this game's board until we're done, and we know the board is correct

	board, err := NextMove([8][8]piece{}, move)
	if err != nil {
		return [8][8]piece{}, fmt.Errorf("invalid move: %w", err), http.StatusBadRequest
	}
	game.Board = board
	return board, nil, http.StatusOK
}

func SomeHandler(s *Sessions) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		board, err, statusCode := handle(r, s)
		if err != nil {
			slog.InfoContext(r.Context(), "somehandler: error", "err", err, "status", statusCode, "method", r.Method, "url", r.URL)
			w.WriteHeader(statusCode)
			if _, err := w.Write([]byte(err.Error())); err != nil {
				middleware.LogOrDefault(r.Context()).Error("somehandler: failed write", "err", err)
			}
			return
		}
		if err := WriteJSON(w, board); err != nil {
			slog.ErrorContext(r.Context(), "somehandler: error writing JSON", "err", err, "method", r.Method, "url", r.URL)
			return
		}
		slog.DebugContext(r.Context(), "somehandler: success", "method", r.Method, "url", r.URL)
	}
}
