package books

import (
	"github.com/productiongo/bestbooktolearn/amazon"
)

type API struct {
	amazon amazon.AmazonProductAPI
}

func New(amz amazon.AmazonProductAPI) *API {
	return &API{
		amazon: amz,
	}
}

type BookImage struct {
	URL    string
	Width  int
	Height int
}

type Book struct {
	Title      string
	ISBN       string
	URL        string
	LargeImage *BookImage
}

func (api API) Search(keywords string, page int) (books []Book, err error) {
	r, err := api.amazon.Search("Books", keywords, page)
	if err != nil {
		return
	}

	books = []Book{}
	for _, item := range r.Items.Items {
		var img *BookImage
		if item.LargeImage != nil {
			img = &BookImage{
				URL:    item.LargeImage.URL,
				Width:  int(item.LargeImage.Width),
				Height: int(item.LargeImage.Height),
			}
		}
		b := Book{
			Title:      item.ItemAttributes.Title,
			ISBN:       item.ItemAttributes.EAN,
			URL:        item.DetailPageURL,
			LargeImage: img,
		}
		books = append(books, b)
	}
	return books, nil
}
