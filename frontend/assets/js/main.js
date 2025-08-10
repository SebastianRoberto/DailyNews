/**
 * DAILYNEWS - FRONTEND PRINCIPAL
 * NO-SPA con Vue 3 como librería de componentes
 * 
 * Este archivo maneja:
 * - Inicialización de Vue 3 sin router (empaquetado localmente)
 * - Interactividad de filtros y búsqueda
 * - Comunicación con API REST del backend
 * - Estados de carga y notificaciones
 */

// 🎯 IMPORTAR VUE 3 LOCALMENTE (se empaqueta en el bundle)
import { createApp, ref, computed, onMounted, nextTick } from 'vue';

// Importar CSS principal (Tailwind se procesará aquí)
import '../css/main.css';

// ========================================
// CONFIGURACIÓN GLOBAL DE LA APLICACIÓN
// ========================================

const DailyNewsApp = {
    data() {
        return {
            // Estado de la aplicación
            isLoading: false,
            isRefreshing: false,
            
            // Filtros actuales
            currentFilters: {
                lang: 'es',
                category: '',
                search: '',
                page: 1
            },
            
            // Datos de la página
            languages: [],
            categories: [],
            news: [],
            pagination: null,
            
            // Estados UI
            notifications: [],
            searchTimeout: null,
            
            // Estado del modo oscuro
            isDarkMode: false
        }
    },
    
    computed: {
        // Verificar si hay filtros activos
        hasActiveFilters() {
            return this.currentFilters.category || this.currentFilters.search;
        },
        
        // Contar noticias mostradas
        newsCount() {
            return this.news ? this.news.length : 0;
        },
        
        // Texto del botón de modo oscuro
        // darkModeText() y darkModeIcon() ELIMINADAS - El botón siempre dice "Cambiar tema"
    },
    
    // ========================================
    // CICLO DE VIDA
    // ========================================
    
    mounted() {
        console.log('🚀 DailyNews App montada');
        
        // Inicializar desde URL
        this.initializeFromURL();
        
        // Configurar eventos globales
        this.setupGlobalEvents();
        
        // Configurar elementos DOM
        this.setupDOMElements();
        
        // Inicializar modo oscuro
        this.initializeDarkMode();
        
        console.log('✅ DailyNews App lista');
    },
    
    methods: {
        // ========================================
        // INICIALIZACIÓN
        // ========================================
        
        initializeFromURL() {
            const urlParams = new URLSearchParams(window.location.search);
            const pathParts = window.location.pathname.split('/');
            
            // Extraer filtros de la URL
            this.currentFilters.lang = urlParams.get('lang') || 'es';
            this.currentFilters.search = urlParams.get('q') || urlParams.get('search') || '';
            this.currentFilters.page = parseInt(urlParams.get('page')) || 1;
            
            // Extraer categoría del path si estamos en /categoria/xxx
            if (pathParts[1] === 'categoria' && pathParts[2]) {
                this.currentFilters.category = pathParts[2];
            } else {
                this.currentFilters.category = urlParams.get('category') || '';
            }
            
            console.log('🔧 Filtros inicializados desde URL:', this.currentFilters);
        },
        
        setupGlobalEvents() {
            // Manejar cambios en el historial del navegador
            window.addEventListener('popstate', () => {
                this.initializeFromURL();
            });
            
            // Manejar teclas especiales
            document.addEventListener('keydown', (e) => {
                // ESC para limpiar búsqueda
                if (e.key === 'Escape' && this.currentFilters.search) {
                    this.clearSearch();
                }
                
                // Enter en búsqueda
                if (e.key === 'Enter' && e.target.id === 'search-input') {
                    e.preventDefault();
                    this.performSearch(e.target.value);
                }
            });
            
            console.log('🎯 Eventos globales configurados');
        },
        
        setupDOMElements() {
            // Configurar búsqueda si existe
            const searchInput = document.getElementById('search-input');
            if (searchInput && this.currentFilters.search) {
                searchInput.value = this.currentFilters.search;
            }
            
            // Configurar selects si existen
            const langSelect = document.getElementById('language-select');
            if (langSelect) {
                langSelect.value = this.currentFilters.lang;
            }
            
            const categorySelect = document.getElementById('category-select');
            if (categorySelect) {
                categorySelect.value = this.currentFilters.category;
            }
        },
        
        // ========================================
        // NAVEGACIÓN (Server-Side)
        // ========================================
        
        navigateTo(url) {
            this.showLoading();
            console.log('🧭 Navegando a:', url);
            window.location.href = url;
        },
        
        buildURL(filters = {}) {
            const finalFilters = { ...this.currentFilters, ...filters };
            const params = new URLSearchParams();
            
            // Solo agregar parámetros no vacíos
            Object.entries(finalFilters).forEach(([key, value]) => {
                if (value && value !== '' && key !== 'page') {
                    params.append(key, value);
                }
            });
            
            // Agregar página solo si no es 1
            if (finalFilters.page && finalFilters.page > 1) {
                params.append('page', finalFilters.page);
            }
            
            // Construir URL base según el tipo de página
            let baseURL = '/';
            if (finalFilters.category) {
                baseURL = `/categoria/${finalFilters.category}`;
            } else if (finalFilters.search) {
                baseURL = '/buscar';
            }
            
            const queryString = params.toString();
            return queryString ? `${baseURL}?${queryString}` : baseURL;
        },
        
        // ========================================
        // FILTROS Y BÚSQUEDA
        // ========================================
        
        onLanguageChange(event) {
            const newLang = event.target.value;
            console.log('🌐 Cambiando idioma a:', newLang);
            
            const newURL = this.buildURL({ lang: newLang, page: 1 });
            this.navigateTo(newURL);
        },
        
        onCategoryChange(event) {
            const newCategory = event.target.value;
            console.log('📂 Cambiando categoría a:', newCategory);
            
            if (newCategory) {
                // Navegar a página de categoría específica
                const newURL = this.buildURL({ 
                    category: newCategory, 
                    search: '', // Limpiar búsqueda al cambiar categoría
                    page: 1 
                });
                this.navigateTo(newURL);
            } else {
                // Navegar a página principal
                const newURL = this.buildURL({ 
                    category: '', 
                    search: '', 
                    page: 1 
                });
                this.navigateTo(newURL);
            }
        },
        
        onSearchInput(event) {
            const query = event.target.value.trim();
            
            // Limpiar timeout anterior
            if (this.searchTimeout) {
                clearTimeout(this.searchTimeout);
            }
            
            // Debounce de 500ms
            this.searchTimeout = setTimeout(() => {
                this.performSearch(query);
            }, 500);
        },
        
        performSearch(query) {
            console.log('🔍 Realizando búsqueda:', query);
            
            if (query.length >= 3) {
                // Buscar con query
                const newURL = this.buildURL({ 
                    search: query, 
                    category: '', // Limpiar categoría al buscar
                    page: 1 
                });
                this.navigateTo(newURL);
            } else if (query.length === 0) {
                // Volver a página principal o categoría actual
                this.clearSearch();
            }
        },
        
        clearSearch() {
            console.log('🧹 Limpiando búsqueda');
            
            const searchInput = document.getElementById('search-input');
            if (searchInput) {
                searchInput.value = '';
            }
            
            const newURL = this.buildURL({ search: '', page: 1 });
            this.navigateTo(newURL);
        },
        
        // ========================================
        // API Y DATOS
        // ========================================
        
        async refreshNews() {
            this.isRefreshing = true;
            this.showNotification('Actualizando noticias...', 'info');
            
            try {
                console.log('🔄 Iniciando actualización de noticias...');
                
                const response = await fetch('/api/news/refresh', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    }
                });
                
                if (response.ok) {
                    this.showNotification('✅ Noticias actualizadas correctamente', 'success');
                    
                    // Recargar página después de 2 segundos
                    setTimeout(() => {
                        window.location.reload();
                    }, 2000);
                } else {
                    throw new Error(`Error ${response.status}: ${response.statusText}`);
                }
                
            } catch (error) {
                console.error('❌ Error al refrescar noticias:', error);
                this.showNotification('❌ Error al actualizar noticias', 'error');
            } finally {
                this.isRefreshing = false;
            }
        },
        
        async loadNewsAjax(filters = {}) {
            // Para futuras implementaciones de carga AJAX sin recarga de página
            const params = new URLSearchParams({
                ...this.currentFilters,
                ...filters
            });
            
            try {
                const response = await fetch(`/api/news?${params}`);
                const data = await response.json();
                
                this.news = data.news || [];
                this.pagination = data.pagination || null;
                
                return data;
            } catch (error) {
                console.error('❌ Error cargando noticias:', error);
                throw error;
            }
        },
        
        // ========================================
        // UI Y ESTADOS
        // ========================================
        
        showLoading() {
            this.isLoading = true;
            const loadingEl = document.getElementById('loading');
            if (loadingEl) {
                loadingEl.classList.remove('hidden');
                loadingEl.classList.add('fade-in');
            }
        },
        
        hideLoading() {
            this.isLoading = false;
            const loadingEl = document.getElementById('loading');
            if (loadingEl) {
                loadingEl.classList.add('hidden');
                loadingEl.classList.remove('fade-in');
            }
        },
        
        showNotification(message, type = 'info', duration = 5000) {
            const notification = {
                id: Date.now(),
                message,
                type,
                visible: true
            };
            
            this.notifications.push(notification);
            
            // Auto-remover después del tiempo especificado
            setTimeout(() => {
                this.removeNotification(notification.id);
            }, duration);
            
            console.log(`[${type.toUpperCase()}] ${message}`);
        },
        
        removeNotification(id) {
            const index = this.notifications.findIndex(n => n.id === id);
            if (index > -1) {
                this.notifications.splice(index, 1);
            }
        },
        
        // ========================================
        // PAGINACIÓN
        // ========================================
        
        goToPage(page) {
            console.log('📄 Navegando a página:', page);
            
            const newURL = this.buildURL({ page });
            this.navigateTo(newURL);
        },
        
        nextPage() {
            const next = this.currentFilters.page + 1;
            this.goToPage(next);
        },
        
        prevPage() {
            const prev = this.currentFilters.page - 1;
            if (prev > 0) {
                this.goToPage(prev);
            }
        },
        
        // ========================================
        // MODO OSCURO
        // ========================================
        
                    toggleDarkMode() {
                        // Guardar la referencia al contexto de Vue
                        const self = this;
                        
                        // Cambiar el estado
                        self.isDarkMode = !self.isDarkMode;
                        
                        // Aplicar cambios al DOM
                        const html = document.documentElement;
                        
                        if (self.isDarkMode) {
                            html.classList.add('dark');
                            localStorage.setItem('darkMode', 'true');
                            console.log('🌙 Modo oscuro activado');
                        } else {
                            html.classList.remove('dark');
                            localStorage.setItem('darkMode', 'false');
                            console.log('🌞 Modo claro activado');
                        }
                        
                        // NO MÁS MANIPULACIÓN DEL TEXTO - El botón siempre dice "Cambiar tema"
                    },
        
        // ========================================
        // INICIALIZACIÓN
        // ========================================
        
        initializeDarkMode() {
            // Verificar localStorage
            const savedMode = localStorage.getItem('darkMode');
            const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;

            // Determinar modo inicial
            this.isDarkMode = savedMode === 'true' || (savedMode === null && prefersDark);

            // Aplicar modo inicial
            const html = document.documentElement;
            if (this.isDarkMode) {
                html.classList.add('dark');
            } else {
                html.classList.remove('dark');
            }

            // NO MÁS MANIPULACIÓN DEL TEXTO - El botón siempre dice "Cambiar tema"
            console.log('🌙 Modo oscuro inicializado:', this.isDarkMode ? 'Oscuro' : 'Claro');
        },
        
        // updateDarkModeButton() ELIMINADA - No más manipulación directa del DOM
        
        // ========================================
        // UTILIDADES
        // ========================================
        
        truncateText(text, maxLength = 100) {
            if (!text || text.length <= maxLength) return text;
            return text.substring(0, maxLength) + '...';
        },
        
        // ========================================
        // DEBUG Y DESARROLLO
        // ========================================
        
        debugApp() {
            console.log('🐛 Estado actual de la aplicación:', {
                filters: this.currentFilters,
                newsCount: this.newsCount,
                pagination: this.pagination,
                loading: this.isLoading,
                notifications: this.notifications
            });
        }
    }
};

