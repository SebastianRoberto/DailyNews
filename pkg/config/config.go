package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Database     DatabaseConfig         `mapstructure:"database"`
	Server       ServerConfig           `mapstructure:"server"`
	Logger       LoggerConfig           `mapstructure:"logger"`
	NewsCount    map[string]interface{} `mapstructure:"newsCount"`
	MaxPerSource map[string]interface{} `mapstructure:"maxPerSource"`
	MaxDays      map[string]interface{} `mapstructure:"maxDays"`
	Cron         CronConfig             `mapstructure:"cron"`
	Filters      FiltersConfig          `mapstructure:"filters"`
}

type DatabaseConfig struct {
	NewsDB Database `mapstructure:"news_db"`
}

type Database struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Schema       string `mapstructure:"schema"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	CustomLogger bool   `mapstructure:"custom_logger"`
	Ensure       bool   `mapstructure:"ensure"`
	AutoMigrate  bool   `mapstructure:"auto_migrate"`
}

type ServerConfig struct {
	HTTP HTTPServer `mapstructure:"http"`
}

type HTTPServer struct {
	Mode    string `mapstructure:"mode"`
	Port    int    `mapstructure:"port"`
	Timeout string `mapstructure:"timeout"`
	Swagger bool   `mapstructure:"swagger"`
}

type LoggerConfig struct {
	Mode         string `mapstructure:"mode"`
	DetailedLogs bool   `mapstructure:"detailedLogs"`
}

type CronConfig struct {
	Logger bool   `mapstructure:"logger"`
	Expr   string `mapstructure:"expr"`
}

type FiltersConfig struct {
	MinTitle                     int     `mapstructure:"minTitle"`
	MaxTitle                     int     `mapstructure:"maxTitle"`
	MaxDays                      int     `mapstructure:"maxDays"`
	MaxDaysForNewsWithFewSources int     `mapstructure:"maxDaysForNewsWithFewSources"`
	AspectTolerance              float64 `mapstructure:"aspectTolerance"`
	TargetAspect                 float64 `mapstructure:"targetAspect"`
}

// LoadConfig carga la configuración desde el archivo YAML
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		// 1. Intentar ./config.yaml en el directorio de trabajo (útil para 'go run .')
		if _, err := os.Stat("config.yaml"); err == nil {
			configPath = "config.yaml"
		} else if _, err := os.Stat(filepath.Join("config", "config.yaml")); err == nil {
			// 2. Intentar ./config/config.yaml
			configPath = filepath.Join("config", "config.yaml")
		} else {
			// 3. Fallback relativo al ejecutable (caso binario instalado)
			exePath, err := os.Executable()
			if err != nil {
				return nil, fmt.Errorf("error al obtener la ruta del ejecutable: %v", err)
			}
			exeDir := filepath.Dir(exePath)
			configPath = filepath.Join(exeDir, "config", "config.yaml")
		}
	}

	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error al leer el archivo de configuración: %v", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error al deserializar la configuración: %v", err)
	}

	return &config, nil
}

// GetNewsCount obtiene el número de noticias para un idioma y categoría específicos
func (c *Config) GetNewsCount(lang, category string) int {
	return getIntValueFromNestedMap(c.NewsCount, lang, category, 10)
}

// GetMaxPerSource obtiene el límite de noticias por fuente para un idioma y categoría específicos
func (c *Config) GetMaxPerSource(lang, category string) int {
	return getIntValueFromNestedMap(c.MaxPerSource, lang, category, 7)
}

// GetMaxDays obtiene la antigüedad máxima para un idioma y categoría específicos
func (c *Config) GetMaxDays(lang, category string) int {
	return getIntValueFromNestedMap(c.MaxDays, lang, category, 5)
}

// getIntValueFromNestedMap es una función helper para extraer valores enteros de mapas anidados
// con la lógica: lang.category > lang.default > default
func getIntValueFromNestedMap(nestedMap map[string]interface{}, lang, category string, fallback int) int {
	if nestedMap == nil {
		return fallback
	}

	// 1. Intentar obtener valor específico para lang.category
	if langMap, exists := nestedMap[lang]; exists {
		if langMapCast, ok := langMap.(map[string]interface{}); ok {
			if categoryValue, exists := langMapCast[category]; exists {
				if intValue, ok := categoryValue.(int); ok {
					return intValue
				}
			}
			// 2. Intentar obtener valor default para el idioma
			if defaultValue, exists := langMapCast["default"]; exists {
				if intValue, ok := defaultValue.(int); ok {
					return intValue
				}
			}
		}
	}

	// 3. Intentar obtener valor default general
	if defaultValue, exists := nestedMap["default"]; exists {
		if intValue, ok := defaultValue.(int); ok {
			return intValue
		}
	}

	// 4. Fallback si no se encuentra nada
	return fallback
}
