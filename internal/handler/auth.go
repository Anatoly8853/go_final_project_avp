package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// TokenTimeHour время жизни токена.
const TokenTimeHour = 8

// SignIn обработчик введенного пароля.
func (h *Handler) SignIn(c *gin.Context) {
	// Получаем пароль из запроса
	var request struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		h.app.Log.Debugf("SignIn неверный формат запроса: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный формат запроса"})
		return
	}

	// Проверяем пароль
	storedPassword := h.config.Password
	if storedPassword == "" || request.Password != storedPassword {
		h.app.Log.Debugf("SignIn Неверный пароль: %v", request.Password)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный пароль"})
		return
	}

	// Округляем текущее время до начала ближайшего 8-часового периода
	now := time.Now()
	startOf8HourPeriod := now.Truncate(TokenTimeHour * time.Hour)

	// Формируем JWT-токен с хэшем пароля
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		// Вставляем контрольную сумму пароля
		"password_hash": fmt.Sprintf("%x", request.Password),
		// Устанавливаем срок жизни токена 8 часов
		"exp": startOf8HourPeriod.Add(TokenTimeHour * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(h.config.JwtSecret))
	if err != nil {
		h.app.Log.Debugf("SignIn Ошибка при создании токена: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при создании токена"})
		return
	}

	// Устанавливаем токен в куку
	c.SetCookie("token", tokenString, TokenTimeHour*3600, "/", "", false, true)

	// Возвращаем токен в JSON-ответе
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// AuthMiddleware проверка JWT-токена.
func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем токен из куки
		tokenString, err := c.Cookie("token")
		if err != nil {
			h.app.Log.Debugf("AuthMiddleware Токен отсутствует: %v", err)
			// Перенаправляем на страницу логина
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Токен отсутствует"})
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
			return []byte(h.config.JwtSecret), nil
		})

		// Если произошла ошибка или токен недействителен
		if err != nil || !token.Valid {
			h.app.Log.Debugf("AuthMiddleware Неверный токен: %v", err)
			// Перенаправляем на страницу логина
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный токен"})
			c.Abort()
			return
		}

		// Проверяем хэш пароля
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			h.app.Log.Debugf("AuthMiddleware Неверный формат claims")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверный формат claims"})
			c.Abort()
			return
		}

		passwordHash, ok := claims["password_hash"].(string)
		if !ok {
			h.app.Log.Debugf("AuthMiddleware Отсутствует хэш пароля в токене")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Отсутствует хэш пароля в токене"})
			c.Abort()
			return
		}

		// Проверяем, что хэш пароля в токене соответствует текущему паролю
		currentPasswordHash := fmt.Sprintf("%x", h.config.Password)
		if passwordHash != currentPasswordHash {
			h.app.Log.Debugf("AuthMiddleware Несоответствие хэша пароля")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Несоответствие хэша пароля"})
			c.Abort()
			return
		}
		// Если токен валиден и хэш пароля совпадает, продолжаем выполнение запроса
		c.Next()
	}
}
