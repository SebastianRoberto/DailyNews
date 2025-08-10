package infrastructure

import (
	"context"
	"fmt"
	"time"

	"dailynews/internal/domain"

	"github.com/robfig/cron/v3"
)

// CronScheduler implementa la programación de tareas periódicas
type CronScheduler struct {
	cron     *cron.Cron
	logger   domain.Logger
	enabled  bool
	schedule string
}

// NewCronScheduler crea una nueva instancia de CronScheduler
func NewCronScheduler(logger domain.Logger, enabled bool, schedule string) *CronScheduler {
	return &CronScheduler{
		cron:     cron.New(cron.WithLocation(time.UTC)),
		logger:   logger,
		enabled:  enabled,
		schedule: schedule,
	}
}

// ScheduleFetchNews programa la tarea de extracción de noticias
func (s *CronScheduler) ScheduleFetchNews(jobFunc func()) error {
	if !s.enabled {
		s.logger.Info("Programación de tareas periódicas deshabilitada en configuración")
		return nil
	}

	// Validar la expresión de programación
	if s.schedule == "" {
		// Usar valor por defecto si no se especifica
		s.schedule = "@daily"
	}

	// Programar la tarea
	_, err := s.cron.AddFunc(s.schedule, func() {
		s.logger.Info("Ejecutando tarea programada de extracción de noticias")
		start := time.Now()

		// Ejecutar la tarea
		jobFunc()

		duration := time.Since(start)
		s.logger.Info("Tarea de extracción de noticias completada",
			"duracion", duration.String())
	})

	if err != nil {
		return fmt.Errorf("error programando tarea: %w", err)
	}

	s.logger.Info("Tarea programada correctamente", "cron_schedule", s.schedule)
	return nil
}

// Start inicia el planificador de tareas
func (s *CronScheduler) Start() error {
	if !s.enabled {
		s.logger.Info("Planificador de tareas deshabilitado en configuración")
		return nil
	}

	s.logger.Info("Iniciando planificador de tareas")
	s.cron.Start()
	return nil
}

// Stop detiene el planificador de tareas
func (s *CronScheduler) Stop() context.Context {
	if s.cron != nil {
		s.logger.Info("Deteniendo planificador de tareas")
		return s.cron.Stop()
	}
	return nil
}
