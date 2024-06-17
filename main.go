package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const (
	statusPageURL = "https://www.cloudflarestatus.com/"
	userAgent     = "https://github.com/piger/cloudflare-colo-status"
)

type ColoStatus struct {
	Name   string
	Status string
	Group  string
}

func fetchPage(ctx context.Context, client *http.Client, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code while fetching URL %s: %d", url, resp.StatusCode)
	}

	return resp.Body, nil
}

func parseStatusPage(r io.Reader) ([]ColoStatus, error) {
	var colos []ColoStatus

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, err
	}

	doc.Find("div.component-container").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}

		group := strings.TrimSpace(
			s.Find(`div.component-inner-container > span.name > span:not([class~="font-small"])`).Text(),
		)
		if group == "" {
			err = errors.Join(err, errors.New("empty group name"))
		}

		s.Find("div.child-components-container > div.component-inner-container").Each(func(_ int, s *goquery.Selection) {
			if s.HasClass("status-green") {
				return
			}

			colo := ColoStatus{
				Name:   strings.TrimSpace(s.Find("span.name").First().Text()),
				Status: strings.TrimSpace(s.Find("span.component-status").Text()),
				Group:  group,
			}
			colos = append(colos, colo)
		})
	})

	return colos, nil
}

func getColoStatus(ctx context.Context, client *http.Client) ([]ColoStatus, error) {
	body, err := fetchPage(ctx, client, statusPageURL)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	return parseStatusPage(body)
}

func run() error {
	ctx := context.Background()
	client := &http.Client{}

	colos, err := getColoStatus(ctx, client)
	if err != nil {
		return err
	}

	for _, colo := range colos {
		fmt.Printf("%s (%s): %s\n", colo.Name, colo.Group, colo.Status)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
