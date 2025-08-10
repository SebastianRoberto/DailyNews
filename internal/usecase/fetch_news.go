package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"dailynews/internal/domain"
	"dailynews/pkg/config"
	"dailynews/pkg/utils"
)

var blacklist = []string{"oróscopo", "horóscopo"}

func cleanText(text string) string {
	// Eliminar etiquetas HTML
	re := regexp.MustCompile("<[^>]*>")
	text = re.ReplaceAllString(text, "")
	// Decodifica entidades HTML como &nbsp; &amp; etc.
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&#160;", " ") // Espacio no separador
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&apos;", "'")
	// Reemplaza múltiples espacios con uno solo
	text = strings.Join(strings.Fields(text), " ")
	return strings.TrimSpace(text)
}

func isBlacklisted(title string) bool {
	title = strings.ToLower(title)
	for _, word := range blacklist {
		if strings.Contains(title, word) {
			return true
		}
	}
	return false
}

// FetchNewsUseCase orquesta la extracción, validación y almacenamiento de noticias.
type FetchNewsUseCase struct {
	newsItemRepo      domain.NewsItemRepository
	categoryRepo      domain.CategoryRepository
	countryRepo       domain.CountryRepository
	newsSourceRepo    domain.NewsSourceRepository
	fallbackImageRepo domain.FallbackImageRepository // NUEVO
	rssFetcher        domain.RSSFetcher
	imageDownloader   domain.ImageDownloader
	config            *config.Config
}

// NewFetchNewsUseCase crea una nueva instancia de FetchNewsUseCase.
func NewFetchNewsUseCase(
	newsItemRepo domain.NewsItemRepository,
	categoryRepo domain.CategoryRepository,
	countryRepo domain.CountryRepository,
	newsSourceRepo domain.NewsSourceRepository,
	fallbackImageRepo domain.FallbackImageRepository, // NUEVO
	rssFetcher domain.RSSFetcher,
	imageDownloader domain.ImageDownloader,
	config *config.Config,
) *FetchNewsUseCase {
	return &FetchNewsUseCase{
		newsItemRepo:      newsItemRepo,
		categoryRepo:      categoryRepo,
		countryRepo:       countryRepo,
		newsSourceRepo:    newsSourceRepo,
		fallbackImageRepo: fallbackImageRepo, // NUEVO
		rssFetcher:        rssFetcher,
		imageDownloader:   imageDownloader,
		config:            config,
	}
}

