package bot

import (
	"context"
	"fmt"
	"github.com/clambin/tado-exporter/internal/controller/rules/action"
	"github.com/clambin/tado-exporter/internal/poller"
	"github.com/clambin/tado/v2"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	"log/slog"
	"sync"
)

type Bot struct {
	TadoClient
	SocketModeHandler
	poller     poller.Poller
	controller Controller
	logger     *slog.Logger
	update     poller.Update
	lock       sync.RWMutex
	updated    bool
}

type TadoClient interface {
	action.TadoClient
	DeletePresenceLockWithResponse(ctx context.Context, homeId tado.HomeId, reqEditors ...tado.RequestEditorFn) (*tado.DeletePresenceLockResponse, error)
}

type SocketModeHandler interface {
	HandleSlashCommand(command string, f socketmode.SocketmodeHandlerFunc)
	HandleDefault(f socketmode.SocketmodeHandlerFunc)
	RunEventLoopContext(ctx context.Context) error
}

type SlackSender interface {
	PostEphemeral(channelID string, userID string, options ...slack.MsgOption) (string, error)
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
}

type Controller interface {
	ReportTasks() []string
}

func New(tadoClient TadoClient, handler SocketModeHandler, p poller.Poller, c Controller, logger *slog.Logger) *Bot {
	b := Bot{
		TadoClient:        tadoClient,
		SocketModeHandler: handler,
		poller:            p,
		controller:        c,
		logger:            logger,
	}

	b.SocketModeHandler.HandleSlashCommand("/rooms", b.runCommand(b.listRooms))
	b.SocketModeHandler.HandleSlashCommand("/users", b.runCommand(b.listUsers))
	b.SocketModeHandler.HandleSlashCommand("/rules", b.runCommand(b.listRules))
	b.SocketModeHandler.HandleSlashCommand("/refresh", b.runCommand(b.refresh))
	b.SocketModeHandler.HandleSlashCommand("/setroom", b.runCommand(b.setRoom))
	b.SocketModeHandler.HandleSlashCommand("/sethome", b.runCommand(b.setHome))
	b.SocketModeHandler.HandleDefault(func(event *socketmode.Event, _ *socketmode.Client) {
		logger.Debug("event received", "type", event.Type)
	})

	return &b
}

// Run the controller
func (b *Bot) Run(ctx context.Context) error {
	b.logger.Debug("bot started")
	defer b.logger.Debug("bot stopped")
	errCh := make(chan error)
	go func() { errCh <- b.SocketModeHandler.RunEventLoopContext(ctx) }()

	ch := b.poller.Subscribe()
	defer b.poller.Unsubscribe(ch)

	for {
		select {
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("bot: %w", err)
			}
		case <-ctx.Done():
			return nil
		case update := <-ch:
			b.setUpdate(update)
		}
	}
}

func (b *Bot) runCommand(f func(command slack.SlashCommand, sender SlackSender) error) socketmode.SocketmodeHandlerFunc {
	return func(event *socketmode.Event, client *socketmode.Client) {
		client.Ack(*event.Request)
		data := event.Data.(slack.SlashCommand)
		err := f(data, client)
		if err != nil {
			_, err = client.PostEphemeral(data.ChannelID, data.UserID, slack.MsgOptionText("command failed: "+err.Error(), false))
		}
		if err == nil {
			b.logger.Warn("command failed", "cmd", data.Command, "err", err)
		}
	}
}

func (b *Bot) setUpdate(update poller.Update) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.update = update
	b.updated = true
}

func (b *Bot) getUpdate() (poller.Update, bool) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.update, b.updated
}
