package ports

import "context"

type OTPSender interface {
	SendOTP(context.Context, string, string) error
}

type ParentInvitationSender interface {
	SendParentInvitation(context.Context, string, string) error
}