// Execute ejecuta el caso de uso.
func (uc *FetchNewsUseCase) Execute(ctx context.Context) error {
	utils.AppInfo("FETCH_NEWS", "Iniciando proceso de extracción de noticias", nil)

	// Limpiar noticias anteriores para evitar sobreescritura
	if err := uc.cleanOldNews(ctx); err != nil {
		utils.AppWarn("FETCH_NEWS", "Error limpiando noticias anteriores", map[string]interface{}{
			"error": err.Error(),
		})
	}

	sources, err := uc.newsSourceRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("error al obtener las fuentes de noticias: %w", err)
	}

	utils.AppInfo("FETCH_NEWS", "Fuentes RSS obtenidas", map[string]interface{}{
		"total_sources": len(sources),
	})

	groups := make(map[string][]domain.NewsSource) // key: <categoryCode>_<langCode>
	for _, src := range sources {
		lang := src.Lang.Code
		cat := src.News.Code
		key := cat + "_" + lang
		groups[key] = append(groups[key], src)
	}

	for key, groupSources := range groups {
		parts := strings.SplitN(key, "_", 2)
		if len(parts) != 2 {
			continue
		}
		cat, lang := parts[0], parts[1]
		tope := uc.getNewsCount(lang, cat)

		// Log de inicio de procesamiento con color por categoría
		utils.ProcessingInfo(cat, lang, tope, len(groupSources))

		var noticias []domain.NewsItem
		// Usar la configuración dinámica por categoría+idioma
		maxDays := uc.config.GetMaxDays(lang, cat)
		if len(groupSources) <= 3 {
			// Para categorías con pocas fuentes, usar el límite extendido
			extendedDays := uc.config.Filters.MaxDaysForNewsWithFewSources
			if extendedDays > maxDays {
				maxDays = extendedDays
			}
		}

		linksVistos := make(map[string]struct{})
		titulosVistos := make(map[string]struct{})
		descartadas := 0
		sourceCounts := make(map[string]int) // Contador por fuente para maxPerSource

		for _, src := range groupSources {
			utils.SourceProcessing(src.SourceName, src.RSSURL)

			// Llamar a Fetch pasando el patrón y los campos personalizados
			feedItems, err := uc.rssFetcher.Fetch(
				ctx,
				src.RSSURL,
				getString(src.Filter),
				getString(src.TitleField),
				getString(src.ImageField),
				getString(src.LinkField),
				getString(src.CampoFecha),
			)
			if err != nil {
				utils.SourceError(src.RSSURL, err.Error())
				continue
			}

			sourceValidCount := 0
			maxPerSource := uc.config.GetMaxPerSource(lang, cat)

			for _, item := range feedItems {
				if len(noticias) >= tope {
					utils.LimitReached(cat, lang)
					break
				}

				// Verificar límite por fuente
				if sourceCounts[src.SourceName] >= maxPerSource {
					utils.SourceLimitReached(src.SourceName, maxPerSource)
					break
				}

				titulo := item.Title
				imagen := item.Image
				link := item.Link
				fecha := item.PubDate
				tituloLimpio := cleanText(titulo)

				// Validaciones con logs específicos
				if isBlacklisted(tituloLimpio) {
					utils.NewsWarn(cat, lang, tituloLimpio, "título en lista negra")
					descartadas++
					continue
				}

				if len(tituloLimpio) < uc.config.Filters.MinTitle || len(tituloLimpio) > uc.config.Filters.MaxTitle {
					utils.NewsWarn(cat, lang, tituloLimpio, fmt.Sprintf("título inválido por longitud: %d caracteres", len(tituloLimpio)))
					descartadas++
					continue
				}

				// Verificar duplicados
				if _, exists := linksVistos[link]; exists {
					utils.NewsWarn(cat, lang, tituloLimpio, "duplicada o paquete lleno")
					descartadas++
					continue
				}
				if _, exists := titulosVistos[tituloLimpio]; exists {
					utils.NewsWarn(cat, lang, tituloLimpio, "duplicada o paquete lleno")
					descartadas++
					continue
				}

				// Verificar edad de la noticia
				antiguedad := time.Since(fecha)
				if antiguedad > time.Duration(maxDays)*24*time.Hour {
					utils.NewsWarn(cat, lang, tituloLimpio, fmt.Sprintf("noticia antigua, ideal: %d días, antigüedad: %.1f días", maxDays, antiguedad.Hours()/24))
					descartadas++
					continue
				}

				// Validar imagen
				if imagen == "" {
					// Si no hay imagen y el patrón es sin imagen, usar fallback
					if strings.Contains(getString(src.Filter), "no_image") {
						fallbackImage := uc.getFallbackImage(ctx, cat, lang)
						if fallbackImage != "" {
							imagen = fallbackImage
							utils.NewsInfo(cat, lang, tituloLimpio, src.SourceName, map[string]interface{}{
								"using_fallback": true,
								"fallback_image": fallbackImage,
							})
						} else {
							utils.NewsWarn(cat, lang, tituloLimpio, "sin imagen y sin fallback configurado")
							descartadas++
							continue
						}
					} else {
						utils.NewsWarn(cat, lang, tituloLimpio, "imagen no encontrada")
						descartadas++
						continue
					}
				}

				// Validar imagen (excepto si es una imagen de fallback local)
				if !strings.Contains(imagen, "/images/fallback/") {
					valid, err := uc.imageDownloader.ValidateImage(imagen)
					if err != nil {
						utils.NewsError(cat, lang, tituloLimpio, fmt.Sprintf("error al procesar imagen: %s", err.Error()))
						descartadas++
						continue
					}
					if !valid {
						utils.NewsWarn(cat, lang, tituloLimpio, "imagen inválida")
						descartadas++
						continue
					}
				} else {
					// Para imágenes de fallback, solo verificar que el archivo existe
					projectRoot := uc.getProjectRoot()
					imagePath := filepath.Join(projectRoot, "frontend", "assets", "images", "fallback", filepath.Base(imagen))
					if _, err := os.Stat(imagePath); os.IsNotExist(err) {
						utils.NewsWarn(cat, lang, tituloLimpio, "imagen de fallback no encontrada en disco")
						descartadas++
						continue
					}
					utils.NewsInfo(cat, lang, tituloLimpio, src.SourceName, map[string]interface{}{
						"fallback_validated": true,
						"image_path":         imagePath,
					})
				}

				// Crear noticia para la BD
				newsItem := domain.NewsItem{
					Title:        tituloLimpio,
					Link:         link,
					Image:        imagen,
					PubDate:      fecha,
					LangCode:     lang,
					CategoryCode: cat,
					SourceID:     src.ID,
					Source:       src,
				}

				// Guardar en la BD
				if err := uc.newsItemRepo.Create(ctx, &newsItem); err != nil {
					utils.NewsError(cat, lang, tituloLimpio, fmt.Sprintf("error guardando en BD: %s", err.Error()))
					continue
				}

				noticias = append(noticias, newsItem)
				linksVistos[link] = struct{}{}
				titulosVistos[tituloLimpio] = struct{}{}
				sourceValidCount++
				sourceCounts[src.SourceName]++ // Incrementar contador por fuente

				// Log de noticia añadida con formato limpio
				utils.NewsInfo(cat, lang, tituloLimpio, src.SourceName, map[string]interface{}{
					"count": len(noticias),
				})
			}

			// Log de finalización de fuente
			if sourceValidCount == 0 {
				utils.NoValidNewsFromSource(src.SourceName, "todas las noticias fueron descartadas")
			} else {
				utils.SourceProcessingComplete(src.SourceName, sourceValidCount, len(feedItems))
			}

			if len(noticias) >= tope {
				break
			}
		}

		// Log de finalización de categoría
		utils.ProcessingComplete(cat, lang, len(noticias), descartadas)
	}

	utils.AppInfo("FETCH_NEWS", "Proceso de extracción finalizado exitosamente", nil)
	return nil
}

