package http

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"dailynews/internal/domain"
	"dailynews/pkg/utils"

	"github.com/gin-gonic/gin"
)

func getProjectRoot() string {
	currentDir, err := os.Getwd()
	if err != nil {
		return "."
	}

	// Subir directorios hasta encontrar go.mod
	for {
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			return currentDir
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			// Llegamos a la raíz del sistema
			return "."
		}
		currentDir = parent
	}
}

type PageData struct {
	Title            string
	Description      string
	Lang             string
	PageScript       string
	CurrentLang      string
	CurrentCategory  string
	SearchQuery      string
	Languages        []LanguageData
	Categories       []CategoryData
	News             []NewsData
	Pagination       *PaginationData
	NewsCount        int
	LastUpdate       string
	URL              string
	MainCSS          string   // Ruta del CSS principal con hash
	MainJS           string   // Ruta del JS principal con hash
	AvailableSources []string // Fuentes disponibles para filtros
}

type LanguageData struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type CategoryData struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Count int    `json:"count"`
}

type NewsData struct {
	ID           uint   `json:"id"`
	Title        string `json:"title"`
	Link         string `json:"link"`
	Image        string `json:"image"`
	SourceName   string `json:"source_name"`
	CategoryName string `json:"category_name"`
	Language     string `json:"language"`
	PubDate      string `json:"pub_date"`
	AuthorName   string `json:"author_name,omitempty"`
}

type PaginationData struct {
	CurrentPage int   `json:"current_page"`
	TotalPages  int   `json:"total_pages"`
	TotalItems  int   `json:"total_items"`
	HasNext     bool  `json:"has_next"`
	HasPrev     bool  `json:"has_prev"`
	NextPage    int   `json:"next_page"`
	PrevPage    int   `json:"prev_page"`
	PageRange   []int `json:"page_range"`
}

// GET / - Página principal
func (h *Handler) HomePageHandler(c *gin.Context) {
	// Parámetros de query
	lang := c.DefaultQuery("lang", "es")
	category := c.DefaultQuery("category", "")
	search := c.Query("search")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	limit := 48

	// Obtener datos comunes
	pageData, err := h.buildPageData(c, lang, category, search, page, limit)
	if err != nil {
		h.renderErrorPage(c, "Error interno del servidor", err.Error())
		return
	}

	// Configurar datos específicos de la página principal
	if search != "" {
		pageData.Title = fmt.Sprintf("Búsqueda: %s", search)
		pageData.Description = fmt.Sprintf("Resultados de búsqueda para '%s' en DailyNews", search)
	} else if category != "" {
		categoryName := h.getCategoryNameByCode(category)
		pageData.Title = fmt.Sprintf("%s - Noticias", categoryName)
		pageData.Description = fmt.Sprintf("Últimas noticias de %s en DailyNews", strings.ToLower(categoryName))
	} else {
		pageData.Title = "Noticias de Última Hora"
		pageData.Description = "Las noticias más relevantes de tecnología, deportes, cultura, entretenimiento y economía"
	}

	pageData.PageScript = "home.js"
	pageData.URL = c.Request.URL.String()

	c.HTML(http.StatusOK, "base", pageData)
}

// GET /categoria/:category - Página de categoría específica
func (h *Handler) CategoryPageHandler(c *gin.Context) {
	category := c.Param("category")
	lang := c.DefaultQuery("lang", "es")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	limit := 60

	// Validar que la categoría exista
	categoryData, err := h.getCategoryByCode(c.Request.Context(), category)
	if err != nil {
		h.renderErrorPage(c, "Categoría no encontrada", "La categoría solicitada no existe")
		return
	}

	pageData, err := h.buildPageData(c, lang, category, "", page, limit)
	if err != nil {
		h.renderErrorPage(c, "Error interno del servidor", err.Error())
		return
	}

	pageData.Title = fmt.Sprintf("%s - Noticias", categoryData.Name)
	pageData.Description = fmt.Sprintf("Últimas noticias de %s - Mantente informado con DailyNews", categoryData.Name)
	pageData.PageScript = "category.js"
	pageData.URL = c.Request.URL.String()

	c.HTML(http.StatusOK, "base", pageData) // Usamos el template base
}

// Página de búsqueda
func (h *Handler) SearchPageHandler(c *gin.Context) {
	query := c.Query("q")
	lang := c.DefaultQuery("lang", "es")
	category := c.Query("category")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	limit := 24

	pageData, err := h.buildPageData(c, lang, category, query, page, limit)
	if err != nil {
		h.renderErrorPage(c, "Error interno del servidor", err.Error())
		return
	}

	if query != "" {
		pageData.Title = fmt.Sprintf("Búsqueda: %s", query)
		pageData.Description = fmt.Sprintf("Resultados de búsqueda para '%s' en DailyNews", query)
	} else {
		pageData.Title = "Buscar Noticias"
		pageData.Description = "Busca noticias por palabra clave en DailyNews"
	}

	pageData.PageScript = "search.js"
	pageData.URL = c.Request.URL.String()

	c.HTML(http.StatusOK, "base", pageData) // Usamos el template base
}

