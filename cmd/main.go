package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	http_delivery "dailynews/internal/delivery/http"
	"dailynews/internal/infrastructure"
	"dailynews/internal/repository"
	"dailynews/internal/usecase"
	"dailynews/pkg/config"
	"dailynews/pkg/database"

	"github.com/joho/godotenv"
)

type simpleLogger struct{}

func (l *simpleLogger) Debug(msg string, fields ...interface{}) { log.Println("DEBUG:", msg, fields) }
func (l *simpleLogger) Info(msg string, fields ...interface{})  { log.Println("INFO:", msg, fields) }
func (l *simpleLogger) Warn(msg string, fields ...interface{})  { log.Println("WARN:", msg, fields) }
func (l *simpleLogger) Error(msg string, fields ...interface{}) { log.Println("ERROR:", msg, fields) }

// buildFrontendAssets compila los assets del frontend automáticamente
func buildFrontendAssets() error {
	// Permitir omitir el build en runtime
	if os.Getenv("SKIP_FRONTEND_BUILD") == "true" {
		log.Println("⏭️  SKIP_FRONTEND_BUILD=true → Omitiendo compilación de frontend")
		return nil
	}
	log.Println("🔨 Compilando assets del frontend...")

	// Verificar si existe el directorio frontend
	frontendDir := "frontend"
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		return fmt.Errorf("directorio frontend no encontrado: %v", err)
	}

	// Verificar si existe package.json
	packageJSON := filepath.Join(frontendDir, "package.json")
	if _, err := os.Stat(packageJSON); os.IsNotExist(err) {
		return fmt.Errorf("package.json no encontrado en frontend: %v", err)
	}

	// Verificar binarios básicos
	if _, err := exec.LookPath("node"); err != nil {
		log.Printf("⚠️  Node.js no encontrado en PATH: %v", err)
	}
	if _, err := exec.LookPath("npm"); err != nil {
		log.Printf("⚠️  npm no encontrado en PATH: %v", err)
	}

	// Ejecutar npm install/ci si node_modules no existe
	nodeModules := filepath.Join(frontendDir, "node_modules")
	if _, err := os.Stat(nodeModules); os.IsNotExist(err) {
		log.Println("📦 Instalando dependencias de npm...")
		installCmd := "install"
		// Si existe package-lock.json preferimos 'npm ci'
		if _, err := os.Stat(filepath.Join(frontendDir, "package-lock.json")); err == nil {
			installCmd = "ci"
		}
		cmd := exec.Command("npm", installCmd)
		cmd.Dir = frontendDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("error instalando dependencias: %v", err)
		}
	}

	// Si node_modules existe, verificar vite utilizable y reparar si hace falta
	viteBin := filepath.Join(frontendDir, "node_modules", ".bin", "vite")
	viteJs := filepath.Join(frontendDir, "node_modules", "vite", "bin", "vite.js")
	needRepair := false
	if fi, err := os.Stat(viteBin); err != nil {
		// faltante o inaccesible: reintentar con npm ci
		needRepair = true
	} else if runtime.GOOS != "windows" {
		// en *nix verificar bit de ejecución
		if fi.Mode()&0111 == 0 {
			needRepair = true
		}
	}
	if _, err := os.Stat(viteJs); err != nil {
		// si falta el script principal, necesitamos reparar
		needRepair = true
	}
	if needRepair {
		log.Println("🔧 Reparando dependencias de frontend (npm ci)...")
		cmd := exec.Command("npm", "ci")
		cmd.Dir = frontendDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("⚠️  Error reparando dependencias con 'npm ci': %v", err)
		}
	}

	// Ejecutar npm run build
	log.Println("🏗️  Ejecutando build de assets...")
	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = frontendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Printf("⚠️  Falló 'npm run build': %v", err)
		// Fallback 1: npx vite build
		log.Println("↩️  Reintentando con 'npx --yes vite build'...")
		cmd1 := exec.Command("npx", "--yes", "vite", "build")
		cmd1.Dir = frontendDir
		cmd1.Stdout = os.Stdout
		cmd1.Stderr = os.Stderr
		if err1 := cmd1.Run(); err1 != nil {
			log.Printf("⚠️  Falló 'npx --yes vite build': %v", err1)
			// Fallback 2: node node_modules/vite/bin/vite.js build
			log.Println("↩️  Reintentando con 'node node_modules/vite/bin/vite.js build'...")
			cmd2 := exec.Command("node", viteJs, "build")
			cmd2.Dir = frontendDir
			cmd2.Stdout = os.Stdout
			cmd2.Stderr = os.Stderr
			if err2 := cmd2.Run(); err2 != nil {
				return fmt.Errorf("error compilando assets (npm/npx/node fallbacks fallidos): %v | %v | %v", err, err1, err2)
			}
		}
	}

	log.Println("✅ Assets del frontend compilados exitosamente")
	return nil
}

