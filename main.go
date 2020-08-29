package main

import (
	"encoding/base64"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ChimeraCoder/anaconda"
	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-lambda-go/lambda"
	"gopkg.in/headzoo/surf.v1"
)

type comic struct {
	title   string
	date    string
	imgLink string
}

const comicsLink = "https://net.hr/kategorija/webcafe/overkloking/"

var comics []comic

var (
	apiKey            = os.Getenv("TWITTER_APIKEY")
	apiSecret         = os.Getenv("TWITTER_APISECRET")
	accessToken       = os.Getenv("TWITTER_ACCESS_TOKEN")
	accessTokenSecret = os.Getenv("TWITTER_ACCESS_TOKEN_SECRET")
)

func getlastComic() (comic, error) {
	browser := surf.NewBrowser()
	err := browser.Open(comicsLink)
	if err != nil {
		panic(err)
	}

	browser.Find(".article-feed").Each(func(_ int, s *goquery.Selection) {
		title := s.Find(".article-text>a>h1").Text()
		date := strings.Replace(strings.TrimSpace(s.Find(".article-text>p.undertitle").Text()), "Overkloking", "", -1)
		imgLink, _ := s.Find(".thumb>img").Attr("src")
		date = strings.TrimSpace(date)

		questionMarkIndex := strings.Index(imgLink, "?")
		imgLink = imgLink[:questionMarkIndex]

		comic := comic{
			title:   title,
			date:    date,
			imgLink: imgLink,
		}
		comics = append(comics, comic)
	})

	lastComic := comics[0]

	if lastComic.title == "" {
		return lastComic, errors.New("Title is missing")
	} else if lastComic.imgLink == "" {
		return lastComic, errors.New("Image link is missing")
	} else {
		return lastComic, nil
	}
}

func getLastTweet(api *anaconda.TwitterApi) string {
	vars := url.Values{}
	vars.Set("screen_name", "overkloking")
	vars.Set("count", "1")

	lastTwits, err := api.GetUserTimeline(vars)
	if err != nil {
		log.Fatal(err)
	}

	reg, err := regexp.Compile(" #(.*)$")
	if err != nil {
		log.Fatal(err)
	}
	lastTwit := reg.ReplaceAllString(lastTwits[0].Text, "")
	lastTwit = strings.ToLower(lastTwit)

	return lastTwit
}

func postTwit(lastComic comic, api *anaconda.TwitterApi) {
	tweet := lastComic.title + " #overkloking"

	response, err := http.Get(lastComic.imgLink)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal("Trouble reading web response body!")
	}

	imageBase64 := base64.StdEncoding.EncodeToString(contents)
	mediaResponse, err := api.UploadMedia(imageBase64)

	vars := url.Values{}
	vars.Set("media_ids", strconv.FormatInt(mediaResponse.MediaID, 10))

	_, err = api.PostTweet(tweet, vars)
	if err != nil {
		log.Fatal(err)
	}
}

func overkloking() {
	anaconda.SetConsumerKey(apiKey)
	anaconda.SetConsumerSecret(apiSecret)

	api := anaconda.NewTwitterApi(accessToken, accessTokenSecret)

	lastTwit := getLastTweet(api)

	lastComic, err := getlastComic()
	if err != nil {
		log.Fatal(err)
	}
	lastComicTitle := strings.TrimSpace(strings.ToLower(lastComic.title))

	if lastTwit != lastComicTitle {
		postTwit(lastComic, api)
	}
}

func main() {
	lambda.Start(overkloking)
}