// GET /api/sources/user - Obtener fuentes del usuario
func (h *Handler) GetUserSourcesHandler(c *gin.Context) {
	ctx := c.Request.Context()

	utils.AppInfo("GET_USER_SOURCES", "Solicitud de fuentes del usuario recibida", nil)

	// Obtener todas las fuentes (incluyendo inactivas) PD: lo de inactivas es para mas adelante que el usuario pueda activar/desactivar las fuentes por defecto por si alguna fuente en concreto no le gusta
	allSources, err := h.SourceRepo.ListAll(ctx)
	if err != nil {
		utils.AppError("GET_USER_SOURCES", "Error al obtener fuentes", err, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener fuentes"})
		return
	}

	utils.AppInfo("GET_USER_SOURCES", "Fuentes obtenidas de BD", map[string]interface{}{
		"total_sources": len(allSources),
	})

	// Filtrar solo las fuentes del usuario que estén activas
	var userSources []map[string]interface{}
	for _, source := range allSources {
		utils.AppInfo("GET_USER_SOURCES", "Procesando fuente", map[string]interface{}{
			"id":          source.ID,
			"source_name": source.SourceName,
			"user_added":  source.UserAdded,
			"is_active":   source.IsActive,
		})

		if source.UserAdded && source.IsActive {
			userSources = append(userSources, map[string]interface{}{
				"id":         source.ID,
				"sourceName": source.SourceName,
				"rssUrl":     source.RSSURL,
				"category":   source.News.Name,
				"language":   source.Lang.Name,
				"isActive":   source.IsActive,
			})
			utils.AppInfo("GET_USER_SOURCES", "Fuente agregada a respuesta", map[string]interface{}{
				"id": source.ID,
			})
		}
	}

	utils.AppInfo("GET_USER_SOURCES", "Respuesta enviada", map[string]interface{}{
		"user_sources_count": len(userSources),
	})

	c.JSON(http.StatusOK, userSources)
}

// detectBestPattern detecta automáticamente el mejor patrón para una URL RSS
// Implementa detección híbrida: primero intenta patrones con imagen, luego sin imagen
func (h *Handler) detectBestPattern(ctx context.Context, rssURL string) (string, error) {
	rssURL = strings.TrimSpace(rssURL)
	// FASE 1: Probar patrones con imagen (prioridad alta)
	patternsWithImage := []string{"patron1", "patron2", "patron3"}
	bestPattern, err := h.testPatternsWithImage(ctx, rssURL, patternsWithImage)
	if err == nil && bestPattern != "" {
		return bestPattern, nil
	}

	// FASE 2: Probar patrones sin imagen (fallback)
	patternsWithoutImage := []string{"patron1_no_image", "patron2_no_image", "patron3_no_image"}
	bestPattern, err = h.testPatternsWithoutImage(ctx, rssURL, patternsWithoutImage)
	if err == nil && bestPattern != "" {
		return bestPattern, nil
	}

	return "", fmt.Errorf("no se pudo detectar un patrón válido para esta URL")
}

// testPatternsWithImage prueba patrones que incluyen extracción de imagen
func (h *Handler) testPatternsWithImage(ctx context.Context, rssURL string, patterns []string) (string, error) {
	for _, pattern := range patterns {
		items, err := h.RSSFetcher.Fetch(ctx, rssURL, pattern, "", "", "", "")
		if err == nil && len(items) > 0 {
			validItems := 0
			for _, item := range items {
				// Validación completa: título, link, imagen
				if item.Title != "" && item.Link != "" && item.Image != "" && len(item.Title) > 10 {
					validItems++
				}
			}

			if validItems >= 2 {
				return pattern, nil
			}
		}
	}
	return "", fmt.Errorf("no se encontró patrón válido con imagen")
}

// testPatternsWithoutImage prueba patrones que no incluyen extracción de imagen
func (h *Handler) testPatternsWithoutImage(ctx context.Context, rssURL string, patterns []string) (string, error) {
	for _, pattern := range patterns {
		items, err := h.RSSFetcher.Fetch(ctx, rssURL, pattern, "", "", "", "")
		if err == nil && len(items) > 0 {
			validItems := 0
			for _, item := range items {
				// Validación sin imagen: solo título y link
				if item.Title != "" && item.Link != "" && len(item.Title) > 10 {
					validItems++
				}
			}

			if validItems >= 2 {
				return pattern, nil
			}
		}
	}
	return "", fmt.Errorf("no se encontró patrón válido sin imagen")
}

// Probar URL RSS con detección automática
func (h *Handler) TestSourceHandler(c *gin.Context) {
	var req struct {
		RSSURL string `json:"url" binding:"required"`
	}

	// Log de la solicitud recibida
	utils.AppInfo("TEST_SOURCE", "Solicitud de prueba RSS recibida", map[string]interface{}{
		"content_type": c.GetHeader("Content-Type"),
		"body_size":    c.Request.ContentLength,
	})

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.AppError("TEST_SOURCE", "Error al parsear JSON", err, map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL RSS requerida"})
		return
	}

	// Sanear URL: eliminar espacios en blanco accidentales
	req.RSSURL = strings.TrimSpace(req.RSSURL)

	utils.AppInfo("TEST_SOURCE", "Datos parseados correctamente", map[string]interface{}{
		"url": req.RSSURL,
	})

	ctx := c.Request.Context()

	// Detectar mejor patrón
	utils.AppInfo("TEST_SOURCE", "Iniciando detección de patrón", map[string]interface{}{
		"url": req.RSSURL,
	})

	bestPattern, err := h.detectBestPattern(ctx, req.RSSURL)
	if err != nil {
		utils.AppError("TEST_SOURCE", "Error al detectar patrón", err, map[string]interface{}{
			"url": req.RSSURL,
		})
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "No se pudo detectar un patrón válido para esta URL",
			"details": err.Error(),
		})
		return
	}

	utils.AppInfo("TEST_SOURCE", "Patrón detectado exitosamente", map[string]interface{}{
		"url":     req.RSSURL,
		"pattern": bestPattern,
	})

	// Obtener noticias con el patrón detectado
	items, err := h.RSSFetcher.Fetch(ctx, req.RSSURL, bestPattern, "", "", "", "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener noticias"})
		return
	}

	// Preparar respuesta
	var sampleTitles []string
	validCount := 0

	for _, item := range items {
		if item.Title != "" && item.Link != "" && len(item.Title) > 10 {
			validCount++
			if len(sampleTitles) < 3 {
				sampleTitles = append(sampleTitles, item.Title)
			}
		}
	}

	// Determinar tipo de patrón
	patternType := "con imagen"
	if strings.Contains(bestPattern, "no_image") {
		patternType = "sin imagen (requerirá imagen de fallback)"
	}

	utils.AppInfo("TEST_SOURCE", "Prueba completada exitosamente", map[string]interface{}{
		"url":          req.RSSURL,
		"pattern":      bestPattern,
		"pattern_type": patternType,
		"valid_items":  validCount,
		"total_items":  len(items),
		"sample_count": len(sampleTitles),
	})

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"valid_items":      validCount,
		"total_items":      len(items),
		"detected_pattern": bestPattern,
		"pattern_type":     patternType,
		"sample_titles":    sampleTitles,
	})
}

