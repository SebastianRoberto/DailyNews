package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"dailynews/internal/domain"
)

type countryRepository struct {
	db *gorm.DB
}

// NewCountryRepository crea una nueva instancia de CountryRepository
func NewCountryRepository(db *gorm.DB) domain.CountryRepository {
	return &countryRepository{
		db: db,
	}
}

// FindByCode busca un país por su código
func (r *countryRepository) FindByCode(ctx context.Context, code string) (*domain.Country, error) {
	var country domain.Country
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&country).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &country, nil
}

// ListAll devuelve todos los países
func (r *countryRepository) ListAll(ctx context.Context) ([]domain.Country, error) {
	var countries []domain.Country
	err := r.db.WithContext(ctx).Find(&countries).Error
	if err != nil {
		return nil, err
	}

	return countries, nil
}
