package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"dailynews/internal/domain"
)

type categoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository crea una nueva instancia de CategoryRepository
func NewCategoryRepository(db *gorm.DB) domain.CategoryRepository {
	return &categoryRepository{
		db: db,
	}
}

// FindByCode busca una categoría por su código
func (r *categoryRepository) FindByCode(ctx context.Context, code string) (*domain.Category, error) {
	if code == "" {
		return nil, errors.New("el código de categoría no puede estar vacío")
	}

	var category domain.Category
	err := r.db.WithContext(ctx).
		Where("code = ?", code).
		First(&category).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &category, nil
}

// ListAll devuelve todas las categorías disponibles
func (r *categoryRepository) ListAll(ctx context.Context) ([]domain.Category, error) {
	var categories []domain.Category

	err := r.db.WithContext(ctx).
		Order("code ASC").
		Find(&categories).Error

	if err != nil {
		return nil, err
	}

	return categories, nil
}
