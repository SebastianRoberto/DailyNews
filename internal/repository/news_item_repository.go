package repository

import (
	"context"
	"errors"
	"time"

	"dailynews/internal/domain"

	"gorm.io/gorm"
)

type newsItemRepository struct {
	db *gorm.DB
}

// NewNewsItemRepository crea una nueva instancia de NewsItemRepository
func NewNewsItemRepository(db *gorm.DB) domain.NewsItemRepository {
	return &newsItemRepository{
		db: db,
	}
}

// Create guarda una nueva noticia en la base de datos
func (r *newsItemRepository) Create(ctx context.Context, item *domain.NewsItem) error {
	if item == nil {
		return errors.New("el item de noticia no puede ser nulo")
	}

	// Validar campos requeridos
	if item.Title == "" || item.Link == "" || item.Image == "" || item.LangCode == "" || item.CategoryCode == "" {
		return errors.New("faltan campos requeridos en el item de noticia")
	}

	// Establecer la fecha de creación si no está definida
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}

	// Usar el contexto proporcionado
	db := r.db.WithContext(ctx)

	// Crear el registro en la base de datos
	if err := db.Create(item).Error; err != nil {
		return err
	}

	return nil
}

// BatchCreate guarda múltiples noticias en la base de datos en una sola transacción
func (r *newsItemRepository) BatchCreate(ctx context.Context, items []domain.NewsItem) error {
	if len(items) == 0 {
		return nil
	}

	// Validar items
	for i, item := range items {
		if item.Title == "" || item.Link == "" || item.Image == "" || item.LangCode == "" || item.CategoryCode == "" {
			return errors.New("faltan campos requeridos en uno o más items de noticia")
		}
		if item.CreatedAt.IsZero() {
			items[i].CreatedAt = time.Now()
		}
	}

	// Usar una transacción para garantizar la integridad de los datos
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.CreateInBatches(items, 100).Error // Procesar en lotes de 100
	})

	return err
}

