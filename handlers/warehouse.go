package handlers

import (
  "fmt"
  "log/slog"
  "net/http"
  "strconv"
  models "warehouse-service/models/sqlc"

  "github.com/gin-gonic/gin"
  "go.opentelemetry.io/otel/attribute"
  )

func (h *Handlers) GetWarehouse(ctx *gin.Context) {
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
      "error": "Invalid warehouse ID format",
    })
    return
  }
  warehouse, err := h.queries.GetWarehouse(ctx, id)
  if err != nil {
    slog.Error("Got an error while getting inventories: ", slog.Any(err.Error(), "err"))
  } else {
    ctx.JSON(200, gin.H{
      "message": "Get Warehouse Successfully",
      "data":    warehouse,
    })
  }

  }

func (h *Handlers) ListWarehouse(ctx *gin.Context) {
  // Start a new span for this operation
  spanCtx, span := h.tracer.Start(ctx.Request.Context(), "ListWarehouse")
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
    attribute.Int("warehouse.limit", 10),
    attribute.Int("warehouse.offset", 0),
  )

  warehouses, err := h.queries.ListWarehouse(spanCtx, models.ListWarehouseParams{
    Limit:  10,
    Offset: 0,
  })
  if err != nil {
    span.RecordError(err)
    span.SetAttributes(attribute.String("error", "database_query_failed"))
    slog.Error("Got an error while listing warehouses: ", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to list warehouses",
    })
    return
  }

  // Record successful operation
  span.SetAttributes(
    attribute.Int("warehouse.count", len(warehouses)),
    attribute.String("operation.status", "success"),
  )

  ctx.JSON(200, gin.H{
    "message": "List Warehouse Successfully",
    "data":    warehouses,
  })
  }

func (h *Handlers) UpdateWarehouse(ctx *gin.Context) {
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
      "error": "Invalid warehouse ID",
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

  // Check if warehouse exists before updating
  _, err = qtx.GetWarehouse(ctx, id)
  if err != nil {
    slog.Error("Warehouse not found", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusNotFound, gin.H{
      "error": "Warehouse not found",
    })
    return
  }

  // Update warehouse within transaction
  param := models.UpdateWarehouseParams{
    ID:      id,
    Name:    ctx.PostForm("Name"),
    Address: ctx.PostForm("Address"),
    Ward:    ctx.PostForm("Ward"),
    City:    ctx.PostForm("City"),
    Country: ctx.PostForm("Country"),
  }

  warehouse, err := qtx.UpdateWarehouse(ctx, param)
  if err != nil {
    slog.Error("Could not update warehouse", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to update warehouse",
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
    "message": "Update Warehouse Successfully",
    "data":    warehouse,
  })
  }

func (h *Handlers) CreateWarehouse(ctx *gin.Context) {
  _, existed := ctx.Get("claims")
  if !existed {
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Claims not found in context",
    })
    return
  }

  param := models.CreateWarehouseParams{
    Name:    ctx.PostForm("Name"),
    Address: ctx.PostForm("Address"),
    Ward:    ctx.PostForm("Ward"),
    City:    ctx.PostForm("City"),
    Country: ctx.PostForm("Country"),
  }

  warehouse, err := h.queries.CreateWarehouse(ctx, param)
  if err != nil {
    slog.Error("Could not create warehouse: ", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to create warehouse",
    })
    return
  }

  ctx.JSON(200, gin.H{
    "message": "Create Warehouse Successfully",
    "data":    warehouse,
  })
  }

  func (h *Handlers) DeleteWarehouse(ctx *gin.Context) {
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
      "error": "Invalid warehouse ID format",
    })
    return
  }

  err = h.queries.DeleteWarehouse(ctx, id)
  if err != nil {
    slog.Error("Failed to delete warehouse: ", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to delete warehouse",
    })
    return
  } else {
    ctx.JSON(200, gin.H{"message": "Delete Warehouse Successfully"})
  }

}
