package infrastructure

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"

	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	// "github.com/chai2010/webp" // Eliminado porque no se usa

	"dailynews/internal/domain"
)

// imageDownloader ahora recibe los parámetros de aspecto y tolerancia
// y el tamaño objetivo para redimensionar
type imageDownloader struct {
	httpClient      *http.Client
	targetAspect    float64
	aspectTolerance float64
	width           int
	height          int
}

func NewImageDownloader(targetAspect, aspectTolerance float64, width, height int) domain.ImageDownloader {
	return &imageDownloader{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		targetAspect:    targetAspect,
		aspectTolerance: aspectTolerance,
		width:           width,
		height:          height,
	}
}

func (d *imageDownloader) DownloadAndValidate(ctx context.Context, imageURL, savePath string) (string, error) {
	// 1. Descargar la imagen
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("error creando petición: %w", err)
	}
	// Configurar headers para evitar ser bloqueado
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error descargando imagen: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("código de estado inesperado: %d", resp.StatusCode)
	}
	// 2. Validar el tipo MIME
	contentType := resp.Header.Get("Content-Type")
	if !isValidImageType(contentType) {
		log.Printf("[WARN] Imagen descartada por tipo MIME no soportado: %s", contentType)
		return "", fmt.Errorf("tipo de imagen no soportado: %s", contentType)
	}
	// 3. Leer y decodificar la imagen
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		log.Printf("[WARN] Imagen descartada por error de decodificación: %v", err)
		return "", fmt.Errorf("error decodificando imagen: %w", err)
	}
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	aspectRatio := float64(width) / float64(height)
	minAspect := d.targetAspect - (d.targetAspect * d.aspectTolerance)
	maxAspect := d.targetAspect + (d.targetAspect * d.aspectTolerance)
	if aspectRatio < minAspect || aspectRatio > maxAspect {
		log.Printf("[WARN] Imagen descartada por relación de aspecto: %.3f (esperado %.3f ±%.2f)", aspectRatio, d.targetAspect, d.aspectTolerance)
		return "", fmt.Errorf("relación de aspecto no soportada: %.3f (esperado %.3f ±%.2f)", aspectRatio, d.targetAspect, d.aspectTolerance)
	}
	// 4. Redimensionar si es necesario
	if width != d.width || height != d.height {
		// Redimensionar usando image/draw (o imaging si está disponible)
		// Aquí usamos image.NewRGBA y draw.Draw para mantener dependencias estándar
		newImg := image.NewRGBA(image.Rect(0, 0, d.width, d.height))
		// Escalado simple (nearest neighbor)
		for y := 0; y < d.height; y++ {
			for x := 0; x < d.width; x++ {
				srcX := x * width / d.width
				srcY := y * height / d.height
				newImg.Set(x, y, img.At(srcX, srcY))
			}
		}
		img = newImg
	}
	// 5. Crear directorio de destino si no existe
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return "", fmt.Errorf("error creando directorio: %w", err)
	}
	savePath = strings.TrimSuffix(savePath, filepath.Ext(savePath)) + ".webp"
	outputFile, err := os.Create(savePath)
	if err != nil {
		return "", fmt.Errorf("error creando archivo: %w", err)
	}
	defer outputFile.Close()

	if err := png.Encode(outputFile, img); err != nil {
		return "", fmt.Errorf("error codificando a PNG: %w", err)
	}
	log.Printf("[INFO] Imagen procesada y guardada en: %s", savePath)
	return savePath, nil
}

func (d *imageDownloader) ValidateImage(imageURL string) (bool, error) {
	// 1. Descargar la imagen
	req, err := http.NewRequest(http.MethodGet, imageURL, nil)
	if err != nil {
		return false, fmt.Errorf("error creando petición: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("error descargando imagen: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("código de estado inesperado: %d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if !isValidImageType(contentType) {
		log.Printf("[WARN] Imagen descartada por tipo MIME no soportado: %s", contentType)
		return false, fmt.Errorf("tipo de imagen no soportado: %s", contentType)
	}

	// 3. Leer y decodificar la imagen
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		log.Printf("[WARN] Imagen descartada por error de decodificación: %v", err)
		return false, fmt.Errorf("error decodificando imagen: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Validar tamaño mínimo
	if width < 400 || height < 225 {
		log.Printf("[WARN] Imagen descartada por tamaño insuficiente: %dx%d", width, height)
		return false, nil
	}

	// Validar relación de aspecto
	aspectRatio := float64(width) / float64(height)
	minAspect := d.targetAspect - (d.targetAspect * d.aspectTolerance)
	maxAspect := d.targetAspect + (d.targetAspect * d.aspectTolerance)
	if aspectRatio < minAspect || aspectRatio > maxAspect {
		log.Printf("[WARN] Imagen descartada por relación de aspecto: %.3f (esperado %.3f ±%.2f)", aspectRatio, d.targetAspect, d.aspectTolerance)
		return false, nil
	}

	log.Printf("[DEBUG] Imagen válida: %dx%d, aspecto: %.3f", width, height, aspectRatio)
	return true, nil
}

// isValidImageType verifica si el tipo MIME es una imagen soportada
func isValidImageType(mimeType string) bool {
	// Obtener la extensión del tipo MIME
	ext, err := mime.ExtensionsByType(mimeType)
	if err != nil || len(ext) == 0 {
		return false
	}

	// Verificar si la extensión está en la lista de formatos soportados
	supported := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
		".gif":  true,
	}

	for _, e := range ext {
		if supported[strings.ToLower(e)] {
			return true
		}
	}

	return false
}
