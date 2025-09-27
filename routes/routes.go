package routes

import (
	handlers "warehouse-service/handlers"
	"warehouse-service/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Route struct {
	db       *pgx.Conn
	handlers *handlers.Handlers
}

func NewRoute(db *pgx.Conn) *Route {
	return &Route{
		db:       db,
		handlers: handlers.NewHandlers(db),
	}
}

func (r *Route) AddWarehouseRoutes(router *gin.Engine) {
	v1 := router.Group("/v1")
	{
		inventory := v1.Group("/warehouse")
		inventory.Use(middlewares.ClerkAuth(r.db))
		{
			inventory.GET("/:id", r.handlers.GetWarehouse)
			inventory.GET("/list", r.handlers.ListWarehouse)
			inventory.POST("/create", r.handlers.CreateWarehouse)
			inventory.PUT("/:id", r.handlers.UpdateWarehouse)
			inventory.DELETE("/:id", r.handlers.DeleteWarehouse)
		}
	}
}

func (r *Route) AddStorageRoomRoutes(router *gin.Engine) {
	v1 := router.Group("/v1")
	{
		inventory := v1.Group("/storageRoom")
		inventory.Use(middlewares.ClerkAuth(r.db))
		{
			inventory.GET("/:id", r.handlers.GetStorageRoom)
			inventory.GET("/list", r.handlers.ListStorageRoom)
			inventory.POST("/create", r.handlers.CreateStorageRoom)
			inventory.PUT("/:id", r.handlers.UpdateStorageRoom)
			inventory.DELETE("/:id", r.handlers.DeleteStorageRoom)
		}
	}
}
