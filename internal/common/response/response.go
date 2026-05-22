package response

import "github.com/gin-gonic/gin"

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

type ListEnvelope[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

func JSONError(c *gin.Context, status int, code, message string) {
	requestID, _ := c.Get("request_id")
	c.AbortWithStatusJSON(status, ErrorResponse{
		Error: ErrorBody{
			Code:      code,
			Message:   message,
			RequestID: toString(requestID),
		},
	})
}

func JSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}

func toString(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