// ExecuteForSource extrae noticias de una fuente específica
func (uc *FetchNewsUseCase) ExecuteForSource(ctx context.Context, sourceID uint) error {
	utils.AppInfo("FETCH_NEWS_SOURCE", "Iniciando extracción de noticias para fuente específica", map[string]interface{}{
		"source_id": sourceID,
	})

	// Obtener la fuente específica
	source, err := uc.newsSourceRepo.FindByID(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("error al obtener la fuente: %w", err)
	}

	if source == nil {
		return fmt.Errorf("fuente no encontrada")
	}

	utils.AppInfo("FETCH_NEWS_SOURCE", "Fuente encontrada", map[string]interface{}{
		"source_id":   source.ID,
		"source_name": source.SourceName,
		"category":    source.News.Code,
		"language":    source.Lang.Code,
	})

	// Obtener configuración para esta categoría+idioma
	cat := source.News.Code
	lang := source.Lang.Code
	maxDays := uc.config.GetMaxDays(lang, cat)
	maxPerSource := uc.config.GetMaxPerSource(lang, cat)

	utils.AppInfo("FETCH_NEWS_SOURCE", "Configuración de extracción", map[string]interface{}{
		"max_days":       maxDays,
		"max_per_source": maxPerSource,
	})

	// Obtener noticias del RSS
	feedItems, err := uc.rssFetcher.Fetch(
		ctx,
		source.RSSURL,
		getString(source.Filter),
		getString(source.TitleField),
		getString(source.ImageField),
		getString(source.LinkField),
		getString(source.CampoFecha),
	)
	if err != nil {
		return fmt.Errorf("error obteniendo RSS: %w", err)
	}

	utils.AppInfo("FETCH_NEWS_SOURCE", "Items RSS obtenidos", map[string]interface{}{
		"total_items": len(feedItems),
	})

	// Procesar items
	extractedCount := 0
	linksVistos := make(map[string]struct{})
	titulosVistos := make(map[string]struct{})

	for _, item := range feedItems {
		if extractedCount >= maxPerSource {
			utils.AppInfo("FETCH_NEWS_SOURCE", "Límite por fuente alcanzado", map[string]interface{}{
				"max_per_source": maxPerSource,
			})
			break
		}

		titulo := item.Title
		imagen := item.Image
		link := item.Link
		fecha := item.PubDate
		tituloLimpio := cleanText(titulo)

		// Validaciones básicas
		if isBlacklisted(tituloLimpio) {
			utils.AppWarn("FETCH_NEWS_SOURCE", "Título en lista negra", map[string]interface{}{
				"title": tituloLimpio,
			})
			continue
		}

		if len(tituloLimpio) < uc.config.Filters.MinTitle || len(tituloLimpio) > uc.config.Filters.MaxTitle {
			utils.AppWarn("FETCH_NEWS_SOURCE", "Título inválido por longitud", map[string]interface{}{
				"title":  tituloLimpio,
				"length": len(tituloLimpio),
				"min":    uc.config.Filters.MinTitle,
				"max":    uc.config.Filters.MaxTitle,
			})
			continue
		}

		// Verificar duplicados
		if _, exists := linksVistos[link]; exists {
			utils.AppWarn("FETCH_NEWS_SOURCE", "Link duplicado", map[string]interface{}{
				"link": link,
			})
			continue
		}
		if _, exists := titulosVistos[tituloLimpio]; exists {
			utils.AppWarn("FETCH_NEWS_SOURCE", "Título duplicado", map[string]interface{}{
				"title": tituloLimpio,
			})
			continue
		}

		// Verificar edad de la noticia
		antiguedad := time.Since(fecha)
		if antiguedad > time.Duration(maxDays)*24*time.Hour {
			utils.AppWarn("FETCH_NEWS_SOURCE", "Noticia antigua", map[string]interface{}{
				"pub_date": fecha,
				"max_days": maxDays,
			})
			continue
		}

		// Validar imagen
		if imagen == "" {
			// Si no hay imagen y el patrón es sin imagen, usar fallback
			if strings.Contains(getString(source.Filter), "no_image") {
				fallbackImage := uc.getFallbackImage(ctx, cat, lang)
				if fallbackImage != "" {
					imagen = fallbackImage
					utils.AppInfo("FETCH_NEWS_SOURCE", "Usando imagen de fallback", map[string]interface{}{
						"fallback_image": fallbackImage,
					})
				} else {
					utils.AppWarn("FETCH_NEWS_SOURCE", "Sin imagen y sin fallback", map[string]interface{}{
						"title": tituloLimpio,
					})
					continue
				}
			} else {
				utils.AppWarn("FETCH_NEWS_SOURCE", "Imagen no encontrada", map[string]interface{}{
					"title": tituloLimpio,
				})
				continue
			}
		}

		// Validar imagen (excepto si es una imagen de fallback local)
		if !strings.Contains(imagen, "/images/fallback/") {
			valid, err := uc.imageDownloader.ValidateImage(imagen)
			if err != nil {
				utils.AppError("FETCH_NEWS_SOURCE", "Error validando imagen", err, map[string]interface{}{
					"title": tituloLimpio,
					"image": imagen,
				})
				continue
			}
			if !valid {
				utils.AppWarn("FETCH_NEWS_SOURCE", "Imagen inválida", map[string]interface{}{
					"title": tituloLimpio,
					"image": imagen,
				})
				continue
			}
		} else {
			// Para imágenes de fallback, solo verificar que el archivo existe
			projectRoot := uc.getProjectRoot()
			imagePath := filepath.Join(projectRoot, "frontend", "assets", "images", "fallback", filepath.Base(imagen))
			if _, err := os.Stat(imagePath); os.IsNotExist(err) {
				utils.AppWarn("FETCH_NEWS_SOURCE", "Imagen de fallback no encontrada", map[string]interface{}{
					"title":      tituloLimpio,
					"image_path": imagePath,
				})
				continue
			}
		}

		// Crear noticia para la BD
		newsItem := domain.NewsItem{
			Title:        tituloLimpio,
			Link:         link,
			Image:        imagen,
			PubDate:      fecha,
			LangCode:     lang,
			CategoryCode: cat,
			SourceID:     source.ID,
			Source:       *source,
		}

		// Guardar en la BD
		if err := uc.newsItemRepo.Create(ctx, &newsItem); err != nil {
			utils.AppError("FETCH_NEWS_SOURCE", "Error guardando noticia", err, map[string]interface{}{
				"title": tituloLimpio,
			})
			continue
		}

		// Marcar como vistos
		linksVistos[link] = struct{}{}
		titulosVistos[tituloLimpio] = struct{}{}
		extractedCount++

		utils.AppInfo("FETCH_NEWS_SOURCE", "Noticia extraída exitosamente", map[string]interface{}{
			"title":           tituloLimpio,
			"extracted_count": extractedCount,
		})
	}

	utils.AppInfo("FETCH_NEWS_SOURCE", "Extracción completada", map[string]interface{}{
		"source_id":       source.ID,
		"extracted_count": extractedCount,
	})

	return nil
}

