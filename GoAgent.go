package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/jamespearly/loggly"
)

type AirQuality struct {
	Status string `json:"status"`
	Data   struct {
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

	var aqData AirQuality

	ticker := time.NewTicker(5 * time.Minute)

	for {

		time := <-ticker.C
		fmt.Println(time)

		request, reqErr := http.Get("http://api.airvisual.com/v2/city?city=Sacramento&state=California&country=USA&key=9f5d9c77-3aaa-44e0-98c3-a24e67884a93")

		if reqErr != nil {
			panic(reqErr)
		}

		requestData, reqDataErr := io.ReadAll(request.Body)

		if reqDataErr != nil {
			panic(reqDataErr)
		}

		Data := []byte(requestData)

		jsonErr := json.Unmarshal(Data, &aqData)

		if jsonErr != nil {
			panic(jsonErr)
		}

		var tag string
		tag = "IQAir"

		client := loggly.New(tag)

		err := client.EchoSend("info", "Status: "+aqData.Status+"\nCity: "+aqData.Data.City+"\nState: "+aqData.Data.State+"\nCountry: "+aqData.Data.Country+"\nType: "+aqData.Data.Location.Type+"\nCoordinates: "+strconv.FormatFloat(aqData.Data.Location.Coordinates[0], 'E', -1, 64)+", "+strconv.FormatFloat(aqData.Data.Location.Coordinates[1], 'E', -1, 64)+
			"\nTimestamp: "+aqData.Data.Current.Pollution.Ts.String()+"\nAQI US: "+strconv.Itoa(aqData.Data.Current.Pollution.Aqius)+
			"\nMain Pollutant US: "+aqData.Data.Current.Pollution.Mainus+"\nAQI China: "+strconv.Itoa(aqData.Data.Current.Pollution.Aqicn)+
			"\nMain Pollutant China: "+aqData.Data.Current.Pollution.Maincn+"\nTimestamp: "+aqData.Data.Current.Weather.Ts.String()+
			"\nTemperature: "+strconv.Itoa(aqData.Data.Current.Weather.Tp)+"\nAir Pressure: "+strconv.Itoa(aqData.Data.Current.Weather.Pr)+
			"\nHumidity: "+strconv.Itoa(aqData.Data.Current.Weather.Hu)+"\nWind Speed: "+strconv.FormatFloat(aqData.Data.Current.Weather.Ws, 'E', -1, 64)+
			"\nWind Direction: "+strconv.Itoa(aqData.Data.Current.Weather.Wd)+"\nWeather Icon Code: "+aqData.Data.Current.Weather.Ic)
		fmt.Println("err:", err)
	}
}
