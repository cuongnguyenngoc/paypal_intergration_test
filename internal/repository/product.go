package repository

import (
	"paypal-integration-demo/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProductRepository interface {
	Seed() error
	FindByID(productID string) (*model.Product, error)
	FindMany(productIDs []string) ([]*model.Product, error)
	GetByType(productType model.ProductType) ([]*model.Product, error)
}

type productRepoImpl struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepoImpl{
		db: db,
	}
}

func (r *productRepoImpl) Seed() error {
	products := []model.Product{
		{ID: "coin_100", Name: "100 Coins", Description: "100 Coins for buying stuff", Price: 100, Currency: "USD", Type: "ONE_TIME"},
		{ID: "coin_200", Name: "200 Coins", Description: "200 Coins for buying stuff", Price: 200, Currency: "USD", Type: "ONE_TIME"},
		{ID: "vip_monthly", Name: "Vip product monthly", Description: "Susbcribe this to earn stuff every month", Price: 999, Currency: "USD", Type: "SUBSCRIPTION"},
	}

	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&products).Error
}

func (r *productRepoImpl) FindByID(productID string) (*model.Product, error) {
	var product model.Product
	err := r.db.
		Where("id = ?", productID).
		First(&product).Error

	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (r *productRepoImpl) FindMany(productIDs []string) ([]*model.Product, error) {
	var products []*model.Product
	err := r.db.
		Where("id IN ?", productIDs).
		Find(&products).
		Error

	if err != nil {
		return nil, err
	}

	return products, nil
}

func (r *productRepoImpl) GetByType(productType model.ProductType) ([]*model.Product, error) {
	var products []*model.Product
	err := r.db.
		Where("type = ?", productType).
		Find(&products).
		Error

	if err != nil {
		return nil, err
	}

	return products, nil
}
