package handlers

import (
  "fmt"
  models "inventory-service/models/sqlc"
  "log/slog"
  "net/http"
  "strconv"
  
  "github.com/gin-gonic/gin"
  "github.com/jackc/pgx/v5"
  "go.opentelemetry.io/otel"
  "go.opentelemetry.io/otel/attribute"
  "go.opentelemetry.io/otel/trace"
)

type Handlers struct {
  db *pgx.Conn
  queries *models.Queries
  tracer trace.Tracer
  getInventoryRequest
}

func NewHandlers(db *pgx.Conn) *Handlers {
  return &Handlers{
    db:      db,
    queries: models.New(db),
    tracer:  otel.Tracer("inventory-service/handlers"),
  }
}

type getInventoryRequest struct {
  Name string `json:"name"`
}

func (h *Handlers) GetInventory(ctx *gin.Context) {
  _, existed := ctx.Get("claims")
  if !existed {
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Claims not found in context",
    })
  }
  idStr := ctx.Param("id")
  id, err := strconv.ParseInt(idStr, 10, 32)
  if err != nil {
    ctx.JSON(http.StatusBadRequest, gin.H{
      "error": "Invalid quantity format",
    })
    return
  }
  inventory, err := h.queries.GetInventory(ctx, id)
  if err != nil {
    slog.Error("Got an error while getting inventories: ", slog.Any(err.Error(), "err"))
  } else {
    ctx.JSON(200, gin.H{
      "message": "Get Inventory Successfully",
      "data": inventory,
    })
  }

}

func (h *Handlers) ListInventory(ctx *gin.Context) {
  // Start a new span for this operation
  spanCtx, span := h.tracer.Start(ctx.Request.Context(), "ListInventory")
  defer span.End()
  
  _, existed := ctx.Get("claims")
  if !existed {
    span.RecordError(fmt.Errorf("claims not found in context"))
    span.SetAttributes(attribute.String("error", "claims_not_found"))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Claims not found in context",
    })
    return
  }
  
  // Add attributes to the span
  span.SetAttributes(
    attribute.Int("inventory.limit", 10),
    attribute.Int("inventory.offset", 0),
  )
  
  inventories, err := h.queries.ListInventory(spanCtx, models.ListInventoryParams{
    Limit: 10,
    Offset: 0,
  })
  if err != nil {
    span.RecordError(err)
    span.SetAttributes(attribute.String("error", "database_query_failed"))
    slog.Error("Got an error while listing inventories: ", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to list inventories",
    })
    return
  }

  // Record successful operation
  span.SetAttributes(
    attribute.Int("inventory.count", len(inventories)),
    attribute.String("operation.status", "success"),
  )

  ctx.JSON(200, gin.H{
    "message": "List Inventory", 
    "data": inventories,
  })
}

func (h *Handlers) UpdateInventory(ctx *gin.Context) {
  _, existed := ctx.Get("claims")
  if !existed {
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Claims not found in context",
    })
    return
  }

  // Get inventory ID from URL parameter
  idStr := ctx.Param("id")
  id, err := strconv.ParseInt(idStr, 10, 64)
  if err != nil {
    ctx.JSON(http.StatusBadRequest, gin.H{
      "error": "Invalid inventory ID",
    })
    return
  }

  // Parse quantity from string to int32
  quantityStr := ctx.PostForm("Quantity")
  quantity, err := strconv.ParseInt(quantityStr, 10, 32)
  if err != nil {
    ctx.JSON(http.StatusBadRequest, gin.H{
      "error": "Invalid quantity format",
    })
    return
  }

  // Start database transaction
  tx, err := h.db.Begin(ctx)
  if err != nil {
    slog.Error("Failed to start transaction", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to start transaction",
    })
    return
  }
  defer tx.Rollback(ctx) // This will be ignored if tx.Commit() succeeds

  // Create queries with transaction
  qtx := h.queries.WithTx(tx)

  // Check if inventory exists before updating
  _, err = qtx.GetInventory(ctx, id)
  if err != nil {
    slog.Error("Inventory not found", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusNotFound, gin.H{
      "error": "Inventory not found",
    })
    return
  }

  // Update inventory within transaction
  param := models.UpdateInventoryParams{
    ID:       id,
    Name:     ctx.PostForm("Name"),
    Unit:     ctx.PostForm("Unit"),
    Quantity: int32(quantity),
    Measure:  ctx.PostForm("Measure"),
    Category: ctx.PostForm("Category"),
    Location: ctx.PostForm("Location"),
  }

  inventory, err := qtx.UpdateInventory(ctx, param)
  if err != nil {
    slog.Error("Could not update inventory", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to update inventory",
    })
    return
  }

  // Commit transaction
  if err := tx.Commit(ctx); err != nil {
    slog.Error("Failed to commit transaction", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to commit transaction",
    })
    return
  }

  ctx.JSON(200, gin.H{
    "message": "Update Inventory Successfully",
    "data":    inventory,
  })
}

func (h *Handlers) CreateInventory(ctx *gin.Context) {
  _, existed := ctx.Get("claims")
  if !existed {
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Claims not found in context",
    })
    return
  }
  
  // Parse quantity from string to int32
  quantityStr := ctx.PostForm("Quantity")
  quantity, err := strconv.ParseInt(quantityStr, 10, 32)
  if err != nil {
    ctx.JSON(http.StatusBadRequest, gin.H{
      "error": "Invalid quantity format",
    })
    return
  }
  
  param := models.CreateInventoryParams{
    Name: ctx.PostForm("Name"),
    Unit: ctx.PostForm("Unit"),
    Quantity: int32(quantity),
    Measure: ctx.PostForm("Measure"),
    Category: ctx.PostForm("Category"),
    Location: ctx.PostForm("Location"),
  }
  
  inventory, err := h.queries.CreateInventory(ctx, param)
  if err != nil {
    slog.Error("Could not create inventory: ", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to create inventory",
    })
    return
  }
  
  ctx.JSON(200, gin.H{
    "message": "Create Inventory Successfully",
    "data": inventory,
  })
}

func (h *Handlers) DeleteInventory(ctx *gin.Context) {
  _, existed := ctx.Get("claims")
  if !existed {
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Claims not found in context",
    })
    return
  }

  idStr := ctx.Param("id")
  id, err := strconv.ParseInt(idStr, 10, 32)
  if err != nil {
    ctx.JSON(http.StatusBadRequest, gin.H{
      "error": "Invalid quantity format",
    })
    return
  }

  err = h.queries.DeleteInventory(ctx, id)
  if err != nil {
    slog.Error("Failed to delete inventory: ", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to delete inventory",
    })
    return
  } else {
    ctx.JSON(200, gin.H{"message": "Delete Inventory Successfully"})
  }

}