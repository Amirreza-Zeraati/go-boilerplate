// Package response standardizes JSON output so every endpoint returns the same
// shape: {"data": ...} on success, {"error": {...}} on failure.
package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Envelope is the success shape.
type Envelope struct {
	Data any `json:"data"`
}

// ErrorBody is the failure shape.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// JSON writes a success payload.
func JSON(c *gin.Context, status int, data any) {
	c.JSON(status, Envelope{Data: data})
}

// Error writes an error payload with a message.
func Error(c *gin.Context, status int, message string) {
	c.JSON(status, ErrorBody{Error: ErrorDetail{Message: message}})
}

// AbortError writes an error payload and stops the middleware chain. Use inside
// middleware so no downstream handler runs.
func AbortError(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, ErrorBody{Error: ErrorDetail{Message: message}})
}

// ValidationError inspects a binding error. If it's a validator error it
// returns a 422 with per-field messages; otherwise a generic 400.
func ValidationError(c *gin.Context, err error) {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make(map[string]string, len(ve))
		for _, fe := range ve {
			fields[fe.Field()] = messageFor(fe)
		}
		c.JSON(http.StatusUnprocessableEntity, ErrorBody{Error: ErrorDetail{
			Message: "validation failed",
			Fields:  fields,
		}})
		return
	}
	Error(c, http.StatusBadRequest, "invalid request body")
}

func messageFor(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "min":
		return "is too short (min " + fe.Param() + ")"
	case "max":
		return "is too long (max " + fe.Param() + ")"
	default:
		return "is invalid"
	}
}
