package session

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name            string
		clientID        string
		cleanStart      bool
		expiryInterval  uint32
		protocolVersion byte
	}{
		{
			name:            "create new session with clean start",
			clientID:        "client1",
			cleanStart:      true,
			expiryInterval:  300,
			protocolVersion: 5,
		},
		{
			name:            "create persistent session",
			clientID:        "client2",
			cleanStart:      false,
			expiryInterval:  0,
			protocolVersion: 4,
		},
		{
			name:            "create session with expiry",
			clientID:        "client3",
			cleanStart:      false,
			expiryInterval:  3600,
			protocolVersion: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := New(tt.clientID, tt.cleanStart, tt.expiryInterval, tt.protocolVersion)

			require.NotNil(t, session)
			assert.Equal(t, tt.clientID, session.ClientID)
			assert.Equal(t, tt.cleanStart, session.CleanStart)
			assert.Equal(t, tt.expiryInterval, session.ExpiryInterval)
			assert.Equal(t, tt.protocolVersion, session.ProtocolVersion)
			assert.Equal(t, StateNew, session.State)
			assert.NotNil(t, session.Subscriptions)
			assert.NotNil(t, session.PendingPublish)
			assert.NotNil(t, session.PendingPubrel)
			assert.NotNil(t, session.PendingPubcomp)
			assert.Equal(t, uint16(1), session.nextPacketID)
			assert.Equal(t, uint16(65535), session.ReceiveMaximum)
		})
	}
}

func TestSession_SetActive(t *testing.T) {
	session := New("client1", true, 300, 5)
	assert.Equal(t, StateNew, session.GetState())

	session.SetActive()
	assert.Equal(t, StateActive, session.GetState())
}

func TestSession_SetDisconnected(t *testing.T) {
	session := New("client1", true, 300, 5)
	session.SetActive()

	session.SetDisconnected()
	assert.Equal(t, StateDisconnected, session.GetState())
	assert.False(t, session.DisconnectedAt.IsZero())
}

func TestSession_SetExpired(t *testing.T) {
	session := New("client1", true, 300, 5)

	session.SetExpired()
	assert.Equal(t, StateExpired, session.GetState())
}

func TestSession_IsExpired(t *testing.T) {
	tests := []struct {
		name           string
		setupSession   func() *Session
		expectedExpiry bool
	}{
		{
			name: "persistent session with no expiry never expires",
			setupSession: func() *Session {
				s := New("client1", false, 0, 5)
				s.SetDisconnected()
				time.Sleep(10 * time.Millisecond)
				return s
			},
			expectedExpiry: false,
		},
		{
			name: "session with expiry interval not yet expired",
			setupSession: func() *Session {
				s := New("client2", false, 10, 5)
				s.SetDisconnected()
				return s
			},
			expectedExpiry: false,
		},
		{
			name: "session with expiry interval expired",
			setupSession: func() *Session {
				s := New("client3", false, 1, 5)
				s.SetDisconnected()
				s.DisconnectedAt = time.Now().Add(-2 * time.Second)
				return s
			},
			expectedExpiry: true,
		},
		{
			name: "session marked as expired",
			setupSession: func() *Session {
				s := New("client4", false, 300, 5)
				s.SetExpired()
				return s
			},
			expectedExpiry: true,
		},
		{
			name: "active session not expired",
			setupSession: func() *Session {
				s := New("client5", false, 1, 5)
				s.SetActive()
				return s
			},
			expectedExpiry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := tt.setupSession()
			assert.Equal(t, tt.expectedExpiry, session.IsExpired())
		})
	}
}

func TestSession_Touch(t *testing.T) {
	session := New("client1", true, 300, 5)
	initialTime := session.LastAccessedAt

	time.Sleep(10 * time.Millisecond)
	session.Touch()

	assert.True(t, session.LastAccessedAt.After(initialTime))
}

func TestSession_WillMessage(t *testing.T) {
	tests := []struct {
		name          string
		willMessage   *WillMessage
		delayInterval uint32
	}{
		{
			name: "set will message without delay",
			willMessage: &WillMessage{
				Topic:   "client/status",
				Payload: []byte("offline"),
				QoS:     1,
				Retain:  true,
			},
			delayInterval: 0,
		},
		{
			name: "set will message with delay",
			willMessage: &WillMessage{
				Topic:   "client/status",
				Payload: []byte("offline"),
				QoS:     2,
				Retain:  false,
			},
			delayInterval: 60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := New("client1", true, 300, 5)

			session.SetWillMessage(tt.willMessage, tt.delayInterval)
			will := session.GetWillMessage()
			require.NotNil(t, will)
			assert.Equal(t, tt.willMessage.Topic, will.Topic)
			assert.Equal(t, tt.willMessage.Payload, will.Payload)
			assert.Equal(t, tt.willMessage.QoS, will.QoS)
			assert.Equal(t, tt.willMessage.Retain, will.Retain)
			assert.Equal(t, tt.delayInterval, session.WillDelayInterval)

			session.ClearWillMessage()
			assert.Nil(t, session.GetWillMessage())
		})
	}
}

