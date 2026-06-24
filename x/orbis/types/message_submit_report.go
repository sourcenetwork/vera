package types

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgSubmitReport{}

func NewMsgSubmitReport(
	creator string,
	report ReportEnvelope,
	reportID string,
	signatureScheme string,
	signature []byte,
) *MsgSubmitReport {
	return &MsgSubmitReport{
		Creator:         creator,
		Report:          report,
		ReportId:        reportID,
		SignatureScheme: signatureScheme,
		Signature:       signature,
	}
}

func (msg *MsgSubmitReport) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	if msg.ReportId == "" {
		return ErrInvalidReport
	}
	if msg.SignatureScheme == "" {
		return ErrInvalidSignatureScheme
	}
	if len(msg.Signature) == 0 {
		return ErrInvalidSignaturePayload
	}
	if msg.Report.Domain == "" {
		return errorsmod.Wrap(ErrInvalidReport, "missing report domain")
	}
	if msg.Report.ReportType == "" {
		return errorsmod.Wrap(ErrInvalidReport, "missing report type")
	}
	if msg.Report.RingId == "" {
		return ErrInvalidRingId
	}
	return nil
}
