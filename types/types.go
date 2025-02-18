package types

import "time"

type contextKey string

const (
	UsernameContextKey contextKey = "username"
	APIKeyContextKey   contextKey = "apiKey"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type APIKey struct {
	Key       string    `bson:"key" json:"key"`
	Created   time.Time `bson:"created" json:"created"`
	LastUsed  time.Time `bson:"lastUsed" json:"lastUsed"`
	ExpiresAt time.Time `bson:"expiresAt" json:"expiresAt"`
}

type User struct {
	UserID   string `bson:"userId" json:"userId"`
	Username string `bson:"username" json:"username"`
	Password string `bson:"password" json:"password"`
	Salt     string `bson:"salt" json:"salt"`
	APIKey   APIKey `bson:"apiKey" json:"apiKey"`
	Font     string `bson:"font" json:"font"`
}

type UserListItem struct {
	UserID   string `bson:"userId" json:"userId"`
	Username string `bson:"username" json:"username"`
}

type ValidateUsernameRequest struct {
	Username string `json:"username"`
}

type ValidateUsernameResponse struct {
	UsernameAvailable bool `json:"usernameAvailable"`
}

type CreateUserRequest struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	SessionOption string `json:"sessionOption"`
}

type CreateUserResponse struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Success  bool   `json:"success"`
	Token    string `json:"token,omitempty"`
	APIKey   string `json:"apiKey,omitempty"`
	Font     string `json:"font"`
}

type UpdateUserRequest struct {
	UserID        string `json:"userId"`
	Username      string `json:"username"`
	SessionOption string `json:"sessionOption"`
}

type UpdateUserResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	APIKey  string `json:"apiKey,omitempty"`
}

type DeleteAccountResponse struct {
	Success bool `json:"success"`
}

type LoginRequest struct {
	Username          string `json:"username"`
	Password          string `json:"password"`
	SessionOption     string `json:"sessionOption"`
	RespondWithAPIKey bool   `json:"respondWithApiKey,omitempty"`
	Key               string `json:"key,omitempty"`
}

type LoginResponse struct {
	UserID   string `json:"userId,omitempty"`
	Username string `json:"username,omitempty"`
	Success  bool   `json:"success"`
	Token    string `json:"token,omitempty"`
	APIKey   string `json:"apiKey,omitempty"`
	Font     string `json:"font,omitempty"`
}

type LocationData struct {
	Latitude    float64 `bson:"latitude" json:"latitude"`
	Longitude   float64 `bson:"longitude" json:"longitude"`
	DisplayName string  `bson:"displayName" json:"displayName"`
}

type TagData struct {
	Key   string `bson:"key" json:"key"`
	Value string `bson:"value,omitempty" json:"value,omitempty"`
}

type CreateNewEntryRequest struct {
	UserID    string         `json:"userId"`
	Username  string         `bson:"username" json:"username"`
	Text      string         `bson:"text" json:"text"`
	Timestamp time.Time      `bson:"timestamp" json:"timestamp"`
	Locations []LocationData `bson:"locations" json:"locations"`
	Tags      []TagData      `bson:"tags" json:"tags"`
	Images    []string       `bson:"images" json:"images"`
}

type CreateNewEntryResponse struct {
	ID          string         `bson:"id" json:"id"`
	UserID      string         `json:"userId"`
	Username    string         `bson:"username" json:"username"`
	Text        string         `bson:"text" json:"text"`
	Timestamp   time.Time      `bson:"timestamp" json:"timestamp"`
	LastUpdated time.Time      `bson:"lastUpdated" json:"lastUpdated"`
	Locations   []LocationData `bson:"locations" json:"locations"`
	Tags        []TagData      `bson:"tags" json:"tags"`
	Images      []string       `bson:"images" json:"images"`
}

type UpdateEntryRequest struct {
	ID          string         `bson:"id" json:"id"`
	UserID      string         `json:"userId"`
	Username    string         `bson:"username" json:"username"`
	Text        string         `bson:"text" json:"text"`
	Timestamp   time.Time      `bson:"timestamp" json:"timestamp"`
	LastUpdated time.Time      `bson:"lastUpdated" json:"lastUpdated"`
	Locations   []LocationData `bson:"locations" json:"locations"`
	Tags        []TagData      `bson:"tags" json:"tags"`
	Images      []string       `bson:"images" json:"images"`
}