// FindByID busca una noticia por su ID
func (r *newsItemRepository) FindByID(ctx context.Context, id uint) (*domain.NewsItem, error) {
	if id == 0 {
		return nil, errors.New("el ID no puede ser cero")
	}

	var item domain.NewsItem
	err := r.db.WithContext(ctx).
		Preload("Source").
		Preload("Source.News").
		Preload("Source.Lang").
		First(&item, id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &item, nil
}

// FindBySourceID busca noticias por el ID de la fuente
func (r *newsItemRepository) FindBySourceID(ctx context.Context, sourceID uint) ([]domain.NewsItem, error) {
	if sourceID == 0 {
		return nil, errors.New("el ID de la fuente no puede ser cero")
	}

	var items []domain.NewsItem
	err := r.db.WithContext(ctx).
		Where("source_id = ?", sourceID).
		Order("pub_date DESC").
		Find(&items).Error

	if err != nil {
		return nil, err
	}

	return items, nil
}

// FindByLangAndCategory busca noticias por idioma y categoría con un límite
func (r *newsItemRepository) FindByLangAndCategory(
	ctx context.Context,
	langCode, categoryCode string,
	limit int,
) ([]domain.NewsItem, error) {
	if langCode == "" || categoryCode == "" {
		return nil, errors.New("tanto el código de idioma como el de categoría son requeridos")
	}

	// Validar y ajustar el límite
	if limit <= 0 {
		limit = 10 // Valor por defecto
	} else if limit > 100 {
		limit = 100 // Límite máximo para evitar sobrecarga
	}

	var items []domain.NewsItem

	query := r.db.WithContext(ctx).
		Where("lang_code = ? AND category_code = ?", langCode, categoryCode).
		Order("pub_date DESC").
		Limit(limit)

	// Si es necesario cargar relaciones
	if r.db.Statement.Preloads != nil {
		query = query.Preload("Source").Preload("Source.News").Preload("Source.Lang")
	}

	err := query.Find(&items).Error
	if err != nil {
		return nil, err
	}

	return items, nil
}

// DeleteOlderThan elimina noticias más antiguas que la fecha especificada
func (r *newsItemRepository) DeleteOlderThan(ctx context.Context, date time.Time) error {
	if date.IsZero() {
		return errors.New("la fecha no puede ser cero")
	}

	result := r.db.WithContext(ctx).
		Where("created_at < ?", date).
		Delete(&domain.NewsItem{})

	if result.Error != nil {
		return result.Error
	}

	return nil
}

// ===== MÉTODOS PARA EL FRONTEND (reutilizando lógica existente) =====

// GetLatest obtiene las noticias más recientes para un idioma con paginación
func (r *newsItemRepository) GetLatest(ctx context.Context, lang string, limit, offset int) ([]domain.NewsItem, error) {
	if lang == "" {
		return nil, errors.New("el código de idioma es requerido")
	}

	// Validar y ajustar parámetros
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var items []domain.NewsItem
	err := r.db.WithContext(ctx).
		Where("lang_code = ?", lang).
		Preload("Source").
		Order("pub_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error

	if err != nil {
		return nil, err
	}

	return items, nil
}

// GetByCategory obtiene noticias por categoría e idioma con paginación
func (r *newsItemRepository) GetByCategory(ctx context.Context, category, lang string, limit, offset int) ([]domain.NewsItem, error) {
	if category == "" || lang == "" {
		return nil, errors.New("tanto el código de categoría como el de idioma son requeridos")
	}

	// Validar y ajustar parámetros
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	var items []domain.NewsItem
	err := r.db.WithContext(ctx).
		Where("category_code = ? AND lang_code = ?", category, lang).
		Preload("Source").
		Order("pub_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error

	if err != nil {
		return nil, err
	}

	return items, nil
}

// SearchByTitle busca noticias por título con filtros opcionales
func (r *newsItemRepository) SearchByTitle(ctx context.Context, query, lang, category string, limit, offset int) ([]domain.NewsItem, error) {
	if query == "" {
		return nil, errors.New("el término de búsqueda es requerido")
	}

	// Validar y ajustar parámetros
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Construir query base
	dbQuery := r.db.WithContext(ctx).
		Where("title LIKE ?", "%"+query+"%").
		Preload("Source")

	// Aplicar filtros opcionales
	if lang != "" {
		dbQuery = dbQuery.Where("lang_code = ?", lang)
	}
	if category != "" {
		dbQuery = dbQuery.Where("category_code = ?", category)
	}

	var items []domain.NewsItem
	err := dbQuery.
		Order("pub_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error

	if err != nil {
		return nil, err
	}

	return items, nil
}

// CountTotal cuenta el total de noticias para un idioma
func (r *newsItemRepository) CountTotal(ctx context.Context, lang string) (int, error) {
	if lang == "" {
		return 0, errors.New("el código de idioma es requerido")
	}

	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.NewsItem{}).
		Where("lang_code = ?", lang).
		Count(&count).Error

	return int(count), err
}

// CountByCategory cuenta noticias por categoría e idioma
func (r *newsItemRepository) CountByCategory(ctx context.Context, category, lang string) (int, error) {
	if category == "" || lang == "" {
		return 0, errors.New("tanto el código de categoría como el de idioma son requeridos")
	}

	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.NewsItem{}).
		Where("category_code = ? AND lang_code = ?", category, lang).
		Count(&count).Error

	return int(count), err
}

// CountSearchResults cuenta los resultados de búsqueda
func (r *newsItemRepository) CountSearchResults(ctx context.Context, query, lang, category string) (int, error) {
	if query == "" {
		return 0, errors.New("el término de búsqueda es requerido")
	}

	// Construir query base
	dbQuery := r.db.WithContext(ctx).
		Model(&domain.NewsItem{}).
		Where("title LIKE ?", "%"+query+"%")

	// Aplicar filtros opcionales
	if lang != "" {
		dbQuery = dbQuery.Where("lang_code = ?", lang)
	}
	if category != "" {
		dbQuery = dbQuery.Where("category_code = ?", category)
	}

	var count int64
	err := dbQuery.Count(&count).Error

	return int(count), err
}

// ===== NUEVOS MÉTODOS PARA FILTROS AVANZADOS =====

// GetFilteredNews obtiene noticias con filtros avanzados
func (r *newsItemRepository) GetFilteredNews(ctx context.Context, filters domain.NewsFilters, limit, offset int) ([]domain.NewsItem, error) {
	// Validar y ajustar parámetros
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	// Construir query base
	dbQuery := r.db.WithContext(ctx).
		Preload("Source").
		Preload("Source.News").
		Preload("Source.Lang")

	// Aplicar filtros
	if filters.Lang != "" {
		dbQuery = dbQuery.Where("lang_code = ?", filters.Lang)
	}
	if filters.Category != "" {
		dbQuery = dbQuery.Where("category_code = ?", filters.Category)
	}
	if len(filters.Sources) > 0 {
		// Usar subquery para filtrar por fuentes
		subQuery := r.db.Table("template_news_sources").
			Select("id").
			Where("source_name IN ?", filters.Sources)
		dbQuery = dbQuery.Where("source_id IN (?)", subQuery)
	}
	if filters.DateFrom != nil {
		dbQuery = dbQuery.Where("pub_date >= ?", *filters.DateFrom)
	}
	if filters.DateTo != nil {
		dbQuery = dbQuery.Where("pub_date <= ?", *filters.DateTo)
	}
	if filters.Search != "" {
		dbQuery = dbQuery.Where("title LIKE ?", "%"+filters.Search+"%")
	}
	if len(filters.ExcludeCategories) > 0 {
		dbQuery = dbQuery.Where("category_code NOT IN ?", filters.ExcludeCategories)
	}

	var items []domain.NewsItem
	err := dbQuery.
		Order("pub_date DESC").
		Limit(limit).
		Offset(offset).
		Find(&items).Error

	if err != nil {
		return nil, err
	}

	return items, nil
}

// CountFilteredNews cuenta noticias con filtros avanzados
func (r *newsItemRepository) CountFilteredNews(ctx context.Context, filters domain.NewsFilters) (int, error) {
	// Construir query base
	dbQuery := r.db.WithContext(ctx).
		Model(&domain.NewsItem{})

	// Aplicar filtros
	if filters.Lang != "" {
		dbQuery = dbQuery.Where("lang_code = ?", filters.Lang)
	}
	if filters.Category != "" {
		dbQuery = dbQuery.Where("category_code = ?", filters.Category)
	}
	if len(filters.Sources) > 0 {
		// Usar subquery para filtrar por fuentes
		subQuery := r.db.Table("template_news_sources").
			Select("id").
			Where("source_name IN ?", filters.Sources)
		dbQuery = dbQuery.Where("source_id IN (?)", subQuery)
	}
	if filters.DateFrom != nil {
		dbQuery = dbQuery.Where("pub_date >= ?", *filters.DateFrom)
	}
	if filters.DateTo != nil {
		dbQuery = dbQuery.Where("pub_date <= ?", *filters.DateTo)
	}
	if filters.Search != "" {
		dbQuery = dbQuery.Where("title LIKE ?", "%"+filters.Search+"%")
	}
	if len(filters.ExcludeCategories) > 0 {
		dbQuery = dbQuery.Where("category_code NOT IN ?", filters.ExcludeCategories)
	}

	var count int64
	err := dbQuery.Count(&count).Error

	return int(count), err
}
