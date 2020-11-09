package server

import (
	"github.com/deejross/direktor/internal/config"
	"github.com/gin-gonic/gin"
)

func registerRoutes(router *gin.Engine) {
	v1 := router.Group("/v1")

	v1.GET("/config/domains", handleConfigDomains)
}

func newError(c *gin.Context, code int, err error) {
	c.AbortWithStatusJSON(200, gin.H{
		"code":  code,
		"error": err.Error(),
	})
}

func handleConfigDomains(c *gin.Context) {
	conf, err := config.Get()
	if err != nil {
		newError(c, 500, err)
		return
	}

	domains := []gin.H{}
	for _, d := range conf.Domains {
		domains = append(domains, map[string]interface{}{
			"name": d.Name,
		})
	}

	c.JSON(200, gin.H{
		"domains": domains,
	})
}
