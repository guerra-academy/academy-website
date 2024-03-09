package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
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

type RecaptchaResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
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

type Usuario struct {
	ID         uint      `gorm:"primary_key"`
	Nome       string    `form:"nome"`
	Email      string    `form:"email"`
	Subscribed int       `json:"subscribed"`
	DataHora   time.Time `json:"data_hora"`
	Recaptcha  string    `form:"g-recaptcha-response"`
	CodRec     string    `form:"codRec"`
	GerouCert  int       `json:"gerou_cert"`
}

var (
	SMTPSERVER     = os.Getenv("SMTPSERVER")
	SMTPPORT       = os.Getenv("SMTPPORT")
	SMTPUSER       = os.Getenv("SMTPUSER")
	SMTPPASS       = os.Getenv("SMTPPASS")
	CAPTCHASECRET  = os.Getenv("CAPTCHASECRET")
	DSN            = os.Getenv("DSN")
	SITE           = os.Getenv("SITE")
	CAPTCHASITEKEY = os.Getenv("CAPTCHASITEKEY")
	USERECAPTCHA   = os.Getenv("CAPTCHASITEKEY")
)

func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func validateRecaptcha(response string) bool {
	// Create request body to validate recaptcha
	body := fmt.Sprintf("secret=%s&response=%s", CAPTCHASECRET, response)

	// Criando uma requisição POST
	url := "https://www.google.com/recaptcha/api/siteverify"
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(body))
	if err != nil {
		log.Println("Error to call recaptcha:", err)
		return false
	}

	// header  form urlenconded
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Realizando a chamada HTTP
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error after call HTTP recaptcha:", err)
		return false
	}
	defer resp.Body.Close()

	var recaptchaResponse RecaptchaResponse
	err = json.NewDecoder(resp.Body).Decode(&recaptchaResponse)
	if err != nil {
		log.Println("Error decoding Recaptcha JSON:", err)
		return false
	}

	// Validade recpatcha response
	if recaptchaResponse.Success {
		return true
	} else {
		log.Println("Invalid reCAPTCHA!")
		log.Println(recaptchaResponse)
		return false
	}
}

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
		log.Println("Error to create access token request: ", err)
	}

	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		println(err)
		log.Println("Error to get access token: ", err)
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		println(err)
		log.Println("Error to parse access token: ", err)
	}

	var tokenResponse ResponseToken
	err = json.Unmarshal(bodyText, &tokenResponse)
	if err != nil {
		return "", err
	}
	println("token: ", tokenResponse.AccessToken)
	return tokenResponse.AccessToken, nil
}

func sendEmail(name, from, to, subject, body string) error {
	auth := smtp.PlainAuth("", SMTPUSER, SMTPPASS, SMTPSERVER)
	// Ler o conteúdo do template HTML
	templateContent, err := ioutil.ReadFile("templates/boasvindas.html")
	if err != nil {
		return err
	}
	data := struct {
		Nome  string
		Email string
		Site  string
	}{
		Nome:  name,
		Email: to,
		Site:  SITE,
	}
	bodyMail := new(bytes.Buffer)
	tmpl := template.Must(template.New("bemvindo").Parse(string(templateContent)))
	err = tmpl.Execute(bodyMail, data)
	if err != nil {
		return err
	}

	msg := []byte("To: " + to + "\r\n" +
		"From: " + from + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=\"utf-8\"\r\n" +
		"\r\n" +
		bodyMail.String() + "\r\n")

	err = smtp.SendMail(SMTPSERVER+":"+SMTPPORT, auth, from, []string{to}, msg)
	if err != nil {
		return err
	}
	return nil
}

