package server

import (
	"time"

	"github.com/deejross/direktor/internal/config"
	"github.com/deejross/direktor/pkg/logger"
	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const tokenIssuer = "direktor"

var log = logger.New("internal/server")

// Start the web server.
func Start() error {
	// get configuration
	conf, err := config.Get()
	if err != nil {
		return err
	}

	// setup the router
	router := setupRouter()

	// start listening
	log.Info("listening on port " + conf.ListenPort)
	return router.Run(":" + conf.ListenPort)
}

func setupRouter() *gin.Engine {
	// configure router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(loggingMiddleware())
	router.Use(gin.Recovery())
	router.Use(static.Serve("/", static.LocalFile("./ui", true)))

	// configure basic endpoints
	router.GET("/health", routeHealth)

	// register other endpoints
	registerRoutes(router)

	return router
}

func routeHealth(c *gin.Context) {
	c.JSON(200, gin.H{"status": "OK"})
}

func loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// start timer
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// process the request
		c.Next()

		// append the query string to the URL if there is one
		if len(query) > 0 {
			path += "?" + query
		}

		// build fields
		fields := []zapcore.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", time.Now().Sub(start)),
			zap.Int("size", c.Writer.Size()),
			zap.String("client", c.ClientIP()),
		}

		// include error field if applicable
		errString := c.Errors.ByType(gin.ErrorTypePrivate).String()
		if len(errString) > 0 {
			fields = append(fields, zap.String("error", errString))
		}

		log.Info("request", fields...)
	}
}
