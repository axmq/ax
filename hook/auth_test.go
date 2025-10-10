package hook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicAuthHook(t *testing.T) {
	hook := NewBasicAuthHook()

	assert.Equal(t, "basic-auth", hook.ID())
	assert.True(t, hook.Provides(OnConnectAuthenticate))
	assert.False(t, hook.Provides(OnPublish))
	assert.Equal(t, 0, hook.UserCount())
}

func TestBasicAuthHookAddUser(t *testing.T) {
	hook := NewBasicAuthHook()

	hook.AddUser("user1", "pass1")
	assert.Equal(t, 1, hook.UserCount())
	assert.True(t, hook.HasUser("user1"))
	assert.False(t, hook.HasUser("user2"))

	hook.AddUser("user2", "pass2")
	assert.Equal(t, 2, hook.UserCount())
	assert.True(t, hook.HasUser("user2"))
}

func TestBasicAuthHookRemoveUser(t *testing.T) {
	hook := NewBasicAuthHook()

	hook.AddUser("user1", "pass1")
	hook.AddUser("user2", "pass2")
	assert.Equal(t, 2, hook.UserCount())

	hook.RemoveUser("user1")
	assert.Equal(t, 1, hook.UserCount())
	assert.False(t, hook.HasUser("user1"))
	assert.True(t, hook.HasUser("user2"))

	hook.RemoveUser("user2")
	assert.Equal(t, 0, hook.UserCount())
	assert.False(t, hook.HasUser("user2"))
}

func TestBasicAuthHookClear(t *testing.T) {
	hook := NewBasicAuthHook()

	hook.AddUser("user1", "pass1")
	hook.AddUser("user2", "pass2")
	hook.AddUser("user3", "pass3")
	assert.Equal(t, 3, hook.UserCount())

	hook.Clear()
	assert.Equal(t, 0, hook.UserCount())
	assert.False(t, hook.HasUser("user1"))
	assert.False(t, hook.HasUser("user2"))
	assert.False(t, hook.HasUser("user3"))
}

func TestBasicAuthHookLoadUsers(t *testing.T) {
	hook := NewBasicAuthHook()

	users := map[string]string{
		"user1": "pass1",
		"user2": "pass2",
		"user3": "pass3",
	}

	hook.LoadUsers(users)
	assert.Equal(t, 3, hook.UserCount())
	assert.True(t, hook.HasUser("user1"))
	assert.True(t, hook.HasUser("user2"))
	assert.True(t, hook.HasUser("user3"))
}

