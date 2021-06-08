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

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/tidwall/gjson"
	"gopkg.in/headzoo/surf.v1"
)

type comic struct {
	title   string
	imgLink string
}

const comicsLink = "https://net.hr/webcafe/overkloking"

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

	browser.Click("a.cardInner")
	json := browser.Find("#__NEXT_DATA__").Text()
	imageLink := gjson.Get(json, "props.initialProps.pageProps.entityData.image.original_url").String()
	title := browser.Find(".title_title").Text()

	lastComic := comic{
		title:   title,
		imgLink: imageLink,
	}

	if lastComic.title == "" {
		return lastComic, errors.New("title is missing")
	} else if lastComic.imgLink == "" {
		return lastComic, errors.New("image link is missing")
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
	if err != nil {
		log.Fatal(err)
	}

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
	println("lastComicTitle: ", lastComicTitle)
	println("lastComicImgLink: ", lastComic.imgLink)
	if lastTwit != lastComicTitle {
		postTwit(lastComic, api)
	}
}

func main() {
	lambda.Start(overkloking)
}
