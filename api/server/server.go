package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/photon-storage/go-common/log"

	"github.com/photo-storage/dropbox/api/service"
)

// Server defines an instance of a server that handles the requests of
// the third-party application.
type Server struct {
	port   int
	engine *gin.Engine
}

// New returns a new instance of the server.
func New(port int, service *service.Service) *Server {
	server := &Server{
		port:   port,
		engine: gin.Default(),
	}

	server.registerRouter(service)
	return server
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}

		c.Next()
	}
}

func (s *Server) registerRouter(service *service.Service) {
	s.engine.Use(handleError(), cors())
	g := s.engine.Group("dropbox/v1")

	g.GET("ping", s.handle(service.Ping))
}

// Run the server
func (s *Server) Run() {
	if err := s.engine.Run(fmt.Sprintf(":%d", s.port)); err != nil {
		log.Error("run the server failed", "error", err)
		os.Exit(1)
	}
}
