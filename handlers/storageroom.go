package handlers

import (
  "fmt"
  models "warehouse-service/models/sqlc"
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
}

func NewHandlers(db *pgx.Conn) *Handlers {
  return &Handlers{
    db:      db,
    queries: models.New(db),
    tracer:  otel.Tracer("warehouse-service/handlers"),
  }
}

func (h *Handlers) GetStorageRoom(ctx *gin.Context) {
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
  storageRoom, err := h.queries.GetStorageRoom(ctx, id)
  if err != nil {
    slog.Error("Got an error while getting inventories: ", slog.Any(err.Error(), "err"))
  } else {
    ctx.JSON(200, gin.H{
      "message": "Get Storage Room Successfully",
      "data": storageRoom,
    })
  }

}

func (h *Handlers) ListStorageRoom(ctx *gin.Context) {
  // Start a new span for this operation
  spanCtx, span := h.tracer.Start(ctx.Request.Context(), "List Storage Room")
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
    attribute.Int("storageRoom.limit", 10),
    attribute.Int("storageRoom.offset", 0),
  )
  
  storageRooms, err := h.queries.ListStorageRoom(spanCtx, models.ListStorageRoomParams{
    Limit: 10,
    Offset: 0,
  })
  if err != nil {
    span.RecordError(err)
    span.SetAttributes(attribute.String("error", "database_query_failed"))
    slog.Error("Got an error while listing storage rooms: ", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to list storage rooms",
    })
    return
  }

  // Record successful operation
  span.SetAttributes(
    attribute.Int("storageRoom.count", len(inventories)),
    attribute.String("operation.status", "success"),
  )

  ctx.JSON(200, gin.H{
    "message": "List Storage Room Succesfully", 
    "data": storageRooms,
  })
}

func (h *Handlers) UpdateStorageRoom(ctx *gin.Context) {
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
      "error": "Invalid storage room ID",
    })
    return
  }

  // Parse quantity from string to int32
  storageRoomNumberStr := ctx.PostForm("Number")
  storageRoomNumber, err := strconv.ParseInt(storageRoomNumberStr, 10, 32)
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
  _, err = qtx.GetStorageRoom(ctx, id)
  if err != nil {
    slog.Error("Storage room not found", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusNotFound, gin.H{
      "error": "Storage room not found",
    })
    return
  }

  // Update inventory within transaction
  param := models.UpdateStorageRoomParams{
    ID:       id,
    Name:     ctx.PostForm("Name"),
    Number:   int32(number),
    Warehouse: ctx.PostForm("Warehouse"),
  }

  storageRoom, err := qtx.UpdateStorageRoom(ctx, param)
  if err != nil {
    slog.Error("Could not update storage room", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to update storage room",
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
    "message": "Update Storage Room Successfully",
    "data":    storageRoom,
  })
}

func (h *Handlers) CreateStorageRoom(ctx *gin.Context) {
  _, existed := ctx.Get("claims")
  if !existed {
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Claims not found in context",
    })
    return
  }
  
  // Parse quantity from string to int32
  storageRoomNumberStr := ctx.PostForm("Number")
  storageRoomNumber, err := strconv.ParseInt(storageRoomNumberStr, 10, 32)
  if err != nil {
    ctx.JSON(http.StatusBadRequest, gin.H{
      "error": "Invalid quantity format",
    })
    return
  }
  
  param := models.CreateStorageRoomParams{
    Name: ctx.PostForm("Name"),
    Number: int32(storageRoomNumber),
    Warehouse: ctx.PostForm("Warehouse"),
  }
  
  storageRoom, err := h.queries.createStorageRoom(ctx, param)
  if err != nil {
    slog.Error("Could not create inventory: ", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to create inventory",
    })
    return
  }
  
  ctx.JSON(200, gin.H{
    "message": "Create Storage Room Successfully",
    "data": storageRoom,
  })
}

func (h *Handlers) DeleteStorageRoom(ctx *gin.Context) {
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

  err = h.queries.DeleteStorageRoom(ctx, id)
  if err != nil {
    slog.Error("Failed to delete storage room: ", slog.Any("err", err.Error()))
    ctx.JSON(http.StatusInternalServerError, gin.H{
      "error": "Failed to delete storage room",
    })
    return
  } else {
    ctx.JSON(200, gin.H{"message": "Delete Storage Room Successfully"})
  }

}