func (h *Handler) AddSourceHandler(c *gin.Context) {
	var req struct {
		SourceName      string `json:"sourceName" binding:"required"`
		RSSURL          string `json:"rssUrl" binding:"required"`
		Category        string `json:"category" binding:"required"`
		Language        string `json:"language" binding:"required"`
		FallbackImageID *uint  `json:"fallbackImageId"` // NUEVO: ID de imagen de fallback
	}

	// Log de la solicitud recibida
	utils.AppInfo("ADD_SOURCE", "Solicitud de agregar fuente RSS recibida", map[string]interface{}{
		"content_type": c.GetHeader("Content-Type"),
		"body_size":    c.Request.ContentLength,
	})

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.AppError("ADD_SOURCE", "Error al parsear JSON", err, map[string]interface{}{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: " + err.Error()})
		return
	}

	utils.AppInfo("ADD_SOURCE", "Datos parseados correctamente", map[string]interface{}{
		"source_name": req.SourceName,
		"rss_url":     req.RSSURL,
		"category":    req.Category,
		"language":    req.Language,
	})

	// Sanear entradas
	req.RSSURL = strings.TrimSpace(req.RSSURL)
	req.SourceName = strings.TrimSpace(req.SourceName)
	req.Category = strings.TrimSpace(req.Category)
	req.Language = strings.TrimSpace(req.Language)

	ctx := c.Request.Context()

	// Validar que la categoría existe
	category, err := h.getCategoryByCode(ctx, req.Category)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Categoría no válida"})
		return
	}

	// Validar que el idioma existe
	lang, err := h.CountryRepo.FindByCode(ctx, req.Language)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Idioma no válido"})
		return
	}

	// 1. Verificar duplicado exacto (misma URL + categoría + idioma)
	exists, err := h.SourceRepo.ExistsByURLCategoryLang(ctx, req.RSSURL, category.ID, lang.ID)
	if err != nil {
		utils.AppError("ADD_SOURCE", "Error al verificar duplicado", err, map[string]interface{}{
			"rss_url":  req.RSSURL,
			"category": req.Category,
			"language": req.Language,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al validar duplicado"})
		return
	}
	if exists {
		// Si venía una imagen de fallback, NO la persistimos ni la asociamos
		c.JSON(http.StatusConflict, gin.H{"error": "Esta fuente ya fue agregada para la misma categoría e idioma"})
		return
	}

	// 2. Detectar el mejor patrón automáticamente
	bestPattern, err := h.detectBestPattern(ctx, req.RSSURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se pudo procesar esta fuente RSS. Verifica que la URL sea correcta."})
		return
	}

	// 3. Crear fuente con el patrón detectado
	newSource := &domain.NewsSource{
		SourceName: req.SourceName,
		RSSURL:     req.RSSURL,
		NewsID:     category.ID,
		LangID:     lang.ID,
		IsActive:   true,
		UserAdded:  true,         // ← MARCA COMO FUENTE DEL USUARIO
		Filter:     &bestPattern, // ← PATRÓN DETECTADO AUTOMÁTICAMENTE
	}

	// 4. Guardar en la base de datos
	if err := h.SourceRepo.Create(ctx, newSource); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar la fuente: " + err.Error()})
		return
	}

	// 5. Si se subió una imagen de fallback, asociarla a la fuente
	if req.FallbackImageID != nil {
		newSource.FallbackImageID = req.FallbackImageID
		if err := h.SourceRepo.Update(ctx, newSource); err != nil {
			utils.AppWarn("ADD_SOURCE", "Error al asociar imagen de fallback", map[string]interface{}{
				"source_id":   newSource.ID,
				"fallback_id": *req.FallbackImageID,
			})
		} else {
			utils.AppInfo("ADD_SOURCE", "Imagen de fallback asociada exitosamente", map[string]interface{}{
				"source_id":   newSource.ID,
				"fallback_id": *req.FallbackImageID,
			})
		}
	}

	// 6. EXTRAER NOTICIAS DE LA NUEVA FUENTE AUTOMÁTICAMENTE
	utils.AppInfo("ADD_SOURCE", "Iniciando extracción automática de noticias", map[string]interface{}{
		"source_id":   newSource.ID,
		"source_name": newSource.SourceName,
		"category":    req.Category,
		"language":    req.Language,
	})

	// Usar el FetchNewsUseCase específico para esta fuente
	if err := h.FetchUseCaseForSource(ctx, newSource.ID); err != nil {
		utils.AppWarn("ADD_SOURCE", "Error al extraer noticias automáticamente", map[string]interface{}{
			"source_id": newSource.ID,
			"error":     err.Error(),
		})
	} else {
		utils.AppInfo("ADD_SOURCE", "Extracción automática completada", map[string]interface{}{
			"source_id": newSource.ID,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Fuente agregada exitosamente",
		"id":      newSource.ID,
		"pattern": bestPattern,
	})
}

// POST /api/sources/check-duplicate - Verificar duplicado exacto antes de subir fallback
func (h *Handler) CheckDuplicateSourceHandler(c *gin.Context) {
	var req struct {
		RSSURL   string `json:"rssUrl" binding:"required"`
		Category string `json:"category" binding:"required"`
		Language string `json:"language" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	ctx := c.Request.Context()

	// Sanear
	req.RSSURL = strings.TrimSpace(req.RSSURL)
	req.Category = strings.TrimSpace(req.Category)
	req.Language = strings.TrimSpace(req.Language)

	// Validar categoría e idioma
	category, err := h.getCategoryByCode(ctx, req.Category)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Categoría no válida"})
		return
	}
	lang, err := h.CountryRepo.FindByCode(ctx, req.Language)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Idioma no válido"})
		return
	}

	exists, err := h.SourceRepo.ExistsByURLCategoryLang(ctx, req.RSSURL, category.ID, lang.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al validar duplicado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"exists": exists})
}

// DELETE /api/sources/:id - Eliminar fuente RSS
func (h *Handler) DeleteSourceHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.AppError("DELETE_SOURCE", "ID inválido", err, map[string]interface{}{
			"id_str": idStr,
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	utils.AppInfo("DELETE_SOURCE", "Solicitud de eliminación recibida", map[string]interface{}{
		"id": id,
	})

	ctx := c.Request.Context()

	// Buscar la fuente
	source, err := h.SourceRepo.FindByID(ctx, uint(id))
	if err != nil {
		utils.AppError("DELETE_SOURCE", "Error al buscar fuente", err, map[string]interface{}{
			"id": id,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al buscar la fuente"})
		return
	}

	if source == nil {
		utils.AppWarn("DELETE_SOURCE", "Fuente no encontrada", map[string]interface{}{
			"id": id,
		})
		c.JSON(http.StatusNotFound, gin.H{"error": "Fuente no encontrada"})
		return
	}

	utils.AppInfo("DELETE_SOURCE", "Fuente encontrada", map[string]interface{}{
		"id":          source.ID,
		"source_name": source.SourceName,
		"is_active":   source.IsActive,
		"user_added":  source.UserAdded,
	})

	// ELIMINAR FÍSICAMENTE la fuente
	if err := h.SourceRepo.Delete(ctx, uint(id)); err != nil {
		utils.AppError("DELETE_SOURCE", "Error al eliminar fuente", err, map[string]interface{}{
			"id": source.ID,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar la fuente"})
		return
	}

	// Si la fuente tenía imagen de fallback, eliminarla también
	if source.FallbackImageID != nil {
		// Buscar la imagen de fallback
		fallbackImage, err := h.FallbackImageRepo.GetByID(ctx, *source.FallbackImageID)
		if err == nil && fallbackImage != nil {
			// Eliminar el archivo físico
			filePath := filepath.Join(getProjectRoot(), "frontend", "assets", "images", "fallback", fallbackImage.Filename)
			if err := os.Remove(filePath); err != nil {
				utils.AppWarn("DELETE_SOURCE", "Error al eliminar archivo de imagen", map[string]interface{}{
					"file_path": filePath,
					"error":     err.Error(),
				})
			} else {
				utils.AppInfo("DELETE_SOURCE", "Archivo de imagen eliminado del disco", map[string]interface{}{
					"file_path": filePath,
				})
			}

			// Eliminar registro de la base de datos
			if err := h.FallbackImageRepo.DeleteByID(ctx, fallbackImage.ID); err != nil {
				utils.AppWarn("DELETE_SOURCE", "Error al eliminar registro de imagen", map[string]interface{}{
					"fallback_id": fallbackImage.ID,
					"error":       err.Error(),
				})
			} else {
				utils.AppInfo("DELETE_SOURCE", "Registro de imagen eliminado de la BD", map[string]interface{}{
					"fallback_id": fallbackImage.ID,
				})
			}
		}
	}

	utils.AppInfo("DELETE_SOURCE", "Fuente eliminada exitosamente", map[string]interface{}{
		"id": source.ID,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Fuente eliminada exitosamente"})
}

// PUT /api/sources/:id - Actualizar nombre de fuente (solo user-added)
func (h *Handler) UpdateSourceHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	var req struct {
		SourceName string `json:"sourceName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}
	req.SourceName = strings.TrimSpace(req.SourceName)
	if req.SourceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "El nombre no puede estar vacío"})
		return
	}

	ctx := c.Request.Context()
	source, err := h.SourceRepo.FindByID(ctx, uint(id))
	if err != nil || source == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fuente no encontrada"})
		return
	}
	if !source.UserAdded {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo se pueden editar fuentes del usuario"})
		return
	}

	source.SourceName = req.SourceName
	if err := h.SourceRepo.Update(ctx, source); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error actualizando fuente"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// POST /api/sources/:id/fallback-image - Actualiza imagen fallback de la fuente
func (h *Handler) UpdateSourceFallbackImageHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}

	// Debe recibir FormData con `image`
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Archivo de imagen requerido"})
		return
	}
	if err := validateImageFile(file); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	source, err := h.SourceRepo.FindByID(ctx, uint(id))
	if err != nil || source == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fuente no encontrada"})
		return
	}
	if !source.UserAdded {
		c.JSON(http.StatusForbidden, gin.H{"error": "Solo se pueden editar fuentes del usuario"})
		return
	}

	// Guardar archivo en carpeta fallback y registrar/actualizar tabla
	timestamp := time.Now().Format("20060102_150405")
	ext := getFileExtension(file.Filename)
	filename := fmt.Sprintf("%s_%s_%s%s", source.News.Code, source.Lang.Code, timestamp, ext)
	projectRoot := getProjectRoot()
	uploadDir := filepath.Join(projectRoot, "frontend", "assets", "images", "fallback")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear directorio"})
		return
	}
	uploadPath := filepath.Join(uploadDir, filename)
	if err := c.SaveUploadedFile(file, uploadPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar imagen"})
		return
	}

	// Si tenía imagen anterior, eliminar archivo y registro
	if source.FallbackImageID != nil {
		oldImg, _ := h.FallbackImageRepo.GetByID(ctx, *source.FallbackImageID)
		if oldImg != nil {
			oldPath := filepath.Join(projectRoot, "frontend", "assets", "images", "fallback", oldImg.Filename)
			os.Remove(oldPath)
			_ = h.FallbackImageRepo.DeleteByID(ctx, oldImg.ID)
		}
	}

	// Crear nuevo registro
	newImg := &domain.FallbackImage{
		CategoryCode: source.News.Code,
		LanguageCode: source.Lang.Code,
		Filename:     filename,
		OriginalName: file.Filename,
		MimeType:     file.Header.Get("Content-Type"),
		FileSize:     file.Size,
	}
	if err := h.FallbackImageRepo.Create(ctx, newImg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al registrar imagen"})
		return
	}

	// Asociar a la fuente
	source.FallbackImageID = &newImg.ID
	if err := h.SourceRepo.Update(ctx, source); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar fuente"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "filename": filename})
}

