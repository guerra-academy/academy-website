package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mmcdole/gofeed"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type ApiResponse struct {
	SliderMenu struct {
		Data CourseData `json:"data"`
	} `json:"slider_menu"`
}

type CourseData struct {
	gorm.Model
	CourseID    int     `json:"course_id"`
	Title       string  `json:"title"`
	Rating      float64 `json:"rating"`
	NumReviews  int     `json:"num_reviews"`
	NumStudents int     `json:"num_students"`
	Hours       float64 `json:"hours"`
	DiscountURL string  `json:"discount_url"`
	ImageURL    string  `json:"image_url"`
}

type FeedItem struct {
	Title       string
	Link        string
	Description template.HTML
	ImageURL    string
	Published   string
}
type TotalStudents struct {
	Sum int
}

type TotalReviews struct {
	Sum int
}

func main() {
	logFile, err := os.OpenFile("application.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Erro ao abrir o arquivo de log: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	db, err := gorm.Open(sqlite.Open("cursos.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados SQLite: %v", err)
	}

	if err := db.AutoMigrate(&CourseData{}); err != nil {
		log.Fatalf("Erro ao migrar banco de dados: %v", err)
	}

	r := gin.Default()

	r.Static("/static", "./templates")
	r.LoadHTMLGlob("templates/*.html")

	r.GET("/", func(c *gin.Context) {
		var courses []CourseData
		if err := db.Find(&courses).Error; err != nil {
			log.Printf("Erro ao buscar cursos: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar cursos"})
			return
		}

		feedItems := loadFeed()
		totalStudents := getTotalStudents(db)
		totalReviews := getTotalReviews(db)

		c.HTML(http.StatusOK, "index.html", gin.H{
			"courses":       courses,
			"feed":          feedItems,
			"totalStudents": totalStudents,
			"totalReviews":  totalReviews,
		})
	})

	r.Run() // Executar o servidor na porta 8080
}

func getTotalStudents(db *gorm.DB) int {
	var total TotalStudents
	db.Model(&CourseData{}).Select("sum(num_students) as sum").Scan(&total)
	return total.Sum
}

func getTotalReviews(db *gorm.DB) int {
	var total TotalReviews
	db.Model(&CourseData{}).Select("sum(num_reviews) as sum").Scan(&total)
	return total.Sum
}

func loadFeed() []FeedItem {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL("https://blog.guerra.academy/rss/")
	if err != nil {
		log.Println("Erro ao analisar o feed RSS:", err)
		return nil
	}
	var feedItems []FeedItem
	for _, item := range feed.Items {
		parsedTime, _ := time.Parse(time.RFC1123, item.Published)
		newFeedItem := FeedItem{
			Title:       item.Title,
			Link:        item.Link,
			Description: template.HTML(item.Description),
			Published:   parsedTime.Format("02/01/2006"), // Formata a data
		}

		if len(item.Extensions["media"]["content"]) > 0 {
			mediaContent := item.Extensions["media"]["content"][0]
			if url, ok := mediaContent.Attrs["url"]; ok {
				newFeedItem.ImageURL = url
			}
		}

		feedItems = append(feedItems, newFeedItem)

		if len(feedItems) == 3 {
			break
		}
	}
	return feedItems
}
