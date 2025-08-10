package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"dailynews/internal/domain"
	"dailynews/pkg/utils"
)

type newsSourceRepository struct {
	db *gorm.DB
}

// NewNewsSourceRepository crea una nueva instancia de NewsSourceRepository
func NewNewsSourceRepository(db *gorm.DB) domain.NewsSourceRepository {
	return &newsSourceRepository{
		db: db,
	}
}

// FindByID busca una fuente de noticias por su ID
func (r *newsSourceRepository) FindByID(ctx context.Context, id uint) (*domain.NewsSource, error) {
	if id == 0 {
		return nil, errors.New("el ID no puede ser cero")
	}

	var source domain.NewsSource
	err := r.db.WithContext(ctx).
		Preload("News").
		Preload("Lang").
		First(&source, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &source, nil
}

// Bbusca fuentes activas por idioma y categoría
func (r *newsSourceRepository) FindActiveByLangAndCategory(
	ctx context.Context,
	langID, categoryID uint,
) ([]domain.NewsSource, error) {
	if langID == 0 || categoryID == 0 {
		return nil, errors.New("tanto el ID de idioma como el de categoría son requeridos")
	}

	var sources []domain.NewsSource

	err := r.db.WithContext(ctx).
		Where("lang_id = ? AND news_id = ? AND is_active = ?", langID, categoryID, true).
		Preload("News").
		Preload("Lang").
		Find(&sources).Error

	if err != nil {
		return nil, err
	}

	return sources, nil
}

// ListActive devuelve todas las fuentes de noticias activas
func (r *newsSourceRepository) ListActive(ctx context.Context) ([]domain.NewsSource, error) {
	var sources []domain.NewsSource

	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Preload("News").
		Preload("Lang").
		Find(&sources).Error

	if err != nil {
		return nil, err
	}

	return sources, nil
}

// ListAll devuelve todas las fuentes de noticias (activas e inactivas)
func (r *newsSourceRepository) ListAll(ctx context.Context) ([]domain.NewsSource, error) {
	var sources []domain.NewsSource

	err := r.db.WithContext(ctx).
		Preload("News").
		Preload("Lang").
		Find(&sources).Error

	if err != nil {
		return nil, err
	}

	return sources, nil
}

// Create crea una nueva fuente de noticias
func (r *newsSourceRepository) Create(ctx context.Context, source *domain.NewsSource) error {
	if source == nil {
		return errors.New("la fuente no puede ser nil")
	}

	return r.db.WithContext(ctx).Create(source).Error
}

// ExistsByURLCategoryLang verifica si ya existe una fuente con la misma URL en la misma categoría e idioma
func (r *newsSourceRepository) ExistsByURLCategoryLang(ctx context.Context, rssURL string, categoryID, langID uint) (bool, error) {
	if rssURL == "" || categoryID == 0 || langID == 0 {
		return false, errors.New("parámetros inválidos para verificación de duplicado")
	}

	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.NewsSource{}).
		Where("rss_url = ? AND news_id = ? AND lang_id = ?", rssURL, categoryID, langID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Update actualiza una fuente de noticias existente
func (r *newsSourceRepository) Update(ctx context.Context, source *domain.NewsSource) error {
	if source == nil {
		return errors.New("la fuente no puede ser nil")
	}

	if source.ID == 0 {
		return errors.New("el ID de la fuente no puede ser cero")
	}

	// Log antes de la actualización
	utils.AppInfo("REPOSITORY_UPDATE", "Actualizando fuente", map[string]interface{}{
		"id":          source.ID,
		"source_name": source.SourceName,
		"is_active":   source.IsActive,
		"user_added":  source.UserAdded,
	})

	err := r.db.WithContext(ctx).Save(source).Error

	if err != nil {
		utils.AppError("REPOSITORY_UPDATE", "Error al actualizar fuente", err, map[string]interface{}{
			"id": source.ID,
		})
	} else {
		utils.AppInfo("REPOSITORY_UPDATE", "Fuente actualizada exitosamente", map[string]interface{}{
			"id": source.ID,
		})
	}

	return err
}

// Delete elimina físicamente una fuente de noticias
func (r *newsSourceRepository) Delete(ctx context.Context, id uint) error {
	utils.AppInfo("REPOSITORY_DELETE", "Iniciando eliminación de fuente", map[string]interface{}{
		"id": id,
	})

	// Primero eliminar las noticias asociadas a esta fuente
	if err := r.db.Where("source_id = ?", id).Delete(&domain.NewsItem{}).Error; err != nil {
		utils.AppError("REPOSITORY_DELETE", "Error al eliminar noticias asociadas", err, map[string]interface{}{
			"id": id,
		})
		return fmt.Errorf("error al eliminar noticias asociadas: %w", err)
	}

	utils.AppInfo("REPOSITORY_DELETE", "Noticias asociadas eliminadas", map[string]interface{}{
		"id": id,
	})

	// Luego eliminar la fuente
	if err := r.db.Delete(&domain.NewsSource{}, id).Error; err != nil {
		utils.AppError("REPOSITORY_DELETE", "Error al eliminar fuente", err, map[string]interface{}{
			"id": id,
		})
		return fmt.Errorf("error al eliminar fuente: %w", err)
	}

	utils.AppInfo("REPOSITORY_DELETE", "Fuente eliminada exitosamente", map[string]interface{}{
		"id": id,
	})

	return nil
}