// ========================================
// INICIALIZACIÓN GLOBAL
// ========================================

// Esperar a que el DOM esté listo
document.addEventListener('DOMContentLoaded', () => {
    console.log('🎯 Iniciando DailyNews App con Vue 3...');
    
    // Inicializar modo oscuro al cargar
    const savedMode = localStorage.getItem('darkMode');
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
    const isDarkMode = savedMode === 'true' || (savedMode === null && prefersDark);
    
    if (isDarkMode) {
        document.documentElement.classList.add('dark');
        // NO MÁS MANIPULACIÓN DIRECTA DEL DOM - El botón siempre dice "Cambiar tema"
    }
    
    // Crear instancia de Vue solo si el elemento existe
    const appElement = document.getElementById('app');
    if (appElement) {
        console.log('🎯 Montando Vue 3 en #app (empaquetado localmente)...');
        
        // LOGS DETALLADOS PARA DEBUGGING
        console.log('🔍 DEBUG: Contenido del elemento #app ANTES de Vue:');
        console.log('  - innerHTML length:', appElement.innerHTML.length);
        console.log('  - children count:', appElement.children.length);
        console.log('  - innerHTML preview:', appElement.innerHTML.substring(0, 200) + '...');
        console.log('  - Elementos hijos:', Array.from(appElement.children).map(child => child.tagName));
        
        // Verificar si ya hay contenido renderizado por el servidor
        const hasServerContent = appElement.children.length > 0 || 
                               appElement.innerHTML.trim() !== '';
        
        console.log('🔍 DEBUG: hasServerContent =', hasServerContent);
        
        if (hasServerContent) {
            console.log('📄 Contenido del servidor detectado, Vue solo añadirá interactividad');
            
            // Crear un elemento vacío para Vue (NO montar en #app)
            const vueContainer = document.createElement('div');
            vueContainer.id = 'vue-app';
            vueContainer.style.display = 'none'; // Oculto, solo para funcionalidad
            document.body.appendChild(vueContainer);
            
            // Crear una instancia de Vue más simple que NO reemplace el contenido
            const app = createApp({
                data() {
                    return {
                        // Estado mínimo para interactividad
                        isLoading: false,
                        isRefreshing: false,
                        notifications: [],
                        currentFilters: {
                            lang: 'es',
                            category: '',
                            search: '',
                            page: 1
                        }
                    }
                },
                
                mounted() {
                    console.log('🚀 Vue 3 iniciado (modo interactividad)');
                    
                    // LOGS DETALLADOS DESPUÉS DE MONTAR
                    console.log('🔍 DEBUG: Contenido del elemento #app DESPUÉS de montar Vue:');
                    console.log('  - innerHTML length:', appElement.innerHTML.length);
                    console.log('  - children count:', appElement.children.length);
                    console.log('  - innerHTML preview:', appElement.innerHTML.substring(0, 200) + '...');
                    console.log('  - Elementos hijos:', Array.from(appElement.children).map(child => child.tagName));
                    
                    this.initializeFromURL();
                    this.setupGlobalEvents();
                },
                
                methods: {
                    initializeFromURL() {
                        const urlParams = new URLSearchParams(window.location.search);
                        const pathParts = window.location.pathname.split('/');
                        
                        this.currentFilters.lang = urlParams.get('lang') || 'es';
                        this.currentFilters.search = urlParams.get('q') || urlParams.get('search') || '';
                        this.currentFilters.page = parseInt(urlParams.get('page')) || 1;
                        
                        if (pathParts[1] === 'categoria' && pathParts[2]) {
                            this.currentFilters.category = pathParts[2];
                        } else {
                            this.currentFilters.category = urlParams.get('category') || '';
                        }
                        
                        console.log('🔧 Filtros inicializados desde URL:', this.currentFilters);
                    },
                    
                    setupGlobalEvents() {
                        // Manejar teclas especiales
                        document.addEventListener('keydown', (e) => {
                            if (e.key === 'Escape' && this.currentFilters.search) {
                                this.clearSearch();
                            }
                        });
                        
                        console.log('🎯 Eventos globales configurados');
                    },
                    
                    async refreshNews() {
                        this.isRefreshing = true;
                        console.log('🔄 Actualizando noticias...');
                        
                        try {
                            const response = await fetch('/api/news/refresh', {
                                method: 'POST',
                                headers: {
                                    'Content-Type': 'application/json'
                                }
                            });
                            
                            if (response.ok) {
                                console.log('✅ Noticias actualizadas correctamente');
                                setTimeout(() => {
                                    window.location.reload();
                                }, 2000);
                            } else {
                                throw new Error(`Error ${response.status}: ${response.statusText}`);
                            }
                            
                        } catch (error) {
                            console.error('❌ Error al refrescar noticias:', error);
                        } finally {
                            this.isRefreshing = false;
                        }
                    },
                    
                    clearSearch() {
                        console.log('🧹 Limpiando búsqueda');
                        const searchInput = document.getElementById('search-input');
                        if (searchInput) {
                            searchInput.value = '';
                        }
                        window.location.href = '/';
                    }
                }
            });
            
            // Montar en el contenedor vacío (NO en #app)
            console.log('🔍 DEBUG: Antes de app.mount()');
            app.mount('#vue-app'); // ← MONTAR EN ELEMENTO VACÍO
            console.log('🔍 DEBUG: Después de app.mount()');
            
            // Hacer app disponible globalmente para debugging
            window.app = app;
            
        } else {
            console.log('⚠️ No hay contenido del servidor, usando Vue completo');
            const app = createApp(DailyNewsApp);
            app.mount('#app');
            window.app = app;
        }
    } else {
        console.warn('⚠️ Elemento #app no encontrado');
    }
});

