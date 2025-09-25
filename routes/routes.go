package routes

import (
	handlers "inventory-service/handlers"
	"inventory-service/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Route struct {
  db *pgx.Conn
  handlers *handlers.Handlers
}

func NewRoute(db *pgx.Conn) *Route {
  return &Route{
    db: db,
    handlers: handlers.NewHandlers(db),
  }
}


func (r *Route) AddWarehouseRoutes(router *gin.Engine) {
  v1 := router.Group("/v1")
  {
    inventory := v1.Group("/warehouse")
    inventory.Use(middlewares.ClerkAuth(r.db))
    {
      inventory.GET("/:id", r.handlers.GetInventory)
      inventory.GET("/list", r.handlers.ListInventory)
      inventory.POST("/create", r.handlers.CreateInventory)
      inventory.PUT("/:id", r.handlers.UpdateInventory)
      inventory.DELETE("/:id", r.handlers.DeleteInventory)
    }
  }
}

func (r *Route) AddStorageRoomRoutes(router *gin.Engine) {
  v1 := router.Group("/v1")
  {
    inventory := v1.Group("/storageRoom")
    inventory.Use(middlewares.ClerkAuth(r.db))
    {
      inventory.GET("/:id", r.handlers.GetInventory)
      inventory.GET("/list", r.handlers.ListInventory)
      inventory.POST("/create", r.handlers.CreateInventory)
      inventory.PUT("/:id", r.handlers.UpdateInventory)
      inventory.DELETE("/:id", r.handlers.DeleteInventory)
    }
  }
}
