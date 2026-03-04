package main

import "context"

type shortUrl struct {
	ID string `json:"id"`
	URL string `json:"url"`
}
func shortenUrl(ctx context.Context, url string) (*shortUrl, error) {
	return nil, nil
}
func resolveUrl(ctx context.Context, url shortUrl) (*string, error) {
	return nil, nil
}
