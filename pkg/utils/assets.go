package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// AssetWithHash encuentra un archivo asset con hash en el directorio especificado
func AssetWithHash(baseDir, prefix, extension string) string {
	pattern := fmt.Sprintf("%s.*%s", prefix, extension)

	matches, err := filepath.Glob(filepath.Join(baseDir, pattern))
	if err != nil || len(matches) == 0 {
		// Fallback sin hash
		return fmt.Sprintf("/%s%s", prefix, extension)
	}

	// Tomar el primer match y convertir a ruta web
	relativePath := strings.TrimPrefix(matches[0], baseDir)
	relativePath = strings.TrimPrefix(relativePath, string(os.PathSeparator))
	relativePath = filepath.ToSlash(relativePath)

	// Asegurar que la ruta sea relativa al directorio static
	if strings.HasPrefix(relativePath, "frontend/dist/") {
		relativePath = strings.TrimPrefix(relativePath, "frontend/dist/")
	}

	return "/" + relativePath
}

// findHashedFile busca en baseDir un archivo cuyo nombre empiece por prefix y termine con extension,
// devolviendo únicamente el nombre de archivo (basename). Si no encuentra, devuelve prefix+extension.
func findHashedFile(baseDir, prefix, extension string) string {
	pattern := fmt.Sprintf("%s.*%s", prefix, extension)
	matches, err := filepath.Glob(filepath.Join(baseDir, pattern))
	if err != nil || len(matches) == 0 {
		return prefix + extension
	}
	return filepath.Base(matches[0])
}

// GetCSSAsset obtiene la ruta del archivo CSS principal con hash
func GetCSSAsset() string {
	filename := findHashedFile("frontend/dist/css", "main", ".css")
	return "/css/" + filename
}

// GetJSAsset obtiene la ruta del archivo JS principal con hash
func GetJSAsset() string {
	filename := findHashedFile("frontend/dist/js", "main", ".js")
	return "/js/" + filename
}

// AssetMapper mantiene un cache de assets para evitar búsquedas repetidas
type AssetMapper struct {
	assets  map[string]string
	distDir string
}

// NewAssetMapper crea una nueva instancia del mapeador de assets
func NewAssetMapper(distDir string) *AssetMapper {
	mapper := &AssetMapper{
		assets:  make(map[string]string),
		distDir: distDir,
	}
	mapper.scanAssets()
	return mapper
}

// scanAssets escanea el directorio dist y mapea todos los assets
func (am *AssetMapper) scanAssets() {
	filepath.WalkDir(am.distDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(am.distDir, path)
		if err != nil {
			return nil
		}

		// Convertir a ruta web
		webPath := "/static/" + filepath.ToSlash(relPath)

		// Extraer el nombre base sin hash para el mapeo
		filename := d.Name()
		if strings.Contains(filename, ".") {
			parts := strings.Split(filename, ".")
			if len(parts) >= 3 {
				// Formato: main.hash.js o main.hash.css
				baseName := parts[0] + "." + parts[len(parts)-1]
				am.assets[baseName] = webPath
			}
		}

		return nil
	})
}

// GetAsset obtiene la ruta de un asset por su nombre base (ej: "main.js", "main.css")
func (am *AssetMapper) GetAsset(baseName string) string {
	if path, exists := am.assets[baseName]; exists {
		return path
	}

	// Fallback
	return "/static/" + baseName
}

// GetMainCSS obtiene la ruta del CSS principal
func (am *AssetMapper) GetMainCSS() string {
	return am.GetAsset("main.css")
}

// GetMainJS obtiene la ruta del JS principal
func (am *AssetMapper) GetMainJS() string {
	return am.GetAsset("main.js")
}
