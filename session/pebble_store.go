package session

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/cockroachdb/pebble"
)

var (
	sessionPrefix = []byte("session:")
)

// PebbleStore is a Pebble-based implementation of the Store interface
type PebbleStore struct {
	db     *pebble.DB
	mu     sync.RWMutex
	closed bool
}

// PebbleStoreConfig configures the Pebble store
type PebbleStoreConfig struct {
	Path string
	Opts *pebble.Options
}

// sessionData is the serializable representation of a session
type sessionData struct {
	ClientID          string                     `json:"client_id"`
	CleanStart        bool                       `json:"clean_start"`
	State             State                      `json:"state"`
	ExpiryInterval    uint32                     `json:"expiry_interval"`
	CreatedAt         time.Time                  `json:"created_at"`
	LastAccessedAt    time.Time                  `json:"last_accessed_at"`
	DisconnectedAt    time.Time                  `json:"disconnected_at"`
	WillMessage       *WillMessage               `json:"will_message,omitempty"`
	WillDelayInterval uint32                     `json:"will_delay_interval"`
	Subscriptions     map[string]*Subscription   `json:"subscriptions"`
	PendingPublish    map[uint16]*PendingMessage `json:"pending_publish"`
	PendingPubrel     map[uint16]bool            `json:"pending_pubrel"`
	PendingPubcomp    map[uint16]bool            `json:"pending_pubcomp"`
	NextPacketID      uint16                     `json:"next_packet_id"`
	MaxPacketSize     uint32                     `json:"max_packet_size"`
	ReceiveMaximum    uint16                     `json:"receive_maximum"`
	ProtocolVersion   byte                       `json:"protocol_version"`
}

// NewPebbleStore creates a new Pebble-based session store
func NewPebbleStore(config PebbleStoreConfig) (*PebbleStore, error) {
	opts := config.Opts
	if opts == nil {
		opts = &pebble.Options{
			ErrorIfExists: false,
		}
	}

	db, err := pebble.Open(config.Path, opts)
	if err != nil {
		return nil, err
	}

	return &PebbleStore{
		db: db,
	}, nil
}

// sessionToData converts a Session to sessionData for serialization
func sessionToData(s *Session) *sessionData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := &sessionData{
		ClientID:          s.ClientID,
		CleanStart:        s.CleanStart,
		State:             s.State,
		ExpiryInterval:    s.ExpiryInterval,
		CreatedAt:         s.CreatedAt,
		LastAccessedAt:    s.LastAccessedAt,
		DisconnectedAt:    s.DisconnectedAt,
		WillMessage:       s.WillMessage,
		WillDelayInterval: s.WillDelayInterval,
		Subscriptions:     s.Subscriptions,
		PendingPublish:    s.PendingPublish,
		NextPacketID:      s.nextPacketID,
		MaxPacketSize:     s.MaxPacketSize,
		ReceiveMaximum:    s.ReceiveMaximum,
		ProtocolVersion:   s.ProtocolVersion,
	}

	// Convert map[uint16]struct{} to map[uint16]bool for JSON
	data.PendingPubrel = make(map[uint16]bool, len(s.PendingPubrel))
	for id := range s.PendingPubrel {
		data.PendingPubrel[id] = true
	}

	data.PendingPubcomp = make(map[uint16]bool, len(s.PendingPubcomp))
	for id := range s.PendingPubcomp {
		data.PendingPubcomp[id] = true
	}

	return data
}