func TestSession_ShouldPublishWill(t *testing.T) {
	tests := []struct {
		name          string
		setupSession  func() *Session
		shouldPublish bool
	}{
		{
			name: "no will message",
			setupSession: func() *Session {
				return New("client1", true, 300, 5)
			},
			shouldPublish: false,
		},
		{
			name: "will message without delay",
			setupSession: func() *Session {
				s := New("client2", true, 300, 5)
				s.SetWillMessage(&WillMessage{
					Topic:   "test",
					Payload: []byte("test"),
				}, 0)
				s.SetDisconnected()
				return s
			},
			shouldPublish: true,
		},
		{
			name: "will message with delay not yet passed",
			setupSession: func() *Session {
				s := New("client3", true, 300, 5)
				s.SetWillMessage(&WillMessage{
					Topic:   "test",
					Payload: []byte("test"),
				}, 10)
				s.SetDisconnected()
				return s
			},
			shouldPublish: false,
		},
		{
			name: "will message with delay passed",
			setupSession: func() *Session {
				s := New("client4", true, 300, 5)
				s.SetWillMessage(&WillMessage{
					Topic:   "test",
					Payload: []byte("test"),
				}, 1)
				s.SetDisconnected()
				s.DisconnectedAt = time.Now().Add(-2 * time.Second)
				return s
			},
			shouldPublish: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := tt.setupSession()
			assert.Equal(t, tt.shouldPublish, session.ShouldPublishWill())
		})
	}
}

func TestSession_Subscriptions(t *testing.T) {
	session := New("client1", true, 300, 5)

	sub1 := &Subscription{
		TopicFilter:       "test/topic1",
		QoS:               1,
		NoLocal:           false,
		RetainAsPublished: true,
		RetainHandling:    0,
	}
	sub2 := &Subscription{
		TopicFilter:       "test/topic2",
		QoS:               2,
		NoLocal:           true,
		RetainAsPublished: false,
		RetainHandling:    1,
	}

	session.AddSubscription(sub1)
	session.AddSubscription(sub2)

	retrieved, ok := session.GetSubscription("test/topic1")
	require.True(t, ok)
	assert.Equal(t, sub1.TopicFilter, retrieved.TopicFilter)
	assert.Equal(t, sub1.QoS, retrieved.QoS)

	allSubs := session.GetAllSubscriptions()
	assert.Len(t, allSubs, 2)

	session.RemoveSubscription("test/topic1")
	_, ok = session.GetSubscription("test/topic1")
	assert.False(t, ok)

	session.ClearSubscriptions()
	allSubs = session.GetAllSubscriptions()
	assert.Len(t, allSubs, 0)
}

func TestSession_NextPacketID(t *testing.T) {
	session := New("client1", true, 300, 5)

	id1 := session.NextPacketID()
	assert.Equal(t, uint16(1), id1)

	id2 := session.NextPacketID()
	assert.Equal(t, uint16(2), id2)

	session.AddPendingPublish(&PendingMessage{PacketID: 3})
	id3 := session.NextPacketID()
	assert.NotEqual(t, uint16(3), id3)

	session.nextPacketID = 65535
	id4 := session.NextPacketID()
	assert.NotEqual(t, uint16(0), id4)
}

func TestSession_PendingPublish(t *testing.T) {
	session := New("client1", true, 300, 5)

	msg := &PendingMessage{
		PacketID:  1,
		Topic:     "test/topic",
		Payload:   []byte("test payload"),
		QoS:       1,
		Retain:    false,
		Timestamp: time.Now(),
	}

	session.AddPendingPublish(msg)

	retrieved, ok := session.GetPendingPublish(1)
	require.True(t, ok)
	assert.Equal(t, msg.PacketID, retrieved.PacketID)
	assert.Equal(t, msg.Topic, retrieved.Topic)
	assert.Equal(t, msg.Payload, retrieved.Payload)

	allPending := session.GetAllPendingPublish()
	assert.Len(t, allPending, 1)

	session.RemovePendingPublish(1)
	_, ok = session.GetPendingPublish(1)
	assert.False(t, ok)
}

func TestSession_PendingPubrel(t *testing.T) {
	session := New("client1", true, 300, 5)

	assert.False(t, session.HasPendingPubrel(1))

	session.AddPendingPubrel(1)
	assert.True(t, session.HasPendingPubrel(1))

	session.RemovePendingPubrel(1)
	assert.False(t, session.HasPendingPubrel(1))
}

func TestSession_PendingPubcomp(t *testing.T) {
	session := New("client1", true, 300, 5)

	assert.False(t, session.HasPendingPubcomp(1))

	session.AddPendingPubcomp(1)
	assert.True(t, session.HasPendingPubcomp(1))

	session.RemovePendingPubcomp(1)
	assert.False(t, session.HasPendingPubcomp(1))
}

func TestSession_Clear(t *testing.T) {
	session := New("client1", true, 300, 5)

	session.AddSubscription(&Subscription{TopicFilter: "test/topic", QoS: 1})
	session.AddPendingPublish(&PendingMessage{PacketID: 1, Topic: "test", Payload: []byte("test")})
	session.AddPendingPubrel(2)
	session.AddPendingPubcomp(3)
	session.SetWillMessage(&WillMessage{Topic: "will", Payload: []byte("will")}, 0)

	session.Clear()

	assert.Len(t, session.Subscriptions, 0)
	assert.Len(t, session.PendingPublish, 0)
	assert.Len(t, session.PendingPubrel, 0)
	assert.Len(t, session.PendingPubcomp, 0)
	assert.Nil(t, session.WillMessage)
}

func TestSession_UpdateExpiryInterval(t *testing.T) {
	session := New("client1", true, 300, 5)
	assert.Equal(t, uint32(300), session.ExpiryInterval)

	session.UpdateExpiryInterval(600)
	assert.Equal(t, uint32(600), session.ExpiryInterval)
}

func TestSession_ConcurrentAccess(t *testing.T) {
	session := New("client1", true, 300, 5)
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				session.AddSubscription(&Subscription{
					TopicFilter: "test/topic",
					QoS:         1,
				})
				session.GetAllSubscriptions()
				session.Touch()
				session.NextPacketID()
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
