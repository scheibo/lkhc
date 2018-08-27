package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Results struct {
	Week      int
	SegmentID int64    `json:"segment_id"`
	Male      []*Entry `json:"male"`
	Female    []*Entry `json:"female"`
}

type Entry struct {
	Rank  int64         `json:"rank"`
	Rider Rider         `json:"rider"`
	Time  time.Duration `json:"time"`
	Score float64       `json:"score"`
}

type Rider struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func main() {
	var year int

	flag.IntVar(&year, "year", 2016, "year to parse results for")

	flag.Parse()

	if !((year >= 1995 && year <= 1998) || (year >= 2006 && year <= 2016)) {
		exit(fmt.Errorf("year must be in the range [1995, 1998] or [2006, 2016] but was %d", year))
	}

	results, err := getResultsForYear(year)
	if err != nil {
		exit(err)
	}

	fmt.Printf("year,week,segment,gender,rank,id,name,time,score\n")
	for _, r := range results {
		for _, m := range r.Male {
			fmt.Printf("%d,%d,%d,M,%d,%d,%s,%d,%.f\n",
				year, r.Week, r.SegmentID, m.Rank, m.Rider.ID, m.Rider.Name, int(m.Time.Seconds()), m.Score)
		}
		for _, f := range r.Female {
			fmt.Printf("%d,%d,%d,F,%d,%d,%s,%d,%.f\n",
				year, r.Week, r.SegmentID, f.Rank, f.Rider.ID, f.Rider.Name, int(f.Time.Seconds()), f.Score)
		}
	}
}

func getResultsForYear(int year) ([]*Results, err) {
	var results []Results
	var err error

	for i := 0; i < 10; i++ {
		r, err := getResults(year, week)
		if err != nil {
			fmt.Printf(os.Stderr, "year: %d week: %d err: %s", year, week, err)
		}
		if r != nil {
			results := append(results, week)
		}
	}
	if len(results) == 0 {
		return nil, err
	}
	return results, nil
}

func getResultsForWeek(int year, int week) (*Results, error) {
	entry, results, err := getRawResults(year, week)
	if err != nil {
		return nil, err
	}

	id := getSegmentID(entry)
	if err != nil {
		return nil, err
	}

	male, female, err := getMaleAndFemaleResults(results)
	if err != nil {
		return nil, err
	}

	return Results{Week: int, ID: id, Male: male, Female: female}
}

func getSegmentID(doc *goquery.Document) (int64, error) {
	href, ok := doc.Find("a[target=Strava]").Attr("href")
	if !ok {
		return nil, fmt.Errorf("could not find Strava URL")
	}

	split := strings.Split(href, "/")
	val, err := parseInt(strings.TrimSpace(split[len(split)-1]))
	if err != nil {
		return nil, err
	}
	return val, nil
}

func getMaleAndFemaleResults(doc *goquery.Document) ([]*Entry, []*Entry, error) {
	doc.Find(".results").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		// TODO
	})
	return nil, nil, nil
}

func getRawResults(int year, int week) (*goquery.Document, *goquery.Document, error) {
	entry, err := ioutil.ReadFile(fmt.Sprintf("lowkeyhillclimbs.com/%d/week%d.html", year, week))
	if err != nil {
		return nil, nil, err
	}

	results, err := ioutil.ReadFile(fmt.Sprintf("lowkeyhillclimbs.com/%d/week%d/results.html", year))
	if err != nil {
		return nil, nil, err
	}

	return entry, results, nil
}

func parseInt(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 0)
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func parseElapsedTime(str string) (int64, error) {
	var x string
	var h, m, s int64
	var err error

	a := strings.Split(str, ":")

	if len(a) == 3 {
		x, a = a[0], a[1:]
		h, err = parseInt(x)
		if err != nil {
			return 0, err
		}
	}
	if len(a) == 2 {
		x, a = a[0], a[1:]
		m, err = parseInt(x)
		if err != nil {
			return 0, err
		}
	}
	s, err = parseInt(strings.TrimSuffix(a[0], "s"))
	if err != nil {
		return 0, err
	}
	return time.Seconds * (h*3600 + m*60 + s), nil
}

func exit(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	flag.PrintDefaults()
	os.Exit(1)
}
