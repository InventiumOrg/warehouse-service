package routes

import (
	handlers "warehouse-service/handlers"
	// "warehouse-service/middlewares"
	"warehouse-service/observability"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Route struct {
	db                *pgx.Conn
	handlers          *handlers.Handlers
	prometheusMetrics *observability.PrometheusMetrics
}

func NewRoute(db *pgx.Conn, prometheusMetrics *observability.PrometheusMetrics) *Route {
	return &Route{
		db:                db,
		handlers:          handlers.NewHandlers(db, prometheusMetrics),
		prometheusMetrics: prometheusMetrics,
	}
}

func (r *Route) AddWarehouseRoutes(router *gin.Engine) {
	v1 := router.Group("/v1")
	{
		inventory := v1.Group("/warehouse")
		// inventory.Use(middlewares.ClerkAuth(r.db))
		{
			inventory.GET("/:id", r.handlers.GetWarehouse)
			inventory.GET("/list", r.handlers.ListWarehouse)
			inventory.POST("/create", r.handlers.CreateWarehouse)
			inventory.PUT("/:id", r.handlers.UpdateWarehouse)
			inventory.DELETE("/:id", r.handlers.DeleteWarehouse)
		}
	}
}

func (r *Route) AddHealthRoutes(router *gin.Engine) {
	// Health check endpoints (no authentication required)
	router.GET("/healthz", r.handlers.HealthzHandler)
	router.GET("/readyz", r.handlers.ReadyzHandler)
}
