package middleware

import (
	"JourneyAppServer/db"
	"JourneyAppServer/types"
	"JourneyAppServer/utils"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strings"
	"time"
)

func ValidateJWTMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			sendError(w, "Authorization header missing", http.StatusUnauthorized)
			return
		}

		splitToken := strings.Split(authHeader, "Bearer ")
		if len(splitToken) != 2 {
			sendError(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		tokenStr := splitToken[1]

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(utils.GetJWTSecret()), nil
		})

		if err != nil {
			utils.LM.Logger.Printf("Invalid JWT token: %v", err)
			sendError(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if exp, ok := claims["exp"].(float64); ok {
				if time.Now().Unix() > int64(exp) {
					utils.LM.Logger.Printf("JWT token expired for token: %s", tokenStr)
					sendError(w, "Token has expired", http.StatusUnauthorized)
					return
				}
			}

			username, ok := claims["username"].(string)
			if !ok {
				utils.LM.Logger.Printf("Invalid JWT claims: missing username")
				sendError(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), types.UsernameContextKey, username)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			utils.LM.Logger.Printf("Invalid JWT claims for token: %s", tokenStr)
			sendError(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}
	}
}

func ValidateAPIKeyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			sendError(w, "API key missing", http.StatusUnauthorized)
			return
		}

		var user types.User
		query := `
            SELECT user_id, username, password, salt, 
                   api_key, api_key_created, api_key_last_used, api_key_expires_at, font
            FROM users WHERE api_key = ?
        `
		err := db.SDB.QueryRow(query, apiKey).Scan(
			&user.UserID, &user.Username, &user.Password, &user.Salt,
			&user.APIKey.Key, &user.APIKey.Created, &user.APIKey.LastUsed, &user.APIKey.ExpiresAt, &user.Font,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				utils.LM.Logger.Printf("No user found for API key: %s", apiKey)
				sendError(w, "Invalid API key", http.StatusUnauthorized)
				return
			}
			utils.LM.Logger.Printf("Database error validating API key %s: %v", apiKey, err)
			sendError(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		if err := utils.ValidateAPIKey(&user.APIKey); err != nil {
			utils.LM.Logger.Printf("API key validation failed for %s: %v", apiKey, err)
			sendError(w, fmt.Sprintf("API key validation failed: %v", err), http.StatusUnauthorized)
			return
		}

		limiter := utils.NewRateLimiter().GetLimiter(apiKey)
		if !limiter.Allow() {
			utils.LM.Logger.Printf("Rate limit exceeded for API key: %s", apiKey)
			sendError(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		updateQuery := `
            UPDATE users 
            SET api_key_last_used = NOW()
            WHERE api_key = ?
        `
		_, err = db.SDB.Exec(updateQuery, apiKey)
		if err != nil {
			utils.LM.Logger.Printf("Error updating API key last used time for %s: %v", apiKey, err)
		}

		//ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		//defer cancel()
		//
		//collection := db.MongoClient.Database(db.DbName).Collection(db.UserCollection)
		//var user types.User
		//err := collection.FindOne(ctx, bson.M{"apiKey.key": apiKey}).Decode(&user)
		//if err != nil {
		//	sendError(w, "Invalid API key", http.StatusUnauthorized)
		//	return
		//}
		//
		//if err := utils.ValidateAPIKey(&user.APIKey); err != nil {
		//	sendError(w, fmt.Sprintf("API key validation failed: %v", err), http.StatusUnauthorized)
		//	return
		//}
		//
		//limiter := utils.NewRateLimiter().GetLimiter(apiKey)
		//if !limiter.Allow() {
		//	sendError(w, "Rate limit exceeded", http.StatusTooManyRequests)
		//	return
		//}
		//
		//_, err = collection.UpdateOne(
		//	ctx,
		//	bson.M{"apiKey.key": apiKey},
		//	bson.M{"$set": bson.M{"apiKey.last_used": time.Now()}},
		//)
		//if err != nil {
		//	fmt.Printf("Error updating API key last used time: %v\n", err)
		//}

		ctx := context.WithValue(r.Context(), types.APIKeyContextKey, apiKey)
		ctx = context.WithValue(ctx, types.UsernameContextKey, user.Username)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func CombinedAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return ValidateJWTMiddleware(ValidateAPIKeyMiddleware(next))
}

func sendError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(types.ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}

func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(types.UsernameContextKey).(string)
	return username, ok
}

func GetAPIKeyFromContext(ctx context.Context) (string, bool) {
	apiKey, ok := ctx.Value(types.APIKeyContextKey).(string)
	return apiKey, ok
}
