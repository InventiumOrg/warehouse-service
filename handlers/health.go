package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthzHandler handles the /healthz endpoint for basic liveness check
func (h *Handlers) HealthzHandler(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "warehouse-service",
	})
}

// ReadyzHandler handles the /readyz endpoint for readiness check
func (h *Handlers) ReadyzHandler(ctx *gin.Context) {
	// Check database connection
	if err := h.db.Ping(context.Background()); err != nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "not ready",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"service":   "warehouse-service",
			"error":     "database connection failed",
			"details":   err.Error(),
		})
		return
	}

	// All checks passed
	ctx.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "warehouse-service",
		"checks": gin.H{
			"database": "ok",
		},
	})
}