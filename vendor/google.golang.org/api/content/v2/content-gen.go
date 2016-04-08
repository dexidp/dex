// Package content provides access to the Content API for Shopping.
//
// See https://developers.google.com/shopping-content/v2/
//
// Usage example:
//
//   import "google.golang.org/api/content/v2"
//   ...
//   contentService, err := content.New(oauthHttpClient)
package content

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/api/googleapi"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Always reference these packages, just in case the auto-generated code
// below doesn't.
var _ = bytes.NewBuffer
var _ = strconv.Itoa
var _ = fmt.Sprintf
var _ = json.NewDecoder
var _ = io.Copy
var _ = url.Parse
var _ = googleapi.Version
var _ = errors.New
var _ = strings.Replace

const apiId = "content:v2"
const apiName = "content"
const apiVersion = "v2"
const basePath = "https://www.googleapis.com/content/v2/"

// OAuth2 scopes used by this API.
const (
	// Manage your product listings and accounts for Google Shopping
	ContentScope = "https://www.googleapis.com/auth/content"
)

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Accounts = NewAccountsService(s)
	s.Accountstatuses = NewAccountstatusesService(s)
	s.Datafeeds = NewDatafeedsService(s)
	s.Datafeedstatuses = NewDatafeedstatusesService(s)
	s.Inventory = NewInventoryService(s)
	s.Products = NewProductsService(s)
	s.Productstatuses = NewProductstatusesService(s)
	return s, nil
}

type Service struct {
	client   *http.Client
	BasePath string // API endpoint base URL

	Accounts *AccountsService

	Accountstatuses *AccountstatusesService

	Datafeeds *DatafeedsService

	Datafeedstatuses *DatafeedstatusesService

	Inventory *InventoryService

	Products *ProductsService

	Productstatuses *ProductstatusesService
}

func NewAccountsService(s *Service) *AccountsService {
	rs := &AccountsService{s: s}
	return rs
}

type AccountsService struct {
	s *Service
}

func NewAccountstatusesService(s *Service) *AccountstatusesService {
	rs := &AccountstatusesService{s: s}
	return rs
}

type AccountstatusesService struct {
	s *Service
}

func NewDatafeedsService(s *Service) *DatafeedsService {
	rs := &DatafeedsService{s: s}
	return rs
}

type DatafeedsService struct {
	s *Service
}

func NewDatafeedstatusesService(s *Service) *DatafeedstatusesService {
	rs := &DatafeedstatusesService{s: s}
	return rs
}

type DatafeedstatusesService struct {
	s *Service
}

func NewInventoryService(s *Service) *InventoryService {
	rs := &InventoryService{s: s}
	return rs
}

type InventoryService struct {
	s *Service
}

func NewProductsService(s *Service) *ProductsService {
	rs := &ProductsService{s: s}
	return rs
}

type ProductsService struct {
	s *Service
}

func NewProductstatusesService(s *Service) *ProductstatusesService {
	rs := &ProductstatusesService{s: s}
	return rs
}

type ProductstatusesService struct {
	s *Service
}

type Account struct {
	// AdultContent: Indicates whether the merchant sells adult content.
	AdultContent bool `json:"adultContent,omitempty"`

	// AdwordsLinks: List of linked AdWords accounts.
	AdwordsLinks []*AccountAdwordsLink `json:"adwordsLinks,omitempty"`

	// Id: Merchant Center account ID.
	Id uint64 `json:"id,omitempty,string"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#account".
	Kind string `json:"kind,omitempty"`

	// Name: Display name for the account.
	Name string `json:"name,omitempty"`

	// ReviewsUrl: URL for individual seller reviews, i.e., reviews for each
	// child account.
	ReviewsUrl string `json:"reviewsUrl,omitempty"`

	// SellerId: Client-specific, locally-unique, internal ID for the child
	// account.
	SellerId string `json:"sellerId,omitempty"`

	// Users: Users with access to the account. Every account (except for
	// subaccounts) must have at least one admin user.
	Users []*AccountUser `json:"users,omitempty"`

	// WebsiteUrl: The merchant's website.
	WebsiteUrl string `json:"websiteUrl,omitempty"`
}

type AccountAdwordsLink struct {
	// AdwordsId: Customer ID of the AdWords account.
	AdwordsId uint64 `json:"adwordsId,omitempty,string"`

	// Status: Status of the link between this Merchant Center account and
	// the AdWords account.
	Status string `json:"status,omitempty"`
}

type AccountStatus struct {
	// AccountId: The ID of the account for which the status is reported.
	AccountId string `json:"accountId,omitempty"`

	// DataQualityIssues: A list of data quality issues.
	DataQualityIssues []*AccountStatusDataQualityIssue `json:"dataQualityIssues,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#accountStatus".
	Kind string `json:"kind,omitempty"`
}

type AccountStatusDataQualityIssue struct {
	// Country: Country for which this issue is reported.
	Country string `json:"country,omitempty"`

	// DisplayedValue: Actual value displayed on the landing page.
	DisplayedValue string `json:"displayedValue,omitempty"`

	// ExampleItems: Example items featuring the issue.
	ExampleItems []*AccountStatusExampleItem `json:"exampleItems,omitempty"`

	// Id: Issue identifier.
	Id string `json:"id,omitempty"`

	// LastChecked: Last time the account was checked for this issue.
	LastChecked string `json:"lastChecked,omitempty"`

	// NumItems: Number of items in the account found to have the said
	// issue.
	NumItems int64 `json:"numItems,omitempty"`

	// Severity: Severity of the problem.
	Severity string `json:"severity,omitempty"`

	// SubmittedValue: Submitted value that causes the issue.
	SubmittedValue string `json:"submittedValue,omitempty"`
}

type AccountStatusExampleItem struct {
	// ItemId: Unique item ID as specified in the uploaded product data.
	ItemId string `json:"itemId,omitempty"`

	// Link: Landing page of the item.
	Link string `json:"link,omitempty"`

	// SubmittedValue: The item value that was submitted.
	SubmittedValue string `json:"submittedValue,omitempty"`

	// Title: Title of the item.
	Title string `json:"title,omitempty"`

	// ValueOnLandingPage: The actual value on the landing page.
	ValueOnLandingPage string `json:"valueOnLandingPage,omitempty"`
}

type AccountUser struct {
	// Admin: Whether user is an admin.
	Admin bool `json:"admin,omitempty"`

	// EmailAddress: User's email address.
	EmailAddress string `json:"emailAddress,omitempty"`
}

type AccountsCustomBatchRequest struct {
	// Entries: The request entries to be processed in the batch.
	Entries []*AccountsCustomBatchRequestEntry `json:"entries,omitempty"`
}

type AccountsCustomBatchRequestEntry struct {
	// Account: The account to create or update. Only defined if the method
	// is insert or update.
	Account *Account `json:"account,omitempty"`

	// AccountId: The ID of the account to get or delete. Only defined if
	// the method is get or delete.
	AccountId uint64 `json:"accountId,omitempty,string"`

	// BatchId: An entry ID, unique within the batch request.
	BatchId int64 `json:"batchId,omitempty"`

	// MerchantId: The ID of the managing account.
	MerchantId uint64 `json:"merchantId,omitempty,string"`

	Method string `json:"method,omitempty"`
}

type AccountsCustomBatchResponse struct {
	// Entries: The result of the execution of the batch requests.
	Entries []*AccountsCustomBatchResponseEntry `json:"entries,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#accountsCustomBatchResponse".
	Kind string `json:"kind,omitempty"`
}

type AccountsCustomBatchResponseEntry struct {
	// Account: The retrieved, created, or updated account. Not defined if
	// the method was delete.
	Account *Account `json:"account,omitempty"`

	// BatchId: The ID of the request entry this entry responds to.
	BatchId int64 `json:"batchId,omitempty"`

	// Errors: A list of errors defined if and only if the request failed.
	Errors *Errors `json:"errors,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#accountsCustomBatchResponseEntry".
	Kind string `json:"kind,omitempty"`
}

type AccountsListResponse struct {
	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#accountsListResponse".
	Kind string `json:"kind,omitempty"`

	// NextPageToken: The token for the retrieval of the next page of
	// accounts.
	NextPageToken string `json:"nextPageToken,omitempty"`

	Resources []*Account `json:"resources,omitempty"`
}

type AccountstatusesCustomBatchRequest struct {
	// Entries: The request entries to be processed in the batch.
	Entries []*AccountstatusesCustomBatchRequestEntry `json:"entries,omitempty"`
}

type AccountstatusesCustomBatchRequestEntry struct {
	// AccountId: The ID of the (sub-)account whose status to get.
	AccountId uint64 `json:"accountId,omitempty,string"`

	// BatchId: An entry ID, unique within the batch request.
	BatchId int64 `json:"batchId,omitempty"`

	// MerchantId: The ID of the managing account.
	MerchantId uint64 `json:"merchantId,omitempty,string"`

	// Method: The method (get).
	Method string `json:"method,omitempty"`
}

