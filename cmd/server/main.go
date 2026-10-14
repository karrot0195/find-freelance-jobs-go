package main

import (
	"log"
	"net/http"

	"job-aggregator/internal/db"
	"job-aggregator/internal/scraper"

	"github.com/gin-gonic/gin"
)

const PORT = ":8080"

func main() {
	db.InitDB()

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/GetAccounts", func(c *gin.Context) {
		c.JSON(http.StatusOK, db.GetAccounts())
	})

	r.POST("/AddAccount", func(c *gin.Context) {
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

	r.GET("/GetJobs", func(c *gin.Context) {
		c.JSON(http.StatusOK, db.GetJobs())
	})

	r.POST("/StartCrawlJobs", func(c *gin.Context) {
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

	if err := r.Run(PORT); err != nil {
		log.Fatal(err)
	}
}
