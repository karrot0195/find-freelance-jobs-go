package main

import (
	"log"
	"net/http"
	"os"

	"job-aggregator/internal/db"
	"job-aggregator/internal/scraper"

	"github.com/gin-gonic/gin"
)

var PORT = os.Getenv("APP_PORT")
var API_SECRET_KEY = os.Getenv("API_SECRET_KEY")

// apiKeyAuth returns a Gin middleware that enforces the X-API-Key header.
func apiKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if API_SECRET_KEY == "" {
			// No key configured — refuse to serve protected routes.
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "server misconfigured: API_SECRET_KEY not set",
			})
			return
		}
		key := c.GetHeader("X-API-Key")
		if key == "" || key != API_SECRET_KEY {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized: invalid or missing API key",
			})
			return
		}
		c.Next()
	}
}

func main() {
	db.InitDB()

	if PORT == "" {
		PORT = "8080"
	}
	if API_SECRET_KEY == "" {
		log.Fatal("API_SECRET_KEY environment variable is required")
	}

	r := gin.Default()

	// Public health check — no key required.
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// All routes below require the X-API-Key header.
	auth := r.Group("/", apiKeyAuth())
	{
		auth.GET("/GetAccounts", func(c *gin.Context) {
			c.JSON(http.StatusOK, db.GetAccounts())
		})

		auth.POST("/AddAccount", func(c *gin.Context) {
			var req struct {
				Provider string `json:"provider"`
				Username string `json:"username"`
			}
			if err := c.ShouldBindJSON(&req); err != nil || req.Provider == "" || req.Username == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "provider and username required"})
				return
			}
			db.AddAccount(req.Provider, req.Username)
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		auth.GET("/GetJobs", func(c *gin.Context) {
			c.JSON(http.StatusOK, db.GetJobs())
		})

		auth.POST("/StartCrawlJobs", func(c *gin.Context) {
			db.ClearJobs()

			accounts := db.GetEnabledAccounts()
			for _, acc := range accounts {
				jobs, err := scraper.FetchJobs(acc.Provider, acc.Username)
				if err != nil {
					log.Printf("Error fetching for account %s: %v", acc.Username, err)
					continue
				}
				log.Printf("Fetched %d jobs for %s (%s)", len(jobs), acc.Username, acc.Provider)
				for _, job := range jobs {
					db.SaveJob(job.Title, job.URL, job.Category, job.Price, job.Description, acc.Provider)
				}
			}
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}

	if err := r.Run(PORT); err != nil {
		log.Fatal(err)
	}
}