// buildPageData construye los datos comunes para todas las páginas
func (h *Handler) buildPageData(c *gin.Context, lang, category, search string, page, limit int) (*PageData, error) {
	ctx := c.Request.Context()

	// Crear un contexto con el gin.Context para acceder a los query parameters
	ctxWithGin := context.WithValue(ctx, "gin_context", c)

	// Obtener idiomas disponibles
	languages, err := h.getLanguagesData(ctx)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo idiomas: %w", err)
	}

	// Obtener categorías disponibles
	categories, err := h.getCategoriesData(ctx, lang)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo categorías: %w", err)
	}

	// Obtener noticias según filtros
	news, pagination, err := h.getFilteredNews(ctxWithGin, lang, category, search, page, limit)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo noticias: %w", err)
	}

	// Construir fuentes disponibles a partir de las noticias mostradas
	sourceSet := make(map[string]struct{})
	for _, n := range news {
		if n.SourceName != "" {
			sourceSet[n.SourceName] = struct{}{}
		}
	}
	availableSources := make([]string, 0, len(sourceSet))
	for name := range sourceSet {
		availableSources = append(availableSources, name)
	}

	return &PageData{
		Lang:             lang,
		CurrentLang:      lang,
		CurrentCategory:  category,
		SearchQuery:      search,
		Languages:        languages,
		Categories:       categories,
		News:             news,
		Pagination:       pagination,
		NewsCount:        len(news),
		LastUpdate:       time.Now().Format("2006-01-02 15:04"),
		MainCSS:          utils.GetCSSAsset(),
		MainJS:           utils.GetJSAsset(),
		AvailableSources: availableSources,
	}, nil
}

