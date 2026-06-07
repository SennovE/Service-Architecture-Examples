package cassandra

import (
	"context"
	"fmt"
	"strings"

	"consumer/internal/models"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

type inventoryState struct {
	available int
	reserved  int
}

type zoneKey struct {
	productID string
	zoneID    string
}

type totalState struct {
	available int
	reserved  int
}

type mutationPlan struct {
	zones          map[zoneKey]inventoryState
	totals         map[string]totalState
	zoneSuppliers  map[zoneKey]string
	totalSuppliers map[string]string
	orderStatuses  map[string]string
	orderItems     map[string][]models.OrderItem
}

func (s *Store) Health(ctx context.Context) error {
	if s == nil || s.session == nil {
		return fmt.Errorf("cassandra session is not initialized")
	}
	if err := s.session.Query("SELECT now() FROM system.local").Consistency(gocql.One).ExecContext(ctx); err != nil {
		return models.WithCode(models.ErrorCodeCassandra, fmt.Errorf("cassandra health: %w", err))
	}
	return nil
}

func (s *Store) ApplyWarehouseEvent(
	ctx context.Context, event models.WarehouseEvent, meta models.KafkaMetadata) (models.ApplyResult, error) {
	if err := validateEnvelope(event); err != nil {
		return models.ApplyResult{}, err
	}

	processed, err := s.isProcessed(ctx, event.EventID)
	if err != nil {
		return models.ApplyResult{}, err
	}
	if processed {
		return models.ApplyResult{Duplicate: true}, nil
	}

	aggregateID, err := aggregateID(event)
	if err != nil {
		return models.ApplyResult{}, err
	}
	isStale, err := s.isStale(ctx, aggregateID, event.SequenceNumber)
	if err != nil {
		return models.ApplyResult{}, err
	}
	if isStale {
		return models.ApplyResult{Skipped: true}, nil
	}

	plan, err := s.buildMutationPlan(ctx, event)
	if err != nil {
		return models.ApplyResult{}, err
	}

	batch := s.session.Batch(gocql.LoggedBatch).Consistency(gocql.Quorum)

	for key, state := range plan.zones {
		if supplierID, ok := plan.zoneSuppliers[key]; ok {
			batch.Query(
				`UPDATE inventory_by_product_zone
				 SET available_quantity = ?, reserved_quantity = ?, supplier_id = ?
				 WHERE product_id = ? AND zone_id = ?`,
				state.available, state.reserved, supplierID, key.productID, key.zoneID,
			)
			batch.Query(
				`UPDATE inventory_by_zone
				 SET available_quantity = ?, reserved_quantity = ?, supplier_id = ?
				 WHERE zone_id = ? AND product_id = ?`,
				state.available, state.reserved, supplierID, key.zoneID, key.productID,
			)
			continue
		}

		batch.Query(
			`UPDATE inventory_by_product_zone
			 SET available_quantity = ?, reserved_quantity = ?
			 WHERE product_id = ? AND zone_id = ?`,
			state.available, state.reserved, key.productID, key.zoneID,
		)
		batch.Query(
			`UPDATE inventory_by_zone
			 SET available_quantity = ?, reserved_quantity = ?
			 WHERE zone_id = ? AND product_id = ?`,
			state.available, state.reserved, key.zoneID, key.productID,
		)
	}

	for productID, state := range plan.totals {
		if supplierID, ok := plan.totalSuppliers[productID]; ok {
			batch.Query(
				`UPDATE inventory_totals_by_product
				 SET total_available_quantity = ?, total_reserved_quantity = ?, supplier_id = ?
				 WHERE product_id = ?`,
				state.available, state.reserved, supplierID, productID,
			)
			continue
		}

		batch.Query(
			`UPDATE inventory_totals_by_product
			 SET total_available_quantity = ?, total_reserved_quantity = ?
			 WHERE product_id = ?`,
			state.available, state.reserved, productID,
		)
	}

	for orderID, status := range plan.orderStatuses {
		batch.Query(
			"UPDATE orders_by_id SET status = ? WHERE order_id = ?",
			status, orderID,
		)
	}

	for orderID, items := range plan.orderItems {
		for _, item := range items {
			batch.Query(
				`INSERT INTO order_items_by_order (order_id, product_id, zone_id, quantity)
				 VALUES (?, ?, ?, ?)`,
				orderID, item.ProductID, item.ZoneID, item.Quantity,
			)
		}
	}

	batch.Query("INSERT INTO processed_events (event_id) VALUES (?)", event.EventID)
	batch.Query("UPDATE last_seq_number SET sequence_number = ? WHERE enity_id = ?", event.SequenceNumber, aggregateID)

	if err := batch.ExecContext(ctx); err != nil {
		return models.ApplyResult{}, models.WithCode(
			models.ErrorCodeCassandra,
			fmt.Errorf("apply event_id=%s partition=%d offset=%d: %w", event.EventID, meta.Partition, meta.Offset, err),
		)
	}

	return models.ApplyResult{}, nil
}

func (s *Store) buildMutationPlan(ctx context.Context, event models.WarehouseEvent) (mutationPlan, error) {
	plan := mutationPlan{
		zones:          make(map[zoneKey]inventoryState),
		totals:         make(map[string]totalState),
		zoneSuppliers:  make(map[zoneKey]string),
		totalSuppliers: make(map[string]string),
		orderStatuses:  make(map[string]string),
		orderItems:     make(map[string][]models.OrderItem),
	}

	changeTotal := func(productID string, availableDelta, reservedDelta int) error {
		state, ok := plan.totals[productID]
		if !ok {
			current, err := s.getTotal(ctx, productID)
			if err != nil {
				return err
			}
			state = current
		}

		state.available += availableDelta
		state.reserved += reservedDelta
		if state.available < 0 {
			return validationError("total available quantity for product=%s became negative: %d", productID, state.available)
		}
		if state.reserved < 0 {
			return validationError("total reserved quantity for product=%s became negative: %d", productID, state.reserved)
		}
		plan.totals[productID] = state
		return nil
	}

	changeZone := func(productID, zoneID string, availableDelta, reservedDelta int, supplierID *string) error {
		key := zoneKey{productID: productID, zoneID: zoneID}
		state, ok := plan.zones[key]
		if !ok {
			current, err := s.getInventory(ctx, productID, zoneID)
			if err != nil {
				return err
			}
			state = current
		}

		state.available += availableDelta
		state.reserved += reservedDelta
		if state.available < 0 {
			return validationError("available quantity for product=%s zone=%s became negative: %d", productID, zoneID, state.available)
		}
		if state.reserved < 0 {
			return validationError("reserved quantity for product=%s zone=%s became negative: %d", productID, zoneID, state.reserved)
		}
		plan.zones[key] = state

		if supplierID != nil && *supplierID != "" {
			plan.zoneSuppliers[key] = *supplierID
			plan.totalSuppliers[productID] = *supplierID
		}
		return changeTotal(productID, availableDelta, reservedDelta)
	}

	switch event.EventType {
	case models.EventProductReceived:
		productID, zoneID, quantity, err := productZoneQuantity(event)
		if err != nil {
			return mutationPlan{}, err
		}
		if err := changeZone(productID, zoneID, quantity, 0, event.SupplierID); err != nil {
			return mutationPlan{}, err
		}
	case models.EventProductShipped:
		productID, zoneID, quantity, err := productZoneQuantity(event)
		if err != nil {
			return mutationPlan{}, err
		}
		if err := changeZone(productID, zoneID, -quantity, 0, nil); err != nil {
			return mutationPlan{}, err
		}
	case models.EventProductMoved:
		productID, fromZoneID, toZoneID, quantity, err := moveFields(event)
		if err != nil {
			return mutationPlan{}, err
		}
		if fromZoneID == toZoneID {
			return mutationPlan{}, validationError("from_zone_id and to_zone_id must be different")
		}
		if err := changeZone(productID, fromZoneID, -quantity, 0, nil); err != nil {
			return mutationPlan{}, err
		}
		if err := changeZone(productID, toZoneID, quantity, 0, nil); err != nil {
			return mutationPlan{}, err
		}
	case models.EventProductReserved:
		productID, zoneID, quantity, err := productZoneQuantity(event)
		if err != nil {
			return mutationPlan{}, err
		}
		if err := changeZone(productID, zoneID, -quantity, quantity, nil); err != nil {
			return mutationPlan{}, err
		}
	case models.EventProductReleased:
		productID, zoneID, quantity, err := productZoneQuantity(event)
		if err != nil {
			return mutationPlan{}, err
		}
		if err := changeZone(productID, zoneID, quantity, -quantity, nil); err != nil {
			return mutationPlan{}, err
		}
	case models.EventInventoryCounted:
		productID, zoneID, countedQuantity, err := productZoneQuantity(event)
		if err != nil {
			return mutationPlan{}, err
		}
		key := zoneKey{productID: productID, zoneID: zoneID}
		current, ok := plan.zones[key]
		if !ok {
			current, err = s.getInventory(ctx, productID, zoneID)
			if err != nil {
				return mutationPlan{}, err
			}
		}
		availableDelta := countedQuantity - current.available
		current.available = countedQuantity
		plan.zones[key] = current
		if err := changeTotal(productID, availableDelta, 0); err != nil {
			return mutationPlan{}, err
		}
	case models.EventOrderCreated:
		if event.OrderID == nil || *event.OrderID == "" {
			return mutationPlan{}, validationError("order_id is required for %s", event.EventType)
		}
		if len(event.OrderItems) == 0 {
			return mutationPlan{}, validationError("order_items are required for %s", event.EventType)
		}
		status, err := s.getOrderStatus(ctx, *event.OrderID)
		if err != nil {
			return mutationPlan{}, err
		}
		if status != "" {
			return mutationPlan{}, validationError("order_id %s already exists with status %s", *event.OrderID, status)
		}
		for _, item := range event.OrderItems {
			if err := validateOrderItem(item); err != nil {
				return mutationPlan{}, err
			}
			if err := changeZone(item.ProductID, item.ZoneID, -item.Quantity, item.Quantity, nil); err != nil {
				return mutationPlan{}, err
			}
		}
		plan.orderStatuses[*event.OrderID] = "CREATED"
		plan.orderItems[*event.OrderID] = event.OrderItems
	case models.EventOrderCompleted:
		if event.OrderID == nil || *event.OrderID == "" {
			return mutationPlan{}, validationError("order_id is required for %s", event.EventType)
		}
		status, err := s.getOrderStatus(ctx, *event.OrderID)
		if err != nil {
			return mutationPlan{}, err
		}
		if status == "" {
			return mutationPlan{}, validationError("order_id %s was not created", *event.OrderID)
		}
		if status == "COMPLETED" {
			return mutationPlan{}, validationError("order_id %s is already completed", *event.OrderID)
		}

		items := event.OrderItems
		if len(items) == 0 {
			items, err = s.getOrderItems(ctx, *event.OrderID)
			if err != nil {
				return mutationPlan{}, err
			}
		}
		if len(items) == 0 {
			return mutationPlan{}, validationError("order_items are required for %s", event.EventType)
		}
		for _, item := range items {
			if err := validateOrderItem(item); err != nil {
				return mutationPlan{}, err
			}
			if err := changeZone(item.ProductID, item.ZoneID, 0, -item.Quantity, nil); err != nil {
				return mutationPlan{}, err
			}
		}
		plan.orderStatuses[*event.OrderID] = "COMPLETED"
	default:
		return mutationPlan{}, validationError("unsupported event_type %q", event.EventType)
	}

	return plan, nil
}

func (s *Store) getInventory(ctx context.Context, productID, zoneID string) (inventoryState, error) {
	var state inventoryState
	err := s.session.Query(
		`SELECT available_quantity, reserved_quantity
		 FROM inventory_by_product_zone
		 WHERE product_id = ? AND zone_id = ?`,
		productID, zoneID,
	).Consistency(gocql.Quorum).ScanContext(ctx, &state.available, &state.reserved)
	if err == nil {
		return state, nil
	}
	if err == gocql.ErrNotFound {
		return inventoryState{}, nil
	}
	return inventoryState{}, models.WithCode(models.ErrorCodeCassandra, fmt.Errorf("read inventory product=%s zone=%s: %w", productID, zoneID, err))
}

func (s *Store) getTotal(ctx context.Context, productID string) (totalState, error) {
	var state totalState
	err := s.session.Query(
		`SELECT total_available_quantity, total_reserved_quantity
		 FROM inventory_totals_by_product
		 WHERE product_id = ?`,
		productID,
	).Consistency(gocql.Quorum).ScanContext(ctx, &state.available, &state.reserved)
	if err == nil {
		return state, nil
	}
	if err == gocql.ErrNotFound {
		return totalState{}, nil
	}
	return totalState{}, models.WithCode(models.ErrorCodeCassandra, fmt.Errorf("read inventory total product=%s: %w", productID, err))
}

func (s *Store) isProcessed(ctx context.Context, eventID string) (bool, error) {
	var found string
	err := s.session.Query(
		"SELECT event_id FROM processed_events WHERE event_id = ?",
		eventID,
	).Consistency(gocql.Quorum).ScanContext(ctx, &found)
	if err == nil {
		return true, nil
	}
	if err == gocql.ErrNotFound {
		return false, nil
	}
	return false, models.WithCode(models.ErrorCodeCassandra, fmt.Errorf("check processed event_id=%s: %w", eventID, err))
}

func (s *Store) isStale(ctx context.Context, aggregateID string, sequenceNumber int64) (bool, error) {
	var last int64
	err := s.session.Query(
		"SELECT sequence_number FROM last_seq_number WHERE enity_id = ?",
		aggregateID,
	).Consistency(gocql.Quorum).ScanContext(ctx, &last)
	if err == nil {
		return sequenceNumber <= last, nil
	}
	if err == gocql.ErrNotFound {
		return false, nil
	}
	return false, models.WithCode(models.ErrorCodeCassandra, fmt.Errorf("read last sequence aggregate=%s: %w", aggregateID, err))
}

func (s *Store) getOrderStatus(ctx context.Context, orderID string) (string, error) {
	var status string
	err := s.session.Query(
		"SELECT status FROM orders_by_id WHERE order_id = ?",
		orderID,
	).Consistency(gocql.Quorum).ScanContext(ctx, &status)
	if err == nil {
		return status, nil
	}
	if err == gocql.ErrNotFound {
		return "", nil
	}
	return "", models.WithCode(models.ErrorCodeCassandra, fmt.Errorf("read order order_id=%s: %w", orderID, err))
}

func (s *Store) getOrderItems(ctx context.Context, orderID string) ([]models.OrderItem, error) {
	iter := s.session.Query(
		`SELECT product_id, zone_id, quantity
		 FROM order_items_by_order
		 WHERE order_id = ?`,
		orderID,
	).Consistency(gocql.Quorum).IterContext(ctx)

	var items []models.OrderItem
	var item models.OrderItem
	for iter.Scan(&item.ProductID, &item.ZoneID, &item.Quantity) {
		items = append(items, item)
		item = models.OrderItem{}
	}
	if err := iter.Close(); err != nil {
		return nil, models.WithCode(models.ErrorCodeCassandra, fmt.Errorf("read order items order_id=%s: %w", orderID, err))
	}
	return items, nil
}

func validateEnvelope(event models.WarehouseEvent) error {
	if strings.TrimSpace(event.EventID) == "" {
		return validationError("event_id is required")
	}
	if strings.TrimSpace(event.EventType) == "" {
		return validationError("event_type is required")
	}
	if event.SchemaVersion != 1 && event.SchemaVersion != 2 {
		return validationError("unsupported schema_version %d", event.SchemaVersion)
	}
	if event.SequenceNumber < 0 {
		return validationError("sequence_number must be non-negative")
	}
	return nil
}

func aggregateID(event models.WarehouseEvent) (string, error) {
	switch event.EventType {
	case models.EventOrderCreated, models.EventOrderCompleted:
		if event.OrderID == nil || strings.TrimSpace(*event.OrderID) == "" {
			return "", validationError("order_id is required for %s", event.EventType)
		}
		return "order:" + *event.OrderID, nil
	default:
		if event.ProductID == nil || strings.TrimSpace(*event.ProductID) == "" {
			return "", validationError("product_id is required for %s", event.EventType)
		}
		return "product:" + *event.ProductID, nil
	}
}

func productZoneQuantity(event models.WarehouseEvent) (string, string, int, error) {
	if event.ProductID == nil || *event.ProductID == "" {
		return "", "", 0, validationError("product_id is required for %s", event.EventType)
	}
	if event.ZoneID == nil || *event.ZoneID == "" {
		return "", "", 0, validationError("zone_id is required for %s", event.EventType)
	}
	if event.Quantity == nil {
		return "", "", 0, validationError("quantity is required for %s", event.EventType)
	}
	if *event.Quantity <= 0 {
		return "", "", 0, validationError("quantity must be positive for %s: %d", event.EventType, *event.Quantity)
	}
	return *event.ProductID, *event.ZoneID, *event.Quantity, nil
}

func moveFields(event models.WarehouseEvent) (string, string, string, int, error) {
	if event.ProductID == nil || *event.ProductID == "" {
		return "", "", "", 0, validationError("product_id is required for %s", event.EventType)
	}
	if event.FromZoneID == nil || *event.FromZoneID == "" {
		return "", "", "", 0, validationError("from_zone_id is required for %s", event.EventType)
	}
	if event.ToZoneID == nil || *event.ToZoneID == "" {
		return "", "", "", 0, validationError("to_zone_id is required for %s", event.EventType)
	}
	if event.Quantity == nil {
		return "", "", "", 0, validationError("quantity is required for %s", event.EventType)
	}
	if *event.Quantity <= 0 {
		return "", "", "", 0, validationError("quantity must be positive for %s: %d", event.EventType, *event.Quantity)
	}
	return *event.ProductID, *event.FromZoneID, *event.ToZoneID, *event.Quantity, nil
}

func validateOrderItem(item models.OrderItem) error {
	if item.ProductID == "" {
		return validationError("order item product_id is required")
	}
	if item.ZoneID == "" {
		return validationError("order item zone_id is required")
	}
	if item.Quantity <= 0 {
		return validationError("order item quantity must be positive: %d", item.Quantity)
	}
	return nil
}

func validationError(format string, args ...any) error {
	return models.WithCode(models.ErrorCodeValidation, fmt.Errorf(format, args...))
}
