package errorscs

import "errors"

var (
	ErrInternalServer   = errors.New("internal server error")
	ErrQrCodeEmpty      = errors.New("qr-code has not been found or is empty")
	ErrCKM13Empty       = errors.New("empty ckm13 value")
	ErrBdmNotSaved      = errors.New("failed to send file to BDM-API")
	ErrAlreadyReceived  = errors.New("was already received")
	ErrInProgress       = errors.New("record is still in process")
	ErrUnknown          = errors.New("is unknown")
	ErrBdmNoFile        = errors.New("BDM-API did not return any filenames")
	ErrBdmFilenameEmpty = errors.New("BDM-API returned an empty filename")
)