// getNewsCount obtiene el tope de noticias para un idioma y categoría según config.yaml
func (uc *FetchNewsUseCase) getNewsCount(lang, cat string) int {
	// Lógica de fallback
	if uc.config.NewsCount == nil {
		return 10
	}
	if v, ok := uc.config.NewsCount[lang]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			if n, ok := m[cat]; ok {
				if i, ok := n.(int); ok {
					return i
				}
				if f, ok := n.(float64); ok {
					return int(f)
				}
			}
			if n, ok := m["default"]; ok {
				if i, ok := n.(int); ok {
					return i
				}
				if f, ok := n.(float64); ok {
					return int(f)
				}
			}
		}
	}
	if n, ok := uc.config.NewsCount["default"]; ok {
		if i, ok := n.(int); ok {
			return i
		}
		if f, ok := n.(float64); ok {
			return int(f)
		}
	}
	return 10
}

// Función processSource eliminada - no se usa
// La lógica de procesamiento está implementada en Execute()

func (uc *FetchNewsUseCase) processItem(ctx context.Context, item *domain.NewsItem, source domain.NewsSource, category *domain.Category, country *domain.Country) error {
	// 1. Validar si ya existe (por Link)
	// Aquí deberías tener un método FindByLink o similar, si no existe, puedes omitir esta validación o implementarla
	// exists, err := uc.newsItemRepo.ExistsByLink(ctx, item.Link)
	// if err != nil {
	// 	return fmt.Errorf("error al verificar la existencia del link: %w", err)
	// }
	// if exists {
	// 	return fmt.Errorf("la noticia ya existe")
	// }

	// 2. Validar título
	if len(item.Title) < uc.config.Filters.MinTitle || len(item.Title) > uc.config.Filters.MaxTitle {
		return fmt.Errorf("longitud de título inválida")
	}

	// 3. Validar antigüedad
	if time.Since(item.PubDate).Hours() > float64(uc.config.Filters.MaxDays*24) {
		return fmt.Errorf("artículo demasiado antiguo")
	}

	// 4. Validar y procesar imagen
	if item.Image == "" {
		return fmt.Errorf("el artículo no tiene imagen")
	}
	valid, err := uc.imageDownloader.ValidateImage(item.Image)
	if err != nil || !valid {
		return fmt.Errorf("validación de imagen fallida: %w", err)
	}

	// 5. Crear y guardar la noticia
	newsItem := &domain.NewsItem{
		Title:        item.Title,
		Link:         item.Link,
		Image:        item.Image,
		SourceID:     source.ID,
		PubDate:      item.PubDate,
		LangCode:     country.Code,
		CategoryCode: category.Code,
		CreatedAt:    time.Now(),
	}

	if err := uc.newsItemRepo.Create(ctx, newsItem); err != nil {
		return fmt.Errorf("error al guardar la noticia en la BD: %w", err)
	}

	return nil
}

