package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"dailynews/internal/domain"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config contiene la configuración para la conexión a la base de datos
type Config struct {
	Host         string
	Port         int
	User         string
	Password     string
	DatabaseName string
}

// DB es un envoltorio para la conexión a la base de datos
type DB struct {
	*gorm.DB
}

// New crea una nueva instancia de DB con lógica inteligente de creación de BD
func New(cfg Config) (*DB, error) {
	if cfg.Host == "" || cfg.User == "" || cfg.DatabaseName == "" {
		return nil, fmt.Errorf("configuración de base de datos incompleta")
	}

	// Paso 1: Conectar sin base de datos específica para crearla si no existe
	rootDsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
	)

	// Conectar a MySQL sin una base de datos específica
	sqlDB, err := sql.Open("mysql", rootDsn)
	if err != nil {
		return nil, fmt.Errorf("error al conectar a MySQL: %w", err)
	}
	defer sqlDB.Close()

	// Crear base de datos si no existe
	_, err = sqlDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", cfg.DatabaseName))
	if err != nil {
		return nil, fmt.Errorf("error al crear la base de datos: %w", err)
	}
	log.Printf("Base de datos '%s' verificada/creada correctamente", cfg.DatabaseName)

	// Paso 2: Conectar a la base de datos específica
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DatabaseName,
	)

	gormDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error), // Solo errores, no INSERT logs
	})
	if err != nil {
		return nil, fmt.Errorf("error al conectar a la base de datos: %w", err)
	}

	sqlDBGorm, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("error al obtener la instancia de sql.DB: %w", err)
	}

	// Configuración del pool de conexiones
	sqlDBGorm.SetMaxIdleConns(10)
	sqlDBGorm.SetMaxOpenConns(100)
	sqlDBGorm.SetConnMaxLifetime(time.Hour)

	log.Println("Conexión a la base de datos establecida")
	return &DB{gormDB}, nil
}

// Ping verifica la conexión a la base de datos
func (db *DB) Ping(ctx context.Context) error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("error al obtener la instancia de sql.DB: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("error al hacer ping a la base de datos: %w", err)
	}

	return nil
}

// Close cierra la conexión a la base de datos
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return fmt.Errorf("error al obtener la instancia de sql.DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("error al cerrar la conexión a la base de datos: %w", err)
	}

	return nil
}

// Migrate ejecuta las migraciones de la base de datos para crear las tablas necesarias.
func (db *DB) Migrate() error {
	if err := db.DB.AutoMigrate(
		&domain.Country{},
		&domain.Category{},
		&domain.NewsSource{},
		&domain.NewsItem{},
		&domain.FallbackImage{}, // NUEVO
	); err != nil {
		return fmt.Errorf("error al migrar la base de datos: %w", err)
	}

	log.Println("Migraciones de la base de datos completadas")
	return nil
}

// SeedInitialData inserta datos iniciales si no existen
func (db *DB) SeedInitialData(ctx context.Context) {
	createInitialCountries(ctx, db)
	createInitialCategories(ctx, db)
	createInitialNewsSources(ctx, db)
}

// createInitialCountries crea los países/idiomas iniciales si no existen
func createInitialCountries(ctx context.Context, db *DB) {
	countries := []domain.Country{
		{Code: "es", Name: "Español"},
		{Code: "en", Name: "English"},
		{Code: "fr", Name: "Français"},
	}

	for _, country := range countries {
		var count int64
		db.Model(&domain.Country{}).Where("code = ?", country.Code).Count(&count)
		if count == 0 {
			db.Create(&country)
			log.Printf("País/Idioma creado: %s", country.Name)
		}
	}
}

// createInitialCategories crea las categorías iniciales si no existen
func createInitialCategories(ctx context.Context, db *DB) {
	categories := []domain.Category{
		{Code: "technology", Name: "Technology"},
		{Code: "health", Name: "Health"},
		{Code: "sports", Name: "Sports"},
		{Code: "culture", Name: "Culture"},
		{Code: "international", Name: "International"},
		{Code: "entertainment", Name: "Entertainment"},
		{Code: "economy", Name: "Economy"},
		{Code: "breaking", Name: "Breaking News"},
	}

	for _, cat := range categories {
		var count int64
		db.Model(&domain.Category{}).Where("code = ?", cat.Code).Count(&count)
		if count == 0 {
			db.Create(&cat)
			log.Printf("Categoría creada: %s", cat.Name)
		}
	}
}