func TestBasicAuthHookAuthenticate(t *testing.T) {
	tests := []struct {
		name           string
		users          map[string]string
		username       string
		password       string
		expectedResult bool
	}{
		{
			name: "valid credentials",
			users: map[string]string{
				"user1": "pass1",
			},
			username:       "user1",
			password:       "pass1",
			expectedResult: true,
		},
		{
			name: "invalid password",
			users: map[string]string{
				"user1": "pass1",
			},
			username:       "user1",
			password:       "wrongpass",
			expectedResult: false,
		},
		{
			name: "non-existent user",
			users: map[string]string{
				"user1": "pass1",
			},
			username:       "user2",
			password:       "pass1",
			expectedResult: false,
		},
		{
			name:           "empty credentials",
			users:          map[string]string{},
			username:       "",
			password:       "",
			expectedResult: false,
		},
		{
			name: "multiple users",
			users: map[string]string{
				"user1": "pass1",
				"user2": "pass2",
				"user3": "pass3",
			},
			username:       "user2",
			password:       "pass2",
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := NewBasicAuthHook()
			hook.LoadUsers(tt.users)

			client := &Client{ID: "client1"}
			packet := &ConnectPacket{
				Username: tt.username,
				Password: []byte(tt.password),
			}

			result := hook.OnConnectAuthenticate(client, packet)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestBasicAuthHookTimingSafeComparison(t *testing.T) {
	hook := NewBasicAuthHook()
	hook.AddUser("user1", "secretpassword")

	client := &Client{ID: "client1"}

	packet1 := &ConnectPacket{
		Username: "user1",
		Password: []byte("secretpassword"),
	}
	assert.True(t, hook.OnConnectAuthenticate(client, packet1))

	packet2 := &ConnectPacket{
		Username: "user1",
		Password: []byte("secretpasswor"),
	}
	assert.False(t, hook.OnConnectAuthenticate(client, packet2))

	packet3 := &ConnectPacket{
		Username: "user1",
		Password: []byte("secretpasswords"),
	}
	assert.False(t, hook.OnConnectAuthenticate(client, packet3))
}

func TestBasicAuthHookConcurrentAccess(t *testing.T) {
	hook := NewBasicAuthHook()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				hook.AddUser("user", "pass")
				hook.HasUser("user")
				hook.RemoveUser("user")
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestBasicAuthHookWithManager(t *testing.T) {
	manager := NewManager()
	hook := NewBasicAuthHook()
	hook.AddUser("testuser", "testpass")

	require.NoError(t, manager.Add(hook))

	client := &Client{ID: "client1"}
	validPacket := &ConnectPacket{
		Username: "testuser",
		Password: []byte("testpass"),
	}
	assert.True(t, manager.OnConnectAuthenticate(client, validPacket))

	invalidPacket := &ConnectPacket{
		Username: "testuser",
		Password: []byte("wrongpass"),
	}
	assert.False(t, manager.OnConnectAuthenticate(client, invalidPacket))
}

func TestAnonymousAuthHook(t *testing.T) {
	hook := NewAnonymousAuthHook(true)

	assert.Equal(t, "anonymous-auth", hook.ID())
	assert.True(t, hook.Provides(OnConnectAuthenticate))
	assert.True(t, hook.IsAnonymousAllowed())
}

func TestAnonymousAuthHookAllowAnonymous(t *testing.T) {
	tests := []struct {
		name           string
		allowAnonymous bool
		username       string
		password       []byte
		expectedResult bool
	}{
		{
			name:           "allow anonymous with empty credentials",
			allowAnonymous: true,
			username:       "",
			password:       nil,
			expectedResult: true,
		},
		{
			name:           "deny anonymous with empty credentials",
			allowAnonymous: false,
			username:       "",
			password:       nil,
			expectedResult: false,
		},
		{
			name:           "allow with credentials",
			allowAnonymous: false,
			username:       "user1",
			password:       []byte("pass1"),
			expectedResult: true,
		},
		{
			name:           "allow with username only",
			allowAnonymous: true,
			username:       "user1",
			password:       nil,
			expectedResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := NewAnonymousAuthHook(tt.allowAnonymous)

			client := &Client{ID: "client1"}
			packet := &ConnectPacket{
				Username: tt.username,
				Password: tt.password,
			}

			result := hook.OnConnectAuthenticate(client, packet)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestAnonymousAuthHookSetAllowAnonymous(t *testing.T) {
	hook := NewAnonymousAuthHook(false)
	assert.False(t, hook.IsAnonymousAllowed())

	hook.SetAllowAnonymous(true)
	assert.True(t, hook.IsAnonymousAllowed())

	hook.SetAllowAnonymous(false)
	assert.False(t, hook.IsAnonymousAllowed())
}

func TestAnonymousAuthHookConcurrentAccess(t *testing.T) {
	hook := NewAnonymousAuthHook(true)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				hook.SetAllowAnonymous(j%2 == 0)
				hook.IsAnonymousAllowed()
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestCombinedAuthHooks(t *testing.T) {
	manager := NewManager()

	anonymousHook := NewAnonymousAuthHook(false)
	basicAuthHook := NewBasicAuthHook()
	basicAuthHook.AddUser("validuser", "validpass")

	require.NoError(t, manager.Add(anonymousHook))
	require.NoError(t, manager.Add(basicAuthHook))

	client := &Client{ID: "client1"}

	anonymousPacket := &ConnectPacket{
		Username: "",
		Password: nil,
	}
	assert.False(t, manager.OnConnectAuthenticate(client, anonymousPacket))

	validPacket := &ConnectPacket{
		Username: "validuser",
		Password: []byte("validpass"),
	}
	assert.True(t, manager.OnConnectAuthenticate(client, validPacket))

	invalidPacket := &ConnectPacket{
		Username: "validuser",
		Password: []byte("wrongpass"),
	}
	assert.False(t, manager.OnConnectAuthenticate(client, invalidPacket))
}

func TestAuthHooksEmptyPassword(t *testing.T) {
	hook := NewBasicAuthHook()
	hook.AddUser("user", "")

	client := &Client{ID: "client1"}
	packet := &ConnectPacket{
		Username: "user",
		Password: []byte(""),
	}

	assert.True(t, hook.OnConnectAuthenticate(client, packet))

	packet2 := &ConnectPacket{
		Username: "user",
		Password: []byte("notEmpty"),
	}
	assert.False(t, hook.OnConnectAuthenticate(client, packet2))
}

func TestAuthHooksSpecialCharacters(t *testing.T) {
	hook := NewBasicAuthHook()
	hook.AddUser("user@domain.com", "p@$$w0rd!#%")

	client := &Client{ID: "client1"}
	packet := &ConnectPacket{
		Username: "user@domain.com",
		Password: []byte("p@$$w0rd!#%"),
	}

	assert.True(t, hook.OnConnectAuthenticate(client, packet))
}

func TestAuthHooksUnicodePasswords(t *testing.T) {
	hook := NewBasicAuthHook()
	hook.AddUser("user", "密码🔒")

	client := &Client{ID: "client1"}
	packet := &ConnectPacket{
		Username: "user",
		Password: []byte("密码🔒"),
	}

	assert.True(t, hook.OnConnectAuthenticate(client, packet))

	packet2 := &ConnectPacket{
		Username: "user",
		Password: []byte("密码"),
	}
	assert.False(t, hook.OnConnectAuthenticate(client, packet2))
}

func TestAuthHooksMultipleUpdates(t *testing.T) {
	hook := NewBasicAuthHook()

	hook.AddUser("user1", "pass1")
	assert.Equal(t, 1, hook.UserCount())

	hook.AddUser("user1", "newpass1")
	assert.Equal(t, 1, hook.UserCount())

	client := &Client{ID: "client1"}

	oldPacket := &ConnectPacket{
		Username: "user1",
		Password: []byte("pass1"),
	}
	assert.False(t, hook.OnConnectAuthenticate(client, oldPacket))

	newPacket := &ConnectPacket{
		Username: "user1",
		Password: []byte("newpass1"),
	}
	assert.True(t, hook.OnConnectAuthenticate(client, newPacket))
}

func TestAnonymousAuthHookWithManager(t *testing.T) {
	manager := NewManager()
	hook := NewAnonymousAuthHook(true)

	require.NoError(t, manager.Add(hook))

	client := &Client{ID: "client1"}
	anonymousPacket := &ConnectPacket{
		Username: "",
		Password: nil,
	}

	assert.True(t, manager.OnConnectAuthenticate(client, anonymousPacket))

	hook.SetAllowAnonymous(false)
	assert.False(t, manager.OnConnectAuthenticate(client, anonymousPacket))
}
