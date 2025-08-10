package repository

import (
	"context"
	"dailynews/internal/domain"
	"dailynews/pkg/utils"
	"errors"

	"gorm.io/gorm"
)

type fallbackImageRepository struct {
	db *gorm.DB
}

func NewFallbackImageRepository(db *gorm.DB) domain.FallbackImageRepository {
	return &fallbackImageRepository{db: db}
}

func (r *fallbackImageRepository) Create(ctx context.Context, image *domain.FallbackImage) error {
	if image == nil {
		return errors.New("la imagen no puede ser nil")
	}
	return r.db.WithContext(ctx).Create(image).Error
}

func (r *fallbackImageRepository) GetByCategoryAndLang(ctx context.Context, categoryCode, languageCode string) (*domain.FallbackImage, error) {
	var image domain.FallbackImage
	err := r.db.WithContext(ctx).
		Where("category_code = ? AND language_code = ?", categoryCode, languageCode).
		First(&image).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// No loggear este error ya que es normal cuando no hay imagen de fallback
			return nil, nil
		}
		return nil, err
	}

	return &image, nil
}

func (r *fallbackImageRepository) Update(ctx context.Context, image *domain.FallbackImage) error {
	if image == nil {
		return errors.New("la imagen no puede ser nil")
	}
	return r.db.WithContext(ctx).Save(image).Error
}

func (r *fallbackImageRepository) Delete(ctx context.Context, categoryCode, languageCode string) error {
	return r.db.WithContext(ctx).
		Where("category_code = ? AND language_code = ?", categoryCode, languageCode).
		Delete(&domain.FallbackImage{}).Error
}

func (r *fallbackImageRepository) ListAll(ctx context.Context) ([]domain.FallbackImage, error) {
	var images []domain.FallbackImage
	err := r.db.WithContext(ctx).Find(&images).Error
	return images, err
}

// GetByID obtiene una imagen fallback por su ID
func (r *fallbackImageRepository) GetByID(ctx context.Context, id uint) (*domain.FallbackImage, error) {
	var image domain.FallbackImage
	err := r.db.WithContext(ctx).First(&image, id).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// DeleteByID elimina una imagen fallback por su ID
func (r *fallbackImageRepository) DeleteByID(ctx context.Context, id uint) error {
	if id == 0 {
		return errors.New("el ID de la imagen no puede ser cero")
	}

	err := r.db.WithContext(ctx).Delete(&domain.FallbackImage{}, id).Error
	if err != nil {
		utils.AppError("FALLBACK_IMAGE_DELETE", "Error al eliminar imagen de fallback", err, map[string]interface{}{
			"id": id,
		})
	} else {
		utils.AppInfo("FALLBACK_IMAGE_DELETE", "Imagen de fallback eliminada", map[string]interface{}{
			"id": id,
		})
	}

	return err
}