func main() {

	log.SetOutput(os.Stdout)
	log.Println("starting app...")
	r := gin.Default()
	apiurl := os.Getenv("API_URL")
	log.Println("api url: ", apiurl)

	r.Static("/static", "./templates")
	r.LoadHTMLGlob("templates/*.html")
	db, err := gorm.Open("postgres", DSN)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Enable GORM Auto migrate
	db.AutoMigrate(&Usuario{})

	r.POST("/adicionar", func(c *gin.Context) {
		var usuario Usuario
		now := time.Now()

		// Get data from form
		if err := c.ShouldBind(&usuario); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if !validateEmail(usuario.Email) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email inválido"})
			return
		}

		if !validateRecaptcha(usuario.Recaptcha) {
			log.Println("Recaptcha response:" + usuario.Recaptcha)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Recaptcha inválido"})
			return
		}

		// set DATE / HOUR and set user to be subscribed
		usuario.DataHora = now
		usuario.Subscribed = 1

		//insert or update
		if db.Model(&usuario).Where("email = ?", usuario.Email).Updates(&usuario).RowsAffected == 0 {
			db.Create(&usuario)
			emailBody := "Olá " + usuario.Nome + ", welcome to Guerra Academy!"
			err = sendEmail(usuario.Nome, "noreply@guerra.academy", usuario.Email, "Welcome to Guerra Academy", emailBody)
			if err != nil {
				log.Println("Error to send email: " + err.Error())
			}
		}

		if err != nil {
			println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error to send welcome email."})
			return
		}

		// Carregando o template HTML
		tmpl, err := template.ParseFiles("templates/sucesso.html")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Renderizando o template
		var data struct{}
		err = tmpl.Execute(c.Writer, data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	})

	r.GET("/", func(c *gin.Context) {
		var courses []CourseData

		feedItems := loadFeed()

		token, err := FetchAccessToken()
		if err != nil {
			println("Error to get token: %v", err)
			log.Printf("Error to get token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error to get token."})
			return
		}

		courses = getCourses(apiurl, token)
		totalReviews := getTotalReviews(apiurl, token)
		totalStudents := getTotalStudents(apiurl, token)

		c.HTML(http.StatusOK, "index.html", gin.H{
			"courses":       courses,
			"feed":          feedItems,
			"totalStudents": totalStudents,
			"totalReviews":  totalReviews,
			"sitekey":       CAPTCHASITEKEY,
		})
	})

	r.Run() // Executar o servidor na porta 8080
}

func getCourses(apiurl string, token string) []CourseData {
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiurl, nil)
	if err != nil {
		log.Println("Error geting courses: ", err)
		log.Println(err)
	}
	req.Header.Set("accept", ACCEPT)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("getCourses: ", err)
		log.Println(err)
	}
	log.Printf("body getCourses: %s\n", bodyText)

	// Deserializa o corpo da resposta diretamente em um slice de CourseData
	var courses []CourseData
	err = json.Unmarshal(bodyText, &courses)
	if err != nil {
		log.Println(err)
	}

	return courses
}

func getTotalStudents(apiurl string, token string) int {
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiurl+"totalStudents", nil)
	if err != nil {
		log.Println(err)
	}
	req.Header.Set("accept", ACCEPT)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		println(err)
		log.Println(err)
	}

	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		println("erro total students: ", err)
		log.Println(err)
	}
	log.Printf("body total students: %s\n", bodyText)

	// Deserialize response to TotalReviewResponse
	var response TotalStudentsResponse
	err = json.Unmarshal(bodyText, &response)
	if err != nil {
		println(err)
		log.Println(err)
	}
	log.Println("total students: ", response)
	return response.TotalStudents

}

func getTotalReviews(apiurl string, token string) int {
	client := &http.Client{}
	req, err := http.NewRequest("GET", apiurl+"totalReviews", nil)
	if err != nil {
		println("error to get total reviews: ", err)
		log.Println(err)
	}
	req.Header.Set("accept", ACCEPT)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		println("error to get total reviews: ", err)
		log.Println(err)
	}
	println("resposta total review: ", resp.StatusCode)
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		println("erro chamada total review: ", err)
		log.Println(err)
	}
	log.Printf("body total reviews: %s\n\n", bodyText)

	// Deserilize body to TotalReviewResponse
	var response TotalReviewResponse
	err = json.Unmarshal(bodyText, &response)
	if err != nil {
		println(err)
		log.Println(err)
	}
	log.Println("total reviews: ", response)
	return response.TotalReviews

}

func loadFeed() []FeedItem {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL("https://blog.guerra.academy/rss/")
	if err != nil {
		log.Println("Error reading RSS:", err)
		return nil
	}
	var feedItems []FeedItem
	for _, item := range feed.Items {
		parsedTime, _ := time.Parse(time.RFC1123, item.Published)
		newFeedItem := FeedItem{
			Title:       item.Title,
			Link:        item.Link,
			Description: template.HTML(item.Description),
			Published:   parsedTime.Format("02/01/2006"),
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
