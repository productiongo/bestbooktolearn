package amazon

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"

	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"io/ioutil"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Some of the code in this file was copied from github.com/DDRBoxman/go-amazon-product-api
// License: https://github.com/DDRBoxman/go-amazon-product-api/blob/master/LICENSE

// Response describes the generic API Response
// Response describes the generic API Response
type AWSResponse struct {
	OperationRequest struct {
		RequestID             string     `xml:"RequestId"`
		Arguments             []Argument `xml:"Arguments>Argument"`
		RequestProcessingTime float64
	}
}

// Argument todo
type Argument struct {
	Name  string `xml:"Name,attr"`
	Value string `xml:"Value,attr"`
}

// Image todo
type Image struct {
	URL    string
	Height uint16
	Width  uint16
}

// Price describes the product price as
// Amount of cents in CurrencyCode
type Price struct {
	Amount         uint
	CurrencyCode   string
	FormattedPrice string
}

type TopSeller struct {
	ASIN  string
	Title string
}

// Item represents a product returned by the API
type Item struct {
	ASIN             string
	URL              string
	DetailPageURL    string
	ItemAttributes   *ItemAttributes
	OfferSummary     OfferSummary
	Offers           Offers
	SalesRank        int
	SmallImage       *Image
	MediumImage      *Image
	LargeImage       *Image
	ImageSets        *ImageSets
	EditorialReviews EditorialReviews
	BrowseNodes      struct {
		BrowseNode []BrowseNode
	}
}

// BrowseNode represents a browse node returned by API
type BrowseNode struct {
	BrowseNodeID string `xml:"BrowseNodeId"`
	Name         string
	TopSellers   struct {
		TopSeller []TopSeller
	}
	Ancestors struct {
		BrowseNode []BrowseNode
	}
}

// ItemAttributes response group
type ItemAttributes struct {
	Author          string
	Binding         string
	Brand           string
	Color           string
	EAN             string
	Creator         string
	Title           string
	ListPrice       Price
	Manufacturer    string
	Publisher       string
	NumberOfItems   int
	PackageQuantity int
	Feature         string
	Model           string
	ProductGroup    string
	ReleaseDate     string
	Studio          string
	Warranty        string
	Size            string
	UPC             string
}

// Offer response attribute
type Offer struct {
	Condition       string `xml:"OfferAttributes>Condition"`
	ID              string `xml:"OfferListing>OfferListingId"`
	Price           Price  `xml:"OfferListing>Price"`
	PercentageSaved uint   `xml:"OfferListing>PercentageSaved"`
	Availability    string `xml:"OfferListing>Availability"`
}

// Offers response group
type Offers struct {
	TotalOffers     int
	TotalOfferPages int
	MoreOffersURL   string  `xml:"MoreOffersUrl"`
	Offers          []Offer `xml:"Offer"`
}

// OfferSummary response group
type OfferSummary struct {
	LowestNewPrice   Price
	LowerUsedPrice   Price
	TotalNew         int
	TotalUsed        int
	TotalCollectible int
	TotalRefurbished int
}

// EditorialReview response attribute
type EditorialReview struct {
	Source  string
	Content string
}

// EditorialReviews response group
type EditorialReviews struct {
	EditorialReview EditorialReview
}

// BrowseNodeLookupRequest is the confirmation of a BrowseNodeInfo request
type BrowseNodeLookupRequest struct {
	BrowseNodeId  string
	ResponseGroup string
}

// ItemLookupRequest is the confirmation of a ItemLookup request
type ItemLookupRequest struct {
	IDType        string `xml:"IdType"`
	ItemID        string `xml:"ItemId"`
	ResponseGroup string `xml:"ResponseGroup"`
	VariationPage string
}

// ItemLookupResponse describes the API response for the ItemLookup operation
type ItemLookupResponse struct {
	AWSResponse
	Items struct {
		Request struct {
			IsValid           bool
			ItemLookupRequest ItemLookupRequest
		}
		Item Item `xml:"Item"`
	}
}

// ItemSearchRequest is the confirmation of a ItemSearch request
type ItemSearchRequest struct {
	Keywords      string `xml:"Keywords"`
	SearchIndex   string `xml:"SearchIndex"`
	ResponseGroup string `xml:"ResponseGroup"`
}

