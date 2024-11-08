package auth

// import (
// 	"net/http"

// 	"github.com/dgrijalva/jwt-go"
// )

// // GenerateAdminToken generates a JWT token specifically for an admin role
// func  (username string) (string, error) {
// 	// return generateToken(username, "admin", time.Now().Add(15*time.Minute))
// 	return GenerateRSAToken(username, "admin")

// }

// // AdminMiddleware validates JWT tokens for admins and checks their claims
// func AdminMiddleware(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		tokenString := r.Header.Get("Authorization") // Retrieve token from Authorization header

// 		claims := &Claims{}
// 		// Parse token and validate claims
// 		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
// 			return jwtKey, nil
// 		})

// 		// Check for token validity and if the role is "admin"
// 		if err != nil || !token.Valid || claims.Role != "admin" {
// 			http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 			return
// 		}

// 		// Set admin information in the context if needed (optional)
// 		// ctx := context.WithValue(r.Context(), "username", claims.Username)
// 		// r = r.WithContext(ctx)

// 		next.ServeHTTP(w, r)
// 	})
// }
