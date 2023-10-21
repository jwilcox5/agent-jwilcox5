package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"github.com/jamespearly/loggly"
)

type AirQualityData struct {
	DateTime string `json:"datetime"`
	Status   string `json:"status"`
	Data     struct {
		City     string `json:"city"`
		State    string `json:"state"`
		Country  string `json:"country"`
		Location struct {
			Type        string    `json:"type"`
			Coordinates []float64 `json:"coordinates"`
		} `json:"location"`
		Current struct {
			Pollution struct {
				Ts     time.Time `json:"ts"`
				Aqius  int       `json:"aqius"`
				Mainus string    `json:"mainus"`
				Aqicn  int       `json:"aqicn"`
				Maincn string    `json:"maincn"`
			} `json:"pollution"`
			Weather struct {
				Ts time.Time `json:"ts"`
				Tp int       `json:"tp"`
				Pr int       `json:"pr"`
				Hu int       `json:"hu"`
				Ws float64   `json:"ws"`
				Wd int       `json:"wd"`
				Ic string    `json:"ic"`
			} `json:"weather"`
		} `json:"current"`
	} `json:"data"`
}

func main() {

	awsSess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := dynamodb.New(awsSess)

	pollingInterval := flag.Int("interval", 1, "how frequent will API requests be made, in minutes")
	flag.Parse()
	ticker := time.NewTicker(time.Minute * time.Duration(*pollingInterval))

	for {
		<-ticker.C

		request, reqErr := http.Get("http://api.airvisual.com/v2/city?city=Sacramento&state=California&country=USA&key=9f5d9c77-3aaa-44e0-98c3-a24e67884a93")

		if reqErr != nil {
			panic(reqErr)
		}

		requestData, reqDataErr := ioutil.ReadAll(request.Body)

		if reqDataErr != nil {
			panic(reqDataErr)
		}

		var aqData AirQualityData

		aqData.DateTime = time.Now().Format(time.RFC3339)
		jsonErr := json.Unmarshal(requestData, &aqData)

		if jsonErr != nil {
			panic(jsonErr)
		}

		av, dbErr := dynamodbattribute.MarshalMap(aqData)

		if dbErr != nil {
			log.Fatalf("Got error marshalling map: %s", dbErr)
		}

		tableName := "air-quality-data-jwilcox5"

		input := &dynamodb.PutItemInput{
			Item:      av,
			TableName: aws.String(tableName),
		}

		_, putErr := svc.PutItem(input)

		if putErr != nil {
			log.Fatalf("Got error calling PutItem: %s", putErr)
		}

		logTag := "IQAir Air Quality Data"

		client := loggly.New(logTag)

		logErr := client.EchoSend("info", "\nData Size: "+strconv.FormatInt(request.ContentLength, 10)+"\nTime: "+aqData.DateTime+"\nStatus: "+aqData.Status+"\nCity: "+aqData.Data.City+"\nState: "+aqData.Data.State+"\nCountry: "+aqData.Data.Country+"\nType: "+aqData.Data.Location.Type+"\nCoordinates: "+strconv.FormatFloat(aqData.Data.Location.Coordinates[0], 'E', -1, 64)+", "+strconv.FormatFloat(aqData.Data.Location.Coordinates[1], 'E', -1, 64)+
			"\nTimestamp: "+aqData.Data.Current.Pollution.Ts.String()+"\nAQI US: "+strconv.Itoa(aqData.Data.Current.Pollution.Aqius)+
			"\nMain Pollutant US: "+aqData.Data.Current.Pollution.Mainus+"\nAQI China: "+strconv.Itoa(aqData.Data.Current.Pollution.Aqicn)+
			"\nMain Pollutant China: "+aqData.Data.Current.Pollution.Maincn+"\nTimestamp: "+aqData.Data.Current.Weather.Ts.String()+
			"\nTemperature: "+strconv.Itoa(aqData.Data.Current.Weather.Tp)+"\nAir Pressure: "+strconv.Itoa(aqData.Data.Current.Weather.Pr)+
			"\nHumidity: "+strconv.Itoa(aqData.Data.Current.Weather.Hu)+"\nWind Speed: "+strconv.FormatFloat(aqData.Data.Current.Weather.Ws, 'E', -1, 64)+
			"\nWind Direction: "+strconv.Itoa(aqData.Data.Current.Weather.Wd)+"\nWeather Icon Code: "+aqData.Data.Current.Weather.Ic)
		fmt.Println("err:", logErr)
	}
}
