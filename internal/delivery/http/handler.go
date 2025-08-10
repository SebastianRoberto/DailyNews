package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"dailynews/internal/domain"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	FetchUseCase          func(ctx context.Context) error
	FetchUseCaseForSource func(ctx context.Context, sourceID uint) error
	NewsRepo              domain.NewsItemRepository
	CategoryRepo          domain.CategoryRepository
	CountryRepo           domain.CountryRepository
	SourceRepo            domain.NewsSourceRepository
	FallbackImageRepo     domain.FallbackImageRepository // NUEVO
	RSSFetcher            domain.RSSFetcher
}

func NewHandler(fetchUseCase func(ctx context.Context) error,
	fetchUseCaseForSource func(ctx context.Context, sourceID uint) error,
	newsRepo domain.NewsItemRepository, categoryRepo domain.CategoryRepository,
	countryRepo domain.CountryRepository, sourceRepo domain.NewsSourceRepository,
	fallbackImageRepo domain.FallbackImageRepository, rssFetcher domain.RSSFetcher) *Handler {
	return &Handler{
		FetchUseCase:          fetchUseCase,
		FetchUseCaseForSource: fetchUseCaseForSource,
		NewsRepo:              newsRepo,
		CategoryRepo:          categoryRepo,
		CountryRepo:           countryRepo,
		SourceRepo:            sourceRepo,
		FallbackImageRepo:     fallbackImageRepo, // NUEVO
		RSSFetcher:            rssFetcher,
	}
}

// GET /api/news/:lang/:category
func (h *Handler) GetNewsHandler(c *gin.Context) {
	lang := c.Param("lang")
	category := c.Param("category")

	// Parámetros de consulta opcionales
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")
	sourceFilter := c.Query("source")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Obtener noticias desde la BD
	ctx := c.Request.Context()
	newsItems, err := h.NewsRepo.FindByLangAndCategory(ctx, lang, category, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error consultando noticias"})
		return
	}

	// Convertir a formato JSON simplificado para el frontend
	var response []map[string]interface{}
	for _, item := range newsItems {
		// Aplicar filtro de fuente si se especifica
		if sourceFilter != "" && item.Source.SourceName != sourceFilter {
			continue
		}

		newsItem := map[string]interface{}{
			"title":  item.Title,
			"link":   item.Link,
			"image":  item.Image,
			"source": item.Source.SourceName,
			"date":   item.PubDate.Format(time.RFC3339),
		}
		response = append(response, newsItem)
	}

	c.JSON(http.StatusOK, gin.H{
		"news": response,
		"meta": gin.H{
			"total":    len(response),
			"limit":    limit,
			"offset":   offset,
			"language": lang,
			"category": category,
		},
	})
}

// GET /api/news/search
func (h *Handler) SearchNewsHandler(c *gin.Context) {
	query := c.Query("q")
	lang := c.Query("lang")
	category := c.Query("category")
	source := c.Query("source")

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parámetro 'q' requerido"})
		return
	}

	// Implementar búsqueda en BD (simplificado por ahora)
	ctx := c.Request.Context()
	newsItems, err := h.NewsRepo.FindByLangAndCategory(ctx, lang, category, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error en búsqueda"})
		return
	}

	// Filtrar por término de búsqueda y fuente
	var results []map[string]interface{}
	for _, item := range newsItems {
		// Aplicar filtro de fuente si se especifica
		if source != "" && item.Source.SourceName != source {
			continue
		}

		if contains(item.Title, query) || contains(item.Source.SourceName, query) {
			newsItem := map[string]interface{}{
				"title":  item.Title,
				"link":   item.Link,
				"image":  item.Image,
				"source": item.Source.SourceName,
				"date":   item.PubDate.Format(time.RFC3339),
			}
			results = append(results, newsItem)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"query":   query,
		"total":   len(results),
	})
}

// GET /api/categories
func (h *Handler) GetCategoriesHandler(c *gin.Context) {
	ctx := c.Request.Context()
	categories, err := h.CategoryRepo.ListAll(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo categorías"})
		return
	}

	var response []map[string]interface{}
	for _, cat := range categories {
		response = append(response, map[string]interface{}{
			"code": cat.Code,
			"name": cat.Name,
		})
	}

	c.JSON(http.StatusOK, response)
}

// GET /api/languages
func (h *Handler) GetLanguagesHandler(c *gin.Context) {
	ctx := c.Request.Context()
	countries, err := h.CountryRepo.ListAll(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error obteniendo idiomas"})
		return
	}

	var response []map[string]interface{}
	for _, country := range countries {
		response = append(response, map[string]interface{}{
			"code": country.Code,
			"name": country.Name,
		})
	}

	c.JSON(http.StatusOK, response)
}

// POST /api/news/refresh
func (h *Handler) RefreshNewsHandler(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err := h.FetchUseCase(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al refrescar noticias"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "Extracción de noticias iniciada"})
}

// GET /api/health
func (h *Handler) HealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GET /api/news/filtered - Filtros avanzados
func (h *Handler) GetFilteredNewsHandler(c *gin.Context) {
	// Parámetros de query
	lang := c.Query("lang")
	category := c.Query("category")
	sources := c.QueryArray("sources") // Múltiples fuentes
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	search := c.Query("search")

	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Construir filtros
	filters := domain.NewsFilters{
		Lang:     lang,
		Category: category,
		Sources:  sources,
		Search:   search,
	}

	// Parsear fechas si se proporcionan
	if dateFrom != "" {
		if date, err := time.Parse("2006-01-02", dateFrom); err == nil {
			filters.DateFrom = &date
		}
	}
	if dateTo != "" {
		if date, err := time.Parse("2006-01-02", dateTo); err == nil {
			// Ajustar a fin de día
			date := date.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			filters.DateTo = &date
		}
	}

	// Obtener noticias filtradas
	ctx := c.Request.Context()
	newsItems, err := h.NewsRepo.GetFilteredNews(ctx, filters, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error consultando noticias filtradas"})
		return
	}

	// Contar total de resultados
	totalCount, err := h.NewsRepo.CountFilteredNews(ctx, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error contando resultados"})
		return
	}

	// Convertir a formato JSON
	var response []map[string]interface{}
	for _, item := range newsItems {
		newsItem := map[string]interface{}{
			"id":         item.ID,
			"title":      item.Title,
			"link":       item.Link,
			"image":      item.Image,
			"source":     item.Source.SourceName,
			"category":   item.CategoryCode,
			"lang":       item.LangCode,
			"pub_date":   item.PubDate.Format(time.RFC3339),
			"created_at": item.CreatedAt.Format(time.RFC3339),
		}
		response = append(response, newsItem)
	}

	c.JSON(http.StatusOK, gin.H{
		"news": response,
		"meta": gin.H{
			"total":   totalCount,
			"limit":   limit,
			"offset":  offset,
			"filters": filters,
		},
	})
}

// Función helper para búsqueda simple
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(len(s) == len(substr) ||
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
