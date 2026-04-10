package common

// StandardResponse is a common response wrapper
type StandardResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	JobID   string      `json:"job_id,omitempty"`
}

// NewSuccessResponse creates a successful response
func NewSuccessResponse(data interface{}) *StandardResponse {
	return &StandardResponse{
		Success: true,
		Data:    data,
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(err error) *StandardResponse {
	return &StandardResponse{
		Success: false,
		Error:   err.Error(),
	}
}

// NewJobResponse creates a job submission response
func NewJobResponse(jobID string) *StandardResponse {
	return &StandardResponse{
		Success: true,
		JobID:   jobID,
	}
}
