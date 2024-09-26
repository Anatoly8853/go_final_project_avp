package handler

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

var jwtSecret = []byte("my_secret_key")

func (h *Handler) SignIn(c *gin.Context) {
	// Получаем пароль из запроса
	var request struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		h.app.Log.Debugf("SignIn неверный формат запроса: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	// Проверяем пароль
	storedPassword := h.config.Password
	if storedPassword == "" || request.Password != storedPassword {
		h.app.Log.Debugf("SignIn Неверный пароль: %v", request.Password)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный пароль"})
		return
	}

	// Формируем JWT-токен с хэшем пароля
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"password_hash": fmt.Sprintf("%x", request.Password),  // Вставляем контрольную сумму пароля
		"exp":           time.Now().Add(8 * time.Hour).Unix(), // Устанавливаем срок жизни токена 8 часов
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		h.app.Log.Debugf("SignIn Ошибка при создании токена: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании токена"})
		return
	}

	// Возвращаем токен в JSON-ответе
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем токен из куки
		tokenString, err := c.Cookie("token")
		if err != nil {
			h.app.Log.Debugf("AuthMiddleware Токен отсутствует: %v", err)
			// Перенаправляем на страницу логина
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Проверяем токен
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Проверяем метод подписи
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				h.app.Log.Debugf("AuthMiddleware неправильный метод подписи: %v", token.Header["alg"])
				return nil, fmt.Errorf("неправильный метод подписи: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		// Если произошла ошибка или токен недействителен
		if err != nil || !token.Valid {
			h.app.Log.Debugf("AuthMiddleware Неверный токен: %v", err)
			// Перенаправляем на страницу логина
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// Если токен валиден, продолжаем выполнение запроса
		c.Next()
	}
}
