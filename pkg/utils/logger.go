package utils

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

var (
	// AppLogger para logs de aplicación (inicio, errores, métricas, etc.)
	AppLogger *logrus.Logger

	// CategoryColors para colorear logs por categoría
	CategoryColors = map[string]string{
		"technology":      "\033[36m", // Cyan
		"business":        "\033[34m", // Azul
		"sports":          "\033[33m", // Amarillo
		"entertainment":   "\033[35m", // Magenta
		"science":         "\033[32m", // Verde
		"health":          "\033[31m", // Rojo
		"economia":        "\033[34m", // Azul
		"cultura":         "\033[35m", // Magenta
		"salud":           "\033[31m", // Rojo
		"internacional":   "\033[32m", // Verde
		"entretenimiento": "\033[35m", // Magenta
		"destacado":       "\033[33m", // Amarillo
	}

	// LevelColors para colorear niveles de log
	LevelColors = map[string]string{
		"INFO":  "",
		"WARN":  "\033[38;5;208m", // Naranja
		"ERROR": "\033[31m",       // Rojo
	}

	// Reset color
	Reset = "\033[0m"
)

func init() {
	AppLogger = logrus.New()
	AppLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	AppLogger.SetOutput(os.Stdout)
	AppLogger.SetLevel(logrus.InfoLevel)
}

// AppInfo log de aplicación con contexto
func AppInfo(component, message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["component"] = component
	AppLogger.WithFields(fields).Info(message)
}

// AppWarn log de advertencia de aplicación
func AppWarn(component, message string, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["component"] = component
	AppLogger.WithFields(fields).Warn(message)
}

// AppError log de error de aplicación
func AppError(component, message string, err error, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	fields["component"] = component
	if err != nil {
		fields["error"] = err.Error()
	}
	AppLogger.WithFields(fields).Error(message)
}

// NewsInfo log detallado de procesamiento de noticias (INFO)
func NewsInfo(category, lang, title, source string, fields map[string]interface{}) {
	color := CategoryColors[category]
	if color == "" {
		color = "\033[37m" // Blanco por defecto
	}

	status := "añadida"
	if fields != nil {
		if count, ok := fields["count"]; ok {
			status = fmt.Sprintf("añadida (#%v)", count)
		}
	}

	message := fmt.Sprintf("Noticia [%s:%s] → {%s} (Fuente: %s, %s)",
		category, lang, title, source, status)

	fmt.Printf("%s%s%s\n", color, message, Reset)
}

// NewsWarn log de advertencia de noticias
func NewsWarn(category, lang, title, reason string) {
	color := CategoryColors[category]
	if color == "" {
		color = "\033[37m"
	}
	warnColor := LevelColors["WARN"]

	message := fmt.Sprintf("Noticia [%s:%s] → {%s} (%s)",
		category, lang, title, reason)

	fmt.Printf("%s%s%s%s\n", warnColor, color, message, Reset)
}

// NewsError log de error de noticias
func NewsError(category, lang, title, reason string) {
	color := CategoryColors[category]
	if color == "" {
		color = "\033[37m"
	}
	errorColor := LevelColors["ERROR"]

	message := fmt.Sprintf("Noticia [%s:%s] → {%s} (%s)",
		category, lang, title, reason)

	fmt.Printf("%s%s%s%s\n", errorColor, color, message, Reset)
}

// SourceError log de error de fuente RSS
func SourceError(url, reason string) {
	errorColor := LevelColors["ERROR"]
	message := fmt.Sprintf("Error al obtener feed [url:%s]: %s", url, reason)
	fmt.Printf("%s%s%s\n", errorColor, message, Reset)
}

// SourceWarn log de advertencia de fuente RSS
func SourceWarn(url, reason string) {
	warnColor := LevelColors["WARN"]
	message := fmt.Sprintf("Advertencia en feed [url:%s]: %s", url, reason)
	fmt.Printf("%s%s%s\n", warnColor, message, Reset)
}

// ProcessingInfo log de inicio de procesamiento de categoría
func ProcessingInfo(category, lang string, newsLimit int, sourceCount int) {
	color := CategoryColors[category]
	if color == "" {
		color = "\033[37m"
	}

	message := fmt.Sprintf("Iniciando procesamiento para categoría: %s, idioma: %s, newsLimit: %d",
		category, lang, newsLimit)

	fmt.Printf("\n%s%s%s\n", color, message, Reset)

	if sourceCount <= 3 {
		fmt.Printf("%sCategoría %s, idioma %s tiene solo %d fuentes, usando límite extendido de 14 días%s\n",
			color, category, lang, sourceCount, Reset)
	}
}

// ProcessingComplete log de finalización de procesamiento
func ProcessingComplete(category, lang string, validCount, discardedCount int) {
	color := CategoryColors[category]
	if color == "" {
		color = "\033[37m"
	}

	if validCount == 0 {
		message := fmt.Sprintf("No se generaron noticias válidas para %s:%s", category, lang)
		fmt.Printf("%s%s%s\n", color, message, Reset)
	} else {
		message := fmt.Sprintf("Alcanzado límite de noticias para categoría %s, idioma %s", category, lang)
		fmt.Printf("%s%s%s\n", color, message, Reset)
	}
}

// LimitReached log cuando se alcanza el límite de noticias
func LimitReached(category, lang string) {
	color := CategoryColors[category]
	if color == "" {
		color = "\033[37m"
	}

	message := fmt.Sprintf("Alcanzado límite de noticias para categoría %s, idioma %s", category, lang)
	fmt.Printf("%s%s%s\n", color, message, Reset)
}

// SourceProcessing log de procesamiento de fuente individual
func SourceProcessing(sourceName, url string) {
	message := fmt.Sprintf("Procesando fuente: %s (%s)", sourceName, url)
	fmt.Printf("%s\n", message)
}

// SourceProcessingComplete log de finalización de procesamiento de fuente
func SourceProcessingComplete(sourceName string, validCount, totalCount int) {
	// Solo mostrar si hay noticias válidas o si todas fueron descartadas
	if validCount > 0 {
		message := fmt.Sprintf("✅ %s: %d noticias válidas", sourceName, validCount)
		fmt.Printf("%s\n", message)
	} else {
		message := fmt.Sprintf("❌ %s: todas las noticias descartadas", sourceName)
		fmt.Printf("%s\n", message)
	}
}

// SourceLimitReached log cuando se alcanza el límite de noticias por fuente
func SourceLimitReached(sourceName string, maxPerSource int) {
	message := fmt.Sprintf("🔄 %s: límite alcanzado (%d noticias)", sourceName, maxPerSource)
	fmt.Printf("%s\n", message)
}

// NoValidNewsFromSource log cuando una fuente no produce noticias válidas
func NoValidNewsFromSource(sourceName, reason string) {
	warnColor := LevelColors["WARN"]
	message := fmt.Sprintf("No se pudo procesar ninguna noticia válida de %s (%s)", sourceName, reason)
	fmt.Printf("%s%s%s\n", warnColor, message, Reset)
}
