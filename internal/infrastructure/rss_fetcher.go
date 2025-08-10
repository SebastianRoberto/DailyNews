package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"dailynews/internal/domain"
	"dailynews/pkg/utils"
)

// rssFetcher implementa la interfaz RSSFetcher del dominio
type rssFetcher struct {
	parser *gofeed.Parser
}

// NewRSSFetcher crea una nueva instancia de RSSFetcher
func NewRSSFetcher() domain.RSSFetcher {
	return &rssFetcher{
		parser: gofeed.NewParser(),
	}
}

// Definición de patrones de extracción basados en los feeds reales
// PATRONES CON IMAGEN (existentes):
// patron1: title, media:content (con alternativa media:thumbnail), link, pubDate
// patron2: title, enclosure (con alternativa media:content), link, pubDate
// patron3: title, description_img (extraer imagen del HTML), link, pubDate
//
// PATRONES SIN IMAGEN (nuevos):
// patron1_no_image: title, link, pubDate (sin imagen)
// patron2_no_image: title, link, pubDate (sin imagen)
// patron3_no_image: title, link, pubDate (sin imagen)
var extractionPatterns = map[string]struct {
	TitleField string
	ImageField string
	LinkField  string
	DateField  string
}{
	// Patrones con imagen (existentes)
	"patron1": {"title", "media:content|media:thumbnail", "link", "pubDate"},
	"patron2": {"title", "enclosure|media:content", "link", "pubDate"},
	"patron3": {"title", "description_img", "link", "pubDate"},

	// Patrones sin imagen (nuevos)
	"patron1_no_image": {"title", "", "link", "pubDate"},
	"patron2_no_image": {"title", "", "link", "pubDate"},
	"patron3_no_image": {"title", "", "link", "pubDate"},
}

// Fetch obtiene noticias de una fuente RSS
func (f *rssFetcher) Fetch(ctx context.Context, url string, filter string, titleField, imageField, linkField, dateField string) ([]domain.NewsItem, error) {
	url = strings.TrimSpace(url)
	utils.AppInfo("RSS_FETCHER", "Iniciando extracción RSS", map[string]interface{}{
		"filter": filter,
		"url":    url,
	})

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	feed, err := f.parser.ParseURLWithContext(url, ctx)
	if err != nil {
		utils.SourceError(url, err.Error())
		return nil, fmt.Errorf("error al obtener feed RSS: %w", err)
	}

	utils.AppInfo("RSS_FETCHER", "Feed obtenido exitosamente", map[string]interface{}{
		"items_count": len(feed.Items),
		"url":         url,
	})

	var items []domain.NewsItem
	for i, item := range feed.Items {
		newsNum := i + 1
		var title, imageURL, linkURL string
		var pubDate time.Time
		var titleFormat, imageFormat, linkFormat, dateFormat string

		// ===== EXTRACCIÓN DE TÍTULO =====
		if titleField != "" {
			title = extractFieldFromItem(item, titleField)
			titleFormat = titleField
		} else {
			pattern := extractionPatterns[filter]
			title = extractFieldFromItem(item, pattern.TitleField)
			titleFormat = pattern.TitleField
		}

		if title == "" {
			utils.NewsWarn("", "", fmt.Sprintf("Noticia %d", newsNum), fmt.Sprintf("título fallido (%s) → noticia descartada", titleFormat))
			continue
		}

		// ===== EXTRACCIÓN DE IMAGEN =====
		if imageField != "" {
			imageURL = extractFieldFromItem(item, imageField)
			imageFormat = imageField
		} else {
			pattern := extractionPatterns[filter]
			// Solo extraer imagen si el patrón no es "sin imagen"
			if !strings.Contains(filter, "no_image") {
				imageURL = extractFieldFromItem(item, pattern.ImageField)
				imageFormat = pattern.ImageField
			} else {
				imageFormat = "no_image"
			}
		}

		// ===== EXTRACCIÓN DE LINK =====
		if linkField != "" {
			linkURL = extractFieldFromItem(item, linkField)
			linkFormat = linkField
		} else {
			pattern := extractionPatterns[filter]
			linkURL = extractFieldFromItem(item, pattern.LinkField)
			linkFormat = pattern.LinkField
		}

		if linkURL == "" {
			utils.NewsWarn("", "", fmt.Sprintf("Noticia %d", newsNum), fmt.Sprintf("link fallido (%s) → noticia descartada", linkFormat))
			continue
		}

		// ===== EXTRACCIÓN DE FECHA =====
		if dateField != "" {
			dateStr := extractFieldFromItem(item, dateField)
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				pubDate = t
				dateFormat = dateField
			} else {
				pubDate = time.Now()
				dateFormat = dateField + " (fallback)"
			}
		} else {
			pattern := extractionPatterns[filter]
			dateStr := extractFieldFromItem(item, pattern.DateField)
			if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
				pubDate = t
				dateFormat = pattern.DateField
			} else if item.PublishedParsed != nil {
				pubDate = *item.PublishedParsed
				dateFormat = "PublishedParsed"
			} else if item.UpdatedParsed != nil {
				pubDate = *item.UpdatedParsed
				dateFormat = "UpdatedParsed"
			} else {
				pubDate = time.Now()
				dateFormat = "current_time"
			}
		}

		newsItem := domain.NewsItem{
			Title:   cleanCDATA(title),
			Link:    linkURL,
			Image:   imageURL,
			PubDate: pubDate,
		}
		items = append(items, newsItem)

		// NO LOG - Eliminamos el log confuso de "Noticia X procesada"

		_ = imageFormat
		_ = dateFormat
	}

	utils.SourceProcessingComplete(url, len(items), len(feed.Items))
	return items, nil
}

