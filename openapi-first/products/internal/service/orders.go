package service

import (
	"context"
	"errors"
	"products/internal/config"
	"products/internal/gen"
	"products/internal/repository"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrdersService struct {
	db  *repository.OrdersRepository
	cfg *config.Config
}

func NewOrdersService(db *repository.OrdersRepository, cfg *config.Config) *OrdersService {
	return &OrdersService{db: db, cfg: cfg}
}

type OrderService struct {
	Id             uuid.UUID
	CreatedAt      *time.Time
	UpdatedAt      *time.Time
	DiscountAmount decimal.Decimal
	Items          []OrderItemService
	PromoCodeId    *uuid.UUID
	Status         gen.OrderStatus
	TotalAmount    decimal.Decimal
	UserID         uuid.UUID
}

type OrderItemToCreateService struct {
	ProductId uuid.UUID
	Quantity  int
}

type OrderItemInfoService struct {
	PriceAtOrder decimal.Decimal
	OrderItemToCreateService
}

type OrderItemService struct {
	Id *uuid.UUID
	OrderItemInfoService
}

type OrderToCreateService struct {
	Items     []OrderItemToCreateService
	PromoCode *string
}

func (srvc *OrdersService) GetOrderById(
	ctx context.Context, id uuid.UUID, userID uuid.UUID, role string) (*OrderService, error) {
	order, err := srvc.db.GetOrderById(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	if role != "ADMIN" && order.UserID != userID {
		return nil, ErrNoPermission
	}
	items, err := srvc.db.GetOrderItemsByOrderId(ctx, order.Id)
	if err != nil {
		return nil, err
	}
	return mapOrderRepositoryToService(order, items), nil
}

func (srvc *OrdersService) CreateOrder(
	ctx context.Context, values OrderToCreateService, userID uuid.UUID) (*OrderService, error) {
	tx, err := srvc.db.BeginTransaction()
	if err != nil {
		return nil, err
	}
	err = checkOrderOperationLimit(ctx, tx, userID, srvc.cfg.OrderTimeout, "CREATE_ORDER")
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	err = checkForActiveOrder(ctx, tx, userID)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	orderProducts, err := checkAndGetProductsInOrder(ctx, tx, &values.Items)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	err = reserveProducts(ctx, tx, &values.Items)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	totalAmount := getTotalAmount(orderProducts)
	promoId, discount, err := checkAndUsePromoCode(ctx, tx, values.PromoCode, totalAmount)
	totalAmount = totalAmount.Sub(discount)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	order, err := tx.CreateOrder(ctx, userID, promoId, totalAmount, discount)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	itemsToInsert := make([]repository.OrderItemToCreateRepository, len(*orderProducts))
	for i, item := range *orderProducts {
		itemsToInsert[i] = repository.OrderItemToCreateRepository{
			PriceAtOrder: item.PriceAtOrder,
			ProductId:    item.ProductId,
			Quantity:     item.Quantity,
		}
	}
	orderItems, err := tx.CreateOrderItems(ctx, order.Id, &itemsToInsert)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = tx.CreateUserOperation(ctx, userID, "CREATE_ORDER")
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return mapOrderRepositoryToService(order, orderItems), nil
}

func (srvc *OrdersService) CancelOrder(ctx context.Context, id uuid.UUID, userID uuid.UUID, role string) error {
	tx, err := srvc.db.BeginTransaction()
	order, err := tx.GetForUpdateOrderById(ctx, id)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, repository.ErrOrderNotFound) {
			return ErrOrderNotFound
		}
		return err
	}
	if role != "ADMIN" && order.UserID != userID {
		tx.Rollback()
		return ErrNoPermission
	}
	if order.Status != gen.CREATED && order.Status != gen.PAYMENTPENDING {
		tx.Rollback()
		return ErrInvalidStateTransaction
	}
	err = tx.ReturnProductsFromOrder(ctx, order.Id)
	if err != nil {
		tx.Rollback()
		return err
	}
	if order.PromoCodeId != nil {
		err = tx.DecrementPromoUsers(ctx, *order.PromoCodeId)
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	err = tx.UpdateOrderStatus(ctx, id, gen.CANCELED)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (srvc *OrdersService) UpdateOrderItems(
	ctx context.Context, id uuid.UUID, items *[]OrderItemToCreateService, userID uuid.UUID, role string) (*OrderService, error) {
	tx, err := srvc.db.BeginTransaction()
	order, err := tx.GetForUpdateOrderById(ctx, id)
	if err != nil {
		tx.Rollback()
		if errors.Is(err, repository.ErrOrderNotFound) {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	if role != "ADMIN" && order.UserID != userID {
		tx.Rollback()
		return nil, ErrNoPermission
	}
	if order.Status != gen.CREATED {
		tx.Rollback()
		return nil, ErrInvalidStateTransaction
	}
	err = checkOrderOperationLimit(ctx, tx, userID, srvc.cfg.OrderTimeout, "UPDATE_ORDER")
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	err = tx.ReturnProductsFromOrder(ctx, order.Id)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	err = tx.RemoveOrderItems(ctx, order.Id)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	orderProducts, err := checkAndGetProductsInOrder(ctx, tx, items)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	err = reserveProducts(ctx, tx, items)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	totalAmount := getTotalAmount(orderProducts)
	var promoCode *string = nil
	if order.PromoCodeId != nil {
		err = tx.DecrementPromoUsers(ctx, *order.PromoCodeId)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		promo, err := tx.GetForUpdatePromoCodeById(ctx, *order.PromoCodeId)
		promoCode = &promo.Code
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	promoId, discount, err := checkAndUsePromoCode(ctx, tx, promoCode, totalAmount)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	totalAmount = totalAmount.Sub(discount)
	order, err = tx.UpdateOrderById(ctx, id, promoId, totalAmount, discount)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	itemsToInsert := make([]repository.OrderItemToCreateRepository, len(*orderProducts))
	for i, item := range *orderProducts {
		itemsToInsert[i] = repository.OrderItemToCreateRepository{
			PriceAtOrder: item.PriceAtOrder,
			ProductId:    item.ProductId,
			Quantity:     item.Quantity,
		}
	}
	orderItems, err := tx.CreateOrderItems(ctx, order.Id, &itemsToInsert)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	err = tx.CreateUserOperation(ctx, userID, "UPDATE_ORDER")
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	return mapOrderRepositoryToService(order, orderItems), nil
}

func checkForActiveOrder(ctx context.Context, tx *repository.OrdersTransaction, userID uuid.UUID) error {
	activeOrderId, err := tx.GetUserActiveOrder(ctx, userID)
	if err != nil {
		return err
	}
	if activeOrderId != nil {
		return &ErrHasActiveOrdeer{*activeOrderId}
	}
	return nil
}

func checkAndGetProductsInOrder(
	ctx context.Context, tx *repository.OrdersTransaction, items *[]OrderItemToCreateService,
) (*[]OrderItemInfoService, error) {
	ids := make([]uuid.UUID, len(*items))
	notFound := make(map[uuid.UUID]struct{})
	quantityById := make(map[uuid.UUID]int)
	for i, item := range *items {
		ids[i] = item.ProductId
		notFound[item.ProductId] = struct{}{}
		quantityById[item.ProductId] = item.Quantity
	}
	products, err := tx.GetForUpdateProductsByIds(ctx, ids)
	if err != nil {
		return nil, err
	}
	notActive := []uuid.UUID(nil)
	notEnough := []uuid.UUID(nil)
	priceById := []OrderItemInfoService(nil)
	for _, product := range *products {
		if product.Status != "ACTIVE" {
			notActive = append(notActive, product.Id)
		}
		if quantityById[product.Id] > product.Stock {
			notEnough = append(notEnough, product.Id)
		}
		priceById = append(priceById, OrderItemInfoService{
			PriceAtOrder: product.Price,
			OrderItemToCreateService: OrderItemToCreateService{
				Quantity:  quantityById[product.Id],
				ProductId: product.Id,
			},
		})
		delete(notFound, product.Id)
	}
	if len(notActive) != 0 {
		return nil, &ErrProductNotActive{IDs: notActive}
	}
	if len(notFound) != 0 {
		tmp := make([]uuid.UUID, 0, len(notFound))
		for k := range notFound {
			tmp = append(tmp, k)
		}
		return nil, &ErrProductNotFound{IDs: tmp}
	}
	if len(notEnough) != 0 {
		return nil, &ErrProductNotEnough{IDs: notEnough}
	}
	return &priceById, nil
}

func checkOrderOperationLimit(
	ctx context.Context, tx *repository.OrdersTransaction, userID uuid.UUID, timeLimit time.Duration, operationType string) error {
	lastTime, err := tx.GetLastOperationTime(ctx, userID, operationType)
	if err != nil {
		return err
	}
	if lastTime != nil && lastTime.Add(timeLimit).After(time.Now()) {
		return &ErrOrderLimit{*lastTime}
	}
	return nil
}

func reserveProducts(
	ctx context.Context, tx *repository.OrdersTransaction, items *[]OrderItemToCreateService) error {
	productsToBuy := make([]repository.ProductStockToBuyRepository, len(*items))
	for i, item := range *items {
		productsToBuy[i] = repository.ProductStockToBuyRepository{
			Id:    item.ProductId,
			Stock: item.Quantity,
		}
	}
	return tx.UpdateStockByIds(ctx, productsToBuy)
}

func getTotalAmount(orderProducts *[]OrderItemInfoService) decimal.Decimal {
	totalAmount := decimal.Zero
	for _, product := range *orderProducts {
		totalAmount = totalAmount.Add(
			decimal.NewFromInt(int64(product.Quantity)).Mul(product.PriceAtOrder))
	}
	return totalAmount
}

func getDiscount(totalAmount, discountValue decimal.Decimal, discountType gen.DiscountType) decimal.Decimal {
	switch discountType {
	case gen.FIXEDAMOUNT:
		return decimal.Min(totalAmount, discountValue)
	case gen.PERCENTAGE:
		disc := totalAmount.Mul(discountValue).Div(decimal.NewFromInt(100))
		max := totalAmount.Mul(decimal.NewFromFloat(0.70))
		return decimal.Min(disc, max)
	}
	return decimal.Zero
}

func checkAndGetPromoCode(
	ctx context.Context, tx *repository.OrdersTransaction, promoCode string, totalAmount decimal.Decimal,
) (*repository.PromoCodeRepository, error) {
	promo, err := tx.GetPromoCodeByCode(ctx, promoCode)
	if err != nil {
		if errors.Is(err, repository.ErrPromoCodeNotFound) {
			return nil, ErrPromoCodeInvalid
		}
		return nil, err
	}
	currTime := time.Now()
	if !(promo.Active && promo.CurrentUses < promo.MaxUses && currTime.Before(promo.ValidUntil) && currTime.After(promo.ValidFrom)) {
		return nil, ErrPromoCodeInvalid
	}
	if !(totalAmount.GreaterThanOrEqual(promo.MinOrderAmount)) {
		return nil, ErrPromoCodeMinAmount
	}
	return promo, nil
}

func checkAndUsePromoCode(
	ctx context.Context, tx *repository.OrdersTransaction, promoCode *string, totalAmount decimal.Decimal,
) (*uuid.UUID, decimal.Decimal, error) {
	var promoId *uuid.UUID = nil
	discount := decimal.Zero
	if promoCode != nil {
		promo, err := checkAndGetPromoCode(ctx, tx, *promoCode, totalAmount)
		if err != nil {
			return nil, decimal.Zero, err
		}
		promoId = &promo.Id
		discount = getDiscount(totalAmount, promo.DiscountValue, promo.DiscountType)
		err = tx.IncrementPromoUsers(ctx, promo.Code)
		if err != nil {
			return nil, decimal.Zero, err
		}
	}
	return promoId, discount, nil
}

func mapOrderRepositoryToService(
	order *repository.OrderRepository, items *[]repository.OrderItemRepository) *OrderService {
	mappedItems := make([]OrderItemService, len(*items))
	for i, item := range *items {
		mappedItems[i] = OrderItemService{
			Id: &item.Id,
			OrderItemInfoService: OrderItemInfoService{
				PriceAtOrder: item.PriceAtOrder,
				OrderItemToCreateService: OrderItemToCreateService{
					ProductId: item.ProductId,
					Quantity:  item.Quantity,
				},
			},
		}
	}
	return &OrderService{
		CreatedAt:      order.CreatedAt,
		DiscountAmount: order.DiscountAmount,
		Id:             order.Id,
		PromoCodeId:    order.PromoCodeId,
		Status:         order.Status,
		TotalAmount:    order.TotalAmount,
		UpdatedAt:      order.UpdatedAt,
		UserID:         order.UserID,
		Items:          mappedItems,
	}
}
