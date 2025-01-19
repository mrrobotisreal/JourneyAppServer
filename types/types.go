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
