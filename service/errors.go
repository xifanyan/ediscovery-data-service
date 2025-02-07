package service

import "errors"

var (
	ErrApplicationRequired     = errors.New("application is required")
	ErrApplicationNameRequired = errors.New("application name is required")
	ErrRedactionReasonRequired = errors.New("redactionReason is required")
	ErrCustodianRequired       = errors.New("custodian is required")
	ErrTemplateRequired        = errors.New("template is required")

	ErrTemplateNotFound = errors.New("template not found")
)