func main() {
	// Cargar variables de entorno desde .env(en mi caso no lo uso)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// 1. Cargar configuración
	cfg, err := config.LoadConfig(os.Getenv("CONFIG_PATH"))
	if err != nil {
		log.Fatalf("Error cargando la configuración: %v", err)
	}

	// 2. Conectar a la base de datos (crea la BD si no existe)
	dbConfig := database.Config{
		Host:         cfg.Database.NewsDB.Host,
		Port:         cfg.Database.NewsDB.Port,
		User:         cfg.Database.NewsDB.User,
		Password:     cfg.Database.NewsDB.Password,
		DatabaseName: cfg.Database.NewsDB.Schema,
	}
	db, err := database.New(dbConfig)
	if err != nil {
		log.Fatalf("Error conectando a la base de datos: %v", err)
	}

	// 3. Ejecutar migraciones (crea las tablas)
	if err := db.Migrate(); err != nil {
		log.Fatalf("Error ejecutando migraciones: %v", err)
	}

	// 4. Crear datos iniciales (seeds inteligentes)
	ctx := context.Background()
	db.SeedInitialData(ctx)

	// 5. Instanciar Repositorios
	newsItemRepo := repository.NewNewsItemRepository(db.DB)
	categoryRepo := repository.NewCategoryRepository(db.DB)
	countryRepo := repository.NewCountryRepository(db.DB)
	newsSourceRepo := repository.NewNewsSourceRepository(db.DB)
	fallbackImageRepo := repository.NewFallbackImageRepository(db.DB) // NUEVO

	// 6. Instanciar Componentes de Infraestructura
	imageDownloader := infrastructure.NewImageDownloader(cfg.Filters.TargetAspect, cfg.Filters.AspectTolerance, 800, 450)
	rssFetcher := infrastructure.NewRSSFetcher()

	// 7. Instanciar Caso de Uso
	fetchNewsUseCase := usecase.NewFetchNewsUseCase(
		newsItemRepo,
		categoryRepo,
		countryRepo,
		newsSourceRepo,
		fallbackImageRepo, // NUEVO
		rssFetcher,
		imageDownloader,
		cfg,
	)

	// Función anónima para el handler y el cron
	fetchFunc := func(ctx context.Context) error {
		return fetchNewsUseCase.Execute(ctx)
	}

	// Función anónima para extraer noticias de una fuente específica
	fetchFuncForSource := func(ctx context.Context, sourceID uint) error {
		return fetchNewsUseCase.ExecuteForSource(ctx, sourceID)
	}

	// 8. Ejecutar extracción inicial de noticias (para instalaciones nuevas)
	log.Println("Ejecutando extracción inicial de noticias...")
	if err := fetchFunc(ctx); err != nil {
		log.Printf("Error en la extracción inicial de noticias: %v", err)
	} else {
		log.Println("Extracción inicial de noticias completada exitosamente.")
	}

	// 9. Iniciar Cron Scheduler
	cronScheduler := infrastructure.NewCronScheduler(&simpleLogger{}, true, cfg.Cron.Expr)
	cronScheduler.ScheduleFetchNews(func() {
		log.Println("Ejecutando tarea cron de extracción de noticias...")
		if err := fetchFunc(context.Background()); err != nil {
			log.Printf("Error en la ejecución cron de extracción de noticias: %v", err)
		}
		log.Println("Tarea cron de extracción de noticias finalizada.")
	})
	cronScheduler.Start()
	log.Println("Cron scheduler iniciado.")

	// 10. Compilar assets del frontend automáticamente
	if err := buildFrontendAssets(); err != nil {
		log.Printf("⚠️  Advertencia: Error compilando assets del frontend: %v", err)
		log.Println("⚠️  El servidor continuará sin assets compilados")
	}

	// 11. Iniciar Servidor HTTP
	httpHandler := http_delivery.NewHandler(
		fetchFunc,
		fetchFuncForSource,
		newsItemRepo,
		categoryRepo,
		countryRepo,
		newsSourceRepo,
		fallbackImageRepo, // NUEVO
		rssFetcher,
	)
	log.Printf("Iniciando servidor HTTP en el puerto %d...", cfg.Server.HTTP.Port)
	http_delivery.StartHTTPServer(httpHandler, "./noticias", fmt.Sprintf("%d", cfg.Server.HTTP.Port))
}
