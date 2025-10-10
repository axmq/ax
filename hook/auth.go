package hook

import (
	"crypto/subtle"
	"sync"
)

// BasicAuthHook provides username/password authentication
type BasicAuthHook struct {
	*Base
	mu    sync.RWMutex
	users map[string]string
}

// NewBasicAuthHook creates a new basic authentication hook
func NewBasicAuthHook() *BasicAuthHook {
	return &BasicAuthHook{
		Base:  &Base{id: "basic-auth"},
		users: make(map[string]string),
	}
}

// ID returns the hook identifier
func (h *BasicAuthHook) ID() string {
	return h.id
}

// Provides indicates this hook provides authentication
func (h *BasicAuthHook) Provides(event Event) bool {
	return event == OnConnectAuthenticate
}

// AddUser adds a user with username and password
func (h *BasicAuthHook) AddUser(username, password string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.users[username] = password
}

// RemoveUser removes a user by username
func (h *BasicAuthHook) RemoveUser(username string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.users, username)
}

// HasUser checks if a user exists
func (h *BasicAuthHook) HasUser(username string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, exists := h.users[username]
	return exists
}

// UserCount returns the number of registered users
func (h *BasicAuthHook) UserCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.users)
}

// Clear removes all users
func (h *BasicAuthHook) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.users = make(map[string]string)
}

// OnConnectAuthenticate validates username and password
func (h *BasicAuthHook) OnConnectAuthenticate(client *Client, packet *ConnectPacket) bool {
	h.mu.RLock()
	expectedPassword, exists := h.users[packet.Username]
	h.mu.RUnlock()

	if !exists {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(expectedPassword), packet.Password) == 1
}

// LoadUsers loads multiple users at once
func (h *BasicAuthHook) LoadUsers(users map[string]string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for username, password := range users {
		h.users[username] = password
	}
}

// AnonymousAuthHook AllowAnonymous sets whether to allow clients with no username/password
type AnonymousAuthHook struct {
	*Base
	allowAnonymous bool
	mu             sync.RWMutex
}

// NewAnonymousAuthHook creates a hook that controls anonymous access
func NewAnonymousAuthHook(allowAnonymous bool) *AnonymousAuthHook {
	return &AnonymousAuthHook{
		Base:           &Base{id: "anonymous-auth"},
		allowAnonymous: allowAnonymous,
	}
}

// ID returns the hook identifier
func (h *AnonymousAuthHook) ID() string {
	return h.id
}

// Provides indicates this hook provides authentication
func (h *AnonymousAuthHook) Provides(event Event) bool {
	return event == OnConnectAuthenticate
}

// SetAllowAnonymous sets whether to allow anonymous connections
func (h *AnonymousAuthHook) SetAllowAnonymous(allow bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.allowAnonymous = allow
}

// IsAnonymousAllowed returns whether anonymous connections are allowed
func (h *AnonymousAuthHook) IsAnonymousAllowed() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.allowAnonymous
}

// OnConnectAuthenticate checks if anonymous access is allowed
func (h *AnonymousAuthHook) OnConnectAuthenticate(client *Client, packet *ConnectPacket) bool {
	h.mu.RLock()
	allow := h.allowAnonymous
	h.mu.RUnlock()

	if packet.Username == "" && packet.Password == nil {
		return allow
	}

	return true
}