type AccountstatusesCustomBatchResponse struct {
	// Entries: The result of the execution of the batch requests.
	Entries []*AccountstatusesCustomBatchResponseEntry `json:"entries,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#accountstatusesCustomBatchResponse".
	Kind string `json:"kind,omitempty"`
}

type AccountstatusesCustomBatchResponseEntry struct {
	// AccountStatus: The requested account status. Defined if and only if
	// the request was successful.
	AccountStatus *AccountStatus `json:"accountStatus,omitempty"`

	// BatchId: The ID of the request entry this entry responds to.
	BatchId int64 `json:"batchId,omitempty"`

	// Errors: A list of errors defined if and only if the request failed.
	Errors *Errors `json:"errors,omitempty"`
}

type AccountstatusesListResponse struct {
	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#accountstatusesListResponse".
	Kind string `json:"kind,omitempty"`

	// NextPageToken: The token for the retrieval of the next page of
	// account statuses.
	NextPageToken string `json:"nextPageToken,omitempty"`

	Resources []*AccountStatus `json:"resources,omitempty"`
}

type Datafeed struct {
	// AttributeLanguage: The two-letter ISO 639-1 language in which the
	// attributes are defined in the data feed.
	AttributeLanguage string `json:"attributeLanguage,omitempty"`

	// ContentLanguage: The two-letter ISO 639-1 language of the items in
	// the feed.
	ContentLanguage string `json:"contentLanguage,omitempty"`

	// ContentType: The type of data feed.
	ContentType string `json:"contentType,omitempty"`

	// FetchSchedule: Fetch schedule for the feed file.
	FetchSchedule *DatafeedFetchSchedule `json:"fetchSchedule,omitempty"`

	// FileName: The filename of the feed. All feeds must have a unique file
	// name.
	FileName string `json:"fileName,omitempty"`

	// Format: Format of the feed file.
	Format *DatafeedFormat `json:"format,omitempty"`

	// Id: The ID of the data feed.
	Id int64 `json:"id,omitempty,string"`

	// IntendedDestinations: The list of intended destinations (corresponds
	// to checked check boxes in Merchant Center).
	IntendedDestinations []string `json:"intendedDestinations,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#datafeed".
	Kind string `json:"kind,omitempty"`

	// Name: A descriptive name of the data feed.
	Name string `json:"name,omitempty"`

	// TargetCountry: The two-letter ISO 3166 country where the items in the
	// feed will be included in the search index.
	TargetCountry string `json:"targetCountry,omitempty"`
}

type DatafeedFetchSchedule struct {
	// DayOfMonth: The day of the month the feed file should be fetched
	// (1-31).
	DayOfMonth int64 `json:"dayOfMonth,omitempty"`

	// FetchUrl: The URL where the feed file can be fetched. Google Merchant
	// Center will support automatic scheduled uploads using the HTTP,
	// HTTPS, FTP, or SFTP protocols, so the value will need to be a valid
	// link using one of those four protocols.
	FetchUrl string `json:"fetchUrl,omitempty"`

	// Hour: The hour of the day the feed file should be fetched (0-24).
	Hour int64 `json:"hour,omitempty"`

	// Password: An optional password for fetch_url.
	Password string `json:"password,omitempty"`

	// TimeZone: Time zone used for schedule. UTC by default. E.g.,
	// "America/Los_Angeles".
	TimeZone string `json:"timeZone,omitempty"`

	// Username: An optional user name for fetch_url.
	Username string `json:"username,omitempty"`

	// Weekday: The day of the week the feed file should be fetched.
	Weekday string `json:"weekday,omitempty"`
}

type DatafeedFormat struct {
	// ColumnDelimiter: Delimiter for the separation of values in a
	// delimiter-separated values feed. If not specified, the delimiter will
	// be auto-detected. Ignored for non-DSV data feeds.
	ColumnDelimiter string `json:"columnDelimiter,omitempty"`

	// FileEncoding: Character encoding scheme of the data feed. If not
	// specified, the encoding will be auto-detected.
	FileEncoding string `json:"fileEncoding,omitempty"`

	// QuotingMode: Specifies how double quotes are interpreted. If not
	// specified, the mode will be auto-detected. Ignored for non-DSV data
	// feeds.
	QuotingMode string `json:"quotingMode,omitempty"`
}

type DatafeedStatus struct {
	// DatafeedId: The ID of the feed for which the status is reported.
	DatafeedId uint64 `json:"datafeedId,omitempty,string"`

	// Errors: The list of errors occurring in the feed.
	Errors []*DatafeedStatusError `json:"errors,omitempty"`

	// ItemsTotal: The number of items in the feed that were processed.
	ItemsTotal uint64 `json:"itemsTotal,omitempty,string"`

	// ItemsValid: The number of items in the feed that were valid.
	ItemsValid uint64 `json:"itemsValid,omitempty,string"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#datafeedStatus".
	Kind string `json:"kind,omitempty"`

	// ProcessingStatus: The processing status of the feed.
	ProcessingStatus string `json:"processingStatus,omitempty"`

	// Warnings: The list of errors occurring in the feed.
	Warnings []*DatafeedStatusError `json:"warnings,omitempty"`
}

type DatafeedStatusError struct {
	// Code: The code of the error, e.g., "validation/invalid_value".
	Code string `json:"code,omitempty"`

	// Count: The number of occurrences of the error in the feed.
	Count uint64 `json:"count,omitempty,string"`

	// Examples: A list of example occurrences of the error, grouped by
	// product.
	Examples []*DatafeedStatusExample `json:"examples,omitempty"`

	// Message: The error message, e.g., "Invalid price".
	Message string `json:"message,omitempty"`
}

type DatafeedStatusExample struct {
	// ItemId: The ID of the example item.
	ItemId string `json:"itemId,omitempty"`

	// LineNumber: Line number in the data feed where the example is found.
	LineNumber uint64 `json:"lineNumber,omitempty,string"`

	// Value: The problematic value.
	Value string `json:"value,omitempty"`
}

type DatafeedsCustomBatchRequest struct {
	// Entries: The request entries to be processed in the batch.
	Entries []*DatafeedsCustomBatchRequestEntry `json:"entries,omitempty"`
}

type DatafeedsCustomBatchRequestEntry struct {
	// BatchId: An entry ID, unique within the batch request.
	BatchId int64 `json:"batchId,omitempty"`

	// Datafeed: The data feed to insert.
	Datafeed *Datafeed `json:"datafeed,omitempty"`

	// DatafeedId: The ID of the data feed to get or delete.
	DatafeedId uint64 `json:"datafeedId,omitempty,string"`

	// MerchantId: The ID of the managing account.
	MerchantId uint64 `json:"merchantId,omitempty,string"`

	Method string `json:"method,omitempty"`
}

type DatafeedsCustomBatchResponse struct {
	// Entries: The result of the execution of the batch requests.
	Entries []*DatafeedsCustomBatchResponseEntry `json:"entries,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#datafeedsCustomBatchResponse".
	Kind string `json:"kind,omitempty"`
}

type DatafeedsCustomBatchResponseEntry struct {
	// BatchId: The ID of the request entry this entry responds to.
	BatchId int64 `json:"batchId,omitempty"`

	// Datafeed: The requested data feed. Defined if and only if the request
	// was successful.
	Datafeed *Datafeed `json:"datafeed,omitempty"`

	// Errors: A list of errors defined if and only if the request failed.
	Errors *Errors `json:"errors,omitempty"`
}

type DatafeedsListResponse struct {
	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#datafeedsListResponse".
	Kind string `json:"kind,omitempty"`

	// NextPageToken: The token for the retrieval of the next page of
	// datafeeds.
	NextPageToken string `json:"nextPageToken,omitempty"`

	Resources []*Datafeed `json:"resources,omitempty"`
}

type DatafeedstatusesCustomBatchRequest struct {
	// Entries: The request entries to be processed in the batch.
	Entries []*DatafeedstatusesCustomBatchRequestEntry `json:"entries,omitempty"`
}

type DatafeedstatusesCustomBatchRequestEntry struct {
	// BatchId: An entry ID, unique within the batch request.
	BatchId int64 `json:"batchId,omitempty"`

	// DatafeedId: The ID of the data feed to get or delete.
	DatafeedId uint64 `json:"datafeedId,omitempty,string"`

	// MerchantId: The ID of the managing account.
	MerchantId uint64 `json:"merchantId,omitempty,string"`

	Method string `json:"method,omitempty"`
}

type DatafeedstatusesCustomBatchResponse struct {
	// Entries: The result of the execution of the batch requests.
	Entries []*DatafeedstatusesCustomBatchResponseEntry `json:"entries,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#datafeedstatusesCustomBatchResponse".
	Kind string `json:"kind,omitempty"`
}

type DatafeedstatusesCustomBatchResponseEntry struct {
	// BatchId: The ID of the request entry this entry responds to.
	BatchId int64 `json:"batchId,omitempty"`

	// DatafeedStatus: The requested data feed status. Defined if and only
	// if the request was successful.
	DatafeedStatus *DatafeedStatus `json:"datafeedStatus,omitempty"`

	// Errors: A list of errors defined if and only if the request failed.
	Errors *Errors `json:"errors,omitempty"`
}

type DatafeedstatusesListResponse struct {
	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#datafeedstatusesListResponse".
	Kind string `json:"kind,omitempty"`

	// NextPageToken: The token for the retrieval of the next page of
	// datafeed statuses.
	NextPageToken string `json:"nextPageToken,omitempty"`

	Resources []*DatafeedStatus `json:"resources,omitempty"`
}

type Error struct {
	// Domain: The domain of the error.
	Domain string `json:"domain,omitempty"`

	// Message: A description of the error.
	Message string `json:"message,omitempty"`

	// Reason: The error code.
	Reason string `json:"reason,omitempty"`
}

type Errors struct {
	// Code: The HTTP status of the first error in errors.
	Code int64 `json:"code,omitempty"`

	// Errors: A list of errors.
	Errors []*Error `json:"errors,omitempty"`

	// Message: The message of the first error in errors.
	Message string `json:"message,omitempty"`
}

type Inventory struct {
	// Availability: The availability of the product.
	Availability string `json:"availability,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#inventory".
	Kind string `json:"kind,omitempty"`

	// Price: The price of the product.
	Price *Price `json:"price,omitempty"`

	// Quantity: The quantity of the product. Must be equal to or greater
	// than zero. Supported only for local products.
	Quantity int64 `json:"quantity,omitempty"`

	// SalePrice: The sale price of the product. Mandatory if
	// sale_price_effective_date is defined.
	SalePrice *Price `json:"salePrice,omitempty"`

	// SalePriceEffectiveDate: A date range represented by a pair of ISO
	// 8601 dates separated by a space, comma, or slash. Both dates might be
	// specified as 'null' if undecided.
	SalePriceEffectiveDate string `json:"salePriceEffectiveDate,omitempty"`
}

