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
	City     string `json:"city"`
	State    string `json:"state"`
	Country  string `json:"country"`
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

		request1, reqErr1 := http.Get("http://api.airvisual.com/v2/city?city=Sacramento&state=California&country=USA&key=9f5d9c77-3aaa-44e0-98c3-a24e67884a93")
		request2, reqErr2 := http.Get("http://api.airvisual.com/v2/city?city=Ottawa&state=Ontario&country=Canada&key=9f5d9c77-3aaa-44e0-98c3-a24e67884a93")

		if reqErr1 != nil {
			panic(reqErr1)
		}

		if reqErr2 != nil {
			panic(reqErr2)
		}

		requestData1, reqDataErr1 := ioutil.ReadAll(request1.Body)
		requestData2, reqDataErr2 := ioutil.ReadAll(request2.Body)

		if reqDataErr1 != nil {
			panic(reqDataErr1)
		}

		if reqDataErr2 != nil {
			panic(reqDataErr2)
		}

		var aqData1 AirQualityData
		var aqData2 AirQualityData

		aqData1.DateTime = time.Now().Format(time.RFC3339)
		jsonErr1 := json.Unmarshal(requestData1, &aqData1)

		aqData2.DateTime = time.Now().Format(time.RFC3339)
		jsonErr2 := json.Unmarshal(requestData2, &aqData2)

		if jsonErr1 != nil {
			panic(jsonErr1)
		}

		if jsonErr2 != nil {
			panic(jsonErr2)
		}

		aqData1.City = aqData1.Data.City
		aqData1.State = aqData1.Data.State
		aqData1.Country = aqData1.Data.Country

		aqData2.City = aqData2.Data.City
		aqData2.State = aqData2.Data.State
		aqData2.Country = aqData2.Data.Country

		av1, dbErr1 := dynamodbattribute.MarshalMap(aqData1)
		av2, dbErr2 := dynamodbattribute.MarshalMap(aqData2)

		if dbErr1 != nil {
			log.Fatalf("Got error marshalling map: %s", dbErr1)
		}

		if dbErr2 != nil {
			log.Fatalf("Got error marshalling map: %s", dbErr2)
		}

		tableName := "air-quality-data-jwilcox5"

		input1 := &dynamodb.PutItemInput{
			Item:      av1,
			TableName: aws.String(tableName),
		}

		input2 := &dynamodb.PutItemInput{
			Item:      av2,
			TableName: aws.String(tableName),
		}

		_, putErr1 := svc.PutItem(input1)
		_, putErr2 := svc.PutItem(input2)

		if putErr1 != nil {
			log.Fatalf("Got error calling PutItem: %s", putErr1)
		}

		if putErr2 != nil {
			log.Fatalf("Got error calling PutItem: %s", putErr2)
		}

		logTag := "IQAir Air Quality Data"

		client := loggly.New(logTag)

		logErr1 := client.EchoSend("info", "\nData Size: "+strconv.FormatInt(request1.ContentLength, 10)+"\nTime: "+aqData1.DateTime+"\nStatus: "+aqData1.Status+"\nCity: "+aqData1.City+"\nState: "+aqData1.State+"\nCountry: "+aqData1.Country+"\nType: "+aqData1.Data.Location.Type+"\nCoordinates: "+strconv.FormatFloat(aqData1.Data.Location.Coordinates[0], 'E', -1, 64)+", "+strconv.FormatFloat(aqData1.Data.Location.Coordinates[1], 'E', -1, 64)+
			"\nTimestamp: "+aqData1.Data.Current.Pollution.Ts.String()+"\nAQI US: "+strconv.Itoa(aqData1.Data.Current.Pollution.Aqius)+
			"\nMain Pollutant US: "+aqData1.Data.Current.Pollution.Mainus+"\nAQI China: "+strconv.Itoa(aqData1.Data.Current.Pollution.Aqicn)+
			"\nMain Pollutant China: "+aqData1.Data.Current.Pollution.Maincn+"\nTimestamp: "+aqData1.Data.Current.Weather.Ts.String()+
			"\nTemperature: "+strconv.Itoa(aqData1.Data.Current.Weather.Tp)+"\nAir Pressure: "+strconv.Itoa(aqData1.Data.Current.Weather.Pr)+
			"\nHumidity: "+strconv.Itoa(aqData1.Data.Current.Weather.Hu)+"\nWind Speed: "+strconv.FormatFloat(aqData1.Data.Current.Weather.Ws, 'E', -1, 64)+
			"\nWind Direction: "+strconv.Itoa(aqData1.Data.Current.Weather.Wd)+"\nWeather Icon Code: "+aqData1.Data.Current.Weather.Ic+"\n")
		fmt.Println("err:", logErr1)

		logErr2 := client.EchoSend("info", "\nData Size: "+strconv.FormatInt(request2.ContentLength, 10)+"\nTime: "+aqData2.DateTime+"\nStatus: "+aqData2.Status+"\nCity: "+aqData2.City+"\nState: "+aqData2.State+"\nCountry: "+aqData2.Country+"\nType: "+aqData2.Data.Location.Type+"\nCoordinates: "+strconv.FormatFloat(aqData2.Data.Location.Coordinates[0], 'E', -1, 64)+", "+strconv.FormatFloat(aqData2.Data.Location.Coordinates[1], 'E', -1, 64)+
			"\nTimestamp: "+aqData2.Data.Current.Pollution.Ts.String()+"\nAQI US: "+strconv.Itoa(aqData2.Data.Current.Pollution.Aqius)+
			"\nMain Pollutant US: "+aqData2.Data.Current.Pollution.Mainus+"\nAQI China: "+strconv.Itoa(aqData2.Data.Current.Pollution.Aqicn)+
			"\nMain Pollutant China: "+aqData2.Data.Current.Pollution.Maincn+"\nTimestamp: "+aqData2.Data.Current.Weather.Ts.String()+
			"\nTemperature: "+strconv.Itoa(aqData2.Data.Current.Weather.Tp)+"\nAir Pressure: "+strconv.Itoa(aqData2.Data.Current.Weather.Pr)+
			"\nHumidity: "+strconv.Itoa(aqData2.Data.Current.Weather.Hu)+"\nWind Speed: "+strconv.FormatFloat(aqData2.Data.Current.Weather.Ws, 'E', -1, 64)+
			"\nWind Direction: "+strconv.Itoa(aqData2.Data.Current.Weather.Wd)+"\nWeather Icon Code: "+aqData2.Data.Current.Weather.Ic+"\n")
		fmt.Println("err:", logErr2)
	}
}
