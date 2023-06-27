package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

type EventHandler interface {
	HandleEvent(w http.ResponseWriter, r *http.Request)
}

func NewEvent(
	slackClient *slack.Client,
	botUserId string,
	channelId string,
) EventHandler {
	return &eventHandler{
		slackClient: slackClient,
		botUserId:   botUserId,
		channelId:   channelId,
	}
}

type eventHandler struct {
	slackClient *slack.Client
	botUserId   string
	channelId   string
}

func (h *eventHandler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("eventHandler.HandleEvent: io.ReadAll: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	eventsApiEvent, err := slackevents.ParseEvent(body, slackevents.OptionNoVerifyToken())
	if err != nil {
		log.Printf("eventHandler.HandleEvent: slackevents.ParseEvent: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch eventsApiEvent.Type {
	case slackevents.URLVerification:
		h.handleUrlVerificationEvent(ctx, w, body)
	case slackevents.CallbackEvent:
		h.handleCallbackEvent(ctx, w, eventsApiEvent.InnerEvent)
	default:
		log.Printf("eventHandler.HandleEvent: unexpected event type: %s", eventsApiEvent.Type)
		w.WriteHeader(http.StatusOK)
	}
}

func (h *eventHandler) handleUrlVerificationEvent(ctx context.Context, w http.ResponseWriter, body []byte) {
	var r *slackevents.EventsAPIURLVerificationEvent
	if err := json.Unmarshal(body, &r); err != nil {
		log.Printf("eventHandler.HandleEvent: json.Unmarshal: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(r.Challenge))
}

func (h *eventHandler) handleCallbackEvent(ctx context.Context, w http.ResponseWriter, innerEvent slackevents.EventsAPIInnerEvent) {
	switch event := innerEvent.Data.(type) {
	case *slackevents.AppMentionEvent:
		if err := h.handleAppMentionEvent(ctx, event); err != nil {
			log.Printf("eventHandler.HandleEvent: h.handleAppMentionEvent: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		log.Printf("eventHandler.HandleEvent: unexpected event type: %s, event=(%+v)", innerEvent.Type, innerEvent)
	}
	w.WriteHeader(http.StatusOK)
}

func (h *eventHandler) handleAppMentionEvent(ctx context.Context, e *slackevents.AppMentionEvent) error {
	log.Printf("eventHandler.HandleAppMentionEvent: %+v", e)

	msg, found := strings.CutPrefix(e.Text, fmt.Sprintf("<@%s> ", h.botUserId))
	if !found {
		// ignore if message doesn't starts with mention to bot
		return nil
	}

	if e.Channel != h.channelId {
		// alert if mention is not from non-authorized channels.
		message := slack.MsgOptionBlocks(
			slack.SectionBlock{
				Type: slack.MBTSection,
				Text: &slack.TextBlockObject{
					Type: slack.MarkdownType,
					Text: fmt.Sprintf(":warning: Please use this bot in <#%s>.", h.channelId),
				},
			},
		)
		respChannel, respTimestamp, err := h.slackClient.PostMessageContext(ctx, e.Channel, message)
		if err != nil {
			return fmt.Errorf("slackClient.PostMessageContext: %w", err)
		}
		log.Printf("eventHandler.HandleAppMentionEvent: alert that message is not in from non-authorized channel. respChannel=%s, respTimestamp=%s", respChannel, respTimestamp)
		return nil
	}

	message := slack.MsgOptionBlocks(
		slack.SectionBlock{
			Type: slack.MBTSection,
			Text: &slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("<@%s> You said `%s`", e.User, msg),
			},
		},
	)
	respChannel, respTimestamp, err := h.slackClient.PostMessageContext(ctx, e.Channel, message)
	if err != nil {
		return fmt.Errorf("slackClient.PostMessageContext: %w", err)
	}
	log.Printf("eventHandler.HandleAppMentionEvent: message sent. respChannel=%s, respTimestamp=%s", respChannel, respTimestamp)
	return nil
}
