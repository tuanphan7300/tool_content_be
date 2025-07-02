package handler

import (
	"creator-tool-backend/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	googleOauthConfig *oauth2.Config
)

// GoogleUserInfo represents the user info from Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}

// InitGoogleOAuth initializes Google OAuth configuration
func InitGoogleOAuth() {
	conf := config.InfaConfig{}
	conf.LoadConfig()

	googleOauthConfig = &oauth2.Config{
		ClientID:     conf.GoogleClientID,
		ClientSecret: conf.GoogleClientSecret,
		RedirectURL:  conf.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

// GoogleLoginHandler initiates Google OAuth flow
func GoogleLoginHandler(c *gin.Context) {
	if googleOauthConfig == nil {
		InitGoogleOAuth()
	}

	// Generate state parameter for security
	state := generateRandomState()

	// Store state in session or cache (for production, use Redis)
	// For now, we'll use a simple approach

	url := googleOauthConfig.AuthCodeURL(state)
	c.JSON(200, gin.H{
		"auth_url": url,
		"state":    state,
	})
}

// GoogleCallbackHandler handles the OAuth callback from Google
func GoogleCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(400, gin.H{"error": "Authorization code not provided"})
		return
	}

	// Verify state parameter (in production, verify against stored state)
	if state == "" {
		c.JSON(400, gin.H{"error": "State parameter missing"})
		return
	}

	// Exchange code for token
	token, err := googleOauthConfig.Exchange(c, code)
	if err != nil {
		log.Printf("get token failed %s", err)
		c.JSON(400, gin.H{"error": "Failed to exchange token"})
		return
	}

	// Get user info from Google
	userInfo, err := getGoogleUserInfo(token.AccessToken)
	if err != nil {
		c.JSON(400, gin.H{"error": "Failed to get user info"})
		return
	}

	// Check if user exists
	user, err := GetUserByGoogleID(c, userInfo.ID)
	if err != nil {
		// User doesn't exist, create new user
		user, err = CreateGoogleUser(c, userInfo)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create user"})
			return
		}
	}

	// Generate JWT token
	conf := config.InfaConfig{}
	conf.LoadConfig()

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := jwtToken.SignedString([]byte(conf.JWTACCESSKEY))
	if err != nil {
		c.JSON(500, gin.H{"error": "Token creation failed"})
		return
	}

	// Return HTML page that will set the token and close the popup
	html := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Authentication Successful</title>
		</head>
		<body>
			<script>
				// Set token in localStorage
				localStorage.setItem('google_token', '%s');
				localStorage.setItem('google_user', '%s');
				
				// Close popup and notify parent
				if (window.opener) {
					window.opener.postMessage({ type: 'GOOGLE_AUTH_SUCCESS', token: '%s' }, '*');
				}
				window.close();
			</script>
			<p>Authentication successful! You can close this window.</p>
		</body>
		</html>
	`, tokenString, userInfo.Email, tokenString)

	c.Header("Content-Type", "text/html")
	c.String(200, html)
}

// getGoogleUserInfo fetches user information from Google
func getGoogleUserInfo(accessToken string) (*GoogleUserInfo, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo GoogleUserInfo
	err = json.Unmarshal(body, &userInfo)
	if err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// GetUserByGoogleID retrieves user by Google ID
func GetUserByGoogleID(c *gin.Context, googleID string) (config.Users, error) {
	var user config.Users
	if err := config.Db.Where("google_id = ?", googleID).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

// CreateGoogleUser creates a new user from Google OAuth data
func CreateGoogleUser(c *gin.Context, userInfo *GoogleUserInfo) (config.Users, error) {
	user := config.Users{
		GoogleID:      userInfo.ID,
		Email:         userInfo.Email,
		Name:          userInfo.Name,
		Picture:       userInfo.Picture,
		EmailVerified: userInfo.VerifiedEmail,
		AuthProvider:  "google",
		PasswordHash:  "", // No password for OAuth users
	}

	result := config.Db.Create(&user)
	if result.Error != nil {
		return user, result.Error
	}

	return user, nil
}

// generateRandomState generates a random state parameter for OAuth security
func generateRandomState() string {
	// In production, use a proper random generator
	return fmt.Sprintf("state_%d", time.Now().Unix())
}