type InventoryCustomBatchRequest struct {
	// Entries: The request entries to be processed in the batch.
	Entries []*InventoryCustomBatchRequestEntry `json:"entries,omitempty"`
}

type InventoryCustomBatchRequestEntry struct {
	// BatchId: An entry ID, unique within the batch request.
	BatchId int64 `json:"batchId,omitempty"`

	// Inventory: Price and availability of the product.
	Inventory *Inventory `json:"inventory,omitempty"`

	// MerchantId: The ID of the managing account.
	MerchantId uint64 `json:"merchantId,omitempty,string"`

	// ProductId: The ID of the product for which to update price and
	// availability.
	ProductId string `json:"productId,omitempty"`

	// StoreCode: The code of the store for which to update price and
	// availability. Use online to update price and availability of an
	// online product.
	StoreCode string `json:"storeCode,omitempty"`
}

type InventoryCustomBatchResponse struct {
	// Entries: The result of the execution of the batch requests.
	Entries []*InventoryCustomBatchResponseEntry `json:"entries,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#inventoryCustomBatchResponse".
	Kind string `json:"kind,omitempty"`
}

type InventoryCustomBatchResponseEntry struct {
	// BatchId: The ID of the request entry this entry responds to.
	BatchId int64 `json:"batchId,omitempty"`

	// Errors: A list of errors defined if and only if the request failed.
	Errors *Errors `json:"errors,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#inventoryCustomBatchResponseEntry".
	Kind string `json:"kind,omitempty"`
}

type InventorySetRequest struct {
	// Availability: The availability of the product.
	Availability string `json:"availability,omitempty"`

	// Price: The price of the product.
	Price *Price `json:"price,omitempty"`

	// Quantity: The quantity of the product. Must be equal to or greater
	// than zero. Supported only for local products.
	Quantity int64 `json:"quantity,omitempty"`

	// SalePrice: The sale price of the product. Mandatory if
	// sale_price_effective_date is defined.
	SalePrice *Price `json:"salePrice,omitempty"`

	// SalePriceEffectiveDate: A date range represented by a pair of ISO
	// 8601 dates separated by a space, comma, or slash. Both dates might be
	// specified as 'null' if undecided.
	SalePriceEffectiveDate string `json:"salePriceEffectiveDate,omitempty"`
}

type InventorySetResponse struct {
	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#inventorySetResponse".
	Kind string `json:"kind,omitempty"`
}

type LoyaltyPoints struct {
	// Name: Name of loyalty points program. It is recommended to limit the
	// name to 12 full-width characters or 24 Roman characters.
	Name string `json:"name,omitempty"`

	// PointsValue: The retailer's loyalty points in absolute value.
	PointsValue int64 `json:"pointsValue,omitempty,string"`

	// Ratio: The ratio of a point when converted to currency. Google
	// assumes currency based on Merchant Center settings. If ratio is left
	// out, it defaults to 1.0.
	Ratio float64 `json:"ratio,omitempty"`
}

type Price struct {
	// Currency: The currency of the price.
	Currency string `json:"currency,omitempty"`

	// Value: The price represented as a number.
	Value string `json:"value,omitempty"`
}

