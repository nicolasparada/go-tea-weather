package metaweather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Client struct {
	HTTPClient *http.Client
}

type LocationSearch struct {
	WOEID int `json:"woeid"`
}

func (mw *Client) SearchLocation(ctx context.Context, query string) ([]LocationSearch, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.metaweather.com/api/location/search/?query="+url.QueryEscape(query), nil)
	if err != nil {
		return nil, fmt.Errorf("could not create location search http request: %w", err)
	}

	resp, err := mw.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not fetch location search: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("could not fetch location search: status_code=%d", resp.StatusCode)
	}

	var out []LocationSearch
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("could not json decode location search: %w", err)
	}

	return out, nil
}

type Location struct {
	Title string    `json:"title"`
	Time  time.Time `json:"time"`

	ConsolidatedWeather []Weather `json:"consolidated_weather"`
}

type Weather struct {
	ApplicableDateStr    string  `json:"applicable_date"`
	WeatherStateName     string  `json:"weather_state_name"`
	WindSpeed            float64 `json:"wind_speed"`             // mph
	WindDirectionCompass string  `json:"wind_direction_compass"` // N, S, E, W
	MinTemp              float64 `json:"min_temp"`               // C
	MaxTemp              float64 `json:"max_temp"`               // C
	TheTemp              float64 `json:"the_temp"`               // C
	AirPressure          float64 `json:"air_pressure"`           // mbar
	Humidity             float64 `json:"humidity"`               // percent
	Visibility           float64 `json:"visibility"`             // miles
}

func (loc Weather) ApplicableDate() (time.Time, error) {
	t, err := time.Parse("2006/01/02", loc.ApplicableDateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not parse date: %w", err)
	}
	return t, nil
}

func (mw *Client) Location(ctx context.Context, woeid int) (Location, error) {
	var out Location

	u := "https://www.metaweather.com/api/location/" + url.PathEscape(strconv.FormatInt(int64(woeid), 10)) + "/"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return out, fmt.Errorf("could not create location day request: %w", err)
	}

	resp, err := mw.HTTPClient.Do(req)
	if err != nil {
		return out, fmt.Errorf("could not fetch location day: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return out, fmt.Errorf("could not fetch location day: status_code=%d", resp.StatusCode)
	}

	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return out, fmt.Errorf("could not json decode location day: %w", err)
	}

	return out, nil
}

func (mw *Client) LocationByQuery(ctx context.Context, query string) (Location, error) {
	var out Location

	results, err := mw.SearchLocation(ctx, query)
	if err != nil {
		return out, err
	}

	if len(results) == 0 {
		return out, errors.New("not found")
	}

	loc, err := mw.Location(ctx, results[0].WOEID)
	if err != nil {
		return out, err
	}

	if len(loc.ConsolidatedWeather) == 0 {
		return out, errors.New("zero consolidated weather")
	}

	return loc, nil
}
