package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v10/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v10/modules/core/exported"

	acpkeeper "github.com/sourcenetwork/sourcehub/x/acp/keeper"
)

type icaHostRoutingMiddleware struct {
	app       *App
	acpKeeper *acpkeeper.Keeper
	next      porttypes.IBCModule
}

func newICAHostRoutingMiddleware(app *App, acpKeeper *acpkeeper.Keeper, next porttypes.IBCModule) porttypes.IBCModule {
	return &icaHostRoutingMiddleware{
		app:       app,
		acpKeeper: acpKeeper,
		next:      next,
	}
}

func (m *icaHostRoutingMiddleware) OnRecvPacket(
	ctx sdk.Context,
	portID string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	return m.next.OnRecvPacket(ctx, portID, packet, relayer)
}

func (m *icaHostRoutingMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	portID string,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	return m.next.OnAcknowledgementPacket(ctx, portID, packet, acknowledgement, relayer)
}

func (m *icaHostRoutingMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	portID string,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	return m.next.OnTimeoutPacket(ctx, portID, packet, relayer)
}

func (m *icaHostRoutingMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID, channelID string,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return m.next.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, counterparty, version)
}

func (m *icaHostRoutingMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID, channelID string,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	version, err := m.next.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, counterparty, counterpartyVersion)
	if err != nil {
		return "", err
	}

	// Only process ICA host channels during DeliverTx to avoid duplicate processing
	if portID == "icahost" && len(connectionHops) > 0 && !ctx.IsCheckTx() {
		m.acpKeeper.HandleICAChannelOpen(
			ctx,
			connectionHops[0],
			counterparty.PortId,
			m.app.ICAHostKeeper,
			m.app.IBCKeeper.ConnectionKeeper,
			m.app.IBCKeeper.ClientKeeper,
		)
	}

	return version, nil
}

func (m *icaHostRoutingMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID, channelID, counterpartyChannelID, counterpartyVersion string,
) error {
	return m.next.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

func (m *icaHostRoutingMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return m.next.OnChanOpenConfirm(ctx, portID, channelID)
}

func (m *icaHostRoutingMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return m.next.OnChanCloseInit(ctx, portID, channelID)
}

func (m *icaHostRoutingMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return m.next.OnChanCloseConfirm(ctx, portID, channelID)
}