// getLanguagesData obtiene todos los idiomas disponibles
func (h *Handler) getLanguagesData(ctx context.Context) ([]LanguageData, error) {
	countries, err := h.CountryRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	languages := make([]LanguageData, len(countries))
	for i, country := range countries {
		languages[i] = LanguageData{
			Code: country.Code,
			Name: country.Name,
		}
	}

	return languages, nil
}

// getCategoriesData obtiene todas las categorías disponibles con conteos
func (h *Handler) getCategoriesData(ctx context.Context, lang string) ([]CategoryData, error) {
	categories, err := h.CategoryRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	categoriesData := make([]CategoryData, len(categories))
	for i, cat := range categories {
		// Obtener conteo de noticias por categoría usando el idioma actual
		count, _ := h.NewsRepo.CountByCategory(ctx, cat.Code, lang)

		categoriesData[i] = CategoryData{
			Code:  cat.Code,
			Name:  h.getCategoryNameByCodeAndLang(cat.Code, lang),
			Icon:  h.getCategoryIcon(cat.Code),
			Count: count,
		}
	}

	return categoriesData, nil
}

// getCategoryIcon devuelve el emoji/icono para cada categoría
func (h *Handler) getCategoryIcon(categoryCode string) string {
	icons := map[string]string{
		"technology":      "💻",
		"salud":           "🏥",
		"sports":          "⚽",
		"cultura":         "🎭",
		"internacional":   "🌍",
		"entretenimiento": "🎬",
		"economia":        "💰",
		"destacado":       "⭐",
	}

	if icon, exists := icons[categoryCode]; exists {
		return icon
	}
	return "📰"
}

// getFilteredNews obtiene noticias filtradas con paginación
func (h *Handler) getFilteredNews(ctx context.Context, lang, category, search string, page, limit int) ([]NewsData, *PaginationData, error) {
	offset := (page - 1) * limit

	var newsItems []domain.NewsItem
	var totalCount int
	var err error

	// Obtener filtros desde el contexto
	var sources []string
	var dateRange, dateFrom, dateTo string
	if c, ok := ctx.Value("gin_context").(*gin.Context); ok {
		sources = c.QueryArray("sources")
		dateRange = c.Query("date_range")
		dateFrom = c.Query("date_from")
		dateTo = c.Query("date_to")
	}

	// Construir filtros avanzados
	filters := domain.NewsFilters{
		Lang:     lang,
		Category: category,
		Search:   search,
		Sources:  sources,
	}

	if category == "" {
		// Excluir categoría "breaking" de la página principal
		filters.ExcludeCategories = []string{"breaking"}
	}

	// Procesar filtros de fecha
	if dateRange != "" {
		// Usar rangos predefinidos
		start, end := utils.GetDateRange(dateRange)
		filters.DateFrom = &start
		filters.DateTo = &end
	} else if dateFrom != "" || dateTo != "" {
		// Usar fechas personalizadas
		if dateFrom != "" {
			if date, err := time.Parse("2006-01-02", dateFrom); err == nil {
				filters.DateFrom = &date
			}
		}
		if dateTo != "" {
			if date, err := time.Parse("2006-01-02", dateTo); err == nil {
				// Ajustar al final del día
				date = date.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
				filters.DateTo = &date
			}
		}
	}

	// Obtener noticias filtradas usando el nuevo método
	newsItems, err = h.NewsRepo.GetFilteredNews(ctx, filters, limit, offset)
	if err != nil {
		return nil, nil, err
	}

	// Contar total de resultados filtrados
	totalCount, err = h.NewsRepo.CountFilteredNews(ctx, filters)
	if err != nil {
		return nil, nil, err
	}

	// Convertir domain.NewsItem a NewsData
	news := make([]NewsData, len(newsItems))
	for i, item := range newsItems {
		news[i] = NewsData{
			ID:           item.ID,
			Title:        item.Title,
			Link:         item.Link,
			Image:        item.Image,
			SourceName:   item.Source.SourceName,
			CategoryName: h.getCategoryNameByCode(item.CategoryCode),
			Language:     item.LangCode,
			PubDate:      utils.FormatDate(item.PubDate),
		}
	}

	// Calcular paginación
	totalPages := (totalCount + limit - 1) / limit
	pagination := &PaginationData{
		CurrentPage: page,
		TotalPages:  totalPages,
		TotalItems:  totalCount,
		HasNext:     page < totalPages,
		HasPrev:     page > 1,
		NextPage:    page + 1,
		PrevPage:    page - 1,
		PageRange:   calculatePageRange(page, totalPages),
	}

	return news, pagination, nil
}