type Product struct {
	// AdditionalImageLinks: Additional URLs of images of the item.
	AdditionalImageLinks []string `json:"additionalImageLinks,omitempty"`

	// Adult: Set to true if the item is targeted towards adults.
	Adult bool `json:"adult,omitempty"`

	// AdwordsGrouping: Used to group items in an arbitrary way. Only for
	// CPA%, discouraged otherwise.
	AdwordsGrouping string `json:"adwordsGrouping,omitempty"`

	// AdwordsLabels: Similar to adwords_grouping, but only works on CPC.
	AdwordsLabels []string `json:"adwordsLabels,omitempty"`

	// AdwordsRedirect: Allows advertisers to override the item URL when the
	// product is shown within the context of Product Ads.
	AdwordsRedirect string `json:"adwordsRedirect,omitempty"`

	// AgeGroup: Target age group of the item.
	AgeGroup string `json:"ageGroup,omitempty"`

	// Availability: Availability status of the item.
	Availability string `json:"availability,omitempty"`

	// AvailabilityDate: The day a pre-ordered product becomes available for
	// delivery.
	AvailabilityDate string `json:"availabilityDate,omitempty"`

	// Brand: Brand of the item.
	Brand string `json:"brand,omitempty"`

	// Channel: The item's channel (online or local).
	Channel string `json:"channel,omitempty"`

	// Color: Color of the item.
	Color string `json:"color,omitempty"`

	// Condition: Condition or state of the item.
	Condition string `json:"condition,omitempty"`

	// ContentLanguage: The two-letter ISO 639-1 language code for the item.
	ContentLanguage string `json:"contentLanguage,omitempty"`

	// CustomAttributes: A list of custom (merchant-provided) attributes.
	CustomAttributes []*ProductCustomAttribute `json:"customAttributes,omitempty"`

	// CustomGroups: A list of custom (merchant-provided) custom attribute
	// groups.
	CustomGroups []*ProductCustomGroup `json:"customGroups,omitempty"`

	// CustomLabel0: Custom label 0 for custom grouping of items in a
	// Shopping campaign.
	CustomLabel0 string `json:"customLabel0,omitempty"`

	// CustomLabel1: Custom label 1 for custom grouping of items in a
	// Shopping campaign.
	CustomLabel1 string `json:"customLabel1,omitempty"`

	// CustomLabel2: Custom label 2 for custom grouping of items in a
	// Shopping campaign.
	CustomLabel2 string `json:"customLabel2,omitempty"`

	// CustomLabel3: Custom label 3 for custom grouping of items in a
	// Shopping campaign.
	CustomLabel3 string `json:"customLabel3,omitempty"`

	// CustomLabel4: Custom label 4 for custom grouping of items in a
	// Shopping campaign.
	CustomLabel4 string `json:"customLabel4,omitempty"`

	// Description: Description of the item.
	Description string `json:"description,omitempty"`

	// Destinations: Specifies the intended destinations for the product.
	Destinations []*ProductDestination `json:"destinations,omitempty"`

	// EnergyEfficiencyClass: The energy efficiency class as defined in EU
	// directive 2010/30/EU.
	EnergyEfficiencyClass string `json:"energyEfficiencyClass,omitempty"`

	// ExpirationDate: Date that an item will expire.
	ExpirationDate string `json:"expirationDate,omitempty"`

	// Gender: Target gender of the item.
	Gender string `json:"gender,omitempty"`

	// GoogleProductCategory: Google's category of the item.
	GoogleProductCategory string `json:"googleProductCategory,omitempty"`

	// Gtin: Global Trade Item Number (GTIN) of the item.
	Gtin string `json:"gtin,omitempty"`

	// Id: The REST id of the product.
	Id string `json:"id,omitempty"`

	// IdentifierExists: False when the item does not have unique product
	// identifiers appropriate to its category, such as GTIN, MPN, and
	// brand. Required according to the Unique Product Identifier Rules for
	// all target countries except for Canada.
	IdentifierExists bool `json:"identifierExists,omitempty"`

	// ImageLink: URL of an image of the item.
	ImageLink string `json:"imageLink,omitempty"`

	// Installment: Number and amount of installments to pay for an item.
	// Brazil only.
	Installment *ProductInstallment `json:"installment,omitempty"`

	// IsBundle: Whether the item is a merchant-defined bundle. A bundle is
	// a custom grouping of different products sold by a merchant for a
	// single price.
	IsBundle bool `json:"isBundle,omitempty"`

	// ItemGroupId: Shared identifier for all variants of the same product.
	ItemGroupId string `json:"itemGroupId,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#product".
	Kind string `json:"kind,omitempty"`

	// Link: URL directly linking to your item's page on your website.
	Link string `json:"link,omitempty"`

	// LoyaltyPoints: Loyalty points that users receive after purchasing the
	// item. Japan only.
	LoyaltyPoints *LoyaltyPoints `json:"loyaltyPoints,omitempty"`

	// Material: The material of which the item is made.
	Material string `json:"material,omitempty"`

	// MobileLink: Link to a mobile-optimized version of the landing page.
	MobileLink string `json:"mobileLink,omitempty"`

	// Mpn: Manufacturer Part Number (MPN) of the item.
	Mpn string `json:"mpn,omitempty"`

	// Multipack: The number of identical products in a merchant-defined
	// multipack.
	Multipack int64 `json:"multipack,omitempty,string"`

	// OfferId: An identifier of the item.
	OfferId string `json:"offerId,omitempty"`

	// OnlineOnly: Whether an item is available for purchase only online.
	OnlineOnly bool `json:"onlineOnly,omitempty"`

	// Pattern: The item's pattern (e.g. polka dots).
	Pattern string `json:"pattern,omitempty"`

	// Price: Price of the item.
	Price *Price `json:"price,omitempty"`

	// ProductType: Your category of the item.
	ProductType string `json:"productType,omitempty"`

	// SalePrice: Advertised sale price of the item.
	SalePrice *Price `json:"salePrice,omitempty"`

	// SalePriceEffectiveDate: Date range during which the item is on sale.
	SalePriceEffectiveDate string `json:"salePriceEffectiveDate,omitempty"`

	// Shipping: Shipping rules.
	Shipping []*ProductShipping `json:"shipping,omitempty"`

	// ShippingLabel: The shipping label of the product, used to group
	// product in account-level shipping rules.
	ShippingLabel string `json:"shippingLabel,omitempty"`

	// ShippingWeight: Weight of the item for shipping.
	ShippingWeight *ProductShippingWeight `json:"shippingWeight,omitempty"`

	// SizeSystem: System in which the size is specified. Recommended for
	// apparel items.
	SizeSystem string `json:"sizeSystem,omitempty"`

	// SizeType: The cut of the item. Recommended for apparel items.
	SizeType string `json:"sizeType,omitempty"`

	// Sizes: Size of the item.
	Sizes []string `json:"sizes,omitempty"`

	// TargetCountry: The two-letter ISO 3166 country code for the item.
	TargetCountry string `json:"targetCountry,omitempty"`

	// Taxes: Tax information.
	Taxes []*ProductTax `json:"taxes,omitempty"`

	// Title: Title of the item.
	Title string `json:"title,omitempty"`

	// UnitPricingBaseMeasure: The preference of the denominator of the unit
	// price.
	UnitPricingBaseMeasure *ProductUnitPricingBaseMeasure `json:"unitPricingBaseMeasure,omitempty"`

	// UnitPricingMeasure: The measure and dimension of an item.
	UnitPricingMeasure *ProductUnitPricingMeasure `json:"unitPricingMeasure,omitempty"`

	// ValidatedDestinations: The read-only list of intended destinations
	// which passed validation.
	ValidatedDestinations []string `json:"validatedDestinations,omitempty"`

	// Warnings: Read-only warnings.
	Warnings []*Error `json:"warnings,omitempty"`
}

type ProductCustomAttribute struct {
	// Name: The name of the attribute.
	Name string `json:"name,omitempty"`

	// Type: The type of the attribute.
	Type string `json:"type,omitempty"`

	// Unit: Free-form unit of the attribute. Unit can only be used for
	// values of type INT or FLOAT.
	Unit string `json:"unit,omitempty"`

	// Value: The value of the attribute.
	Value string `json:"value,omitempty"`
}

type ProductCustomGroup struct {
	// Attributes: The sub-attributes.
	Attributes []*ProductCustomAttribute `json:"attributes,omitempty"`

	// Name: The name of the group.
	Name string `json:"name,omitempty"`
}

type ProductDestination struct {
	// DestinationName: The name of the destination.
	DestinationName string `json:"destinationName,omitempty"`

	// Intention: Whether the destination is required, excluded or should be
	// validated.
	Intention string `json:"intention,omitempty"`
}

type ProductInstallment struct {
	// Amount: The amount the buyer has to pay per month.
	Amount *Price `json:"amount,omitempty"`

	// Months: The number of installments the buyer has to pay.
	Months int64 `json:"months,omitempty,string"`
}

type ProductShipping struct {
	// Country: The two-letter ISO 3166 country code for the country to
	// which an item will ship.
	Country string `json:"country,omitempty"`

	// LocationGroupName: The location where the shipping is applicable,
	// represented by a location group name.
	LocationGroupName string `json:"locationGroupName,omitempty"`

	// LocationId: The numeric id of a location that the shipping rate
	// applies to as defined in the AdWords API.
	LocationId int64 `json:"locationId,omitempty,string"`

	// PostalCode: The postal code range that the shipping rate applies to,
	// represented by a postal code, a postal code prefix using * wildcard,
	// a range between two postal codes or two postal code prefixes of equal
	// length.
	PostalCode string `json:"postalCode,omitempty"`

	// Price: Fixed shipping price, represented as a number.
	Price *Price `json:"price,omitempty"`

	// Region: The geographic region to which a shipping rate applies (e.g.
	// zip code).
	Region string `json:"region,omitempty"`

	// Service: A free-form description of the service class or delivery
	// speed.
	Service string `json:"service,omitempty"`
}

type ProductShippingWeight struct {
	// Unit: The unit of value.
	Unit string `json:"unit,omitempty"`

	// Value: The weight of the product used to calculate the shipping cost
	// of the item.
	Value float64 `json:"value,omitempty"`
}

type ProductStatus struct {
	// DataQualityIssues: A list of data quality issues associated with the
	// product.
	DataQualityIssues []*ProductStatusDataQualityIssue `json:"dataQualityIssues,omitempty"`

	// DestinationStatuses: The intended destinations for the product.
	DestinationStatuses []*ProductStatusDestinationStatus `json:"destinationStatuses,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#productStatus".
	Kind string `json:"kind,omitempty"`

	// Link: The link to the product.
	Link string `json:"link,omitempty"`

	// ProductId: The id of the product for which status is reported.
	ProductId string `json:"productId,omitempty"`

	// Title: The title of the product.
	Title string `json:"title,omitempty"`
}

type ProductStatusDataQualityIssue struct {
	// Detail: A more detailed error string.
	Detail string `json:"detail,omitempty"`

	// FetchStatus: The fetch status for landing_page_errors.
	FetchStatus string `json:"fetchStatus,omitempty"`

	// Id: The id of the data quality issue.
	Id string `json:"id,omitempty"`

	// Location: The attribute name that is relevant for the issue.
	Location string `json:"location,omitempty"`

	// Timestamp: The time stamp of the data quality issue.
	Timestamp string `json:"timestamp,omitempty"`

	// ValueOnLandingPage: The value of that attribute that was found on the
	// landing page
	ValueOnLandingPage string `json:"valueOnLandingPage,omitempty"`

	// ValueProvided: The value the attribute had at time of evaluation.
	ValueProvided string `json:"valueProvided,omitempty"`
}

type ProductStatusDestinationStatus struct {
	// ApprovalStatus: The destination's approval status.
	ApprovalStatus string `json:"approvalStatus,omitempty"`

	// Destination: The name of the destination
	Destination string `json:"destination,omitempty"`

	// Intention: Whether the destination is required, excluded, selected by
	// default or should be validated.
	Intention string `json:"intention,omitempty"`
}

type ProductTax struct {
	// Country: The country within which the item is taxed, specified with a
	// two-letter ISO 3166 country code.
	Country string `json:"country,omitempty"`

	// LocationId: The numeric id of a location that the tax rate applies to
	// as defined in the Adwords API
	// (https://developers.google.com/adwords/api/docs/appendix/geotargeting)
	// .
	LocationId int64 `json:"locationId,omitempty,string"`

	// PostalCode: The postal code range that the tax rate applies to,
	// represented by a ZIP code, a ZIP code prefix using * wildcard, a
	// range between two ZIP codes or two ZIP code prefixes of equal length.
	// Examples: 94114, 94*, 94002-95460, 94*-95*.
	PostalCode string `json:"postalCode,omitempty"`

	// Rate: The percentage of tax rate that applies to the item price.
	Rate float64 `json:"rate,omitempty"`

	// Region: The geographic region to which the tax rate applies.
	Region string `json:"region,omitempty"`

	// TaxShip: Set to true if tax is charged on shipping.
	TaxShip bool `json:"taxShip,omitempty"`
}

type ProductUnitPricingBaseMeasure struct {
	// Unit: The unit of the denominator.
	Unit string `json:"unit,omitempty"`

	// Value: The denominator of the unit price.
	Value int64 `json:"value,omitempty,string"`
}

type ProductUnitPricingMeasure struct {
	// Unit: The unit of the measure.
	Unit string `json:"unit,omitempty"`

	// Value: The measure of an item.
	Value float64 `json:"value,omitempty"`
}

type ProductsCustomBatchRequest struct {
	// Entries: The request entries to be processed in the batch.
	Entries []*ProductsCustomBatchRequestEntry `json:"entries,omitempty"`
}

type ProductsCustomBatchRequestEntry struct {
	// BatchId: An entry ID, unique within the batch request.
	BatchId int64 `json:"batchId,omitempty"`

	// MerchantId: The ID of the managing account.
	MerchantId uint64 `json:"merchantId,omitempty,string"`

	Method string `json:"method,omitempty"`

	// Product: The product to insert. Only required if the method is
	// insert.
	Product *Product `json:"product,omitempty"`

	// ProductId: The ID of the product to get or delete. Only defined if
	// the method is get or delete.
	ProductId string `json:"productId,omitempty"`
}

