package rest

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/didip/tollbooth"
	"github.com/gin-gonic/gin"
	"github.com/umputun/secrets/app/store"
)

// Server is a rest with store
type Server struct {
	Store store.Interface
}

//Run the lister and request's router, activate rest server
func (s Server) Run() {
	log.Printf("[INFO] activate rest server")

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(s.limiterMiddleware())
	router.Use(s.loggerMiddleware())

	router.POST("/v1/message", s.saveMessageCtrl)
	router.GET("/v1/message/:key/:pin", s.getMessageCtrl)

	log.Fatal(router.Run(":8080"))
}

// /v1/message
func (s Server) saveMessageCtrl(c *gin.Context) {
	request := struct {
		Message string `binding:"required"`
		Exp     int    `binding:"required"`
		Pin     string `binding:"required"`
	}{}

	err := c.BindJSON(&request)
	if err != nil {
		log.Printf("[WARN] can't bind request %v", request)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	r, err := s.Store.Save(time.Minute*time.Duration(request.Exp), request.Message, request.Pin)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"key": r.Key, "exp": r.Exp})
}

// /v1/message/:key/:pin
func (s Server) getMessageCtrl(c *gin.Context) {
	key, pin := c.Param("key"), c.Param("pin")
	if key == "" || pin == "" {
		log.Print("[WARN] no key or pin in get request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "no key passed"})
		return
	}
	r, err := s.Store.Load(key, pin)
	if err != nil {
		log.Printf("[WARN] failed to load key %v", key)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": r.Key, "message": r.Data})
}

func (s Server) limiterMiddleware() gin.HandlerFunc {
	limiter := tollbooth.NewLimiter(5, time.Second)
	return func(c *gin.Context) {
		keys := []string{c.ClientIP(), c.Request.Header.Get("User-Agent")}
		if httpError := tollbooth.LimitByKeys(limiter, keys); httpError != nil {
			c.JSON(httpError.StatusCode, gin.H{"error": httpError.Message})
			c.Abort()
		} else {
			c.Next()
		}
	}
}

func (s Server) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		c.Next()

		body := ""
		if b, ok := c.Get("post"); ok {
			body = fmt.Sprintf("%v", b)
		}

		log.Printf("[INFO] %s %s {%s} %s %v %d",
			c.Request.Method, c.Request.URL.Path, body,
			c.ClientIP(), time.Since(t), c.Writer.Status())

	}
}