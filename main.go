package main

import (
	"context"
	"flag"

	// for local use comment this
	"os"

	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"io"
	"log"
	"net/http"

	// "regexp"

	"strings"

	"github.com/ChimeraCoder/anaconda"
	// for local use comment this
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dghubble/oauth1"
	"github.com/g8rswimmer/go-twitter/v2"

	"github.com/tidwall/gjson"
	"gopkg.in/headzoo/surf.v1"

	// twitterscraper "github.com/n0madic/twitter-scraper"
	sqlitecloud "github.com/sqlitecloud/go-sdk"
)

type authorizer struct{}
func (a *authorizer) Add(req *http.Request) {}

type comic struct {
	title   string
	imgLink string
}

const comicsLink = "https://net.hr/webcafe/overkloking"

var (
	// apiKey            = "...MyI7"
	// apiSecret         = "...xmcD"
	// accessToken       = "...DCuO"
	// accessTokenSecret = "...lvnJ"

	// // for local use uncomment this
	// sqliteCloudConString = ""
	// apiKey = ""
  // apiSecret = ""
  // accessToken = ""
  // accessTokenSecret = ""

	// for local use comment this
	sqliteCloudConString = os.Getenv("SQLITECLOUDCONSTRING")
	apiKey            	 = os.Getenv("TWITTER_APIKEY")
	apiSecret         	 = os.Getenv("TWITTER_APISECRET")
	accessToken       	 = os.Getenv("TWITTER_ACCESS_TOKEN")
	accessTokenSecret 	 = os.Getenv("TWITTER_ACCESS_TOKEN_SECRET")
)

func getlastComic() (comic, error) {
	browser := surf.NewBrowser()
	err := browser.Open(comicsLink)
	if err != nil {
		panic(err)
	}
	
	// // title := gjson.Get(json, "props.pageProps.dehydratedState.queries.4.state.data.name").String()
	// dirtyTitle := gjson.Get(json, "props.pageProps.entityData.image.name").String()
	// title = strings.Replace(dirtyTitle, ".jpg", "", 1)

	title, _ := browser.Find("a.cardInner").Attr("title")

	browser.Click("a.cardInner")
	json := browser.Find("#__NEXT_DATA__").Text()

	// imageLink := gjson.Get(json, "props.pageProps.dehydratedState.queries.4.state.data.original_url").String()
	// imageLink := gjson.Get(json, "props.pageProps.entityData.image.original_url").String()
	imageLink := gjson.Get(json, "props.pageProps.pageProps.ssrData.article.image.original_url").String()

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

func getLastTweet() string {

	db, err := sqlitecloud.Connect(sqliteCloudConString)
	if err != nil {
		fmt.Println("Connect error: ", err)
	}

	result, _ := db.Select("SELECT comic_name FROM comics limit 1;")
	if err != nil {
		fmt.Println("Select error: ", err)
	}
	comic_name, _ := result.GetStringValue(0, 0)
	comic_name = strings.TrimSpace(strings.ToLower(comic_name))

	return comic_name
}

func postTwit(lastComic comic, api *anaconda.TwitterApi) {
	tweet := lastComic.title //+ " #overkloking"

	response, err := http.Get(lastComic.imgLink)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	contents, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal("Trouble reading web response body!")
	}

	imageBase64 := base64.StdEncoding.EncodeToString(contents)
	mediaResponse, err := api.UploadMedia(imageBase64)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("mediaResponse.MediaID:", mediaResponse.MediaIDString)

	text := flag.String("text", tweet, "Tweet")
	flag.Parse()

	oauth1Config := oauth1.NewConfig(apiKey, apiSecret)

	twitterHttpClient := oauth1Config.Client(oauth1.NoContext, &oauth1.Token{
		Token:       accessToken,
		TokenSecret: accessTokenSecret,
	})

	client := &twitter.Client{
		Authorizer: &authorizer{},
		Client:     twitterHttpClient,
		Host:       "https://api.twitter.com",
	}

	req := twitter.CreateTweetRequest{
		Text: *text,
	}

	var mediaIds = []string{mediaResponse.MediaIDString}

	if len(mediaIds) > 0 {
		req.Media = &twitter.CreateTweetMedia{
			IDs: mediaIds,
		}
	}

	fmt.Println("Callout to create tweet callout")

	tweetResponse, err := client.CreateTweet(context.Background(), req)
	if err != nil {
		log.Panicf("create tweet error: %v", err)
	}

	enc, err := json.MarshalIndent(tweetResponse, "", "    ")
	if err != nil {
		log.Panic(err)
	}
	fmt.Println(string(enc))

	// Saving comic name into database
	db, err := sqlitecloud.Connect(sqliteCloudConString)
	if err != nil {
		fmt.Println("Connect error fo update: ", err)
	}

	updateSQL := fmt.Sprintf("UPDATE comics SET comic_name = '%s' where id = 1;", lastComic.title)
	err = db.Execute(updateSQL)
	if err != nil {
		log.Panic(err)
	}
}

func overkloking() {
	anaconda.SetConsumerKey(apiKey)
	anaconda.SetConsumerSecret(apiSecret)

	api := anaconda.NewTwitterApi(accessToken, accessTokenSecret)

	println("1. getLastTweet")
	lastTwit := getLastTweet()

	println("2. getlastComic")
	lastComic, err := getlastComic()
	if err != nil {
		log.Fatal(err)
	}

	lastComicTitle := strings.TrimSpace(strings.ToLower(lastComic.title))
	
	println("lastTwit: ", lastTwit)
	println("lastComicTitle: ", lastComicTitle)
	println("lastComicImgLink: ", lastComic.imgLink)

	println("3. postTwit")
	if lastTwit != lastComicTitle {
		postTwit(lastComic, api)
	}

	println("4. done")
}

func main() {
	// for local use comment this
	lambda.Start(overkloking)

	// // for local use uncomment this
	// overkloking()
}