type ProductsCustomBatchResponse struct {
	// Entries: The result of the execution of the batch requests.
	Entries []*ProductsCustomBatchResponseEntry `json:"entries,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#productsCustomBatchResponse".
	Kind string `json:"kind,omitempty"`
}

type ProductsCustomBatchResponseEntry struct {
	// BatchId: The ID of the request entry this entry responds to.
	BatchId int64 `json:"batchId,omitempty"`

	// Errors: A list of errors defined if and only if the request failed.
	Errors *Errors `json:"errors,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#productsCustomBatchResponseEntry".
	Kind string `json:"kind,omitempty"`

	// Product: The inserted product. Only defined if the method is insert
	// and if the request was successful.
	Product *Product `json:"product,omitempty"`
}

type ProductsListResponse struct {
	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#productsListResponse".
	Kind string `json:"kind,omitempty"`

	// NextPageToken: The token for the retrieval of the next page of
	// products.
	NextPageToken string `json:"nextPageToken,omitempty"`

	Resources []*Product `json:"resources,omitempty"`
}

type ProductstatusesCustomBatchRequest struct {
	// Entries: The request entries to be processed in the batch.
	Entries []*ProductstatusesCustomBatchRequestEntry `json:"entries,omitempty"`
}

type ProductstatusesCustomBatchRequestEntry struct {
	// BatchId: An entry ID, unique within the batch request.
	BatchId int64 `json:"batchId,omitempty"`

	// MerchantId: The ID of the managing account.
	MerchantId uint64 `json:"merchantId,omitempty,string"`

	Method string `json:"method,omitempty"`

	// ProductId: The ID of the product whose status to get.
	ProductId string `json:"productId,omitempty"`
}

type ProductstatusesCustomBatchResponse struct {
	// Entries: The result of the execution of the batch requests.
	Entries []*ProductstatusesCustomBatchResponseEntry `json:"entries,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#productstatusesCustomBatchResponse".
	Kind string `json:"kind,omitempty"`
}

type ProductstatusesCustomBatchResponseEntry struct {
	// BatchId: The ID of the request entry this entry responds to.
	BatchId int64 `json:"batchId,omitempty"`

	// Errors: A list of errors, if the request failed.
	Errors *Errors `json:"errors,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#productstatusesCustomBatchResponseEntry".
	Kind string `json:"kind,omitempty"`

	// ProductStatus: The requested product status. Only defined if the
	// request was successful.
	ProductStatus *ProductStatus `json:"productStatus,omitempty"`
}

type ProductstatusesListResponse struct {
	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "content#productstatusesListResponse".
	Kind string `json:"kind,omitempty"`

	// NextPageToken: The token for the retrieval of the next page of
	// products statuses.
	NextPageToken string `json:"nextPageToken,omitempty"`

	Resources []*ProductStatus `json:"resources,omitempty"`
}

// method id "content.accounts.custombatch":

type AccountsCustombatchCall struct {
	s                          *Service
	accountscustombatchrequest *AccountsCustomBatchRequest
	opt_                       map[string]interface{}
}

// Custombatch: Retrieves, inserts, updates, and deletes multiple
// Merchant Center (sub-)accounts in a single request.
func (r *AccountsService) Custombatch(accountscustombatchrequest *AccountsCustomBatchRequest) *AccountsCustombatchCall {
	c := &AccountsCustombatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountscustombatchrequest = accountscustombatchrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsCustombatchCall) Fields(s ...googleapi.Field) *AccountsCustombatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsCustombatchCall) Do() (*AccountsCustomBatchResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.accountscustombatchrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/batch")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *AccountsCustomBatchResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves, inserts, updates, and deletes multiple Merchant Center (sub-)accounts in a single request.",
	//   "httpMethod": "POST",
	//   "id": "content.accounts.custombatch",
	//   "path": "accounts/batch",
	//   "request": {
	//     "$ref": "AccountsCustomBatchRequest"
	//   },
	//   "response": {
	//     "$ref": "AccountsCustomBatchResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.accounts.delete":

type AccountsDeleteCall struct {
	s          *Service
	merchantId uint64
	accountId  uint64
	opt_       map[string]interface{}
}

// Delete: Deletes a Merchant Center sub-account.
func (r *AccountsService) Delete(merchantId uint64, accountId uint64) *AccountsDeleteCall {
	c := &AccountsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.accountId = accountId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsDeleteCall) Fields(s ...googleapi.Field) *AccountsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/accounts/{accountId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
		"accountId":  strconv.FormatUint(c.accountId, 10),
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Deletes a Merchant Center sub-account.",
	//   "httpMethod": "DELETE",
	//   "id": "content.accounts.delete",
	//   "parameterOrder": [
	//     "merchantId",
	//     "accountId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The ID of the account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/accounts/{accountId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.accounts.get":

type AccountsGetCall struct {
	s          *Service
	merchantId uint64
	accountId  uint64
	opt_       map[string]interface{}
}

// Get: Retrieves a Merchant Center account.
func (r *AccountsService) Get(merchantId uint64, accountId uint64) *AccountsGetCall {
	c := &AccountsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.accountId = accountId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsGetCall) Fields(s ...googleapi.Field) *AccountsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsGetCall) Do() (*Account, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/accounts/{accountId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
		"accountId":  strconv.FormatUint(c.accountId, 10),
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Account
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves a Merchant Center account.",
	//   "httpMethod": "GET",
	//   "id": "content.accounts.get",
	//   "parameterOrder": [
	//     "merchantId",
	//     "accountId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The ID of the account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/accounts/{accountId}",
	//   "response": {
	//     "$ref": "Account"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.accounts.insert":

type AccountsInsertCall struct {
	s          *Service
	merchantId uint64
	account    *Account
	opt_       map[string]interface{}
}

// Insert: Creates a Merchant Center sub-account.
func (r *AccountsService) Insert(merchantId uint64, account *Account) *AccountsInsertCall {
	c := &AccountsInsertCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.account = account
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsInsertCall) Fields(s ...googleapi.Field) *AccountsInsertCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsInsertCall) Do() (*Account, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.account)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/accounts")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Account
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a Merchant Center sub-account.",
	//   "httpMethod": "POST",
	//   "id": "content.accounts.insert",
	//   "parameterOrder": [
	//     "merchantId"
	//   ],
	//   "parameters": {
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/accounts",
	//   "request": {
	//     "$ref": "Account"
	//   },
	//   "response": {
	//     "$ref": "Account"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.accounts.list":

type AccountsListCall struct {
	s          *Service
	merchantId uint64
	opt_       map[string]interface{}
}

// List: Lists the sub-accounts in your Merchant Center account.
func (r *AccountsService) List(merchantId uint64) *AccountsListCall {
	c := &AccountsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	return c
}

// MaxResults sets the optional parameter "maxResults": The maximum
// number of accounts to return in the response, used for paging.
func (c *AccountsListCall) MaxResults(maxResults int64) *AccountsListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": The token returned
// by the previous request.
func (c *AccountsListCall) PageToken(pageToken string) *AccountsListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsListCall) Fields(s ...googleapi.Field) *AccountsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsListCall) Do() (*AccountsListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/accounts")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *AccountsListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists the sub-accounts in your Merchant Center account.",
	//   "httpMethod": "GET",
	//   "id": "content.accounts.list",
	//   "parameterOrder": [
	//     "merchantId"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "description": "The maximum number of accounts to return in the response, used for paging.",
	//       "format": "uint32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "pageToken": {
	//       "description": "The token returned by the previous request.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/accounts",
	//   "response": {
	//     "$ref": "AccountsListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.accounts.patch":

type AccountsPatchCall struct {
	s          *Service
	merchantId uint64
	accountId  uint64
	account    *Account
	opt_       map[string]interface{}
}

// Patch: Updates a Merchant Center account. This method supports patch
// semantics.
func (r *AccountsService) Patch(merchantId uint64, accountId uint64, account *Account) *AccountsPatchCall {
	c := &AccountsPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.accountId = accountId
	c.account = account
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsPatchCall) Fields(s ...googleapi.Field) *AccountsPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsPatchCall) Do() (*Account, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.account)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/accounts/{accountId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
		"accountId":  strconv.FormatUint(c.accountId, 10),
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Account
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a Merchant Center account. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "content.accounts.patch",
	//   "parameterOrder": [
	//     "merchantId",
	//     "accountId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The ID of the account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/accounts/{accountId}",
	//   "request": {
	//     "$ref": "Account"
	//   },
	//   "response": {
	//     "$ref": "Account"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.accounts.update":

type AccountsUpdateCall struct {
	s          *Service
	merchantId uint64
	accountId  uint64
	account    *Account
	opt_       map[string]interface{}
}

// Update: Updates a Merchant Center account.
func (r *AccountsService) Update(merchantId uint64, accountId uint64, account *Account) *AccountsUpdateCall {
	c := &AccountsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.accountId = accountId
	c.account = account
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsUpdateCall) Fields(s ...googleapi.Field) *AccountsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsUpdateCall) Do() (*Account, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.account)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/accounts/{accountId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
		"accountId":  strconv.FormatUint(c.accountId, 10),
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Account
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a Merchant Center account.",
	//   "httpMethod": "PUT",
	//   "id": "content.accounts.update",
	//   "parameterOrder": [
	//     "merchantId",
	//     "accountId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The ID of the account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/accounts/{accountId}",
	//   "request": {
	//     "$ref": "Account"
	//   },
	//   "response": {
	//     "$ref": "Account"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.accountstatuses.custombatch":

type AccountstatusesCustombatchCall struct {
	s                                 *Service
	accountstatusescustombatchrequest *AccountstatusesCustomBatchRequest
	opt_                              map[string]interface{}
}

// Custombatch:
func (r *AccountstatusesService) Custombatch(accountstatusescustombatchrequest *AccountstatusesCustomBatchRequest) *AccountstatusesCustombatchCall {
	c := &AccountstatusesCustombatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountstatusescustombatchrequest = accountstatusescustombatchrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountstatusesCustombatchCall) Fields(s ...googleapi.Field) *AccountstatusesCustombatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountstatusesCustombatchCall) Do() (*AccountstatusesCustomBatchResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.accountstatusescustombatchrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accountstatuses/batch")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *AccountstatusesCustomBatchResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "httpMethod": "POST",
	//   "id": "content.accountstatuses.custombatch",
	//   "path": "accountstatuses/batch",
	//   "request": {
	//     "$ref": "AccountstatusesCustomBatchRequest"
	//   },
	//   "response": {
	//     "$ref": "AccountstatusesCustomBatchResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.accountstatuses.get":

type AccountstatusesGetCall struct {
	s          *Service
	merchantId uint64
	accountId  uint64
	opt_       map[string]interface{}
}

// Get: Retrieves the status of a Merchant Center account.
func (r *AccountstatusesService) Get(merchantId uint64, accountId uint64) *AccountstatusesGetCall {
	c := &AccountstatusesGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.accountId = accountId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountstatusesGetCall) Fields(s ...googleapi.Field) *AccountstatusesGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountstatusesGetCall) Do() (*AccountStatus, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/accountstatuses/{accountId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
		"accountId":  strconv.FormatUint(c.accountId, 10),
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *AccountStatus
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves the status of a Merchant Center account.",
	//   "httpMethod": "GET",
	//   "id": "content.accountstatuses.get",
	//   "parameterOrder": [
	//     "merchantId",
	//     "accountId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The ID of the account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/accountstatuses/{accountId}",
	//   "response": {
	//     "$ref": "AccountStatus"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.accountstatuses.list":

type AccountstatusesListCall struct {
	s          *Service
	merchantId uint64
	opt_       map[string]interface{}
}

// List: Lists the statuses of the sub-accounts in your Merchant Center
// account.
func (r *AccountstatusesService) List(merchantId uint64) *AccountstatusesListCall {
	c := &AccountstatusesListCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	return c
}

// MaxResults sets the optional parameter "maxResults": The maximum
// number of account statuses to return in the response, used for
// paging.
func (c *AccountstatusesListCall) MaxResults(maxResults int64) *AccountstatusesListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": The token returned
// by the previous request.
func (c *AccountstatusesListCall) PageToken(pageToken string) *AccountstatusesListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountstatusesListCall) Fields(s ...googleapi.Field) *AccountstatusesListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountstatusesListCall) Do() (*AccountstatusesListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/accountstatuses")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *AccountstatusesListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists the statuses of the sub-accounts in your Merchant Center account.",
	//   "httpMethod": "GET",
	//   "id": "content.accountstatuses.list",
	//   "parameterOrder": [
	//     "merchantId"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "description": "The maximum number of account statuses to return in the response, used for paging.",
	//       "format": "uint32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "pageToken": {
	//       "description": "The token returned by the previous request.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/accountstatuses",
	//   "response": {
	//     "$ref": "AccountstatusesListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.datafeeds.custombatch":

type DatafeedsCustombatchCall struct {
	s                           *Service
	datafeedscustombatchrequest *DatafeedsCustomBatchRequest
	opt_                        map[string]interface{}
}

// Custombatch:
func (r *DatafeedsService) Custombatch(datafeedscustombatchrequest *DatafeedsCustomBatchRequest) *DatafeedsCustombatchCall {
	c := &DatafeedsCustombatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.datafeedscustombatchrequest = datafeedscustombatchrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatafeedsCustombatchCall) Fields(s ...googleapi.Field) *DatafeedsCustombatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatafeedsCustombatchCall) Do() (*DatafeedsCustomBatchResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.datafeedscustombatchrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "datafeeds/batch")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *DatafeedsCustomBatchResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "httpMethod": "POST",
	//   "id": "content.datafeeds.custombatch",
	//   "path": "datafeeds/batch",
	//   "request": {
	//     "$ref": "DatafeedsCustomBatchRequest"
	//   },
	//   "response": {
	//     "$ref": "DatafeedsCustomBatchResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.datafeeds.delete":

