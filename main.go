package main

import (
	"crypto/tls"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mmcdole/gofeed"
)

type ApiResponse struct {
	SliderMenu struct {
		Data CourseData `json:"data"`
	} `json:"slider_menu"`
}

type CourseData struct {
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
type TotalReviewResponse struct {
	TotalReviews int `json:"totalReviews"`
}

type TotalStudentsResponse struct {
	TotalStudents int `json:"totalStudents"`
}

type ResponseToken struct {
	AccessToken string `json:"access_token"`
}

const ACCEPT = "application/json"

// function que recupera token jwt
func FetchAccessToken() (string, error) {

	apiUrl := os.Getenv("TOKEN_API_URL")
	authorization := os.Getenv("AUTHORIZATION")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	var data = strings.NewReader(`grant_type=client_credentials`)

	req, err := http.NewRequest("POST", apiUrl, data)
	if err != nil {
		println(err)
		log.Fatal(err)
	}

	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		println(err)
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		println(err)
		log.Fatal(err)
	}

	var tokenResponse ResponseToken
	err = json.Unmarshal(bodyText, &tokenResponse)
	if err != nil {
		return "", err
	}
	return tokenResponse.AccessToken, nil
}

func main() {
	log.SetOutput(os.Stdout)
	r := gin.Default()
	apiurl := os.Getenv("API_URL")
	log.Println("api url: ", apiurl)

	r.Static("/static", "./templates")
	r.LoadHTMLGlob("templates/*.html")

	r.GET("/", func(c *gin.Context) {
		var courses []CourseData

		feedItems := loadFeed()

		token, err := FetchAccessToken()
		if err != nil {
			println("Erro ao obter token: %v", err)
			log.Printf("Erro ao obter token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao obter token"})
			return
		}

		totalStudents := getTotalStudents(apiurl, token)
		totalReviews := getTotalReviews(apiurl, token)
		courses = getCourses(apiurl, token)

		c.HTML(http.StatusOK, "index.html", gin.H{
			"courses":       courses,
			"feed":          feedItems,
			"totalStudents": totalStudents,
			"totalReviews":  totalReviews,
		})
	})

	r.Run() // Executar o servidor na porta 8080
}

func getCourses(apiurl string, token string) []CourseData {
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiurl, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("accept", ACCEPT)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("body: %s\n", bodyText)

	// Deserializa o corpo da resposta diretamente em um slice de CourseData
	var courses []CourseData
	err = json.Unmarshal(bodyText, &courses)
	if err != nil {
		log.Fatal(err)
	}

	return courses
}

func getTotalStudents(apiurl string, token string) int {
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiurl+"totalStudents", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("accept", ACCEPT)
	req.Header.Set("Authorization", "Bearer "+token)
	//req.Header.Set("API-Key", apikey)

	resp, err := client.Do(req)
	if err != nil {
		println(err)
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		println(err)
		log.Fatal(err)
	}
	log.Printf("body: %s\n", bodyText)

	// Deserializa o corpo da resposta para a struct TotalReviewResponse
	var response TotalStudentsResponse
	err = json.Unmarshal(bodyText, &response)
	if err != nil {
		println(err)
		log.Fatal(err)
	}
	log.Println("total students: ", response)
	return response.TotalStudents

}

func getTotalReviews(apiurl string, token string) int {
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiurl+"totalReviews", nil)
	if err != nil {
		println(err)
		log.Fatal(err)
	}
	req.Header.Set("accept", ACCEPT)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		println(err)
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		println(err)
		log.Fatal(err)
	}
	log.Printf("body total reviews: %s\n", bodyText)

	// Deserializa o corpo da resposta para a struct TotalReviewResponse
	var response TotalReviewResponse
	err = json.Unmarshal(bodyText, &response)
	if err != nil {
		println(err)
		log.Fatal(err)
	}
	log.Println("total reviews: ", response)
	return response.TotalReviews

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
