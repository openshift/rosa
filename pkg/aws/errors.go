package aws

import (
	"errors"

	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/smithy-go"
)

const (
	invalidClientTokenID         = "InvalidClientTokenId"
	AccessDenied                 = "AccessDenied"
)

func IsErrorCode(err error, code string) bool {
	var apiErr smithy.APIError
	return errors.As(err, &apiErr) && apiErr.ErrorCode() == code
}

func IsEntityAlreadyExistsException(err error) bool {
	var entityAlreadyExists *iamtypes.EntityAlreadyExistsException
	return errors.As(err, &entityAlreadyExists)
}

func IsNoSuchEntityException(err error) bool {
	var noSuchEntity *iamtypes.NoSuchEntityException
	return errors.As(err, &noSuchEntity)
}

func IsDeleteConfictException(err error) bool {
	var deleteConflict *iamtypes.DeleteConflictException
	return errors.As(err, &deleteConflict)
}

func IsAccessDeniedException(err error) bool {
	return IsErrorCode(err, AccessDenied)
}

func IsInvalidTokenException(err error) bool {
	return IsErrorCode(err, invalidClientTokenID)
}