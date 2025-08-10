package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"dailynews/internal/domain"
)

// unitOfWork implementa la interfaz UnitOfWork
type unitOfWork struct {
	db    *gorm.DB
	tx    *gorm.DB
	repos map[string]interface{}
}

// NewUnitOfWork crea una nueva instancia de UnitOfWork
func NewUnitOfWork(db *gorm.DB) domain.UnitOfWork {
	if db == nil {
		panic("db no puede ser nil")
	}

	return &unitOfWork{
		db:    db,
		repos: make(map[string]interface{}),
	}
}

// Begin inicia una nueva transacción
func (u *unitOfWork) Begin(ctx context.Context) (domain.UnitOfWork, error) {
	if u.tx != nil {
		return nil, errors.New("ya existe una transacción activa")
	}

	tx := u.db.Begin()
	if tx.Error != nil {
		return nil, fmt.Errorf("error al iniciar transacción: %w", tx.Error)
	}

	return &unitOfWork{
		db:    u.db,
		tx:    tx,
		repos: make(map[string]interface{}),
	}, nil
}

// Commit confirma la transacción actual
func (u *unitOfWork) Commit() error {
	if u.tx == nil {
		return errors.New("no hay transacción activa para hacer commit")
	}

	if err := u.tx.Commit().Error; err != nil {
		return fmt.Errorf("error al hacer commit: %w", err)
	}

	u.tx = nil
	return nil
}

// Rollback deshace la transacción actual
func (u *unitOfWork) Rollback() error {
	if u.tx == nil {
		return errors.New("no hay transacción activa para hacer rollback")
	}

	if err := u.tx.Rollback().Error; err != nil {
		return fmt.Errorf("error al hacer rollback: %w", err)
	}

	u.tx = nil
	return nil
}

// Countries retorna el repositorio de países
func (u *unitOfWork) Countries() domain.CountryRepository {
	if repo, ok := u.repos["countries"]; ok {
		return repo.(domain.CountryRepository)
	}

	repo := NewCountryRepository(u.getDB())
	u.repos["countries"] = repo
	return repo
}

// Categories retorna el repositorio de categorías
func (u *unitOfWork) Categories() domain.CategoryRepository {
	if repo, ok := u.repos["categories"]; ok {
		return repo.(domain.CategoryRepository)
	}

	repo := NewCategoryRepository(u.getDB())
	u.repos["categories"] = repo
	return repo
}

// NewsSources retorna el repositorio de fuentes de noticias
func (u *unitOfWork) NewsSources() domain.NewsSourceRepository {
	if repo, ok := u.repos["news_sources"]; ok {
		return repo.(domain.NewsSourceRepository)
	}

	repo := NewNewsSourceRepository(u.getDB())
	u.repos["news_sources"] = repo
	return repo
}

// NewsItems retorna el repositorio de noticias
func (u *unitOfWork) NewsItems() domain.NewsItemRepository {
	if repo, ok := u.repos["news_items"]; ok {
		return repo.(domain.NewsItemRepository)
	}

	repo := NewNewsItemRepository(u.getDB())
	u.repos["news_items"] = repo
	return repo
}

// getDB retorna la instancia de base de datos actual (transacción o no)
func (u *unitOfWork) getDB() *gorm.DB {
	if u.tx != nil {
		return u.tx
	}
	return u.db
}