// calculatePageRange calcula el rango de páginas a mostrar en la paginación
func calculatePageRange(currentPage, totalPages int) []int {
	const maxPages = 7 // Mostrar máximo 7 números de página

	if totalPages <= maxPages {
		pages := make([]int, totalPages)
		for i := 0; i < totalPages; i++ {
			pages[i] = i + 1
		}
		return pages
	}

	start := currentPage - 3
	if start < 1 {
		start = 1
	}

	end := start + maxPages - 1
	if end > totalPages {
		end = totalPages
		start = end - maxPages + 1
		if start < 1 {
			start = 1
		}
	}

	pages := make([]int, end-start+1)
	for i := 0; i < len(pages); i++ {
		pages[i] = start + i
	}

	return pages
}

// getCategoryByCode obtiene una categoría por su código
func (h *Handler) getCategoryByCode(ctx context.Context, code string) (*domain.Category, error) {
	categories, err := h.CategoryRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, cat := range categories {
		if cat.Code == code {
			return &cat, nil
		}
	}

	return nil, fmt.Errorf("categoría '%s' no encontrada", code)
}

// getCategoryNameByCode obtiene el nombre de una categoría por su código
func (h *Handler) getCategoryNameByCode(code string) string {
	// Mapa por defecto en español
	names := map[string]string{
		"technology":    "Tecnología",
		"health":        "Salud",
		"sports":        "Deportes",
		"culture":       "Cultura",
		"international": "Internacional",
		"entertainment": "Entretenimiento",
		"economy":       "Economía",
		"breaking":      "Último Momento",
	}

	if name, exists := names[code]; exists {
		return name
	}
	return "Noticias"
}

// getCategoryNameByCodeAndLang devuelve el nombre localizado por código e idioma
func (h *Handler) getCategoryNameByCodeAndLang(code, lang string) string {
	switch lang {
	case "en":
		names := map[string]string{
			"technology":    "Technology",
			"health":        "Health",
			"sports":        "Sports",
			"culture":       "Culture",
			"international": "International",
			"entertainment": "Entertainment",
			"economy":       "Economy",
			"breaking":      "Breaking News",
		}
		if n, ok := names[code]; ok {
			return n
		}
	case "fr":
		names := map[string]string{
			"technology":    "Technologie",
			"health":        "Santé",
			"sports":        "Sports",
			"culture":       "Culture",
			"international": "International",
			"entertainment": "Divertissement",
			"economy":       "Économie",
			"breaking":      "À la une",
		}
		if n, ok := names[code]; ok {
			return n
		}
	default:
		// Español (por defecto)
		return h.getCategoryNameByCode(code)
	}
	return h.getCategoryNameByCode(code)
}

// getAvailableSources obtiene las fuentes disponibles para el filtro según la categoría actual
func (h *Handler) getAvailableSources(ctx context.Context, categoryCode, lang string) ([]string, error) {
	// Si no hay categoría específica, obtener todas las fuentes del idioma actual
	if categoryCode == "" {
		sources, err := h.SourceRepo.ListActive(ctx)
		if err != nil {
			return nil, err
		}

		// Filtrar por idioma actual
		sourceMap := make(map[string]bool)
		for _, source := range sources {
			if source.Lang.Code == lang {
				sourceMap[source.SourceName] = true
			}
		}

		// Convertir a slice
		availableSources := make([]string, 0, len(sourceMap))
		for sourceName := range sourceMap {
			availableSources = append(availableSources, sourceName)
		}

		return availableSources, nil
	}

	// Obtener la categoría por código
	category, err := h.getCategoryByCode(ctx, categoryCode)
	if err != nil {
		return nil, err
	}

	// Obtener el idioma por código
	country, err := h.CountryRepo.FindByCode(ctx, lang)
	if err != nil {
		return nil, err
	}

	// Obtener fuentes específicas de esta categoría e idioma
	sources, err := h.SourceRepo.FindActiveByLangAndCategory(ctx, country.ID, category.ID)
	if err != nil {
		return nil, err
	}

	// Crear un mapa para evitar duplicados
	sourceMap := make(map[string]bool)
	for _, source := range sources {
		sourceMap[source.SourceName] = true
	}

	// Convertir a slice
	availableSources := make([]string, 0, len(sourceMap))
	for sourceName := range sourceMap {
		availableSources = append(availableSources, sourceName)
	}

	return availableSources, nil
}

// renderErrorPage renderiza una página de error
func (h *Handler) renderErrorPage(c *gin.Context, title, message string) {
	errorData := PageData{
		Title:       title,
		Description: message,
		Lang:        c.DefaultQuery("lang", "es"),
		MainCSS:     utils.GetCSSAsset(),
		MainJS:      utils.GetJSAsset(),
	}

	c.HTML(http.StatusNotFound, "error.html", errorData)
}

// ===== HANDLERS PARA IMÁGENES DE FALLBACK =====

