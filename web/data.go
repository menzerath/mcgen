package web

// AchievementRequest is the request body for the achievement endpoint.
type AchievementRequest struct {
	Background string `json:"background"`
	Title      string `json:"title"`
	Text       string `json:"text"`

	Output AchievementOutputType `json:"output"`
}

// ErrorResponse is the response body for errors.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// AchievementOutputType is the type of output for the achievement endpoint.
type AchievementOutputType string

// AchievementOutputType constants.
var (
	AchievementOutputTypeDefault  AchievementOutputType = ""
	AchievementOutputTypeDownload AchievementOutputType = "download"
)
