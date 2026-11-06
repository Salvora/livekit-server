package telemetry_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/livekit/livekit-server/pkg/telemetry"
	"github.com/livekit/livekit-server/pkg/telemetry/telemetryfakes"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
)

// recordingNotifier keeps the webhook events it is asked to send.
type recordingNotifier struct {
	mu     sync.Mutex
	events []*livekit.WebhookEvent
}

func (n *recordingNotifier) RegisterProcessedHook(func(context.Context, *livekit.WebhookInfo)) {}
func (n *recordingNotifier) SetKeys(string, string)                                            {}
func (n *recordingNotifier) SetFilter(webhook.FilterParams)                                    {}
func (n *recordingNotifier) Stop(bool)                                                         {}

func (n *recordingNotifier) QueueNotify(_ context.Context, event *livekit.WebhookEvent, _ ...webhook.NotifyOption) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.events = append(n.events, event)
	return nil
}

func (n *recordingNotifier) sent() []*livekit.WebhookEvent {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]*livekit.WebhookEvent(nil), n.events...)
}

func Test_OnTrackMuteChange_WebhookIsSent(t *testing.T) {
	notifier := &recordingNotifier{}
	sut := telemetry.NewTelemetryService(notifier, &telemetryfakes.FakeAnalyticsService{})

	room := &livekit.Room{Sid: "RoomSid", Name: "RoomName", Metadata: `{"server":"S1"}`}
	track := &livekit.TrackInfo{Sid: "TR_1", Type: livekit.TrackType_AUDIO}
	sut.TrackMuted(context.Background(), room, "PA_1", "user1", track)
	sut.TrackUnmuted(context.Background(), room, "PA_1", "user1", track)

	require.Eventually(t, func() bool { return len(notifier.sent()) == 2 }, time.Second, 10*time.Millisecond)
	for i, name := range []string{"track_muted", "track_unmuted"} {
		event := notifier.sent()[i]
		require.Equal(t, name, event.Event)
		require.Equal(t, room, event.Room)
		require.Equal(t, "PA_1", event.Participant.Sid)
		require.Equal(t, "user1", event.Participant.Identity)
		require.Equal(t, track, event.Track)
	}
}