// dataToSession converts sessionData to a Session
func dataToSession(data *sessionData) *Session {
	s := &Session{
		ClientID:          data.ClientID,
		CleanStart:        data.CleanStart,
		State:             data.State,
		ExpiryInterval:    data.ExpiryInterval,
		CreatedAt:         data.CreatedAt,
		LastAccessedAt:    data.LastAccessedAt,
		DisconnectedAt:    data.DisconnectedAt,
		WillMessage:       data.WillMessage,
		WillDelayInterval: data.WillDelayInterval,
		Subscriptions:     data.Subscriptions,
		PendingPublish:    data.PendingPublish,
		nextPacketID:      data.NextPacketID,
		MaxPacketSize:     data.MaxPacketSize,
		ReceiveMaximum:    data.ReceiveMaximum,
		ProtocolVersion:   data.ProtocolVersion,
	}

	// Initialize maps if nil
	if s.Subscriptions == nil {
		s.Subscriptions = make(map[string]*Subscription)
	}
	if s.PendingPublish == nil {
		s.PendingPublish = make(map[uint16]*PendingMessage)
	}

	// Convert map[uint16]bool to map[uint16]struct{}
	s.PendingPubrel = make(map[uint16]struct{}, len(data.PendingPubrel))
	for id := range data.PendingPubrel {
		s.PendingPubrel[id] = struct{}{}
	}

	s.PendingPubcomp = make(map[uint16]struct{}, len(data.PendingPubcomp))
	for id := range data.PendingPubcomp {
		s.PendingPubcomp[id] = struct{}{}
	}

	return s
}

// makeKey creates a key for a client ID
func makeKey(clientID string) []byte {
	key := make([]byte, len(sessionPrefix)+len(clientID))
	copy(key, sessionPrefix)
	copy(key[len(sessionPrefix):], clientID)
	return key
}

// Save stores or updates a session
func (p *PebbleStore) Save(ctx context.Context, session *Session) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrStoreClosed
	}
	p.mu.RUnlock()

	data := sessionToData(session)
	value, err := json.Marshal(data)
	if err != nil {
		return err
	}

	key := makeKey(session.GetClientID())
	return p.db.Set(key, value, pebble.Sync)
}

// Load retrieves a session by client ID
func (p *PebbleStore) Load(ctx context.Context, clientID string) (*Session, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, ErrStoreClosed
	}
	p.mu.RUnlock()

	key := makeKey(clientID)
	value, closer, err := p.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	defer closer.Close()

	var data sessionData
	if err := json.Unmarshal(value, &data); err != nil {
		return nil, err
	}

	return dataToSession(&data), nil
}

// Delete removes a session
func (p *PebbleStore) Delete(ctx context.Context, clientID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrStoreClosed
	}
	p.mu.RUnlock()

	key := makeKey(clientID)
	return p.db.Delete(key, pebble.Sync)
}

// Exists checks if a session exists
func (p *PebbleStore) Exists(ctx context.Context, clientID string) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return false, ErrStoreClosed
	}
	p.mu.RUnlock()

	key := makeKey(clientID)
	_, closer, err := p.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	closer.Close()
	return true, nil
}

// List returns all session client IDs
func (p *PebbleStore) List(ctx context.Context) ([]string, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, ErrStoreClosed
	}
	p.mu.RUnlock()

	var clientIDs []string

	iter, err := p.db.NewIter(&pebble.IterOptions{
		LowerBound: sessionPrefix,
		UpperBound: append(sessionPrefix, 0xff),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		clientID := string(key[len(sessionPrefix):])
		clientIDs = append(clientIDs, clientID)
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}

	return clientIDs, nil
}

// Close closes the store
func (p *PebbleStore) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrStoreClosed
	}

	p.closed = true
	return p.db.Close()
}

// Count returns the total number of sessions
func (p *PebbleStore) Count(ctx context.Context) (int64, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return 0, ErrStoreClosed
	}
	p.mu.RUnlock()

	var count int64

	iter, err := p.db.NewIter(&pebble.IterOptions{
		LowerBound: sessionPrefix,
		UpperBound: append(sessionPrefix, 0xff),
	})
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	if err := iter.Error(); err != nil {
		return 0, err
	}

	return count, nil
}

// CountByState returns the number of sessions in a given state
func (p *PebbleStore) CountByState(ctx context.Context, state State) (int64, error) {
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return 0, ErrStoreClosed
	}
	p.mu.RUnlock()

	var count int64

	iter, err := p.db.NewIter(&pebble.IterOptions{
		LowerBound: sessionPrefix,
		UpperBound: append(sessionPrefix, 0xff),
	})
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		var data sessionData
		if err := json.Unmarshal(iter.Value(), &data); err != nil {
			continue
		}
		if data.State == state {
			count++
		}
	}

	if err := iter.Error(); err != nil {
		return 0, err
	}

	return count, nil
}