// Helper para obtener el valor string de un *string
func getString(ptr *string) string {
	if ptr != nil {
		return *ptr
	}
	return ""
}

// cleanOldNews limpia noticias anteriores de la BD para evitar duplicación
func (uc *FetchNewsUseCase) cleanOldNews(ctx context.Context) error {
	utils.AppInfo("FETCH_NEWS", "Limpiando noticias anteriores de la base de datos", nil)

	// Limpiar TODAS las noticias de la BD (como hacíamos con las carpetas)
	// Esto asegura que cada ejecución empiece con una BD limpia
	if err := uc.newsItemRepo.DeleteOlderThan(ctx, time.Now()); err != nil {
		return fmt.Errorf("error limpiando noticias de la BD: %w", err)
	}

	utils.AppInfo("FETCH_NEWS", "Limpieza de noticias anteriores completada", nil)
	return nil
}

// getFallbackImage obtiene la imagen de fallback para una categoría+idioma
func (uc *FetchNewsUseCase) getFallbackImage(ctx context.Context, categoryCode, languageCode string) string {
	fallbackImage, err := uc.fallbackImageRepo.GetByCategoryAndLang(ctx, categoryCode, languageCode)
	if err != nil || fallbackImage == nil {
		return ""
	}

	// Usar URL relativa que funcione en cualquier entorno
	// Esto evita problemas de protocolo (HTTP vs HTTPS)
	fallbackURL := fmt.Sprintf("/images/fallback/%s", fallbackImage.Filename)

	utils.AppInfo("FALLBACK_IMAGE", "URL de imagen de fallback generada", map[string]interface{}{
		"category_code": categoryCode,
		"language_code": languageCode,
		"filename":      fallbackImage.Filename,
		"url":           fallbackURL,
	})

	return fallbackURL
}

// getProjectRoot obtiene la ruta raíz del proyecto.
func (uc *FetchNewsUseCase) getProjectRoot() string {
	// Obtener el directorio de trabajo actual
	dir, err := os.Getwd()
	if err != nil {
		utils.AppError("PROJECT_ROOT", "Error al obtener la ruta raíz del proyecto", err, map[string]interface{}{
			"error": err.Error(),
		})
		return ""
	}

	// Buscar la raíz del proyecto (donde está el archivo go.mod)
	projectRoot := dir
	for {
		// Verificar si existe go.mod en el directorio actual
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}

		// Subir un nivel
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			// Llegamos a la raíz del sistema de archivos
			utils.AppError("PROJECT_ROOT", "No se encontró go.mod en ningún directorio padre", nil, map[string]interface{}{
				"current_dir": dir,
			})
			return dir // Fallback al directorio actual
		}
		projectRoot = parent
	}

	utils.AppInfo("PROJECT_ROOT", "Ruta raíz del proyecto obtenida", map[string]interface{}{
		"project_root": projectRoot,
	})

	return projectRoot
}
