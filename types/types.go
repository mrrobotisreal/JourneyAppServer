package types

type User struct {
	Username string `bson:"username" json:"username"`
	Password string `bson:"password" json:"password"`
	Salt     string `bson:"salt" json:"salt"`
}

type ValidateUsernameRequest struct {
	Username string `bson:"username" json:"username"`
}

type ValidateUsernameResponse struct {
	UsernameAvailable bool `bson:"usernameAvailable" json:"usernameAvailable"`
}

type CreateUserRequest struct {
	Username string `bson:"username" json:"username"`
	Password string `bson:"password" json:"password"`
}

type CreateUserResponse struct {
	Success bool `bson:"success" json:"success"`
}

type ListUsersRequest struct{}

type ListUsersResponse struct{}

type GetUserRequest struct{}

type GetUserResponse struct{}

type LoginRequest struct {
	Username string `bson:"username" json:"username"`
	Password string `bson:"password" json:"password"`
}

type LoginResponse struct {
	Success bool `bson:"success" json:"success"`
}

type LocationData struct {
	Latitude    float64 `bson:"latitude" json:"latitude"`
	Longitude   float64 `bson:"longitude" json:"longitude"`
	DisplayName string  `bson:"displayName" json:"displayName"`
}

type TagData struct {
	Key   string `bson:"key" json:"key"`
	Value string `bson:"value" json:"value"`
}

type CreateNewEntryRequest struct {
	Username  string         `bson:"username" json:"username"`
	Text      string         `bson:"text" json:"text"`
	Timestamp string         `bson:"timestamp" json:"timestamp"`
	Locations []LocationData `bson:"locations" json:"locations"`
	Tags      []TagData      `bson:"tags" json:"tags"`
}

type CreateNewEntryResponse struct {
	UUID string `bson:"uuid" json:"uuid"`
}

type UpdateEntryRequest struct {
	ID        string         `bson:"id" json:"id"`
	Username  string         `bson:"username" json:"username"`
	Text      string         `bson:"text" json:"text"`
	Timestamp string         `bson:"timestamp" json:"timestamp"`
	Locations []LocationData `bson:"locations" json:"locations"`
	Tags      []TagData      `bson:"tags" json:"tags"`
	Images    []string       `bson:"images" json:"images"`
}

type UpdateEntryResponse struct {
	Success bool `bson:"success" json:"success"`
}

type Entry struct {
	ID        string         `bson:"id" json:"id"`
	Username  string         `bson:"username" json:"username"`
	Text      string         `bson:"text" json:"text"`
	Timestamp string         `bson:"timestamp" json:"timestamp"`
	Locations []LocationData `bson:"locations" json:"locations"`
	Tags      []TagData      `bson:"tags" json:"tags"`
	Images    []string       `bson:"images" json:"images"`
}

type EntryListItem struct {
	ID        string         `bson:"id" json:"id"`
	Text      string         `bson:"text" json:"text"`
	Timestamp string         `bson:"timestamp" json:"timestamp"`
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