type ItemSearchResponse struct {
	AWSResponse
	Items struct {
		Request struct {
			IsValid           bool
			ItemSearchRequest ItemSearchRequest
		}
		Items                []Item `xml:"Item"`
		TotalResult          int
		TotalPages           int
		MoreSearchResultsUrl string
	}
}

type BrowseNodeLookupResponse struct {
	AWSResponse
	BrowseNodes struct {
		Request struct {
			IsValid                 bool
			BrowseNodeLookupRequest BrowseNodeLookupRequest
		}
		BrowseNode BrowseNode
	}
}

type ImageSets struct {
	ImageSet []ImageSet
}

type ImageSet struct {
	//Category string `xml:"Category,attr"`
	Category       string `xml:",attr"`
	SwatchImage    *Image
	SmallImage     *Image
	ThumbnailImage *Image
	TinyImage      *Image
	MediumImage    *Image
	LargeImage     *Image
}

type AmazonProductAPI struct {
	AccessKey    string
	SecretKey    string
	AssociateTag string
	Host         string
	Client       *http.Client
}

func (api AmazonProductAPI) genSignAndFetch(Operation string, Parameters map[string]string) (string, error) {
	genURL, err := generateAmazonURL(api, Operation, Parameters)
	if err != nil {
		return "", err
	}

	setTimestamp(genURL)

	signedurl, err := signAmazonURL(genURL, api)
	if err != nil {
		return "", err
	}

	if api.Client == nil {
		api.Client = http.DefaultClient
	}

	resp, err := api.Client.Get(signedurl)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// Search for Products on the Amazon API. Index should be a valid Amazon product category,
// e.g. "Books".
func (api AmazonProductAPI) Search(index string, keywords string, page int) (isr ItemSearchResponse, err error) {
	params := map[string]string{
		"Keywords":      url.QueryEscape(keywords),
		"ResponseGroup": "Images,ItemAttributes,Small,EditorialReview",
		"ItemPage":      strconv.FormatInt(int64(page), 10),
		"SearchIndex":   index,
	}
	result, err := api.genSignAndFetch("ItemSearch", params)
	if err != nil {
		return
	}

	err = xml.Unmarshal([]byte(result), &isr)
	if err != nil {
		return
	}
	return isr, err
}

func generateAmazonURL(api AmazonProductAPI, Operation string, Parameters map[string]string) (finalURL *url.URL, err error) {

	result, err := url.Parse(api.Host)
	if err != nil {
		return nil, err
	}

	result.Host = api.Host
	result.Scheme = "http"
	result.Path = "/onca/xml"

	values := url.Values{}
	values.Add("Operation", Operation)
	values.Add("Service", "AWSECommerceService")
	values.Add("AWSAccessKeyId", api.AccessKey)
	values.Add("Version", "2013-08-01")
	values.Add("AssociateTag", api.AssociateTag)

	for k, v := range Parameters {
		values.Set(k, v)
	}

	params := values.Encode()
	result.RawQuery = params

	return result, nil
}

func setTimestamp(origURL *url.URL) (err error) {
	values, err := url.ParseQuery(origURL.RawQuery)
	if err != nil {
		return err
	}
	values.Set("Timestamp", time.Now().UTC().Format(time.RFC3339))
	origURL.RawQuery = values.Encode()

	return nil
}

func signAmazonURL(origURL *url.URL, api AmazonProductAPI) (signedURL string, err error) {
	escapeURL := strings.Replace(origURL.RawQuery, ",", "%2C", -1)
	escapeURL = strings.Replace(escapeURL, ":", "%3A", -1)

	params := strings.Split(escapeURL, "&")
	sort.Strings(params)
	sortedParams := strings.Join(params, "&")

	toSign := fmt.Sprintf("GET\n%s\n%s\n%s", origURL.Host, origURL.Path, sortedParams)

	hasher := hmac.New(sha256.New, []byte(api.SecretKey))
	_, err = hasher.Write([]byte(toSign))
	if err != nil {
		return "", err
	}

	hash := base64.StdEncoding.EncodeToString(hasher.Sum(nil))
	hash = url.QueryEscape(hash)
	newParams := fmt.Sprintf("%s&Signature=%s", sortedParams, hash)
	origURL.RawQuery = newParams
	return origURL.String(), nil
}