type DatafeedsDeleteCall struct {
	s          *Service
	merchantId uint64
	datafeedId uint64
	opt_       map[string]interface{}
}

// Delete: Deletes a datafeed from your Merchant Center account.
func (r *DatafeedsService) Delete(merchantId uint64, datafeedId uint64) *DatafeedsDeleteCall {
	c := &DatafeedsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.datafeedId = datafeedId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatafeedsDeleteCall) Fields(s ...googleapi.Field) *DatafeedsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatafeedsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/datafeeds/{datafeedId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
		"datafeedId": strconv.FormatUint(c.datafeedId, 10),
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Deletes a datafeed from your Merchant Center account.",
	//   "httpMethod": "DELETE",
	//   "id": "content.datafeeds.delete",
	//   "parameterOrder": [
	//     "merchantId",
	//     "datafeedId"
	//   ],
	//   "parameters": {
	//     "datafeedId": {
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "merchantId": {
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/datafeeds/{datafeedId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.datafeeds.get":

type DatafeedsGetCall struct {
	s          *Service
	merchantId uint64
	datafeedId uint64
	opt_       map[string]interface{}
}

// Get: Retrieves a datafeed from your Merchant Center account.
func (r *DatafeedsService) Get(merchantId uint64, datafeedId uint64) *DatafeedsGetCall {
	c := &DatafeedsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.datafeedId = datafeedId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatafeedsGetCall) Fields(s ...googleapi.Field) *DatafeedsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatafeedsGetCall) Do() (*Datafeed, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/datafeeds/{datafeedId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
		"datafeedId": strconv.FormatUint(c.datafeedId, 10),
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Datafeed
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves a datafeed from your Merchant Center account.",
	//   "httpMethod": "GET",
	//   "id": "content.datafeeds.get",
	//   "parameterOrder": [
	//     "merchantId",
	//     "datafeedId"
	//   ],
	//   "parameters": {
	//     "datafeedId": {
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "merchantId": {
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/datafeeds/{datafeedId}",
	//   "response": {
	//     "$ref": "Datafeed"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.datafeeds.insert":

type DatafeedsInsertCall struct {
	s          *Service
	merchantId uint64
	datafeed   *Datafeed
	opt_       map[string]interface{}
}

// Insert: Registers a datafeed with your Merchant Center account.
func (r *DatafeedsService) Insert(merchantId uint64, datafeed *Datafeed) *DatafeedsInsertCall {
	c := &DatafeedsInsertCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.datafeed = datafeed
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatafeedsInsertCall) Fields(s ...googleapi.Field) *DatafeedsInsertCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatafeedsInsertCall) Do() (*Datafeed, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.datafeed)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/datafeeds")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Datafeed
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Registers a datafeed with your Merchant Center account.",
	//   "httpMethod": "POST",
	//   "id": "content.datafeeds.insert",
	//   "parameterOrder": [
	//     "merchantId"
	//   ],
	//   "parameters": {
	//     "merchantId": {
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/datafeeds",
	//   "request": {
	//     "$ref": "Datafeed"
	//   },
	//   "response": {
	//     "$ref": "Datafeed"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.datafeeds.list":

type DatafeedsListCall struct {
	s          *Service
	merchantId uint64
	opt_       map[string]interface{}
}

// List: Lists the datafeeds in your Merchant Center account.
func (r *DatafeedsService) List(merchantId uint64) *DatafeedsListCall {
	c := &DatafeedsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	return c
}

// MaxResults sets the optional parameter "maxResults": The maximum
// number of products to return in the response, used for paging.
func (c *DatafeedsListCall) MaxResults(maxResults int64) *DatafeedsListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": The token returned
// by the previous request.
func (c *DatafeedsListCall) PageToken(pageToken string) *DatafeedsListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatafeedsListCall) Fields(s ...googleapi.Field) *DatafeedsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatafeedsListCall) Do() (*DatafeedsListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/datafeeds")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *DatafeedsListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists the datafeeds in your Merchant Center account.",
	//   "httpMethod": "GET",
	//   "id": "content.datafeeds.list",
	//   "parameterOrder": [
	//     "merchantId"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "description": "The maximum number of products to return in the response, used for paging.",
	//       "format": "uint32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "pageToken": {
	//       "description": "The token returned by the previous request.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/datafeeds",
	//   "response": {
	//     "$ref": "DatafeedsListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.datafeeds.patch":

type DatafeedsPatchCall struct {
	s          *Service
	merchantId uint64
	datafeedId uint64
	datafeed   *Datafeed
	opt_       map[string]interface{}
}

// Patch: Updates a datafeed of your Merchant Center account. This
// method supports patch semantics.
func (r *DatafeedsService) Patch(merchantId uint64, datafeedId uint64, datafeed *Datafeed) *DatafeedsPatchCall {
	c := &DatafeedsPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.datafeedId = datafeedId
	c.datafeed = datafeed
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatafeedsPatchCall) Fields(s ...googleapi.Field) *DatafeedsPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatafeedsPatchCall) Do() (*Datafeed, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.datafeed)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/datafeeds/{datafeedId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
		"datafeedId": strconv.FormatUint(c.datafeedId, 10),
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Datafeed
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a datafeed of your Merchant Center account. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "content.datafeeds.patch",
	//   "parameterOrder": [
	//     "merchantId",
	//     "datafeedId"
	//   ],
	//   "parameters": {
	//     "datafeedId": {
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "merchantId": {
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/datafeeds/{datafeedId}",
	//   "request": {
	//     "$ref": "Datafeed"
	//   },
	//   "response": {
	//     "$ref": "Datafeed"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.datafeeds.update":

type DatafeedsUpdateCall struct {
	s          *Service
	merchantId uint64
	datafeedId uint64
	datafeed   *Datafeed
	opt_       map[string]interface{}
}

// Update: Updates a datafeed of your Merchant Center account.
func (r *DatafeedsService) Update(merchantId uint64, datafeedId uint64, datafeed *Datafeed) *DatafeedsUpdateCall {
	c := &DatafeedsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.datafeedId = datafeedId
	c.datafeed = datafeed
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatafeedsUpdateCall) Fields(s ...googleapi.Field) *DatafeedsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatafeedsUpdateCall) Do() (*Datafeed, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.datafeed)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/datafeeds/{datafeedId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
		"datafeedId": strconv.FormatUint(c.datafeedId, 10),
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Datafeed
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates a datafeed of your Merchant Center account.",
	//   "httpMethod": "PUT",
	//   "id": "content.datafeeds.update",
	//   "parameterOrder": [
	//     "merchantId",
	//     "datafeedId"
	//   ],
	//   "parameters": {
	//     "datafeedId": {
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "merchantId": {
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/datafeeds/{datafeedId}",
	//   "request": {
	//     "$ref": "Datafeed"
	//   },
	//   "response": {
	//     "$ref": "Datafeed"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.datafeedstatuses.custombatch":

type DatafeedstatusesCustombatchCall struct {
	s                                  *Service
	datafeedstatusescustombatchrequest *DatafeedstatusesCustomBatchRequest
	opt_                               map[string]interface{}
}

// Custombatch:
func (r *DatafeedstatusesService) Custombatch(datafeedstatusescustombatchrequest *DatafeedstatusesCustomBatchRequest) *DatafeedstatusesCustombatchCall {
	c := &DatafeedstatusesCustombatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.datafeedstatusescustombatchrequest = datafeedstatusescustombatchrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatafeedstatusesCustombatchCall) Fields(s ...googleapi.Field) *DatafeedstatusesCustombatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatafeedstatusesCustombatchCall) Do() (*DatafeedstatusesCustomBatchResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.datafeedstatusescustombatchrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "datafeedstatuses/batch")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *DatafeedstatusesCustomBatchResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "httpMethod": "POST",
	//   "id": "content.datafeedstatuses.custombatch",
	//   "path": "datafeedstatuses/batch",
	//   "request": {
	//     "$ref": "DatafeedstatusesCustomBatchRequest"
	//   },
	//   "response": {
	//     "$ref": "DatafeedstatusesCustomBatchResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.datafeedstatuses.get":

type DatafeedstatusesGetCall struct {
	s          *Service
	merchantId uint64
	datafeedId uint64
	opt_       map[string]interface{}
}

// Get: Retrieves the status of a datafeed from your Merchant Center
// account.
func (r *DatafeedstatusesService) Get(merchantId uint64, datafeedId uint64) *DatafeedstatusesGetCall {
	c := &DatafeedstatusesGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.datafeedId = datafeedId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatafeedstatusesGetCall) Fields(s ...googleapi.Field) *DatafeedstatusesGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatafeedstatusesGetCall) Do() (*DatafeedStatus, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/datafeedstatuses/{datafeedId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
		"datafeedId": strconv.FormatUint(c.datafeedId, 10),
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *DatafeedStatus
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves the status of a datafeed from your Merchant Center account.",
	//   "httpMethod": "GET",
	//   "id": "content.datafeedstatuses.get",
	//   "parameterOrder": [
	//     "merchantId",
	//     "datafeedId"
	//   ],
	//   "parameters": {
	//     "datafeedId": {
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "merchantId": {
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/datafeedstatuses/{datafeedId}",
	//   "response": {
	//     "$ref": "DatafeedStatus"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.datafeedstatuses.list":

type DatafeedstatusesListCall struct {
	s          *Service
	merchantId uint64
	opt_       map[string]interface{}
}

// List: Lists the statuses of the datafeeds in your Merchant Center
// account.
func (r *DatafeedstatusesService) List(merchantId uint64) *DatafeedstatusesListCall {
	c := &DatafeedstatusesListCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	return c
}

// MaxResults sets the optional parameter "maxResults": The maximum
// number of products to return in the response, used for paging.
func (c *DatafeedstatusesListCall) MaxResults(maxResults int64) *DatafeedstatusesListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": The token returned
// by the previous request.
func (c *DatafeedstatusesListCall) PageToken(pageToken string) *DatafeedstatusesListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DatafeedstatusesListCall) Fields(s ...googleapi.Field) *DatafeedstatusesListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DatafeedstatusesListCall) Do() (*DatafeedstatusesListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/datafeedstatuses")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *DatafeedstatusesListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists the statuses of the datafeeds in your Merchant Center account.",
	//   "httpMethod": "GET",
	//   "id": "content.datafeedstatuses.list",
	//   "parameterOrder": [
	//     "merchantId"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "description": "The maximum number of products to return in the response, used for paging.",
	//       "format": "uint32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "pageToken": {
	//       "description": "The token returned by the previous request.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/datafeedstatuses",
	//   "response": {
	//     "$ref": "DatafeedstatusesListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.inventory.custombatch":

type InventoryCustombatchCall struct {
	s                           *Service
	inventorycustombatchrequest *InventoryCustomBatchRequest
	opt_                        map[string]interface{}
}

// Custombatch: Updates price and availability for multiple products or
// stores in a single request.
func (r *InventoryService) Custombatch(inventorycustombatchrequest *InventoryCustomBatchRequest) *InventoryCustombatchCall {
	c := &InventoryCustombatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.inventorycustombatchrequest = inventorycustombatchrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *InventoryCustombatchCall) Fields(s ...googleapi.Field) *InventoryCustombatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *InventoryCustombatchCall) Do() (*InventoryCustomBatchResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.inventorycustombatchrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "inventory/batch")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *InventoryCustomBatchResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates price and availability for multiple products or stores in a single request.",
	//   "httpMethod": "POST",
	//   "id": "content.inventory.custombatch",
	//   "path": "inventory/batch",
	//   "request": {
	//     "$ref": "InventoryCustomBatchRequest"
	//   },
	//   "response": {
	//     "$ref": "InventoryCustomBatchResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.inventory.set":

type InventorySetCall struct {
	s                   *Service
	merchantId          uint64
	storeCode           string
	productId           string
	inventorysetrequest *InventorySetRequest
	opt_                map[string]interface{}
}

// Set: Updates price and availability of a product in your Merchant
// Center account.
func (r *InventoryService) Set(merchantId uint64, storeCode string, productId string, inventorysetrequest *InventorySetRequest) *InventorySetCall {
	c := &InventorySetCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.storeCode = storeCode
	c.productId = productId
	c.inventorysetrequest = inventorysetrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *InventorySetCall) Fields(s ...googleapi.Field) *InventorySetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *InventorySetCall) Do() (*InventorySetResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.inventorysetrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/inventory/{storeCode}/products/{productId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
		"storeCode":  c.storeCode,
		"productId":  c.productId,
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *InventorySetResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates price and availability of a product in your Merchant Center account.",
	//   "httpMethod": "POST",
	//   "id": "content.inventory.set",
	//   "parameterOrder": [
	//     "merchantId",
	//     "storeCode",
	//     "productId"
	//   ],
	//   "parameters": {
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "productId": {
	//       "description": "The ID of the product for which to update price and availability.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "storeCode": {
	//       "description": "The code of the store for which to update price and availability. Use online to update price and availability of an online product.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/inventory/{storeCode}/products/{productId}",
	//   "request": {
	//     "$ref": "InventorySetRequest"
	//   },
	//   "response": {
	//     "$ref": "InventorySetResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.products.custombatch":

type ProductsCustombatchCall struct {
	s                          *Service
	productscustombatchrequest *ProductsCustomBatchRequest
	opt_                       map[string]interface{}
}

// Custombatch: Retrieves, inserts, and deletes multiple products in a
// single request.
func (r *ProductsService) Custombatch(productscustombatchrequest *ProductsCustomBatchRequest) *ProductsCustombatchCall {
	c := &ProductsCustombatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.productscustombatchrequest = productscustombatchrequest
	return c
}

// DryRun sets the optional parameter "dryRun": Flag to run the request
// in dry-run mode.
func (c *ProductsCustombatchCall) DryRun(dryRun bool) *ProductsCustombatchCall {
	c.opt_["dryRun"] = dryRun
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProductsCustombatchCall) Fields(s ...googleapi.Field) *ProductsCustombatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ProductsCustombatchCall) Do() (*ProductsCustomBatchResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.productscustombatchrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["dryRun"]; ok {
		params.Set("dryRun", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "products/batch")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *ProductsCustomBatchResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves, inserts, and deletes multiple products in a single request.",
	//   "httpMethod": "POST",
	//   "id": "content.products.custombatch",
	//   "parameters": {
	//     "dryRun": {
	//       "description": "Flag to run the request in dry-run mode.",
	//       "location": "query",
	//       "type": "boolean"
	//     }
	//   },
	//   "path": "products/batch",
	//   "request": {
	//     "$ref": "ProductsCustomBatchRequest"
	//   },
	//   "response": {
	//     "$ref": "ProductsCustomBatchResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.products.delete":

type ProductsDeleteCall struct {
	s          *Service
	merchantId uint64
	productId  string
	opt_       map[string]interface{}
}

// Delete: Deletes a product from your Merchant Center account.
func (r *ProductsService) Delete(merchantId uint64, productId string) *ProductsDeleteCall {
	c := &ProductsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.productId = productId
	return c
}

// DryRun sets the optional parameter "dryRun": Flag to run the request
// in dry-run mode.
func (c *ProductsDeleteCall) DryRun(dryRun bool) *ProductsDeleteCall {
	c.opt_["dryRun"] = dryRun
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProductsDeleteCall) Fields(s ...googleapi.Field) *ProductsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ProductsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["dryRun"]; ok {
		params.Set("dryRun", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/products/{productId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
		"productId":  c.productId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return err
	}
	return nil
	// {
	//   "description": "Deletes a product from your Merchant Center account.",
	//   "httpMethod": "DELETE",
	//   "id": "content.products.delete",
	//   "parameterOrder": [
	//     "merchantId",
	//     "productId"
	//   ],
	//   "parameters": {
	//     "dryRun": {
	//       "description": "Flag to run the request in dry-run mode.",
	//       "location": "query",
	//       "type": "boolean"
	//     },
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "productId": {
	//       "description": "The ID of the product.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/products/{productId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.products.get":

type ProductsGetCall struct {
	s          *Service
	merchantId uint64
	productId  string
	opt_       map[string]interface{}
}

// Get: Retrieves a product from your Merchant Center account.
func (r *ProductsService) Get(merchantId uint64, productId string) *ProductsGetCall {
	c := &ProductsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.productId = productId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProductsGetCall) Fields(s ...googleapi.Field) *ProductsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ProductsGetCall) Do() (*Product, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/products/{productId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
		"productId":  c.productId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Product
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves a product from your Merchant Center account.",
	//   "httpMethod": "GET",
	//   "id": "content.products.get",
	//   "parameterOrder": [
	//     "merchantId",
	//     "productId"
	//   ],
	//   "parameters": {
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "productId": {
	//       "description": "The ID of the product.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/products/{productId}",
	//   "response": {
	//     "$ref": "Product"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.products.insert":

type ProductsInsertCall struct {
	s          *Service
	merchantId uint64
	product    *Product
	opt_       map[string]interface{}
}

// Insert: Uploads a product to your Merchant Center account.
func (r *ProductsService) Insert(merchantId uint64, product *Product) *ProductsInsertCall {
	c := &ProductsInsertCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.product = product
	return c
}

// DryRun sets the optional parameter "dryRun": Flag to run the request
// in dry-run mode.
func (c *ProductsInsertCall) DryRun(dryRun bool) *ProductsInsertCall {
	c.opt_["dryRun"] = dryRun
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProductsInsertCall) Fields(s ...googleapi.Field) *ProductsInsertCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ProductsInsertCall) Do() (*Product, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.product)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["dryRun"]; ok {
		params.Set("dryRun", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/products")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
	})
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *Product
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Uploads a product to your Merchant Center account.",
	//   "httpMethod": "POST",
	//   "id": "content.products.insert",
	//   "parameterOrder": [
	//     "merchantId"
	//   ],
	//   "parameters": {
	//     "dryRun": {
	//       "description": "Flag to run the request in dry-run mode.",
	//       "location": "query",
	//       "type": "boolean"
	//     },
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/products",
	//   "request": {
	//     "$ref": "Product"
	//   },
	//   "response": {
	//     "$ref": "Product"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.products.list":

type ProductsListCall struct {
	s          *Service
	merchantId uint64
	opt_       map[string]interface{}
}

// List: Lists the products in your Merchant Center account.
func (r *ProductsService) List(merchantId uint64) *ProductsListCall {
	c := &ProductsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	return c
}

// MaxResults sets the optional parameter "maxResults": The maximum
// number of products to return in the response, used for paging.
func (c *ProductsListCall) MaxResults(maxResults int64) *ProductsListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": The token returned
// by the previous request.
func (c *ProductsListCall) PageToken(pageToken string) *ProductsListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProductsListCall) Fields(s ...googleapi.Field) *ProductsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ProductsListCall) Do() (*ProductsListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/products")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *ProductsListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists the products in your Merchant Center account.",
	//   "httpMethod": "GET",
	//   "id": "content.products.list",
	//   "parameterOrder": [
	//     "merchantId"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "description": "The maximum number of products to return in the response, used for paging.",
	//       "format": "uint32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "pageToken": {
	//       "description": "The token returned by the previous request.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/products",
	//   "response": {
	//     "$ref": "ProductsListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.productstatuses.custombatch":

type ProductstatusesCustombatchCall struct {
	s                                 *Service
	productstatusescustombatchrequest *ProductstatusesCustomBatchRequest
	opt_                              map[string]interface{}
}

// Custombatch: Gets the statuses of multiple products in a single
// request.
func (r *ProductstatusesService) Custombatch(productstatusescustombatchrequest *ProductstatusesCustomBatchRequest) *ProductstatusesCustombatchCall {
	c := &ProductstatusesCustombatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.productstatusescustombatchrequest = productstatusescustombatchrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProductstatusesCustombatchCall) Fields(s ...googleapi.Field) *ProductstatusesCustombatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ProductstatusesCustombatchCall) Do() (*ProductstatusesCustomBatchResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.productstatusescustombatchrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "productstatuses/batch")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *ProductstatusesCustomBatchResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets the statuses of multiple products in a single request.",
	//   "httpMethod": "POST",
	//   "id": "content.productstatuses.custombatch",
	//   "path": "productstatuses/batch",
	//   "request": {
	//     "$ref": "ProductstatusesCustomBatchRequest"
	//   },
	//   "response": {
	//     "$ref": "ProductstatusesCustomBatchResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.productstatuses.get":

type ProductstatusesGetCall struct {
	s          *Service
	merchantId uint64
	productId  string
	opt_       map[string]interface{}
}

// Get: Gets the status of a product from your Merchant Center account.
func (r *ProductstatusesService) Get(merchantId uint64, productId string) *ProductstatusesGetCall {
	c := &ProductstatusesGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	c.productId = productId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProductstatusesGetCall) Fields(s ...googleapi.Field) *ProductstatusesGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ProductstatusesGetCall) Do() (*ProductStatus, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/productstatuses/{productId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
		"productId":  c.productId,
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *ProductStatus
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets the status of a product from your Merchant Center account.",
	//   "httpMethod": "GET",
	//   "id": "content.productstatuses.get",
	//   "parameterOrder": [
	//     "merchantId",
	//     "productId"
	//   ],
	//   "parameters": {
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "productId": {
	//       "description": "The ID of the product.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/productstatuses/{productId}",
	//   "response": {
	//     "$ref": "ProductStatus"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}

// method id "content.productstatuses.list":

type ProductstatusesListCall struct {
	s          *Service
	merchantId uint64
	opt_       map[string]interface{}
}

// List: Lists the statuses of the products in your Merchant Center
// account.
func (r *ProductstatusesService) List(merchantId uint64) *ProductstatusesListCall {
	c := &ProductstatusesListCall{s: r.s, opt_: make(map[string]interface{})}
	c.merchantId = merchantId
	return c
}

// MaxResults sets the optional parameter "maxResults": The maximum
// number of product statuses to return in the response, used for
// paging.
func (c *ProductstatusesListCall) MaxResults(maxResults int64) *ProductstatusesListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": The token returned
// by the previous request.
func (c *ProductstatusesListCall) PageToken(pageToken string) *ProductstatusesListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ProductstatusesListCall) Fields(s ...googleapi.Field) *ProductstatusesListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ProductstatusesListCall) Do() (*ProductstatusesListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{merchantId}/productstatuses")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"merchantId": strconv.FormatUint(c.merchantId, 10),
	})
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *ProductstatusesListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists the statuses of the products in your Merchant Center account.",
	//   "httpMethod": "GET",
	//   "id": "content.productstatuses.list",
	//   "parameterOrder": [
	//     "merchantId"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "description": "The maximum number of product statuses to return in the response, used for paging.",
	//       "format": "uint32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "merchantId": {
	//       "description": "The ID of the managing account.",
	//       "format": "uint64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "pageToken": {
	//       "description": "The token returned by the previous request.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "{merchantId}/productstatuses",
	//   "response": {
	//     "$ref": "ProductstatusesListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/content"
	//   ]
	// }

}