// createInitialNewsSources crea las fuentes RSS iniciales si no existen
func createInitialNewsSources(ctx context.Context, db *DB) {
	// Obtener categorías e idiomas para las relaciones
	var categories []domain.Category
	var countries []domain.Country
	db.Find(&categories)
	db.Find(&countries)

	if len(categories) == 0 || len(countries) == 0 {
		log.Println("No hay categorías o países disponibles para crear fuentes RSS")
		return
	}

	// Crear mapa de categorías por código
	categoryMap := make(map[string]domain.Category)
	for _, cat := range categories {
		categoryMap[cat.Code] = cat
	}

	// Crear mapa de países por código
	countryMap := make(map[string]domain.Country)
	for _, country := range countries {
		countryMap[country.Code] = country
	}

	// TODAS las fuentes RSS del init_db.sql
	sources := []domain.NewsSource{
		// SPORTS - patron1
		// El País Deportes: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://imagenes.elpais.com/resizer/v2/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["sports"].ID,
			SourceName: "El País Deportes",
			RSSURL:     "https://feeds.elpais.com/mrss-s/pages/ep/site/elpais.com/section/deportes/portada",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// ABC Fútbol: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://s1.abcstatics.com/abc/www/multimedia/deportes/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["sports"].ID,
			SourceName: "ABC Fútbol",
			RSSURL:     "https://www.abc.es/rss/2.0/deportes/futbol/",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// ABC Real Madrid: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://s3.abcstatics.com/abc/www/multimedia/deportes/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["sports"].ID,
			SourceName: "ABC Real Madrid",
			RSSURL:     "https://www.abc.es/rss/2.0/deportes/real-madrid/",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// SPORTS - patron2
		// La Vanguardia Deportes: Título en <title>, imagen en <enclosure url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <enclosure type="image/jpeg" url="https://www.lavanguardia.com/files/og_thumbnail/...">
		{
			NewsID:     categoryMap["sports"].ID,
			SourceName: "La Vanguardia Deportes",
			RSSURL:     "https://www.lavanguardia.com/rss/deportes.xml",
			Filter:     stringPtr("patron2"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// France24 Deportes: Título en <title>, imagen en <media:thumbnail url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:thumbnail url="https://s.france24.com/media/display/...">
		{
			NewsID:     categoryMap["sports"].ID,
			SourceName: "France24 Deportes",
			RSSURL:     "https://www.france24.com/es/deportes/rss",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// TECHNOLOGY - patron1
		// El País Tecnología: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://imagenes.elpais.com/resizer/v2/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["technology"].ID,
			SourceName: "El País Tecnología",
			RSSURL:     "https://feeds.elpais.com/mrss-s/pages/ep/site/elpais.com/section/tecnologia/portada",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// ABC Tecnología: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://s3.abcstatics.com/abc/www/multimedia/tecnologia/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["technology"].ID,
			SourceName: "ABC Tecnología",
			RSSURL:     "https://www.abc.es/rss/2.0/tecnologia/",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// El País Ciencia: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://imagenes.elpais.com/resizer/v2/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["technology"].ID,
			SourceName: "El País Ciencia",
			RSSURL:     "https://feeds.elpais.com/mrss-s/pages/ep/site/elpais.com/section/ciencia/portada",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// TECNOLOGÍA - patron2
		// La Vanguardia Tecnología: Título en <title>, imagen en <enclosure url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <enclosure type="image/jpeg" length="..." url="https://www.lavanguardia.com/files/og_thumbnail/...">
		{
			NewsID:     categoryMap["technology"].ID,
			SourceName: "La Vanguardia Tecnología",
			RSSURL:     "https://www.lavanguardia.com/rss/tecnologia.xml",
			Filter:     stringPtr("patron2"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// TECNOLOGÍA - patron3
		// Xataka: Título en <title>, imagen en <description_img> (extraída del HTML), link en <link>, fecha en <pubDate>
		// Estructura: <description><![CDATA[<p><img src="https://i.blogs.es/..." alt="...">]]></description>
		{
			NewsID:     categoryMap["technology"].ID,
			SourceName: "Xataka",
			RSSURL:     "https://www.xataka.com/feedburner.xml",
			Filter:     stringPtr("patron3"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// SALUD - patron1
		// Mejor con Salud: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content medium="image" type="image/jpeg" url="https://mejorconsalud.as.com/wp-content/uploads/..." height="586" width="880"/>
		{
			NewsID:     categoryMap["health"].ID,
			SourceName: "Mejor con Salud",
			RSSURL:     "https://mejorconsalud.as.com/feed/",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// HEALTH - patron1
		// ABC Alimentación: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://s3.abcstatics.com/abc/www/multimedia/bienestar/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["health"].ID,
			SourceName: "ABC Alimentación",
			RSSURL:     "https://www.abc.es/rss/2.0/bienestar/alimentacion/",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// ABC Fitness: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://s1.abcstatics.com/abc/www/multimedia/bienestar/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["health"].ID,
			SourceName: "ABC Fitness",
			RSSURL:     "https://www.abc.es/rss/2.0/bienestar/fitness/",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// HEALTH - patron2
		// La Vanguardia Salud: Título en <title>, imagen en <enclosure url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <enclosure type="image/jpeg" length="..." url="https://www.lavanguardia.com/files/og_thumbnail/...">
		{
			NewsID:     categoryMap["health"].ID,
			SourceName: "La Vanguardia Salud",
			RSSURL:     "https://www.lavanguardia.com/rss/vida/salud.xml",
			Filter:     stringPtr("patron2"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// INTERNATIONAL - patron2
		// France24 Internacional: Título en <title>, imagen en <media:thumbnail url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:thumbnail url="https://s.france24.com/media/display/..." width="1024" height="576"/>
		{
			NewsID:     categoryMap["international"].ID,
			SourceName: "France24 Internacional",
			RSSURL:     "https://www.france24.com/es/econom%C3%ADa/rss",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// La Vanguardia Internacional: Título en <title>, imagen en <enclosure url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <enclosure type="image/jpeg" length="..." url="https://www.lavanguardia.com/files/og_thumbnail/...">
		{
			NewsID:     categoryMap["international"].ID,
			SourceName: "La Vanguardia Internacional",
			RSSURL:     "https://www.lavanguardia.com/rss/internacional.xml",
			Filter:     stringPtr("patron2"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// INTERNATIONAL - patron1
		// El País América: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://imagenes.elpais.com/resizer/v2/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["international"].ID,
			SourceName: "El País América",
			RSSURL:     "https://feeds.elpais.com/mrss-s/pages/ep/site/elpais.com/section/america/portada",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// INTERNATIONAL - campos personalizados
		// ElDiario.es Internacional: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://static.eldiario.es/clip/..." type="image/jpeg" fileSize="..." width="..." height="..."/>
		{
			NewsID:     categoryMap["international"].ID,
			SourceName: "ElDiario.es Internacional",
			RSSURL:     "https://www.eldiario.es/rss/internacional/",
			Filter:     stringPtr(""),
			TitleField: stringPtr("title"),
			ImageField: stringPtr("media:content"),
			LinkField:  stringPtr("link"),
			CampoFecha: stringPtr("pubDate"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// CULTURE - patron1
		// El País Cultura: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://imagenes.elpais.com/resizer/v2/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["culture"].ID,
			SourceName: "El País Cultura",
			RSSURL:     "https://feeds.elpais.com/mrss-s/pages/ep/site/elpais.com/section/cultura/portada",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// ABC Cultura Música: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://s3.abcstatics.com/abc/www/multimedia/cultura/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["culture"].ID,
			SourceName: "ABC Cultura Música",
			RSSURL:     "https://www.abc.es/rss/2.0/cultura/musica/",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// ABC Cultura Cultural: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://s1.abcstatics.com/abc/www/multimedia/cultura/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["culture"].ID,
			SourceName: "ABC Cultura Cultural",
			RSSURL:     "https://www.abc.es/rss/2.0/cultura/cultural/",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// CULTURE - patron2
		// La Vanguardia Cultura: Título en <title>, imagen en <enclosure url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <enclosure type="image/jpeg" url="https://www.lavanguardia.com/files/og_thumbnail/...">
		{
			NewsID:     categoryMap["culture"].ID,
			SourceName: "La Vanguardia Cultura",
			RSSURL:     "https://www.lavanguardia.com/rss/cultura.xml",
			Filter:     stringPtr("patron2"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// ECONOMY - patron1
		// Expansión Portada: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://e00-expansion.uecdn.es/assets/multimedia/imagenes/..." medium="image" width="2048" height="951"/>
		{
			NewsID:     categoryMap["economy"].ID,
			SourceName: "Expansión Portada",
			RSSURL:     "https://e00-expansion.uecdn.es/rss/portada.xml",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// El País Economía: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://imagenes.elpais.com/resizer/v2/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["economy"].ID,
			SourceName: "El País Economía",
			RSSURL:     "https://feeds.elpais.com/mrss-s/pages/ep/site/elpais.com/section/economia/portada",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// ABC Economía: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://s3.abcstatics.com/abc/www/multimedia/economia/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["economy"].ID,
			SourceName: "ABC Economía",
			RSSURL:     "https://www.abc.es/rss/2.0/economia/",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// ECONOMY - patron2
		// La Vanguardia Economía: Título en <title>, imagen en <enclosure url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <enclosure type="image/jpeg" length="..." url="https://www.lavanguardia.com/files/og_thumbnail/...">
		{
			NewsID:     categoryMap["economy"].ID,
			SourceName: "La Vanguardia Economía",
			RSSURL:     "https://www.lavanguardia.com/rss/economia.xml",
			Filter:     stringPtr("patron2"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// ENTERTAINMENT - patron1
		// El País Gente: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://imagenes.elpais.com/resizer/v2/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["entertainment"].ID,
			SourceName: "El País Gente",
			RSSURL:     "https://feeds.elpais.com/mrss-s/pages/ep/site/elpais.com/section/gente/portada",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// ABC Series: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://s3.abcstatics.com/abc/www/multimedia/play/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["entertainment"].ID,
			SourceName: "ABC Series",
			RSSURL:     "https://www.abc.es/rss/2.0/play/series/",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// ABC Cine: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://s2.abcstatics.com/abc/www/multimedia/play/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["entertainment"].ID,
			SourceName: "ABC Cine",
			RSSURL:     "https://www.abc.es/rss/2.0/play/cine/",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// BREAKING - patron1
		// El País Lo Más Visto: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://imagenes.elpais.com/resizer/v2/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["breaking"].ID,
			SourceName: "El País Lo Más Visto",
			RSSURL:     "https://feeds.elpais.com/mrss-s/pages/ep/site/elpais.com/section/lo-mas-visto/portada",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// El País Últimas Noticias: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://imagenes.elpais.com/resizer/v2/..." type="image/jpeg" medium="image">
		{
			NewsID:     categoryMap["breaking"].ID,
			SourceName: "El País Últimas Noticias",
			RSSURL:     "https://feeds.elpais.com/mrss-s/pages/ep/site/elpais.com/section/ultimas-noticias/portada",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// BREAKING - patron2
		// La Vanguardia Portada: Título en <title>, imagen en <enclosure url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <enclosure type="image/jpeg" length="..." url="https://www.lavanguardia.com/files/og_thumbnail/...">
		{
			NewsID:     categoryMap["breaking"].ID,
			SourceName: "La Vanguardia Portada",
			RSSURL:     "https://www.lavanguardia.com/rss/home.xml",
			Filter:     stringPtr("patron2"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// BREAKING - patron1
		// El Mundo Portada: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://e00-elmundo.uecdn.es/assets/multimedia/imagenes/..." medium="image" width="..." height="..."/>
		{
			NewsID:     categoryMap["breaking"].ID,
			SourceName: "El Mundo Portada",
			RSSURL:     "https://e00-elmundo.uecdn.es/elmundo/rss/portada.xml",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["es"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// ===== FUENTES RSS EN INGLÉS =====

		// ECONOMY - patron1 (ENGLISH)
		// Financial Times: Título en <title>, imagen en <media:thumbnail url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:thumbnail url="https://www.ft.com/__origami/service/image/v2/images/raw/..."/>
		{
			NewsID:     categoryMap["economy"].ID,
			SourceName: "Financial Times",
			RSSURL:     "https://www.ft.com/rss/home",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["en"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// SPORTS - patron1 (ENGLISH)
		// Fox Sports: Título en <title>, imagen en <media:content url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <media:content url="https://statics.foxsports.com/www.foxsports.com/content/uploads/..." expression="full" type="image/jpg">
		{
			NewsID:     categoryMap["sports"].ID,
			SourceName: "Fox Sports",
			RSSURL:     "https://api.foxsports.com/v2/content/optimized-rss?partnerKey=MB0Wehpmuj2lUhuRhQaafhBjAJqaPU244mlTDK1i&aggregateId=7f83e8ca-6701-5ea0-96ee-072636b67336",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["en"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// TECHNOLOGY - patron1 (ENGLISH)
		// The New York Times Technology: Título en <title>, imagen en <media:content url="..."> cuando está presente, link en <link>, fecha en <pubDate>
		// Estructura: mezcla de items con/si n <media:content>, por lo que usamos patron1 (media:*)
		{
			NewsID:     categoryMap["technology"].ID,
			SourceName: "The New York Times Technology",
			RSSURL:     "https://rss.nytimes.com/services/xml/rss/nyt/Technology.xml",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["en"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// CULTURE - patron1 (ENGLISH)
		// BBC Entertainment & Arts: Título en <title>, imagen en <media:thumbnail>, link en <link>, fecha en <pubDate>
		// Estructura: usa <media:thumbnail>, por lo que patron1 (media:content|media:thumbnail) encaja
		{
			NewsID:     categoryMap["culture"].ID,
			SourceName: "BBC Entertainment & Arts",
			RSSURL:     "https://feeds.bbci.co.uk/news/entertainment_and_arts/rss.xml",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["en"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
		// TECHNOLOGY - patron1 (ENGLISH)
		// CNET News: Título en <title>, imagen en <media:content|media:thumbnail>, link en <link>, fecha en <pubDate>
		// Estructura: incluye <media:thumbnail> y a menudo <media:content>, por lo que patron1 encaja
		{
			NewsID:     categoryMap["technology"].ID,
			SourceName: "CNET News",
			RSSURL:     "https://www.cnet.com/rss/news/",
			Filter:     stringPtr("patron1"),
			LangID:     countryMap["en"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// ENTERTAINMENT - patron2 (ENGLISH)
		// Sky News Entertainment: Título en <title>, imagen en <enclosure url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <enclosure url="https://e3.365dm.com/24/08/1920x1080/..." length="0" type="image/jpeg"/>
		{
			NewsID:     categoryMap["entertainment"].ID,
			SourceName: "Sky News Entertainment",
			RSSURL:     "https://feeds.skynews.com/feeds/rss/entertainment.xml",
			Filter:     stringPtr("patron2"),
			LangID:     countryMap["en"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// HEALTH - patron3 (ENGLISH)
		// MedPage Today: Título en <title>, imagen extraída del HTML en <description>, link en <link>, fecha en <pubDate>
		// Estructura: <description><![CDATA[ <img src="https://clf1.medpagetoday.com/media/images/116xxx/116846.jpg"/> ]]></description>
		{
			NewsID:     categoryMap["health"].ID,
			SourceName: "MedPage Today",
			RSSURL:     "https://www.medpagetoday.com/rss/headlines.xml",
			Filter:     stringPtr("patron3"),
			LangID:     countryMap["en"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// INTERNATIONAL - patron2 (ENGLISH)
		// Sky News World: Título en <title>, imagen en <enclosure url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <enclosure url="https://e3.365dm.com/25/08/1920x1080/..." length="0" type="image/jpeg"/>
		{
			NewsID:     categoryMap["international"].ID,
			SourceName: "Sky News World",
			RSSURL:     "https://feeds.skynews.com/feeds/rss/world.xml",
			Filter:     stringPtr("patron2"),
			LangID:     countryMap["en"].ID,
			IsActive:   true,
			UserAdded:  false,
		},

		// BREAKING - patron2 (ENGLISH)
		// Sky News Home: Título en <title>, imagen en <enclosure url="...">, link en <link>, fecha en <pubDate>
		// Estructura: <enclosure url="https://e3.365dm.com/25/03/1920x1080/..." length="0" type="image/jpeg"/>
		{
			NewsID:     categoryMap["breaking"].ID,
			SourceName: "Sky News Home",
			RSSURL:     "https://feeds.skynews.com/feeds/rss/home.xml",
			Filter:     stringPtr("patron2"),
			LangID:     countryMap["en"].ID,
			IsActive:   true,
			UserAdded:  false,
		},
	}

	for _, source := range sources {
		var count int64
		db.Model(&domain.NewsSource{}).Where("rss_url = ?", source.RSSURL).Count(&count)
		if count == 0 {
			db.Create(&source)
			log.Printf("Fuente RSS creada: %s", source.SourceName)
		}
	}
}

// Helper para crear punteros a string
func stringPtr(s string) *string {
	return &s
}

// getEnv obtiene una variable de entorno o un valor por defecto
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt obtiene una variable de entorno como entero o un valor por defecto
func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// NewFromEnv crea una nueva instancia de DB desde variables de entorno
func NewFromEnv() (*DB, error) {
	cfg := Config{
		Host:         getEnv("DB_HOST", "localhost"),
		Port:         getEnvAsInt("DB_PORT", 3306),
		User:         getEnv("DB_USER", "root"),
		Password:     getEnv("DB_PASSWORD", ""),
		DatabaseName: getEnv("DB_NAME", "getnews"),
	}

	return New(cfg)
}