// ========================================
// UTILIDADES GLOBALES
// ========================================

// Función de debounce reutilizable
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func.apply(this, args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}



// Funciones globales para compatibilidad con templates
window.trackNewsClick = function(newsId) {
    console.log('📊 Tracking click en noticia:', newsId);
    // Aquí se puede implementar analytics
};

window.toggleDarkMode = function() {
    // Si Vue está disponible, usar el método de Vue
    if (window.app && window.app.toggleDarkMode) {
        window.app.toggleDarkMode();
        return;
    }
    
    // Fallback para cuando Vue no está disponible
    const html = document.documentElement;
    const isDark = html.classList.contains('dark');
    
    if (isDark) {
        // Cambiar a modo claro
        html.classList.remove('dark');
        localStorage.setItem('darkMode', 'false');
        console.log('🌞 Modo claro activado (fallback)');
    } else {
        // Cambiar a modo oscuro
        html.classList.add('dark');
        localStorage.setItem('darkMode', 'true');
        console.log('🌙 Modo oscuro activado (fallback)');
    }
};

window.refreshNews = function() {
    if (window.app) {
        window.app.refreshNews();
    }
};

window.toggleMobileMenu = function() {
    const mobileMenu = document.getElementById('mobile-menu');
    if (mobileMenu) {
        mobileMenu.classList.toggle('hidden');
    }
};

