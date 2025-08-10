package http

import (
	"github.com/gin-gonic/gin"
)

func StartHTTPServer(handler *Handler, staticDir string, port string) {
	router := gin.Default()
	SetupMiddlewares(router)

	// ===== CONFIGURAR TEMPLATES HTML =====
	router.LoadHTMLGlob("frontend/templates/**/*")

	// ===== SERVIR ARCHIVOS ESTÁTICOS =====
	// Servir assets empaquetados (CSS y JS con hash) directamente desde frontend/dist
	router.Static("/css", "frontend/dist/css")
	router.Static("/js", "frontend/dist/js")

	// Servir imágenes y otros assets
	router.Static("/images", "frontend/assets/images")
	router.StaticFile("/favicon.ico", "frontend/assets/images/favicon.ico")

	// ===== CONFIGURAR RUTAS =====
	InitRoutes(router, handler)

	// ===== NO FALLBACK SPA - Cada ruta debe ser específica =====
	// Sin router.NoRoute() - verdadero NO-SPA

	router.Run(":" + port)
}
