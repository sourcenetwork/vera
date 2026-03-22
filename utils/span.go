package utils

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const spanEventType = "msg_span"

type spanCtxKeyType struct{}

var spanCtxKey spanCtxKeyType = struct{}{}

// MsgSpan is a container for execution data generated during the processing of a Msg
// MsgSpan is used as tracing data and must not be relied upon by users, it's an introspection tool only.
// Attributes are not guaranteed to be stable or deterministic
type MsgSpan struct {
	start    time.Time
	duration time.Duration

	message    string
	attributes map[string]string
}

func NewSpan() *MsgSpan {
	return &MsgSpan{
		start:      time.Now(),
		attributes: make(map[string]string),
	}
}

func (s *MsgSpan) End() {
	s.duration = time.Since(s.start)
}

func (s *MsgSpan) SetMessage(msg string) {
	s.message = msg
}

func (s *MsgSpan) Attr(key, value string) {
	s.attributes[key] = value
}

func (s *MsgSpan) ToEvent() sdk.Event {
	var attrs []sdk.Attribute
	attrs = append(attrs, sdk.NewAttribute("start", s.start.String()))
	attrs = append(attrs, sdk.NewAttribute("duration", s.duration.String()))
	if s.message != "" {
		attrs = append(attrs, sdk.NewAttribute("message", s.message))
	}

	for key, value := range s.attributes {
		attrs = append(attrs, sdk.NewAttribute(key, value))
	}

	return sdk.NewEvent(spanEventType, attrs...)
}

// WithMsgSpan returns a new Context with an initialized MsgSpan
func WithMsgSpan(ctx sdk.Context) sdk.Context {
	goCtx := ctx.Context()
	goCtx = context.WithValue(goCtx, spanCtxKey, NewSpan())
	return ctx.WithContext(goCtx)
}

func GetMsgSpan(ctx sdk.Context) *MsgSpan {
	span, ok := ctx.Context().Value(spanCtxKey).(*MsgSpan)
	if !ok {
		return nil
	}
	return span
}

// FinalizeSpan ends the span duration frame, transforms it into an SDK Event and emits it using the event manager
func FinalizeSpan(ctx sdk.Context) {
	event := GetMsgSpan(ctx).ToEvent()
	ctx.EventManager().EmitEvent(event)
}
