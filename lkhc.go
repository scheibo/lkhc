package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"strconv"
	"regexp"
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

var timeonly = regexp.MustCompile("[^0-9:.]+")
var scoreonly = regexp.MustCompile("[^0-9.]+")

func main() {
	var year int

	flag.IntVar(&year, "year", 2016, "year to parse results for in range [2010, 2016]")

	flag.Parse()

	// TODO(kjs): handle years missing segments
	if !(year >= 2010 && year <= 2016) {
		exit(fmt.Errorf("year must be between 2010 and 2016 but was %d", year))
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

		fmt.Fprintf(os.Stderr, "segment: %d male: %d female: %d\n", r.SegmentID, len(r.Male), len(r.Female))
	}
}

// TODO(kjs): also need to include overall.html!
func getResultsForYear(year int) ([]*Results, error) {
	var results []*Results
	var err error

	for week := 1; week < 10; week++ {
		r, err := getResultsForWeek(year, week)
		if err != nil {
			fmt.Fprintf(os.Stderr, "year: %d week: %d err: %s\n", year, week, err)
		}
		if r != nil {
			results = append(results, r)
		}
	}
	if len(results) == 0 {
		return nil, err
	}
	return results, nil
}

func getResultsForWeek(year, week int) (*Results, error) {
	entry, results, err := getRawResults(year, week)
	if err != nil {
		return nil, err
	}

	id, err := getSegmentID(entry)
	if err != nil {
		return nil, err
	}

	male, female, err := getMaleAndFemaleResults(results)
	if err != nil {
		return nil, err
	}

	return &Results{Week: week, SegmentID: id, Male: male, Female: female}, nil
}

func getSegmentID(doc *goquery.Document) (int64, error) {
	href, ok := doc.Find("a[target=Strava]").Attr("href")
	if !ok {
		href, ok = doc.Find("img[src='strava_logo.png']").Parent().Attr("href")
		if !ok {
			return -1, fmt.Errorf("could not find Strava URL")
		}
	}

	split := strings.Split(href, "/")
	val, err := parseInt(strings.TrimSpace(split[len(split)-1]))
	if err != nil {
		return -1, err
	}
	return val, nil
}

func getMaleAndFemaleResults(doc *goquery.Document) ([]*Entry, []*Entry, error) {
	var male, female []*Entry
	var err error
	doc.Find("table.results").EachWithBreak(func(i int, table *goquery.Selection) bool {
		switch table.Find("caption").Text() {
		case "Men":
			male, err = getResults(table)
			if err != nil {
				return false
			}
		case "Women":
			female, err = getResults(table)
			if err != nil {
				return false
			}
		}
		return true
	})
	return male, female, err
}

func getResults(table *goquery.Selection) ([]*Entry, error) {
	var err error
	var entries []*Entry

	table.Find("tr").EachWithBreak(func(i int, tr *goquery.Selection) bool {
		tds := tr.Find("td")
		if tds.Length() == 0 {
			return true // header row
		} else if tds.Length() != 9 {
			err = fmt.Errorf("unexpected number of td elements: %d", tds.Length())
			return false
		}

		entry := Entry{}

		rank, err := parseInt(tds.Eq(0).Text())
		if err != nil {
			return false
		}
		entry.Rank = rank

		entry.Rider = Rider{}
		riderID, err := parseInt(tds.Eq(1).Text())
		if err != nil {
			return false
		}
		entry.Rider.ID = riderID
		entry.Rider.Name = tds.Eq(2).Text()

		// BUG: Remove Runners/Tandems from results.
		t, err := parseElapsedTime(tds.Eq(5).Text())
		if err != nil {
			return false
		}
		entry.Time = t

		str := tds.Eq(8).Text()
		str = timeonly.ReplaceAllString(str, "")
		score, err := parseFloat(str)
		if err != nil {
			return false
		}
		entry.Score = score

		entries = append(entries, &entry)
		return true
	})

	if err != nil {
		return nil, err
	}
	return entries, nil
}

func getRawResults(year, week int) (*goquery.Document, *goquery.Document, error) {
	f, err := os.Open(fmt.Sprintf("lowkeyhillclimbs.com/%d/week%d.html", year, week))
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	entry, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		return nil, nil, err
	}

	f, err = os.Open(fmt.Sprintf("lowkeyhillclimbs.com/%d/week%d/results.html", year, week))
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	results, err := goquery.NewDocumentFromReader(f)
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

func parseElapsedTime(str string) (time.Duration, error) {
	var x string
	var h, m int64
	var s float64
	var err error

	str = timeonly.ReplaceAllString(str, "")
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
	s, err = parseFloat(strings.TrimSuffix(a[0], "s"))
	if err != nil {
		return 0, err
	}
	return time.Duration(float64(h*3600) + float64(m*60) + s) * time.Second, nil
}

func exit(err error) {
	fmt.Fprintf(os.Stderr, "%s\n", err)
	flag.PrintDefaults()
	os.Exit(1)
}
