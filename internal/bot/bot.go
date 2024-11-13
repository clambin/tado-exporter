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
	commandRunner
	shortcuts
	SocketModeHandler
	poller poller.Poller
	logger *slog.Logger
}

type TadoClient interface {
	action.TadoClient
	DeletePresenceLockWithResponse(ctx context.Context, homeId tado.HomeId, reqEditors ...tado.RequestEditorFn) (*tado.DeletePresenceLockResponse, error)
}

type SocketModeHandler interface {
	HandleSlashCommand(command string, f socketmode.SocketmodeHandlerFunc)
	HandleInteraction(et slack.InteractionType, f socketmode.SocketmodeHandlerFunc)
	HandleDefault(f socketmode.SocketmodeHandlerFunc)
	RunEventLoopContext(ctx context.Context) error
}

type SlackSender interface {
	PostEphemeral(channelID string, userID string, options ...slack.MsgOption) (string, error)
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
	OpenView(triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error)
	UpdateView(view slack.ModalViewRequest, externalID string, hash string, viewID string) (*slack.ViewResponse, error)
	Ack(socketmode.Request, ...any)
}

type Controller interface {
	ReportTasks() []string
}

func New(tadoClient TadoClient, handler SocketModeHandler, p poller.Poller, c Controller, logger *slog.Logger) *Bot {
	b := Bot{
		commandRunner: commandRunner{
			TadoClient: tadoClient,
			poller:     p,
			controller: c,
			logger:     logger,
		},
		shortcuts: shortcuts{
			handlers: map[string]shortcutHandler{
				setRoomCallbackID: &setRoomShortcut{
					TadoClient: tadoClient,
					logger:     logger.With("shortcut", "setRoom"),
				},
				setHomeCallbackID: &setHomeShortcut{
					TadoClient: tadoClient,
					logger:     logger.With("shortcut", "setHome"),
				},
			},
		},
		SocketModeHandler: handler,
		poller:            p,
		logger:            logger,
	}

	b.SocketModeHandler.HandleSlashCommand("/rooms", b.runCommand(b.commandRunner.listRooms))
	b.SocketModeHandler.HandleSlashCommand("/users", b.runCommand(b.commandRunner.listUsers))
	b.SocketModeHandler.HandleSlashCommand("/rules", b.runCommand(b.commandRunner.listRules))
	b.SocketModeHandler.HandleSlashCommand("/refresh", b.runCommand(b.commandRunner.refresh))
	b.SocketModeHandler.HandleInteraction(slack.InteractionTypeShortcut, b.runShortcut(b.shortcuts.dispatch))
	b.SocketModeHandler.HandleInteraction(slack.InteractionTypeBlockActions, b.runShortcut(b.shortcuts.dispatch))
	b.SocketModeHandler.HandleInteraction(slack.InteractionTypeViewSubmission, b.runShortcut(b.shortcuts.dispatch))
	b.SocketModeHandler.HandleDefault(func(event *socketmode.Event, _ *socketmode.Client) {
		logger.Debug("unhandled event received", "type", event.Type, "data", fmt.Sprintf("%T", event.Data))
	})

	return &b
}

// Run the controller
func (r *Bot) Run(ctx context.Context) error {
	r.logger.Debug("bot started")
	defer r.logger.Debug("bot stopped")
	errCh := make(chan error)
	go func() { errCh <- r.SocketModeHandler.RunEventLoopContext(ctx) }()

	ch := r.poller.Subscribe()
	defer r.poller.Unsubscribe(ch)

	for {
		select {
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("bot: %w", err)
			}
		case <-ctx.Done():
			return nil
		case u := <-ch:
			r.commandRunner.setUpdate(u)
			r.shortcuts.setUpdate(u)
		}
	}
}

func (r *Bot) runCommand(cmd func(command slack.SlashCommand, sender SlackSender) error) socketmode.SocketmodeHandlerFunc {
	return func(event *socketmode.Event, client *socketmode.Client) {
		client.Ack(*event.Request)
		data := event.Data.(slack.SlashCommand)
		if err := cmd(data, client); err != nil {
			if _, err = client.PostEphemeral(data.ChannelID, data.UserID, slack.MsgOptionText("command failed: "+err.Error(), false)); err != nil {
				r.logger.Warn("failed to post command output", "cmd", data.Command, "err", err)
			}
		}
	}
}

func (r *Bot) runShortcut(shortcut func(data slack.InteractionCallback, sender SlackSender) error) socketmode.SocketmodeHandlerFunc {
	return func(event *socketmode.Event, client *socketmode.Client) {
		data := event.Data.(slack.InteractionCallback)
		if err := shortcut(data, client); err != nil {
			r.logger.Warn("shortcut failed", "err", err, "type", data.Type)
			return
		}
		client.Ack(*event.Request)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type update struct {
	update *poller.Update
	lock   sync.RWMutex
}

func (r *update) setUpdate(u poller.Update) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.update = &u
}

func (r *update) getUpdate() (poller.Update, bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.update == nil {
		return poller.Update{}, false
	}
	return *r.update, true
}
