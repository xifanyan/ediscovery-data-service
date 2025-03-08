package service

import "errors"

var (
	ErrApplicationRequired     = errors.New("application is required")
	ErrApplicationNameRequired = errors.New("application name is required")
	ErrRedactionReasonRequired = errors.New("redactionReason is required")
	ErrCustodianRequired       = errors.New("custodian is required")
	ErrTemplateRequired        = errors.New("template is required")
	ErrValidEntityTypeRequired = errors.New("valid entity type is required")

	ErrUserNotFound     = errors.New("user not found")
	ErrGroupNotFound    = errors.New("groupnot found")
	ErrEntityNotFound   = errors.New("entity not found")
	ErrTemplateNotFound = errors.New("template not found")

	ErrNotImplemented = errors.New("not implemented")
)
