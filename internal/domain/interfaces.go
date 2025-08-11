package domain

import (
	"context"
	"time"
)

// CountryRepository define las operaciones para el repositorio de países/idiomas
type CountryRepository interface {
	FindByCode(ctx context.Context, code string) (*Country, error)
	ListAll(ctx context.Context) ([]Country, error)
}

// CategoryRepository define las operaciones para el repositorio de categorías
type CategoryRepository interface {
	FindByCode(ctx context.Context, code string) (*Category, error)
	ListAll(ctx context.Context) ([]Category, error)
}

// NewsSourceRepository define las operaciones para el repositorio de fuentes de noticias
type NewsSourceRepository interface {
	FindByID(ctx context.Context, id uint) (*NewsSource, error)
	FindActiveByLangAndCategory(ctx context.Context, langID, categoryID uint) ([]NewsSource, error)
	ListActive(ctx context.Context) ([]NewsSource, error)
	ListAll(ctx context.Context) ([]NewsSource, error)
	Create(ctx context.Context, source *NewsSource) error
	Update(ctx context.Context, source *NewsSource) error
	Delete(ctx context.Context, id uint) error // NUEVO
	// Verificación de duplicados por URL + categoría + idioma
	ExistsByURLCategoryLang(ctx context.Context, rssURL string, categoryID, langID uint) (bool, error)
}

// FallbackImageRepository define las operaciones para el repositorio de imágenes de fallback
type FallbackImageRepository interface {
	Create(ctx context.Context, image *FallbackImage) error
	GetByCategoryAndLang(ctx context.Context, categoryCode, languageCode string) (*FallbackImage, error)
	GetByID(ctx context.Context, id uint) (*FallbackImage, error) // NUEVO
	Update(ctx context.Context, image *FallbackImage) error
	Delete(ctx context.Context, categoryCode, languageCode string) error
	DeleteByID(ctx context.Context, id uint) error // NUEVO
	ListAll(ctx context.Context) ([]FallbackImage, error)
}

// NewsItemRepository define las operaciones para el repositorio de noticias
type NewsItemRepository interface {
	Create(ctx context.Context, item *NewsItem) error
	BatchCreate(ctx context.Context, items []NewsItem) error
	FindByID(ctx context.Context, id uint) (*NewsItem, error)
	FindBySourceID(ctx context.Context, sourceID uint) ([]NewsItem, error)
	FindByLangAndCategory(ctx context.Context, langCode, categoryCode string, limit int) ([]NewsItem, error)
	DeleteOlderThan(ctx context.Context, date time.Time) error

	// Métodos para el frontend
	GetLatest(ctx context.Context, lang string, limit, offset int) ([]NewsItem, error)
	GetByCategory(ctx context.Context, category, lang string, limit, offset int) ([]NewsItem, error)
	SearchByTitle(ctx context.Context, query, lang, category string, limit, offset int) ([]NewsItem, error)
	CountTotal(ctx context.Context, lang string) (int, error)
	CountByCategory(ctx context.Context, category, lang string) (int, error)
	CountSearchResults(ctx context.Context, query, lang, category string) (int, error)

	// Nuevos métodos para filtros avanzados
	GetFilteredNews(ctx context.Context, filters NewsFilters, limit, offset int) ([]NewsItem, error)
	CountFilteredNews(ctx context.Context, filters NewsFilters) (int, error)
}

// RSSFetcher define el contrato para obtener noticias desde fuentes RSS
type RSSFetcher interface {
	Fetch(ctx context.Context, url string, filter string, titleField, imageField, linkField, dateField string) ([]NewsItem, error)
}

// ImageDownloader define el contrato para descargar y validar imágenes
type ImageDownloader interface {
	DownloadAndValidate(ctx context.Context, url, savePath string) (string, error)
	ValidateImage(path string) (bool, error)
}

// Logger define el contrato para el sistema de logging
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// UseCase define el contrato para los casos de uso de la aplicación
type UseCase interface {
	FetchAndProcessNews(ctx context.Context) error
	GetNewsByLangAndCategory(ctx context.Context, lang, category string, limit int) ([]NewsItemDTO, error)
}

type UnitOfWork interface {
	Begin(ctx context.Context) (UnitOfWork, error)
	Commit() error
	Rollback() error
	Countries() CountryRepository
	Categories() CategoryRepository
	NewsSources() NewsSourceRepository
	NewsItems() NewsItemRepository
}

// NewsFilters define los filtros avanzados para noticias
type NewsFilters struct {
	Lang              string     `json:"lang"`
	Category          string     `json:"category"`
	Sources           []string   `json:"sources"`            // Múltiples fuentes
	ExcludeCategories []string   `json:"exclude_categories"` // Categorías a excluir
	DateFrom          *time.Time `json:"date_from"`          // Fecha desde
	DateTo            *time.Time `json:"date_to"`            // Fecha hasta
	Search            string     `json:"search"`             // Búsqueda en título
}
