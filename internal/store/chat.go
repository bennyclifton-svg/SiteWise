package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ChatTurn is one typed message and what came of it. A conversation belongs
// to the user who started it; every read and write below is scoped by org,
// project and that user.
type ChatTurn struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	Text           string `json:"text"`
	Status         string `json:"status"` // running | done | stopped | failed
	Outside        bool   `json:"outside"`
	Model          string `json:"-"`
	Rounds         int    `json:"-"`
	InTokens       int64  `json:"-"`
	OutTokens      int64  `json:"-"`
	CacheRead      int64  `json:"-"`
	CacheWrite     int64  `json:"-"`
	FirstMS        int    `json:"-"`
	LatencyMS      int    `json:"-"`
	// Declined: the answer carried chat's fixed words for a request it
	// cannot do yet (agency plan, Task 7).
	Declined bool            `json:"-"`
	Answer   json.RawMessage `json:"answer"`
	At       time.Time       `json:"at"`
}

// ChatPrepared is a change that waits for the PM's click.
type ChatPrepared struct {
	ID      string          `json:"id"`
	TurnID  string          `json:"turn_id"`
	Area    string          `json:"area"`
	Ops     json.RawMessage `json:"ops"`
	Summary string          `json:"summary"`
	Why     string          `json:"why"`
	State   string          `json:"state"` // waiting | applied | dismissed
}

// StartChatTurn records the PM's message. An empty conversation id starts a
// conversation; another user's, another project's or another org's
// conversation is ErrNotFound.
func (s *Store) StartChatTurn(ctx context.Context, org, project, user, conversation, text string) (ChatTurn, error) {
	tx, err := s.pool.Pool.Begin(ctx)
	if err != nil {
		return ChatTurn{}, err
	}
	defer tx.Rollback(ctx)
	if conversation == "" {
		conversation = newID()
		if _, err := tx.Exec(ctx, `INSERT INTO chat_conversations(org_id,id,project_id,user_id)
SELECT $1::uuid,$2::uuid,p.id,$4::uuid FROM projects p WHERE p.org_id=$1::uuid AND p.id=$3::uuid`, org, conversation, project, user); err != nil {
			return ChatTurn{}, err
		}
	}
	var found bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM chat_conversations WHERE org_id=$1::uuid AND id=$2::uuid AND project_id=$3::uuid AND user_id=$4::uuid)`,
		org, conversation, project, user).Scan(&found); err != nil {
		return ChatTurn{}, err
	}
	if !found {
		return ChatTurn{}, ErrNotFound
	}
	t := ChatTurn{ID: newID(), ConversationID: conversation, Text: text, Status: "running"}
	if err := tx.QueryRow(ctx, `INSERT INTO chat_turns(org_id,id,conversation_id,text) VALUES($1::uuid,$2::uuid,$3::uuid,$4) RETURNING at`,
		org, t.ID, conversation, text).Scan(&t.At); err != nil {
		return ChatTurn{}, err
	}
	return t, tx.Commit(ctx)
}

// FinishChatTurn stores how a turn ended and the provider's messages.
func (s *Store) FinishChatTurn(ctx context.Context, org string, t ChatTurn, blocks []json.RawMessage) error {
	tx, err := s.pool.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	answer := t.Answer
	if len(answer) == 0 {
		answer = json.RawMessage(`{}`)
	}
	tag, err := tx.Exec(ctx, `UPDATE chat_turns SET status=$3,outside=$4,model=$5,rounds=$6,in_tokens=$7,out_tokens=$8,cache_read_tokens=$9,
cache_write_tokens=$10,first_ms=$11,latency_ms=$12,answer=$13::jsonb,declined=$14 WHERE org_id=$1::uuid AND id=$2::uuid`,
		org, t.ID, t.Status, t.Outside, t.Model, t.Rounds, t.InTokens, t.OutTokens, t.CacheRead, t.CacheWrite, t.FirstMS, t.LatencyMS, string(answer), t.Declined)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	for i, b := range blocks {
		if _, err := tx.Exec(ctx, `INSERT INTO chat_blocks(org_id,turn_id,ordinal,block) VALUES($1::uuid,$2::uuid,$3,$4::jsonb)`, org, t.ID, i, string(b)); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

const chatTurnColumns = `t.id::text,t.conversation_id::text,t.text,t.status,t.outside,t.answer,t.at`

// ChatTurns is a conversation's turns, oldest first, for its own user. An
// empty conversation id reads the user's latest conversation on the
// project; with none, it returns no turns.
func (s *Store) ChatTurns(ctx context.Context, org, project, user, conversation string, limit int) ([]ChatTurn, error) {
	if conversation == "" {
		err := s.pool.QueryRow(ctx, `SELECT id::text FROM chat_conversations WHERE org_id=$1::uuid AND project_id=$2::uuid AND user_id=$3::uuid