// Funciones de paginación
window.prevPage = function() {
    const urlParams = new URLSearchParams(window.location.search);
    const currentPage = parseInt(urlParams.get('page')) || 1;
    const prevPage = currentPage - 1;
    if (prevPage > 0) {
        window.location.href = `?page=${prevPage}`;
    }
};

window.nextPage = function() {
    const urlParams = new URLSearchParams(window.location.search);
    const currentPage = parseInt(urlParams.get('page')) || 1;
    const nextPage = currentPage + 1;
    window.location.href = `?page=${nextPage}`;
};

window.goToPage = function(page) {
    window.location.href = `?page=${page}`;
};

window.jumpToPage = function() {
    const input = document.getElementById('page-jump-input');
    if (input && input.value) {
        const page = parseInt(input.value);
        if (page > 0) {
            window.goToPage(page);
        }
    }
};

window.onPageJumpKeypress = function(event) {
    if (event.key === 'Enter') {
        window.jumpToPage();
    }
};

// Exportar para uso global
window.DailyNewsUtils = {
    debounce
};

// Copiar enlace de noticia con feedback visual
window.copyNewsLink = async function(link, btnEl) {
    try {
        await navigator.clipboard.writeText(link);
        // Animación breve del botón
        if (btnEl) {
            btnEl.classList.add('scale-105');
            setTimeout(() => btnEl.classList.remove('scale-105'), 110);
            // Tooltip flotante
            const tooltip = document.createElement('div');
            tooltip.textContent = 'Noticia copiada';
            tooltip.className = 'copy-tooltip';
            document.body.appendChild(tooltip);
            // Calcular posición centrada sobre el botón (viewport coords)
            const rect = btnEl.getBoundingClientRect();
            const centerX = rect.left + rect.width / 2;
            const topY = rect.top; // sobre el botón
            tooltip.style.left = `${Math.round(centerX)}px`;
            tooltip.style.top = `${Math.round(topY)}px`;
            requestAnimationFrame(() => tooltip.classList.add('show'));
            setTimeout(() => {
                tooltip.classList.remove('show');
                setTimeout(() => tooltip.remove(), 220);
            }, 800);
        }
    } catch (err) {
        console.error('No se pudo copiar el enlace:', err);
    }
};