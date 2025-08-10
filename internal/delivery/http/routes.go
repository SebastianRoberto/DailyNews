package http

import (
	"github.com/gin-gonic/gin"
)

func InitRoutes(router *gin.Engine, handler *Handler) {

	router.GET("/", handler.HomePageHandler)
	router.GET("/categoria/:category", handler.CategoryPageHandler)
	router.GET("/buscar", handler.SearchPageHandler)

	//  Rutas de API
	api := router.Group("/api")
	{
		// Rutas de noticias
		api.GET("/news/:lang/:category", handler.GetNewsHandler)
		api.GET("/news/search", handler.SearchNewsHandler)
		api.GET("/news/filtered", handler.GetFilteredNewsHandler) // Nueva ruta para filtros avanzados
		// Fuentes RSS del usuario (CRUD)
		api.PUT("/sources/:id", handler.UpdateSourceHandler)                              // actualizar nombre
		api.POST("/sources/:id/fallback-image", handler.UpdateSourceFallbackImageHandler) // actualizar imagen fallback

		// Rutas de metadatos
		api.GET("/categories", handler.GetCategoriesHandler)
		api.GET("/languages", handler.GetLanguagesHandler)

		// Rutas de gestión de fuentes RSS
		api.GET("/sources/user", handler.GetUserSourcesHandler)
		api.POST("/sources/check-duplicate", handler.CheckDuplicateSourceHandler)
		api.POST("/sources/add", handler.AddSourceHandler)
		api.POST("/sources/test", handler.TestSourceHandler)
		api.DELETE("/sources/:id", handler.DeleteSourceHandler)

		// Rutas para gestión de imágenes de fallback
		api.POST("/fallback-image/upload", handler.UploadFallbackImageHandler)
		api.GET("/fallback-image/:category/:lang", handler.GetFallbackImageHandler)
		api.DELETE("/fallback-image/:category/:lang", handler.DeleteFallbackImageHandler)
		api.GET("/fallback-image/list", handler.ListFallbackImagesHandler)

		// Rutas de administración
		api.POST("/news/refresh", handler.RefreshNewsHandler)
		api.GET("/health", handler.HealthHandler)
	}
}