ORDER BY started_at DESC LIMIT 1`, org, project, user).Scan(&conversation)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
	}
	rows, err := s.pool.Query(ctx, `SELECT * FROM (SELECT `+chatTurnColumns+`,t.n FROM chat_turns t
JOIN chat_conversations c ON c.org_id=t.org_id AND c.id=t.conversation_id
WHERE t.org_id=$1::uuid AND c.project_id=$2::uuid AND c.user_id=$3::uuid AND c.id=$4::uuid ORDER BY t.n DESC LIMIT $5) x ORDER BY n`,
		org, project, user, conversation, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ChatTurn
	for rows.Next() {
		var t ChatTurn
		var answer []byte
		var n int64
		if err := rows.Scan(&t.ID, &t.ConversationID, &t.Text, &t.Status, &t.Outside, &answer, &t.At, &n); err != nil {
			return nil, err
		}
		t.Answer = answer
		out = append(out, t)
	}
	return out, rows.Err()
}

// AddChatPrepared stores a change that waits for a click.
func (s *Store) AddChatPrepared(ctx context.Context, org string, p ChatPrepared) (string, error) {
	id := newID()
	_, err := s.pool.Pool.Exec(ctx, `INSERT INTO chat_prepared(org_id,id,turn_id,area,ops,summary,why) VALUES($1::uuid,$2::uuid,$3::uuid,$4,$5::jsonb,$6,$7)`,
		org, id, p.TurnID, p.Area, string(p.Ops), p.Summary, p.Why)
	return id, err
}

const chatPreparedSQL = `SELECT p.id::text,p.turn_id::text,p.area,p.ops,p.summary,p.why,p.state FROM chat_prepared p
JOIN chat_turns t ON t.org_id=p.org_id AND t.id=p.turn_id
JOIN chat_conversations c ON c.org_id=t.org_id AND c.id=t.conversation_id
WHERE p.org_id=$1::uuid AND c.project_id=$2::uuid AND c.user_id=$3::uuid AND p.id=$4::uuid`

// TakeChatPrepared marks a waiting change applied or dismissed and returns
// it, for the user it was prepared for. It is ErrNotFound for anyone else
// and ErrConflict when it was already decided, so a double click applies
// once.
func (s *Store) TakeChatPrepared(ctx context.Context, org, project, user, id, state string) (ChatPrepared, error) {
	tx, err := s.pool.Pool.Begin(ctx)
	if err != nil {
		return ChatPrepared{}, err
	}
	defer tx.Rollback(ctx)
	var p ChatPrepared
	var ops []byte
	err = tx.QueryRow(ctx, chatPreparedSQL+` FOR UPDATE OF p`, org, project, user, id).Scan(&p.ID, &p.TurnID, &p.Area, &ops, &p.Summary, &p.Why, &p.State)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	p.Ops = ops
	if p.State != "waiting" {
		return p, ErrConflict
	}
	if _, err := tx.Exec(ctx, `UPDATE chat_prepared SET state=$3,decided_at=clock_timestamp() WHERE org_id=$1::uuid AND id=$2::uuid`, org, id, state); err != nil {
		return p, err
	}
	p.State = state
	return p, tx.Commit(ctx)
}

// ReopenChatPrepared puts a change back to waiting when applying it failed,
// so the PM can try again or dismiss it.
func (s *Store) ReopenChatPrepared(ctx context.Context, org, id string) error {
	_, err := s.pool.Pool.Exec(ctx, `UPDATE chat_prepared SET state='waiting',decided_at=NULL WHERE org_id=$1::uuid AND id=$2::uuid`, org, id)
	return err
}