// extractFieldFromItem extrae el campo solicitado del item, soportando alternativas con '|'
func extractFieldFromItem(item *gofeed.Item, field string) string {
	for _, f := range strings.Split(field, "|") {
		f = strings.TrimSpace(f)

		switch f {
		case "title":
			if item.Title != "" {
				return item.Title
			}
		case "media:content":
			if result := getMediaContent(item); result != "" {
				return result
			}
		case "media:thumbnail":
			if result := getMediaThumbnail(item); result != "" {
				return result
			}
		case "enclosure":
			if len(item.Enclosures) > 0 && strings.HasPrefix(item.Enclosures[0].Type, "image/") {
				return item.Enclosures[0].URL
			}
		case "description_img":
			if result := extractImgFromDescription(item.Description); result != "" {
				return result
			}
		case "link":
			if item.Link != "" {
				return item.Link
			}
		case "pubDate":
			if item.PublishedParsed != nil {
				return item.PublishedParsed.Format(time.RFC3339)
			}
			if item.UpdatedParsed != nil {
				return item.UpdatedParsed.Format(time.RFC3339)
			}
		}
	}
	return ""
}

// getMediaThumbnail busca media:thumbnail en las extensiones
func getMediaThumbnail(item *gofeed.Item) string {
	if ext, ok := item.Extensions["media"]; ok {
		if thumbs, ok := ext["thumbnail"]; ok && len(thumbs) > 0 {
			if url, ok := thumbs[0].Attrs["url"]; ok {
				return url
			}
		}
	}
	return ""
}

// getMediaContent busca media:content en las extensiones
func getMediaContent(item *gofeed.Item) string {
	if ext, ok := item.Extensions["media"]; ok {
		if contents, ok := ext["content"]; ok && len(contents) > 0 {
			if url, ok := contents[0].Attrs["url"]; ok {
				return url
			}
		}
	}
	return ""
}

// extractImgFromDescription busca la primera imagen en el HTML de la descripción
func extractImgFromDescription(desc string) string {
	start := strings.Index(desc, "<img ")
	if start == -1 {
		return ""
	}
	imgTag := desc[start:]
	end := strings.Index(imgTag, ">")
	if end == -1 {
		return ""
	}
	imgTag = imgTag[:end]
	srcIdx := strings.Index(imgTag, "src=")
	if srcIdx == -1 {
		return ""
	}
	quote := imgTag[srcIdx+4]
	rest := imgTag[srcIdx+5:]
	endQuote := strings.IndexRune(rest, rune(quote))
	if endQuote == -1 {
		return ""
	}
	return rest[:endQuote]
}

// cleanCDATA elimina CDATA y espacios
func cleanCDATA(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "<![CDATA[")
	s = strings.TrimSuffix(s, "]]>")
	return strings.TrimSpace(s)
}
