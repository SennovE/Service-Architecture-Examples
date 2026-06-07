package models

import (
	"errors"
	"time"
)

const (
	EventProductReceived  = "PRODUCT_RECEIVED"
	EventProductShipped   = "PRODUCT_SHIPPED"
	EventProductMoved     = "PRODUCT_MOVED"
	EventProductReserved  = "PRODUCT_RESERVED"
	EventProductReleased  = "PRODUCT_RELEASED"
	EventInventoryCounted = "INVENTORY_COUNTED"
	EventOrderCreated     = "ORDER_CREATED"
	EventOrderCompleted   = "ORDER_COMPLETED"

	ErrorCodeValidation      = "VALIDATION_ERROR"
	ErrorCodeProcessing      = "PROCESSING_ERROR"
	ErrorCodeCassandra       = "CASSANDRA_ERROR"
	ErrorCodeDeserialization = "DESERIALIZATION_ERROR"
)

type OrderItem struct {
	ProductID string `avro:"product_id" json:"product_id"`
	ZoneID    string `avro:"zone_id" json:"zone_id"`
	Quantity  int    `avro:"quantity" json:"quantity"`
}

type WarehouseEvent struct {
	EventID        string      `avro:"event_id" json:"event_id"`
	SchemaVersion  int         `avro:"schema_version" json:"schema_version"`
	EventType      string      `avro:"event_type" json:"event_type"`
	SequenceNumber int64       `avro:"sequence_number" json:"sequence_number"`
	ProductID      *string     `avro:"product_id" json:"product_id,omitempty"`
	ZoneID         *string     `avro:"zone_id" json:"zone_id,omitempty"`
	FromZoneID     *string     `avro:"from_zone_id" json:"from_zone_id,omitempty"`
	ToZoneID       *string     `avro:"to_zone_id" json:"to_zone_id,omitempty"`
	Quantity       *int        `avro:"quantity" json:"quantity,omitempty"`
	OrderID        *string     `avro:"order_id" json:"order_id,omitempty"`
	OrderItems     []OrderItem `avro:"order_items" json:"order_items,omitempty"`
	SupplierID     *string     `avro:"supplier_id" json:"supplier_id,omitempty"`
}

type KafkaMetadata struct {
	Topic     string `avro:"topic" json:"topic"`
	Partition int    `avro:"partition" json:"partition"`
	Offset    int64  `avro:"offset" json:"offset"`
}

type ApplyResult struct {
	Duplicate bool
	Skipped   bool
}

type WarehouseDLQEvent struct {
	OriginalEvent string        `avro:"original_event" json:"original_event"`
	ErrorCode     string        `avro:"error_code" json:"error_code"`
	ErrorReason   string        `avro:"error_reason" json:"error_reason"`
	FailedAt      time.Time     `avro:"failed_at" json:"failed_at"`
	KafkaMetadata KafkaMetadata `avro:"kafka_metadata" json:"kafka_metadata"`
}

type CodedError struct {
	Code string
	Err  error
}

func (e *CodedError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *CodedError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func WithCode(code string, err error) error {
	if err == nil {
		return nil
	}
	return &CodedError{Code: code, Err: err}
}

func ErrorCode(err error) string {
	var coded *CodedError
	if errors.As(err, &coded) && coded.Code != "" {
		return coded.Code
	}
	return ErrorCodeProcessing
}
