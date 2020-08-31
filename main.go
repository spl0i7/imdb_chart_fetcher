package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gocolly/colly"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type ImdbMovie struct {
	Year     string `json:"year"`
	Title    string `json:"title"`
	Rating   string `json:"rating"`
	Summary  string `json:"summary"`
	Duration string `json:"duration"`
	Genre    string `json:"genre"`
}

// Since we are using parallelism in colly, we must protect our result with a mutex.
type Result struct {
	AllMovies []ImdbMovie
	*sync.Mutex
}

// Prints Usage
func printHelp() {
	fmt.Printf("Usage : %s  <url> <count>\n", os.Args[0])
}

// Validates command line argument
func validateInput() (string, int, error) {

	if len(os.Args) != 3 {
		return "", 0, errors.New("incorrect parameters")
	}

	imdbUrl := os.Args[1]
	_, err := url.Parse(imdbUrl)
	if err != nil {
		return "", 0, errors.New("invalid url")
	}

	limit, err := strconv.Atoi(os.Args[2])
	if err != nil || limit <= 0 {
		return "", 0, errors.New("invalid count")
	}

	return imdbUrl, limit, nil

}

// regex to remove date from title
var yearRegex = regexp.MustCompile(`\(\d{4}\)`)


func main() {

	// Take the input from os args
	imdbUrl, count, err := validateInput()
	if err != nil {
		fmt.Println(err)
		printHelp()
		return
	}

	// Only need maximum depth of 2 here at max.
	c := colly.NewCollector(
		colly.MaxDepth(2),
		colly.Async(true),
	)

	// Setting it to 5 but can be taken as a parameter too.
	c.Limit(&colly.LimitRule{Parallelism: 5})

	var result = Result{Mutex: &sync.Mutex{}}

	c.OnHTML("td.posterColumn > a", func(e *colly.HTMLElement) {
		if e.Index+1 > count {
			return
		}

		err := e.Request.Visit(e.Attr("href"))
		if err != nil {
			fmt.Println(err)
		}
	})

	c.OnHTML("#title-overview-widget", func(element *colly.HTMLElement) {

		year := element.ChildText("#titleYear")
		title := element.ChildText(".titleBar h1")
		rating := element.ChildText("div.ratingValue > strong > span")
		summary := element.ChildText(".summary_text")
		duration := element.ChildText("time")
		genre := element.ChildText("div.subtext > a:nth-child(4)")

		// remove () from year
		year = strings.ReplaceAll(year, "(", "")
		year = strings.ReplaceAll(year, ")", "")


		// remove year from title
		title = yearRegex.ReplaceAllString(title, "")
		title = strings.TrimLeft(title, " ")

		var movie = ImdbMovie{
			Year:     year,
			Title:    title,
			Rating:   rating,
			Summary:  summary,
			Duration: duration,
			Genre:    genre,
		}

		result.Lock()
		defer result.Unlock()

		result.AllMovies = append(result.AllMovies, movie)

	})

	c.Visit(imdbUrl)
	c.Wait()

	jsonResult, err := json.Marshal(&result.AllMovies)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(jsonResult))

}