// UploadFallbackImageHandler maneja la subida de imágenes de fallback
func (h *Handler) UploadFallbackImageHandler(c *gin.Context) {
	// Obtener parámetros del formulario
	categoryCode := c.PostForm("categoryCode")
	languageCode := c.PostForm("languageCode")

	utils.AppInfo("UPLOAD_FALLBACK", "Solicitud de subida de imagen recibida", map[string]interface{}{
		"category_code": categoryCode,
		"language_code": languageCode,
		"content_type":  c.GetHeader("Content-Type"),
	})

	if categoryCode == "" || languageCode == "" {
		utils.AppError("UPLOAD_FALLBACK", "Categoría o idioma faltante", nil, map[string]interface{}{
			"category_code": categoryCode,
			"language_code": languageCode,
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "Categoría e idioma son requeridos"})
		return
	}

	// Obtener archivo
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Archivo de imagen requerido"})
		return
	}

	// Validar archivo
	if err := validateImageFile(file); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generar nombre único
	timestamp := time.Now().Format("20060102_150405")
	extension := getFileExtension(file.Filename)
	filename := fmt.Sprintf("%s_%s_%s%s", categoryCode, languageCode, timestamp, extension)

	// Crear directorio si no existe (ruta relativa al proyecto)
	projectRoot := getProjectRoot()
	uploadDir := filepath.Join(projectRoot, "frontend", "assets", "images", "fallback")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear directorio"})
		return
	}

	// Guardar archivo
	uploadPath := filepath.Join(uploadDir, filename)
	utils.AppInfo("UPLOAD_FALLBACK", "Guardando archivo", map[string]interface{}{
		"upload_path": uploadPath,
		"file_size":   file.Size,
		"filename":    filename,
	})

	if err := c.SaveUploadedFile(file, uploadPath); err != nil {
		utils.AppError("UPLOAD_FALLBACK", "Error al guardar archivo", err, map[string]interface{}{
			"upload_path": uploadPath,
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar archivo"})
		return
	}

	utils.AppInfo("UPLOAD_FALLBACK", "Archivo guardado exitosamente", map[string]interface{}{
		"upload_path": uploadPath,
	})

	// Crear registro en BD
	fallbackImage := &domain.FallbackImage{
		CategoryCode: categoryCode,
		LanguageCode: languageCode,
		Filename:     filename,
		OriginalName: file.Filename,
		MimeType:     file.Header.Get("Content-Type"),
		FileSize:     file.Size,
	}

	ctx := c.Request.Context()
	if err := h.FallbackImageRepo.Create(ctx, fallbackImage); err != nil {
		// Eliminar archivo si falla la BD
		os.Remove(uploadPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar en base de datos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"id":       fallbackImage.ID,
		"filename": filename,
		"message":  "Imagen de fallback guardada exitosamente",
	})
}

// GetFallbackImageHandler obtiene información de imagen de fallback
func (h *Handler) GetFallbackImageHandler(c *gin.Context) {
	categoryCode := c.Param("category")
	languageCode := c.Param("lang")

	ctx := c.Request.Context()
	image, err := h.FallbackImageRepo.GetByCategoryAndLang(ctx, categoryCode, languageCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener imagen"})
		return
	}

	if image == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No se encontró imagen de fallback"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"category_code": image.CategoryCode,
		"language_code": image.LanguageCode,
		"filename":      image.Filename,
		"original_name": image.OriginalName,
		"file_size":     image.FileSize,
		"created_at":    image.CreatedAt,
	})
}

// DeleteFallbackImageHandler elimina imagen de fallback
func (h *Handler) DeleteFallbackImageHandler(c *gin.Context) {
	categoryCode := c.Param("category")
	languageCode := c.Param("lang")

	ctx := c.Request.Context()

	// Obtener imagen para eliminar archivo
	image, err := h.FallbackImageRepo.GetByCategoryAndLang(ctx, categoryCode, languageCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener imagen"})
		return
	}

	if image == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No se encontró imagen de fallback"})
		return
	}

	// Eliminar de BD
	if err := h.FallbackImageRepo.Delete(ctx, categoryCode, languageCode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar de base de datos"})
		return
	}

	// Eliminar archivo
	projectRoot := getProjectRoot()
	filePath := filepath.Join(projectRoot, "frontend", "assets", "images", "fallback", image.Filename)
	os.Remove(filePath)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Imagen de fallback eliminada exitosamente",
	})
}

// ListFallbackImagesHandler lista todas las imágenes de fallback
func (h *Handler) ListFallbackImagesHandler(c *gin.Context) {
	ctx := c.Request.Context()
	images, err := h.FallbackImageRepo.ListAll(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener imágenes"})
		return
	}

	var result []gin.H
	for _, img := range images {
		result = append(result, gin.H{
			"category_code": img.CategoryCode,
			"language_code": img.LanguageCode,
			"filename":      img.Filename,
			"original_name": img.OriginalName,
			"file_size":     img.FileSize,
			"created_at":    img.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, result)
}

// Funciones auxiliares para validación de archivos
func validateImageFile(file *multipart.FileHeader) error {
	// Validar tipo MIME
	contentType := file.Header.Get("Content-Type")
	validTypes := []string{"image/jpeg", "image/jpg", "image/png", "image/webp"}

	isValidType := false
	for _, validType := range validTypes {
		if contentType == validType {
			isValidType = true
			break
		}
	}

	if !isValidType {
		return errors.New("solo se permiten archivos de imagen (JPG, PNG, WebP)")
	}

	// Validar tamaño (5MB máximo)
	if file.Size > 5*1024*1024 {
		return errors.New("el archivo debe ser menor a 5MB")
	}

	return nil
}

func getFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ".jpg"
	}
	return ext
}
