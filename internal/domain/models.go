package domain

import (
	"time"
)

// Country representa un país o idioma soportado en el sistema
type Country struct {
	ID   uint   `gorm:"primaryKey"`              // Identificador único del país/idioma
	Code string `gorm:"size:10;unique;not null"` // Código identificativo del país/idioma (ej: "es", "eng")
	Name string `gorm:"size:50"`                 // Nombre descriptivo (ej: "España", "Inglés")
}

// TableName especifica el nombre de la tabla para el modelo Country
func (Country) TableName() string {
	return "template_country"
}

// Category representa una categoría de noticias
type Category struct {
	ID   uint   `gorm:"primaryKey"`              // Identificador único de la categoría
	Code string `gorm:"size:50;unique;not null"` // Código de la categoría (ej: "technology", "sports")
	Name string `gorm:"size:100"`                // Nombre descriptivo de la categoría
}

// TableName especifica el nombre de la tabla para el modelo Category
func (Category) TableName() string {
	return "template_news"
}

// NewsSource representa una fuente RSS de noticias
type NewsSource struct {
	ID              uint     `gorm:"primaryKey"` // Identificador único de la fuente RSS
	NewsID          uint     `gorm:"not null"`   // ID de la categoría asociada
	News            Category // Relación con la categoría
	SourceName      string   `gorm:"size:100"`           // Nombre de la fuente (ej: "BBC Mundo")
	RSSURL          string   `gorm:"type:text;not null"` // URL del feed RSS
	Filter          *string  `gorm:"type:text"`          // Identificador de patrón de extracción ("patron1", "patron2", etc.)
	TitleField      *string  `gorm:"size:255"`           // Campo personalizado para el titular (si el RSS es único)
	ImageField      *string  `gorm:"size:255"`           // Campo personalizado para la imagen
	LinkField       *string  `gorm:"size:255"`           // Campo personalizado para el link
	CampoFecha      *string  `gorm:"size:255"`           // Campo personalizado para la fecha
	LangID          uint     `gorm:"not null"`           // ID del país/idioma asociado
	Lang            Country  // Relación con el país/idioma
	IsActive        bool     `gorm:"default:true"`  // Lo de is IsActive esta pensado para que en un futuro el usuario pueda desactivar fuentes por defecto.
	UserAdded       bool     `gorm:"default:false"` // Indica si la fuente fue agregada por el usuario
	FallbackImageID *uint    `gorm:"index"`         // NUEVO: FK a FallbackImage
}

// TableName especifica el nombre de la tabla para el modelo NewsSource
func (NewsSource) TableName() string {
	return "template_news_sources"
}

// NewsItem representa una noticia procesada
type NewsItem struct {
	ID           uint       `gorm:"primaryKey"`          // Identificador único de la noticia
	SourceID     uint       `gorm:"not null"`            // ID de la fuente RSS de origen
	Source       NewsSource `gorm:"foreignKey:SourceID"` // Relación con la fuente RSS
	Title        string     `gorm:"type:text;not null"`  // Titular de la noticia
	Link         string     `gorm:"type:text;not null"`  // Link a la noticia original
	Image        string     `gorm:"type:text;not null"`  // URL de la imagen principal
	PubDate      time.Time  `gorm:"not null"`            // Fecha de publicación de la noticia
	LangCode     string     `gorm:"size:10;not null"`    // Código de idioma (ej: "es", "en")
	CategoryCode string     `gorm:"size:50;not null"`    // Código de categoría (ej: "technology")
	CreatedAt    time.Time  `gorm:"autoCreateTime"`      // Fecha de creación en el sistema
}

// TableName especifica el nombre de la tabla para el modelo NewsItem
func (NewsItem) TableName() string {
	return "news_items"
}

// NewsItemDTO es una representación simplificada de NewsItem para la API
type NewsItemDTO struct {
	ID       uint      `json:"id"`        // Identificador único de la noticia
	Title    string    `json:"title"`     // Titular de la noticia
	Link     string    `json:"link"`      // Link a la noticia original
	Image    string    `json:"image"`     // URL de la imagen principal
	Source   string    `json:"source"`    // Nombre de la fuente RSS
	Date     time.Time `json:"date"`      // Fecha de publicación
	LangCode string    `json:"lang_code"` // Código de idioma
	Category string    `json:"category"`  // Código de categoría
}

// ToDTO convierte un NewsItem a NewsItemDTO
func (n *NewsItem) ToDTO() *NewsItemDTO {
	return &NewsItemDTO{
		ID:       n.ID,
		Title:    n.Title,
		Link:     n.Link,
		Image:    n.Image,
		Source:   n.Source.SourceName,
		Date:     n.PubDate,
		LangCode: n.LangCode,
		Category: n.CategoryCode,
	}
}

// FallbackImage representa una imagen de respaldo para una categoría+idioma
type FallbackImage struct {
	ID           uint      `gorm:"primaryKey"`
	CategoryCode string    `gorm:"size:50;not null;index"`
	LanguageCode string    `gorm:"size:10;not null;index"`
	Filename     string    `gorm:"size:255;not null"`
	OriginalName string    `gorm:"size:255;not null"`
	MimeType     string    `gorm:"size:100;not null"`
	FileSize     int64     `gorm:"not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

// TableName especifica el nombre de la tabla para el modelo FallbackImage
func (FallbackImage) TableName() string {
	return "fallback_images"
}

// GetNewsItemField permite obtener campos dinámicamente de un NewsItem
func GetNewsItemField(item *NewsItem, field string) string {
	switch field {
	case "title", "titulo":
		return item.Title
	case "image", "imagen", "imagesrc":
		return item.Image
	case "link", "enlace":
		return item.Link
	case "date", "fecha":
		return item.PubDate.Format("2006-01-02T15:04:05Z07:00")
	default:
		return ""
	}
}
