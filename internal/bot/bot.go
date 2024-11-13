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
)

type Bot struct {
	commandRunner
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
		SocketModeHandler: handler,
		poller:            p,
		logger:            logger,
	}

	b.SocketModeHandler.HandleSlashCommand("/rooms", b.runCommand(b.commandRunner.listRooms))
	b.SocketModeHandler.HandleSlashCommand("/users", b.runCommand(b.commandRunner.listUsers))
	b.SocketModeHandler.HandleSlashCommand("/rules", b.runCommand(b.commandRunner.listRules))
	b.SocketModeHandler.HandleSlashCommand("/refresh", b.runCommand(b.commandRunner.refresh))
	b.SocketModeHandler.HandleSlashCommand("/setroom", b.runCommand(b.commandRunner.setRoom))
	b.SocketModeHandler.HandleInteraction(slack.InteractionTypeShortcut, b.handleShortcut)
	b.SocketModeHandler.HandleInteraction(slack.InteractionTypeViewSubmission, b.handleShortcutSubmission)
	b.SocketModeHandler.HandleDefault(func(event *socketmode.Event, _ *socketmode.Client) {
		logger.Debug("unhandled event received", "type", event.Type)
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
		case update := <-ch:
			r.commandRunner.setUpdate(update)
		}
	}
}

func (r *Bot) runCommand(f func(command slack.SlashCommand, sender SlackSender) error) socketmode.SocketmodeHandlerFunc {
	return func(event *socketmode.Event, client *socketmode.Client) {
		client.Ack(*event.Request)
		data := event.Data.(slack.SlashCommand)
		if err := f(data, client); err != nil {
			if _, err = client.PostEphemeral(data.ChannelID, data.UserID, slack.MsgOptionText("command failed: "+err.Error(), false)); err != nil {
				r.logger.Warn("failed to post command output", "cmd", data.Command, "err", err)
			}
		}
	}
}

func (r *Bot) handleShortcut(event *socketmode.Event, client *socketmode.Client) {
	data := event.Data.(slack.InteractionCallback)
	switch data.CallbackID {
	case setRoomCallbackID:
		// what to do when we get this event but we don't have an update yet?  we need it to populate the rooms dropdown.
		client.Ack(*event.Request)

		resp, err := client.OpenView(data.TriggerID, setRoomView())
		if err != nil {
			r.logger.Warn("failed to open view", "err", err, "callbackID", data.CallbackID)
			return
		}
		r.logger.Debug("opened view", "callbackID", resp.CallbackID)
	default:
		r.logger.Warn("received unexpected shortcut CallbackID", "callbackID", data.CallbackID)
	}
}

func (r *Bot) handleShortcutSubmission(event *socketmode.Event, client *socketmode.Client) {
	data := event.Data.(slack.InteractionCallback)
	switch data.CallbackID {
	case setRoomCallbackID:
		client.Ack(*event.Request)
		r.logger.Info("received shortcut input", "callbackID", data.CallbackID, "input", data.Submission)
	default:
		r.logger.Warn("received unexpected shortcut CallbackID", "callbackID", data.CallbackID)
	}
}
