## 📰 DailyNews - Portal de Noticias RSS

![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)
![MySQL](https://img.shields.io/badge/MySQL-8.0+-green.svg)
![Vue.js](https://img.shields.io/badge/Vue.js-3.5+-green.svg)
![Tailwind CSS](https://img.shields.io/badge/Tailwind_CSS-4.x-38B2AC.svg)
![Vite](https://img.shields.io/badge/Vite-7.x-646CFF.svg)

<div align="center">
  <img src="https://img.shields.io/badge/Status-Production%20Ready-brightgreen" alt="Status: Production Ready">
  <img src="https://img.shields.io/badge/Architecture-Clean%20Architecture%20%2B%20DDD-blue" alt="Architecture: Clean Architecture + DDD">
</div>

---

## 🌍 Idioma / Language

<div align="center">
  <a href="#español">🇪🇸 Español</a> | 
  <a href="#english">🇺🇸 English</a>
</div>

---

<a id="español"></a>
## 🇪🇸 Español

### 📖 Descripción

**DailyNews** es una aplicación web moderna que obtiene noticias desde múltiples fuentes RSS(Hay unas que vienen por defecto con la aplicación y el usuario puede agregar las que quiera) y las organiza por categoría e idioma. Desarrollada con **Clean Architecture** y **Domain-Driven Design (DDD)**, ofrece una interfaz elegante y responsive para estar al tanto de las noticias de tu interes sin sesgos ni patrones preestablecidos.

### ✨ Características Principales

- 🔄 Extracción automática via cron configurable (por defecto cada 24h, tambien se puede ejecutar manualmente la extraccion con un simple go run cmd/main.go desde la raiz del proyecto)
- ⚙️ Sistema por patrones para la deteccion y extraccion automatica de las noticias de las distintas fuentes RSS
- 🖼️ Imágenes fallback por categoría+idioma para fuentes añadidas por el usuario(No todos los feed RSS incluyen imagenes)
- 🔍 Filtros avanzados: Por fuente, fecha, categoría e idioma
- 🌍 Multiidioma (ES/EN/FR) con persistencia de `lang` en la navegación
- 🎨 UI moderna con Tailwind 4 + Vue 3
- 🧹 Gestion por parte del usuario de sus propias fuentes con borrado real de fuentes (fuente, noticias asociadas e imagen fallback) con efectos inmediatos en el frontend
- 🎯 Clean Architecture + DDD

### 🏗️ Arquitectura

```
📁 DailyNews/
├── 📁 cmd/                    # Punto de entrada de la aplicación
├── 📁 internal/
│   ├── 📁 domain/            # Modelos y contratos (interfaces)
│   ├── 📁 repository/        # Adaptadores GORM (MySQL)
│   ├── 📁 usecase/           # Orquestación (fetch/validación/guardado)
│   ├── 📁 delivery/          # HTTP (handlers, rutas, middleware, templates)
│   └── 📁 infrastructure/    # RSS, validación de imágenes, cron
├── 📁 frontend/
│   ├── 📁 templates/         # Plantillas HTML
│   ├── 📁 assets/            # JS (Vue 3), CSS (Tailwind 4), imágenes
│   └── 📁 dist/              # Build Vite (assets con hash)
├── 📁 pkg/
│   ├── 📁 config/            # Carga de config (Viper)
│   ├── 📁 database/          # Conexión y migraciones (GORM)
│   └── 📁 utils/             # Logging, assets helpers, fechas
└── 📁 config/                # config.yaml
```

### 🛠️ Tecnologías

**Backend:** Go 1.21+, Gin, GORM (MySQL 8), gofeed, robfig/cron

**Frontend:** Vue.js 3 (no SPA), Tailwind CSS 4, Vite 7, JavaScript (ES6+)

**Base de Datos:** MySQL

### 📋 Requisitos Previos

- **Go 1.21** o superior
- **MySQL 8.0** o superior
- **Node.js**
- **npm** o **yarn**

### 🚀 Instalación

#### 1. Clonar el Repositorio
```bash
git clone https://github.com/SebastianRoberto/DailyNews.git
cd DailyNews
```

#### 2. Configurar config.yaml
- Ve a `config/config_example.yaml` 
- Edita las credenciales que necesites(Usuario, contraseña, puerto en el que quieras que corra la app o en el que tengas tu Mysql)
- Ajusta las configuraciones para la obtencion de noticias que gustes
 - Cambia el nombre de `config_example.yaml` a `config.yaml`

La app crea la base de datos automáticamente si el usuario tiene permisos

#### 3. Ejecutar la Aplicación
```bash
# Desde la raíz del proyecto
go run cmd/main.go
```
La aplicación estará disponible en `http://localhost:3020`, esto es configurable en `config/config.yaml`(config_example cuando recien clones el repo)

`go run cmd/main.go`:
- Compila assets del frontend con Vite
- Ejecuta migraciones GORM y seeds
- Inicia servidor HTTP y sirve estáticos (`/css`, `/js`, `/images`)
- Ejecuta extracción inicial y arranca el cron
- Basta con ejecutar esto para poder ver la aplicacion en tu navegador 🙂.

### 📊 Estructura de Base de Datos

**Tablas Principales (nombres reales):**
- `news_items` — Noticias procesadas
- `template_news_sources` — Fuentes RSS
- `template_news` — Categorías
- `template_country` — Idiomas
- `fallback_images` — Imágenes de respaldo para las fuentes RSS que añada el usuario

### ⚙️ Configuración

#### Seeds por defecto
En el primer arranque(go run cmd/main.go) se crean idiomas, categorías y un conjunto de fuentes por defecto por categoría/idioma.

#### Agregar Fuentes Personalizadas (flujo real)
1. Abre el panel flotante de “Configuración”.
2. Agrega los datos de tu fuente(nombre, link, categoria e idioma)
3. Puedes darle click a probar antes de agregar fuente para saber si vas a necesitar subir una imagen fallback y el sistema analizara las reglas de extraccion necesarias para la fuente(sistema de patrones)
4. Subes la imagen fallback(o no, depende de tu caso) y le das click a agregar fuente RSS
5. Se ejecuta extracción inmediata SOLO para esa fuente y se recarga la página.
 
Lo de la imagen de fallback es fundamental por que no todas las fuentes RSS incluyen imagenes y si quieres una interfaz bonita es lo ideal

Patrones soportados para la extraccion de elementos de una fuente(asignacion automatica):
- `patron1`: title, media:content|media:thumbnail, link, pubDate
- `patron2`: title, enclosure|media:content, link, pubDate
- `patron3`: title, description_img (HTML), link, pubDate
- `*_no_image`: 3 variantes de los patrones de arriba pero sin imagen (requieren imagen de fallback)

Imágenes fallback:
- Se suben a `/images/fallback/<filename>` y se gestionan vía API.

### 🔧 Desarrollo

```

#### Variables de Entorno
```bash
CONFIG_PATH=./config/config.yaml  # Ruta del archivo de configuración
```

### 📈 Monitoreo

#### Logs y monitoreo
- Logs detallados para extracción/validación y operaciones de BD.
- Información detallada asociada a la imagen en fallback_images(en un futuro me gustaria establecer limites de subida de archivos o conversion automatica de imagenes a webp para menor tiempo de carga)

### 🔌 API (resumen)
- GET `/api/news/:lang/:category`
- GET `/api/news/filtered`
- GET `/api/categories`
- GET `/api/languages`
- POST `/api/sources/test` — body: `{ "url": "..." }`
- POST `/api/sources/add` — body: `{ sourceName, rssUrl, category, language, fallbackImageId? }`
- DELETE `/api/sources/:id`
- POST `/api/fallback-image/upload` (FormData: image, categoryCode, languageCode)
- GET `/api/fallback-image/:category/:lang`
- DELETE `/api/fallback-image/:category/:lang`
- GET `/api/fallback-image/list`
- POST `/api/news/refresh`
- GET `/api/health`


<a id="english"></a>
## 🇺🇸 English


PD: Todas las rutas para la gestión de imagenes y dependencias estan optimizadas para ser compatibles y no dar problemas ni con Windows ni con Linux, si tienes algun problema o hay algo que creas que se pueda/deba mejorar puedes escribirme a mi correo:
sebastian.roberto.pp@gmail.com
### 📖 Description

**DailyNews** is a modern web application that aggregates news from multiple RSS sources (some are included by default and users can add their own) and organizes them by category and language. Built with **Clean Architecture** and **Domain-Driven Design (DDD)**, it provides an elegant, responsive interface to keep up with unbiased news without pre-set patterns.

### ✨ Key Features

- 🔄 Automatic extraction via configurable cron (default every 24h)
- ⚙️ Pattern-based system to detect and extract news from various RSS feeds
- 🖼️ Fallback images per category+language for user-added sources (not all RSS feeds include images)
- 🔍 Advanced filters: by source, date, category and language
- 🌍 Multi-language (ES/EN/FR) with `lang` persistence
- 🎨 Modern UI with Tailwind 4 + Vue 3
- 🧹 User-managed sources with hard delete (source, related news and fallback image) with immediate effects on the frontend
- 🎯 Clean Architecture + DDD

### 🏗️ Architecture

```
📁 DailyNews/
├── 📁 cmd/                    # Entry (bootstrap & DI)
├── 📁 internal/
│   ├── 📁 domain/            # Models & contracts
│   ├── 📁 repository/        # GORM adapters (MySQL)
│   ├── 📁 usecase/           # Business orchestration
│   ├── 📁 delivery/          # HTTP (handlers, routes, templates)
│   └── 📁 infrastructure/    # RSS, image validation, cron
├── 📁 frontend/
│   ├── 📁 templates/         # Server-rendered templates
│   ├── 📁 assets/            # JS (Vue 3), CSS (Tailwind 4), images
│   └── 📁 dist/              # Vite build (hashed assets)
├── 📁 pkg/                   # Config, database, utils
└── 📁 config/                # YAML configuration
```

### 🛠️ Technologies

**Backend:** Go 1.21+, Gin, GORM (MySQL 8), gofeed, robfig/cron

**Frontend:** Vue 3 (non-SPA), Tailwind 4, Vite 7, ES6+

**Database:**
- **MySQL 8.0+** - Main database

### 📋 Prerequisites

- **Go 1.21** or higher
- **MySQL 8.0** or higher
- **Node.js 18** or higher
- **npm** or **yarn**

### 🚀 Installation

#### 1. Clone Repository
```bash
git clone https://github.com/SebastianRoberto/DailyNews.git
cd DailyNews
```

#### 2. Configure config.yaml
- Go to `config/config_example.yaml`.
- Edit the credentials you need (MySQL user, password, and the port you want the app to run on).
- Adjust news fetching settings as you like.
- Rename `config_example.yaml` to `config.yaml`.

Note: The app creates the database automatically if your MySQL user has permissions.

#### 3. Run Application
```bash
# From project root
go run cmd/main.go
```

Application will be available at `http://localhost:3020` (configurable in `config/config.yaml`).

`go run cmd/main.go` will:
- Build frontend assets with Vite
- Run GORM migrations & seeds
- Start HTTP server and static serving
- Run initial extraction and start cron

### 📊 Database Structure

**Real table names:**
- `news_items` — Processed news
- `template_news_sources` — RSS sources
- `template_news` — Categories
- `template_country` — Languages/countries
- `fallback_images` — Fallback images for user-added RSS sources

### ⚙️ Configuration

#### Default seeds
On first run, languages, categories and a set of sources per category/language are created.

#### Add Custom Sources (real flow)
1. Open the floating “Settings” panel.
2. Add your source data (name, link, category and language).
3. You can click Test before adding to know if you will need to upload a fallback image; the system will analyze and auto-assign the extraction rules.
4. Upload the fallback image (or not, depending on your case) and click “Add RSS Source”.
5. Immediate extraction ONLY for that source and the page reloads.

Note: the fallback image is important because not all RSS feeds include images; it helps keep a nice UI.

Supported patterns for extracting elements from a source (auto-assigned):
- `patron1`: title, media:content|media:thumbnail, link, pubDate
- `patron2`: title, enclosure|media:content, link, pubDate
- `patron3`: title, description_img (HTML), link, pubDate
- `*_no_image`: 3 variants of the above patterns without image (require a fallback image)

### 🔧 Development

#### Main Commands
```bash
# Run in development mode
go run cmd/main.go

# Build for production
go build -o dailynews cmd/main.go

# Clean cache
go clean -cache
```

#### Environment Variables
```bash
CONFIG_PATH=./config/config.yaml  # Configuration file path
```

### 📈 Monitoring

#### Logs & monitoring
- Detailed logs for extraction/validation and DB operations.
- Detailed info associated with fallback_images (in the future we may enforce upload limits or auto-convert to WebP for faster load times).

---