type UpdateEntryResponse struct {
	Success bool `bson:"success" json:"success"`
}

type Entry struct {
	ID          string         `bson:"id" json:"id"`
	UserID      string         `bson:"userId" json:"userId"`
	Username    string         `bson:"username" json:"username"`
	Text        string         `bson:"text" json:"text"`
	Timestamp   time.Time      `bson:"timestamp" json:"timestamp"`
	LastUpdated time.Time      `bson:"lastUpdated" json:"lastUpdated"`
	Locations   []LocationData `bson:"locations" json:"locations"`
	Tags        []TagData      `bson:"tags" json:"tags"`
	Images      []string       `bson:"images" json:"images"`
}

type EntryListItem struct {
	ID        string         `bson:"id" json:"id"`
	Text      string         `bson:"text" json:"text"`
	Timestamp time.Time      `bson:"timestamp" json:"timestamp"`
	Locations []LocationData `bson:"locations" json:"locations"`
	Tags      []TagData      `bson:"tags" json:"tags"`
	Images    []string       `bson:"images" json:"images"`
}

type ListEntriesParams struct {
	User      string
	Locations []LocationData
	Tags      []TagData
	Limit     int64
	Page      int64
	SortRule  string
}

type GetEntryRequest struct{}

type GetEntryResponse struct{}

type SearchEntriesRequest struct {
	User        string         `bson:"user" json:"user"`
	Page        int64          `bson:"page" json:"page"`
	Limit       int64          `bson:"limit" json:"limit"`
	SearchQuery string         `bson:"searchQuery" json:"searchQuery"`
	Locations   []LocationData `bson:"locations" json:"locations"`
	Tags        []TagData      `bson:"tags" json:"tags"`
	SortRule    string         `bson:"sortRule" json:"sortRule"`
	Timeframe   string         `bson:"timeframe" json:"timeframe"`
	FromDate    string         `bson:"fromDate" json:"fromDate"`
	ToDate      string         `bson:"toDate" json:"toDate"`
}

type SearchEntriesResponse struct{} // May not need this. TODO: remove if not needed

type AddTagRequest struct {
	Username  string    `json:"username"`
	UserID    string    `json:"userId"`
	Timestamp time.Time `json:"timestamp"`
	EntryID   string    `json:"entryId"`
	Tags      []TagData `json:"tags"`
}

type AddTagResponse struct {
	Success bool `json:"success"`
}

type DeleteTagRequest struct {
	Username  string    `json:"username"`
	UserID    string    `json:"userId"`
	Timestamp time.Time `json:"timestamp"`
	EntryID   string    `json:"entryId"`
	Tags      []TagData `json:"tags"`
}

type DeleteTagResponse struct {
	Success bool `json:"success"`
}

type AddLocationRequest struct {
	Username  string         `json:"username"`
	UserID    string         `json:"userId"`
	Timestamp time.Time      `json:"timestamp"`
	EntryID   string         `json:"entryId"`
	Locations []LocationData `json:"locations"`
}

type AddLocationResponse struct {
	Success bool `json:"success"`
}

type DeleteLocationRequest struct {
	Username  string         `json:"username"`
	UserID    string         `json:"userId"`
	Timestamp time.Time      `json:"timestamp"`
	EntryID   string         `json:"entryId"`
	Locations []LocationData `json:"locations"`
}

type DeleteLocationResponse struct {
	Success bool `json:"success"`
}

type AddImageRequest struct {
	Username  string    `json:"username"`
	UserID    string    `json:"userId"`
	Timestamp time.Time `json:"timestamp"`
	EntryID   string    `json:"entryId"`
	Images    []string  `json:"images"`
}

type AddImageResponse struct {
	Success bool `json:"success"`
}

type DeleteImageRequest struct {
	Username      string    `json:"username"`
	UserID        string    `json:"userId"`
	Timestamp     time.Time `json:"timestamp"`
	EntryID       string    `json:"entryId"`
	Images        []string  `json:"images"`
	ImageToDelete string    `json:"imageToDelete"`
}

type DeleteImageResponse struct {
	Success bool `json:"success"`
}
