package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrOrderNotFound           = errors.New("order not found")
	ErrNoPermission            = errors.New("no permission")
	ErrSamePromoExists         = errors.New("promo with same code exists")
	ErrPromoCodeInvalid        = errors.New("promo code invalid")
	ErrPromoCodeMinAmount      = errors.New("order sum below minimum")
	ErrInvalidStateTransaction = errors.New("invalid order state transition")
)

type ErrHasActiveOrdeer struct {
	ID uuid.UUID
}

func (e *ErrHasActiveOrdeer) Error() string {
	return fmt.Sprintf("already has active order: %s", e.ID)
}

type ErrOrderLimit struct {
	LastOperationTime time.Time
}

func (e *ErrOrderLimit) Error() string {
	return fmt.Sprintf("last creation time: %s", e.LastOperationTime)
}

type ErrProductNotFound struct {
	IDs []uuid.UUID
}

func (e *ErrProductNotFound) Error() string {
	if len(e.IDs) == 0 {
		return "product(s) not found"
	}
	return fmt.Sprintf("product(s) not found: %v", e.IDs)
}

type ErrProductNotActive struct {
	IDs []uuid.UUID
}

func (e *ErrProductNotActive) Error() string {
	if len(e.IDs) == 0 {
		return "product(s) not active"
	}
	return fmt.Sprintf("product(s) not active: %v", e.IDs)
}

type ErrProductNotEnough struct {
	IDs []uuid.UUID
}

func (e *ErrProductNotEnough) Error() string {
	if len(e.IDs) == 0 {
		return "not enough product(s)"
	}
	return fmt.Sprintf("not enough product(s): %v", e.IDs)
}
