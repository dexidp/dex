// Package androidpublisher provides access to the Google Play Android Developer API.
//
// See https://developers.google.com/android-publisher
//
// Usage example:
//
//   import "google.golang.org/api/androidpublisher/v2"
//   ...
//   androidpublisherService, err := androidpublisher.New(oauthHttpClient)
package androidpublisher

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

const apiId = "androidpublisher:v2"
const apiName = "androidpublisher"
const apiVersion = "v2"
const basePath = "https://www.googleapis.com/androidpublisher/v2/applications/"

// OAuth2 scopes used by this API.
const (
	// View and manage your Google Play Android Developer account
	AndroidpublisherScope = "https://www.googleapis.com/auth/androidpublisher"
)

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Edits = NewEditsService(s)
	s.Inappproducts = NewInappproductsService(s)
	s.Purchases = NewPurchasesService(s)
	return s, nil
}

type Service struct {
	client   *http.Client
	BasePath string // API endpoint base URL

	Edits *EditsService

	Inappproducts *InappproductsService

	Purchases *PurchasesService
}

func NewEditsService(s *Service) *EditsService {
	rs := &EditsService{s: s}
	rs.Apklistings = NewEditsApklistingsService(s)
	rs.Apks = NewEditsApksService(s)
	rs.Details = NewEditsDetailsService(s)
	rs.Expansionfiles = NewEditsExpansionfilesService(s)
	rs.Images = NewEditsImagesService(s)
	rs.Listings = NewEditsListingsService(s)
	rs.Testers = NewEditsTestersService(s)
	rs.Tracks = NewEditsTracksService(s)
	return rs
}

type EditsService struct {
	s *Service

	Apklistings *EditsApklistingsService

	Apks *EditsApksService

	Details *EditsDetailsService

	Expansionfiles *EditsExpansionfilesService

	Images *EditsImagesService

	Listings *EditsListingsService

	Testers *EditsTestersService

	Tracks *EditsTracksService
}

func NewEditsApklistingsService(s *Service) *EditsApklistingsService {
	rs := &EditsApklistingsService{s: s}
	return rs
}

type EditsApklistingsService struct {
	s *Service
}

func NewEditsApksService(s *Service) *EditsApksService {
	rs := &EditsApksService{s: s}
	return rs
}

type EditsApksService struct {
	s *Service
}

func NewEditsDetailsService(s *Service) *EditsDetailsService {
	rs := &EditsDetailsService{s: s}
	return rs
}

type EditsDetailsService struct {
	s *Service
}

func NewEditsExpansionfilesService(s *Service) *EditsExpansionfilesService {
	rs := &EditsExpansionfilesService{s: s}
	return rs
}

type EditsExpansionfilesService struct {
	s *Service
}

func NewEditsImagesService(s *Service) *EditsImagesService {
	rs := &EditsImagesService{s: s}
	return rs
}

type EditsImagesService struct {
	s *Service
}

func NewEditsListingsService(s *Service) *EditsListingsService {
	rs := &EditsListingsService{s: s}
	return rs
}

type EditsListingsService struct {
	s *Service
}

func NewEditsTestersService(s *Service) *EditsTestersService {
	rs := &EditsTestersService{s: s}
	return rs
}

type EditsTestersService struct {
	s *Service
}

func NewEditsTracksService(s *Service) *EditsTracksService {
	rs := &EditsTracksService{s: s}
	return rs
}

type EditsTracksService struct {
	s *Service
}

func NewInappproductsService(s *Service) *InappproductsService {
	rs := &InappproductsService{s: s}
	return rs
}

type InappproductsService struct {
	s *Service
}

func NewPurchasesService(s *Service) *PurchasesService {
	rs := &PurchasesService{s: s}
	rs.Products = NewPurchasesProductsService(s)
	rs.Subscriptions = NewPurchasesSubscriptionsService(s)
	return rs
}

type PurchasesService struct {
	s *Service

	Products *PurchasesProductsService

	Subscriptions *PurchasesSubscriptionsService
}

func NewPurchasesProductsService(s *Service) *PurchasesProductsService {
	rs := &PurchasesProductsService{s: s}
	return rs
}

type PurchasesProductsService struct {
	s *Service
}

func NewPurchasesSubscriptionsService(s *Service) *PurchasesSubscriptionsService {
	rs := &PurchasesSubscriptionsService{s: s}
	return rs
}

type PurchasesSubscriptionsService struct {
	s *Service
}

type Apk struct {
	// Binary: Information about the binary payload of this APK.
	Binary *ApkBinary `json:"binary,omitempty"`

	// VersionCode: The version code of the APK, as specified in the APK's
	// manifest file.
	VersionCode int64 `json:"versionCode,omitempty"`
}

type ApkBinary struct {
	// Sha1: A sha1 hash of the APK payload, encoded as a hex string and
	// matching the output of the sha1sum command.
	Sha1 string `json:"sha1,omitempty"`
}

type ApkListing struct {
	// Language: The language code, in BCP 47 format (eg "en-US").
	Language string `json:"language,omitempty"`

	// RecentChanges: Describe what's new in your APK.
	RecentChanges string `json:"recentChanges,omitempty"`
}

type ApkListingsListResponse struct {
	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "androidpublisher#apkListingsListResponse".
	Kind string `json:"kind,omitempty"`

	Listings []*ApkListing `json:"listings,omitempty"`
}

type ApksListResponse struct {
	Apks []*Apk `json:"apks,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "androidpublisher#apksListResponse".
	Kind string `json:"kind,omitempty"`
}

type AppDetails struct {
	// ContactEmail: The user-visible support email for this app.
	ContactEmail string `json:"contactEmail,omitempty"`

	// ContactPhone: The user-visible support telephone number for this app.
	ContactPhone string `json:"contactPhone,omitempty"`

	// ContactWebsite: The user-visible website for this app.
	ContactWebsite string `json:"contactWebsite,omitempty"`

	// DefaultLanguage: Default language code, in BCP 47 format (eg
	// "en-US").
	DefaultLanguage string `json:"defaultLanguage,omitempty"`
}

type AppEdit struct {
	// ExpiryTimeSeconds: The time at which the edit will expire and will be
	// no longer valid for use in any subsequent API calls (encoded as
	// seconds since the Epoch).
	ExpiryTimeSeconds string `json:"expiryTimeSeconds,omitempty"`

	// Id: The ID of the edit that can be used in subsequent API calls.
	Id string `json:"id,omitempty"`
}

type ExpansionFile struct {
	// FileSize: If set this field indicates that this APK has an Expansion
	// File uploaded to it: this APK does not reference another APK's
	// Expansion File. The field's value is the size of the uploaded
	// Expansion File in bytes.
	FileSize int64 `json:"fileSize,omitempty,string"`

	// ReferencesVersion: If set this APK's Expansion File references
	// another APK's Expansion File. The file_size field will not be set.
	ReferencesVersion int64 `json:"referencesVersion,omitempty"`
}

type ExpansionFilesUploadResponse struct {
	ExpansionFile *ExpansionFile `json:"expansionFile,omitempty"`
}

type Image struct {
	// Id: A unique id representing this image.
	Id string `json:"id,omitempty"`

	// Sha1: A sha1 hash of the image that was uploaded.
	Sha1 string `json:"sha1,omitempty"`

	// Url: A URL that will serve a preview of the image.
	Url string `json:"url,omitempty"`
}

type ImagesDeleteAllResponse struct {
	Deleted []*Image `json:"deleted,omitempty"`
}

type ImagesListResponse struct {
	Images []*Image `json:"images,omitempty"`
}

type ImagesUploadResponse struct {
	Image *Image `json:"image,omitempty"`
}

type InAppProduct struct {
	// DefaultLanguage: The default language of the localized data, as
	// defined by BCP 47. e.g. "en-US", "en-GB".
	DefaultLanguage string `json:"defaultLanguage,omitempty"`

	// DefaultPrice: Default price cannot be zero. In-app products can never
	// be free. Default price is always in the developer's Checkout merchant
	// currency.
	DefaultPrice *Price `json:"defaultPrice,omitempty"`

	// Listings: List of localized title and description data.
	Listings map[string]InAppProductListing `json:"listings,omitempty"`

	// PackageName: The package name of the parent app.
	PackageName string `json:"packageName,omitempty"`

	// Prices: Prices per buyer region. None of these prices should be zero.
	// In-app products can never be free.
	Prices map[string]Price `json:"prices,omitempty"`

	// PurchaseType: Purchase type enum value. Unmodifiable after creation.
	PurchaseType string `json:"purchaseType,omitempty"`

	// Season: Definition of a season for a seasonal subscription. Can be
	// defined only for yearly subscriptions.
	Season *Season `json:"season,omitempty"`

	// Sku: The stock-keeping-unit (SKU) of the product, unique within an
	// app.
	Sku string `json:"sku,omitempty"`

	Status string `json:"status,omitempty"`

	// SubscriptionPeriod: The period of the subscription (if any), i.e.
	// period at which payments must happen. Defined as ISO 8601 duration,
	// i.e. "P1M" for 1 month period.
	SubscriptionPeriod string `json:"subscriptionPeriod,omitempty"`

	// TrialPeriod: Trial period, specified in ISO 8601 format. Acceptable
	// values are anything between "P7D" (seven days) and "P999D" (999
	// days). Seasonal subscriptions cannot have a trial period.
	TrialPeriod string `json:"trialPeriod,omitempty"`
}

type InAppProductListing struct {
	Description string `json:"description,omitempty"`

	Title string `json:"title,omitempty"`
}

type InappproductsBatchRequest struct {
	Entrys []*InappproductsBatchRequestEntry `json:"entrys,omitempty"`
}

type InappproductsBatchRequestEntry struct {
	BatchId int64 `json:"batchId,omitempty"`

	Inappproductsinsertrequest *InappproductsInsertRequest `json:"inappproductsinsertrequest,omitempty"`

	Inappproductsupdaterequest *InappproductsUpdateRequest `json:"inappproductsupdaterequest,omitempty"`

	MethodName string `json:"methodName,omitempty"`
}

type InappproductsBatchResponse struct {
	Entrys []*InappproductsBatchResponseEntry `json:"entrys,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "androidpublisher#inappproductsBatchResponse".
	Kind string `json:"kind,omitempty"`
}

type InappproductsBatchResponseEntry struct {
	BatchId int64 `json:"batchId,omitempty"`

	Inappproductsinsertresponse *InappproductsInsertResponse `json:"inappproductsinsertresponse,omitempty"`

	Inappproductsupdateresponse *InappproductsUpdateResponse `json:"inappproductsupdateresponse,omitempty"`
}

type InappproductsInsertRequest struct {
	Inappproduct *InAppProduct `json:"inappproduct,omitempty"`
}

type InappproductsInsertResponse struct {
	Inappproduct *InAppProduct `json:"inappproduct,omitempty"`
}

type InappproductsListResponse struct {
	Inappproduct []*InAppProduct `json:"inappproduct,omitempty"`

	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "androidpublisher#inappproductsListResponse".
	Kind string `json:"kind,omitempty"`

	PageInfo *PageInfo `json:"pageInfo,omitempty"`

	TokenPagination *TokenPagination `json:"tokenPagination,omitempty"`
}

type InappproductsUpdateRequest struct {
	Inappproduct *InAppProduct `json:"inappproduct,omitempty"`
}

type InappproductsUpdateResponse struct {
	Inappproduct *InAppProduct `json:"inappproduct,omitempty"`
}

type Listing struct {
	// FullDescription: Full description of the app; this may be up to 4000
	// characters in length.
	FullDescription string `json:"fullDescription,omitempty"`

	// Language: Language localization code (for example, "de-AT" for
	// Austrian German).
	Language string `json:"language,omitempty"`

	// ShortDescription: Short description of the app (previously known as
	// promo text); this may be up to 80 characters in length.
	ShortDescription string `json:"shortDescription,omitempty"`

	// Title: App's localized title.
	Title string `json:"title,omitempty"`

	// Video: URL of a promotional YouTube video for the app.
	Video string `json:"video,omitempty"`
}

type ListingsListResponse struct {
	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "androidpublisher#listingsListResponse".
	Kind string `json:"kind,omitempty"`

	Listings []*Listing `json:"listings,omitempty"`
}

type MonthDay struct {
	// Day: Day of a month, value in [1, 31] range. Valid range depends on
	// the specified month.
	Day int64 `json:"day,omitempty"`

	// Month: Month of a year. e.g. 1 = JAN, 2 = FEB etc.
	Month int64 `json:"month,omitempty"`
}

type PageInfo struct {
	ResultPerPage int64 `json:"resultPerPage,omitempty"`

	StartIndex int64 `json:"startIndex,omitempty"`

	TotalResults int64 `json:"totalResults,omitempty"`
}

type Price struct {
	// Currency: 3 letter Currency code, as defined by ISO 4217.
	Currency string `json:"currency,omitempty"`

	// PriceMicros: The price in millionths of the currency base unit
	// represented as a string.
	PriceMicros string `json:"priceMicros,omitempty"`
}

type ProductPurchase struct {
	// ConsumptionState: The consumption state of the inapp product.
	// Possible values are:
	// - Yet to be consumed
	// - Consumed
	ConsumptionState int64 `json:"consumptionState,omitempty"`

	// DeveloperPayload: A developer-specified string that contains
	// supplemental information about an order.
	DeveloperPayload string `json:"developerPayload,omitempty"`

	// Kind: This kind represents an inappPurchase object in the
	// androidpublisher service.
	Kind string `json:"kind,omitempty"`

	// PurchaseState: The purchase state of the order. Possible values are:
	//
	// - Purchased
	// - Cancelled
	PurchaseState int64 `json:"purchaseState,omitempty"`

	// PurchaseTimeMillis: The time the product was purchased, in
	// milliseconds since the epoch (Jan 1, 1970).
	PurchaseTimeMillis int64 `json:"purchaseTimeMillis,omitempty,string"`
}

type Season struct {
	// End: Inclusive end date of the recurrence period.
	End *MonthDay `json:"end,omitempty"`

	// Start: Inclusive start date of the recurrence period.
	Start *MonthDay `json:"start,omitempty"`
}

type SubscriptionDeferralInfo struct {
	// DesiredExpiryTimeMillis: The desired next expiry time for the
	// subscription in milliseconds since Epoch. The given time must be
	// after the current expiry time for the subscription.
	DesiredExpiryTimeMillis int64 `json:"desiredExpiryTimeMillis,omitempty,string"`

	// ExpectedExpiryTimeMillis: The expected expiry time for the
	// subscription. If the current expiry time for the subscription is not
	// the value specified here, the deferral will not occur.
	ExpectedExpiryTimeMillis int64 `json:"expectedExpiryTimeMillis,omitempty,string"`
}

type SubscriptionPurchase struct {
	// AutoRenewing: Whether the subscription will automatically be renewed
	// when it reaches its current expiry time.
	AutoRenewing bool `json:"autoRenewing,omitempty"`

	// ExpiryTimeMillis: Time at which the subscription will expire, in
	// milliseconds since Epoch.
	ExpiryTimeMillis int64 `json:"expiryTimeMillis,omitempty,string"`

	// Kind: This kind represents a subscriptionPurchase object in the
	// androidpublisher service.
	Kind string `json:"kind,omitempty"`

	// StartTimeMillis: Time at which the subscription was granted, in
	// milliseconds since Epoch.
	StartTimeMillis int64 `json:"startTimeMillis,omitempty,string"`
}

type SubscriptionPurchasesDeferRequest struct {
	// DeferralInfo: The information about the new desired expiry time for
	// the subscription.
	DeferralInfo *SubscriptionDeferralInfo `json:"deferralInfo,omitempty"`
}

type SubscriptionPurchasesDeferResponse struct {
	// NewExpiryTimeMillis: The new expiry time for the subscription in
	// milliseconds since the Epoch.
	NewExpiryTimeMillis int64 `json:"newExpiryTimeMillis,omitempty,string"`
}

type Testers struct {
	GoogleGroups []string `json:"googleGroups,omitempty"`

	GooglePlusCommunities []string `json:"googlePlusCommunities,omitempty"`
}

type TokenPagination struct {
	NextPageToken string `json:"nextPageToken,omitempty"`

	PreviousPageToken string `json:"previousPageToken,omitempty"`
}

type Track struct {
	Track string `json:"track,omitempty"`

	UserFraction float64 `json:"userFraction,omitempty"`

	VersionCodes []int64 `json:"versionCodes,omitempty"`
}

type TracksListResponse struct {
	// Kind: Identifies what kind of resource this is. Value: the fixed
	// string "androidpublisher#tracksListResponse".
	Kind string `json:"kind,omitempty"`

	Tracks []*Track `json:"tracks,omitempty"`
}

// method id "androidpublisher.edits.commit":

type EditsCommitCall struct {
	s             *Service
	packageNameid string
	editId        string
	opt_          map[string]interface{}
}

// Commit: Commits/applies the changes made in this edit back to the
// app.
func (r *EditsService) Commit(packageNameid string, editId string) *EditsCommitCall {
	c := &EditsCommitCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsCommitCall) Fields(s ...googleapi.Field) *EditsCommitCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsCommitCall) Do() (*AppEdit, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}:commit")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
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
	var ret *AppEdit
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Commits/applies the changes made in this edit back to the app.",
	//   "httpMethod": "POST",
	//   "id": "androidpublisher.edits.commit",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}:commit",
	//   "response": {
	//     "$ref": "AppEdit"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.delete":

type EditsDeleteCall struct {
	s             *Service
	packageNameid string
	editId        string
	opt_          map[string]interface{}
}

// Delete: Deletes an edit for an app. Creating a new edit will
// automatically delete any of your previous edits so this method need
// only be called if you want to preemptively abandon an edit.
func (r *EditsService) Delete(packageNameid string, editId string) *EditsDeleteCall {
	c := &EditsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsDeleteCall) Fields(s ...googleapi.Field) *EditsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
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
	//   "description": "Deletes an edit for an app. Creating a new edit will automatically delete any of your previous edits so this method need only be called if you want to preemptively abandon an edit.",
	//   "httpMethod": "DELETE",
	//   "id": "androidpublisher.edits.delete",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.get":

type EditsGetCall struct {
	s             *Service
	packageNameid string
	editId        string
	opt_          map[string]interface{}
}

// Get: Returns information about the edit specified. Calls will fail if
// the edit is no long active (e.g. has been deleted, superseded or
// expired).
func (r *EditsService) Get(packageNameid string, editId string) *EditsGetCall {
	c := &EditsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsGetCall) Fields(s ...googleapi.Field) *EditsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsGetCall) Do() (*AppEdit, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
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
	var ret *AppEdit
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Returns information about the edit specified. Calls will fail if the edit is no long active (e.g. has been deleted, superseded or expired).",
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.edits.get",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}",
	//   "response": {
	//     "$ref": "AppEdit"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.insert":

type EditsInsertCall struct {
	s             *Service
	packageNameid string
	appedit       *AppEdit
	opt_          map[string]interface{}
}

// Insert: Creates a new edit for an app, populated with the app's
// current state.
func (r *EditsService) Insert(packageNameid string, appedit *AppEdit) *EditsInsertCall {
	c := &EditsInsertCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.appedit = appedit
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsInsertCall) Fields(s ...googleapi.Field) *EditsInsertCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsInsertCall) Do() (*AppEdit, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.appedit)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
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
	var ret *AppEdit
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a new edit for an app, populated with the app's current state.",
	//   "httpMethod": "POST",
	//   "id": "androidpublisher.edits.insert",
	//   "parameterOrder": [
	//     "packageName"
	//   ],
	//   "parameters": {
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits",
	//   "request": {
	//     "$ref": "AppEdit"
	//   },
	//   "response": {
	//     "$ref": "AppEdit"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.validate":

type EditsValidateCall struct {
	s             *Service
	packageNameid string
	editId        string
	opt_          map[string]interface{}
}

// Validate: Checks that the edit can be successfully committed. The
// edit's changes are not applied to the live app.
func (r *EditsService) Validate(packageNameid string, editId string) *EditsValidateCall {
	c := &EditsValidateCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsValidateCall) Fields(s ...googleapi.Field) *EditsValidateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsValidateCall) Do() (*AppEdit, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}:validate")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
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
	var ret *AppEdit
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Checks that the edit can be successfully committed. The edit's changes are not applied to the live app.",
	//   "httpMethod": "POST",
	//   "id": "androidpublisher.edits.validate",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}:validate",
	//   "response": {
	//     "$ref": "AppEdit"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.apklistings.delete":

type EditsApklistingsDeleteCall struct {
	s              *Service
	packageNameid  string
	editId         string
	apkVersionCode int64
	language       string
	opt_           map[string]interface{}
}

// Delete: Deletes the APK-specific localized listing for a specified
// APK and language code.
func (r *EditsApklistingsService) Delete(packageNameid string, editId string, apkVersionCode int64, language string) *EditsApklistingsDeleteCall {
	c := &EditsApklistingsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.apkVersionCode = apkVersionCode
	c.language = language
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsApklistingsDeleteCall) Fields(s ...googleapi.Field) *EditsApklistingsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsApklistingsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/apks/{apkVersionCode}/listings/{language}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":    c.packageNameid,
		"editId":         c.editId,
		"apkVersionCode": strconv.FormatInt(c.apkVersionCode, 10),
		"language":       c.language,
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
	//   "description": "Deletes the APK-specific localized listing for a specified APK and language code.",
	//   "httpMethod": "DELETE",
	//   "id": "androidpublisher.edits.apklistings.delete",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "apkVersionCode",
	//     "language"
	//   ],
	//   "parameters": {
	//     "apkVersionCode": {
	//       "description": "The APK version code whose APK-specific listings should be read or modified.",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     },
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "language": {
	//       "description": "The language code (a BCP-47 language tag) of the APK-specific localized listing to read or modify. For example, to select Austrian German, pass \"de-AT\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/apks/{apkVersionCode}/listings/{language}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.apklistings.deleteall":

type EditsApklistingsDeleteallCall struct {
	s              *Service
	packageNameid  string
	editId         string
	apkVersionCode int64
	opt_           map[string]interface{}
}

// Deleteall: Deletes all the APK-specific localized listings for a
// specified APK.
func (r *EditsApklistingsService) Deleteall(packageNameid string, editId string, apkVersionCode int64) *EditsApklistingsDeleteallCall {
	c := &EditsApklistingsDeleteallCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.apkVersionCode = apkVersionCode
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsApklistingsDeleteallCall) Fields(s ...googleapi.Field) *EditsApklistingsDeleteallCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsApklistingsDeleteallCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/apks/{apkVersionCode}/listings")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":    c.packageNameid,
		"editId":         c.editId,
		"apkVersionCode": strconv.FormatInt(c.apkVersionCode, 10),
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
	//   "description": "Deletes all the APK-specific localized listings for a specified APK.",
	//   "httpMethod": "DELETE",
	//   "id": "androidpublisher.edits.apklistings.deleteall",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "apkVersionCode"
	//   ],
	//   "parameters": {
	//     "apkVersionCode": {
	//       "description": "The APK version code whose APK-specific listings should be read or modified.",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     },
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/apks/{apkVersionCode}/listings",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.apklistings.get":

type EditsApklistingsGetCall struct {
	s              *Service
	packageNameid  string
	editId         string
	apkVersionCode int64
	language       string
	opt_           map[string]interface{}
}

// Get: Fetches the APK-specific localized listing for a specified APK
// and language code.
func (r *EditsApklistingsService) Get(packageNameid string, editId string, apkVersionCode int64, language string) *EditsApklistingsGetCall {
	c := &EditsApklistingsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.apkVersionCode = apkVersionCode
	c.language = language
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsApklistingsGetCall) Fields(s ...googleapi.Field) *EditsApklistingsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsApklistingsGetCall) Do() (*ApkListing, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/apks/{apkVersionCode}/listings/{language}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":    c.packageNameid,
		"editId":         c.editId,
		"apkVersionCode": strconv.FormatInt(c.apkVersionCode, 10),
		"language":       c.language,
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
	var ret *ApkListing
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Fetches the APK-specific localized listing for a specified APK and language code.",
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.edits.apklistings.get",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "apkVersionCode",
	//     "language"
	//   ],
	//   "parameters": {
	//     "apkVersionCode": {
	//       "description": "The APK version code whose APK-specific listings should be read or modified.",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     },
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "language": {
	//       "description": "The language code (a BCP-47 language tag) of the APK-specific localized listing to read or modify. For example, to select Austrian German, pass \"de-AT\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/apks/{apkVersionCode}/listings/{language}",
	//   "response": {
	//     "$ref": "ApkListing"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.apklistings.list":

type EditsApklistingsListCall struct {
	s              *Service
	packageNameid  string
	editId         string
	apkVersionCode int64
	opt_           map[string]interface{}
}

// List: Lists all the APK-specific localized listings for a specified
// APK.
func (r *EditsApklistingsService) List(packageNameid string, editId string, apkVersionCode int64) *EditsApklistingsListCall {
	c := &EditsApklistingsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.apkVersionCode = apkVersionCode
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsApklistingsListCall) Fields(s ...googleapi.Field) *EditsApklistingsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsApklistingsListCall) Do() (*ApkListingsListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/apks/{apkVersionCode}/listings")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":    c.packageNameid,
		"editId":         c.editId,
		"apkVersionCode": strconv.FormatInt(c.apkVersionCode, 10),
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
	var ret *ApkListingsListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all the APK-specific localized listings for a specified APK.",
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.edits.apklistings.list",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "apkVersionCode"
	//   ],
	//   "parameters": {
	//     "apkVersionCode": {
	//       "description": "The APK version code whose APK-specific listings should be read or modified.",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     },
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/apks/{apkVersionCode}/listings",
	//   "response": {
	//     "$ref": "ApkListingsListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.apklistings.patch":

type EditsApklistingsPatchCall struct {
	s              *Service
	packageNameid  string
	editId         string
	apkVersionCode int64
	language       string
	apklisting     *ApkListing
	opt_           map[string]interface{}
}

// Patch: Updates or creates the APK-specific localized listing for a
// specified APK and language code. This method supports patch
// semantics.
func (r *EditsApklistingsService) Patch(packageNameid string, editId string, apkVersionCode int64, language string, apklisting *ApkListing) *EditsApklistingsPatchCall {
	c := &EditsApklistingsPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.apkVersionCode = apkVersionCode
	c.language = language
	c.apklisting = apklisting
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsApklistingsPatchCall) Fields(s ...googleapi.Field) *EditsApklistingsPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsApklistingsPatchCall) Do() (*ApkListing, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.apklisting)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/apks/{apkVersionCode}/listings/{language}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":    c.packageNameid,
		"editId":         c.editId,
		"apkVersionCode": strconv.FormatInt(c.apkVersionCode, 10),
		"language":       c.language,
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
	var ret *ApkListing
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates or creates the APK-specific localized listing for a specified APK and language code. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "androidpublisher.edits.apklistings.patch",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "apkVersionCode",
	//     "language"
	//   ],
	//   "parameters": {
	//     "apkVersionCode": {
	//       "description": "The APK version code whose APK-specific listings should be read or modified.",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     },
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "language": {
	//       "description": "The language code (a BCP-47 language tag) of the APK-specific localized listing to read or modify. For example, to select Austrian German, pass \"de-AT\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/apks/{apkVersionCode}/listings/{language}",
	//   "request": {
	//     "$ref": "ApkListing"
	//   },
	//   "response": {
	//     "$ref": "ApkListing"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.apklistings.update":

type EditsApklistingsUpdateCall struct {
	s              *Service
	packageNameid  string
	editId         string
	apkVersionCode int64
	language       string
	apklisting     *ApkListing
	opt_           map[string]interface{}
}

// Update: Updates or creates the APK-specific localized listing for a
// specified APK and language code.
func (r *EditsApklistingsService) Update(packageNameid string, editId string, apkVersionCode int64, language string, apklisting *ApkListing) *EditsApklistingsUpdateCall {
	c := &EditsApklistingsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.apkVersionCode = apkVersionCode
	c.language = language
	c.apklisting = apklisting
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsApklistingsUpdateCall) Fields(s ...googleapi.Field) *EditsApklistingsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsApklistingsUpdateCall) Do() (*ApkListing, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.apklisting)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/apks/{apkVersionCode}/listings/{language}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":    c.packageNameid,
		"editId":         c.editId,
		"apkVersionCode": strconv.FormatInt(c.apkVersionCode, 10),
		"language":       c.language,
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
	var ret *ApkListing
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates or creates the APK-specific localized listing for a specified APK and language code.",
	//   "httpMethod": "PUT",
	//   "id": "androidpublisher.edits.apklistings.update",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "apkVersionCode",
	//     "language"
	//   ],
	//   "parameters": {
	//     "apkVersionCode": {
	//       "description": "The APK version code whose APK-specific listings should be read or modified.",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     },
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "language": {
	//       "description": "The language code (a BCP-47 language tag) of the APK-specific localized listing to read or modify. For example, to select Austrian German, pass \"de-AT\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/apks/{apkVersionCode}/listings/{language}",
	//   "request": {
	//     "$ref": "ApkListing"
	//   },
	//   "response": {
	//     "$ref": "ApkListing"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.apks.list":

type EditsApksListCall struct {
	s             *Service
	packageNameid string
	editId        string
	opt_          map[string]interface{}
}

// List:
func (r *EditsApksService) List(packageNameid string, editId string) *EditsApksListCall {
	c := &EditsApksListCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsApksListCall) Fields(s ...googleapi.Field) *EditsApksListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsApksListCall) Do() (*ApksListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/apks")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
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
	var ret *ApksListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.edits.apks.list",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/apks",
	//   "response": {
	//     "$ref": "ApksListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.apks.upload":

type EditsApksUploadCall struct {
	s             *Service
	packageNameid string
	editId        string
	opt_          map[string]interface{}
	media_        io.Reader
}

// Upload:
func (r *EditsApksService) Upload(packageNameid string, editId string) *EditsApksUploadCall {
	c := &EditsApksUploadCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	return c
}
func (c *EditsApksUploadCall) Media(r io.Reader) *EditsApksUploadCall {
	c.media_ = r
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsApksUploadCall) Fields(s ...googleapi.Field) *EditsApksUploadCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsApksUploadCall) Do() (*Apk, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/apks")
	if c.media_ != nil {
		urls = strings.Replace(urls, "https://www.googleapis.com/", "https://www.googleapis.com/upload/", 1)
		params.Set("uploadType", "multipart")
	}
	urls += "?" + params.Encode()
	body = new(bytes.Buffer)
	ctype := "application/json"
	contentLength_, hasMedia_ := googleapi.ConditionallyIncludeMedia(c.media_, &body, &ctype)
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
	})
	if hasMedia_ {
		req.ContentLength = contentLength_
	}
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
	var ret *Apk
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "httpMethod": "POST",
	//   "id": "androidpublisher.edits.apks.upload",
	//   "mediaUpload": {
	//     "accept": [
	//       "application/octet-stream",
	//       "application/vnd.android.package-archive"
	//     ],
	//     "maxSize": "50MB",
	//     "protocols": {
	//       "resumable": {
	//         "multipart": true,
	//         "path": "/resumable/upload/androidpublisher/v2/applications/{packageName}/edits/{editId}/apks"
	//       },
	//       "simple": {
	//         "multipart": true,
	//         "path": "/upload/androidpublisher/v2/applications/{packageName}/edits/{editId}/apks"
	//       }
	//     }
	//   },
	//   "parameterOrder": [
	//     "packageName",
	//     "editId"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/apks",
	//   "response": {
	//     "$ref": "Apk"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ],
	//   "supportsMediaUpload": true
	// }

}

// method id "androidpublisher.edits.details.get":

type EditsDetailsGetCall struct {
	s             *Service
	packageNameid string
	editId        string
	opt_          map[string]interface{}
}

// Get: Fetches app details for this edit. This includes the default
// language and developer support contact information.
func (r *EditsDetailsService) Get(packageNameid string, editId string) *EditsDetailsGetCall {
	c := &EditsDetailsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsDetailsGetCall) Fields(s ...googleapi.Field) *EditsDetailsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsDetailsGetCall) Do() (*AppDetails, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/details")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
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
	var ret *AppDetails
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Fetches app details for this edit. This includes the default language and developer support contact information.",
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.edits.details.get",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/details",
	//   "response": {
	//     "$ref": "AppDetails"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.details.patch":

type EditsDetailsPatchCall struct {
	s             *Service
	packageNameid string
	editId        string
	appdetails    *AppDetails
	opt_          map[string]interface{}
}

// Patch: Updates app details for this edit. This method supports patch
// semantics.
func (r *EditsDetailsService) Patch(packageNameid string, editId string, appdetails *AppDetails) *EditsDetailsPatchCall {
	c := &EditsDetailsPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.appdetails = appdetails
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsDetailsPatchCall) Fields(s ...googleapi.Field) *EditsDetailsPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsDetailsPatchCall) Do() (*AppDetails, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.appdetails)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/details")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
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
	var ret *AppDetails
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates app details for this edit. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "androidpublisher.edits.details.patch",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/details",
	//   "request": {
	//     "$ref": "AppDetails"
	//   },
	//   "response": {
	//     "$ref": "AppDetails"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.details.update":

type EditsDetailsUpdateCall struct {
	s             *Service
	packageNameid string
	editId        string
	appdetails    *AppDetails
	opt_          map[string]interface{}
}

// Update: Updates app details for this edit.
func (r *EditsDetailsService) Update(packageNameid string, editId string, appdetails *AppDetails) *EditsDetailsUpdateCall {
	c := &EditsDetailsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.appdetails = appdetails
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsDetailsUpdateCall) Fields(s ...googleapi.Field) *EditsDetailsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsDetailsUpdateCall) Do() (*AppDetails, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.appdetails)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/details")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
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
	var ret *AppDetails
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates app details for this edit.",
	//   "httpMethod": "PUT",
	//   "id": "androidpublisher.edits.details.update",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/details",
	//   "request": {
	//     "$ref": "AppDetails"
	//   },
	//   "response": {
	//     "$ref": "AppDetails"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.expansionfiles.get":

type EditsExpansionfilesGetCall struct {
	s                 *Service
	packageNameid     string
	editId            string
	apkVersionCode    int64
	expansionFileType string
	opt_              map[string]interface{}
}

// Get: Fetches the Expansion File configuration for the APK specified.
func (r *EditsExpansionfilesService) Get(packageNameid string, editId string, apkVersionCode int64, expansionFileType string) *EditsExpansionfilesGetCall {
	c := &EditsExpansionfilesGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.apkVersionCode = apkVersionCode
	c.expansionFileType = expansionFileType
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsExpansionfilesGetCall) Fields(s ...googleapi.Field) *EditsExpansionfilesGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsExpansionfilesGetCall) Do() (*ExpansionFile, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/apks/{apkVersionCode}/expansionFiles/{expansionFileType}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":       c.packageNameid,
		"editId":            c.editId,
		"apkVersionCode":    strconv.FormatInt(c.apkVersionCode, 10),
		"expansionFileType": c.expansionFileType,
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
	var ret *ExpansionFile
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Fetches the Expansion File configuration for the APK specified.",
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.edits.expansionfiles.get",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "apkVersionCode",
	//     "expansionFileType"
	//   ],
	//   "parameters": {
	//     "apkVersionCode": {
	//       "description": "The version code of the APK whose Expansion File configuration is being read or modified.",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     },
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "expansionFileType": {
	//       "enum": [
	//         "main",
	//         "patch"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         ""
	//       ],
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/apks/{apkVersionCode}/expansionFiles/{expansionFileType}",
	//   "response": {
	//     "$ref": "ExpansionFile"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.expansionfiles.patch":

type EditsExpansionfilesPatchCall struct {
	s                 *Service
	packageNameid     string
	editId            string
	apkVersionCode    int64
	expansionFileType string
	expansionfile     *ExpansionFile
	opt_              map[string]interface{}
}

// Patch: Updates the APK's Expansion File configuration to reference
// another APK's Expansion Files. To add a new Expansion File use the
// Upload method. This method supports patch semantics.
func (r *EditsExpansionfilesService) Patch(packageNameid string, editId string, apkVersionCode int64, expansionFileType string, expansionfile *ExpansionFile) *EditsExpansionfilesPatchCall {
	c := &EditsExpansionfilesPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.apkVersionCode = apkVersionCode
	c.expansionFileType = expansionFileType
	c.expansionfile = expansionfile
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsExpansionfilesPatchCall) Fields(s ...googleapi.Field) *EditsExpansionfilesPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsExpansionfilesPatchCall) Do() (*ExpansionFile, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.expansionfile)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/apks/{apkVersionCode}/expansionFiles/{expansionFileType}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":       c.packageNameid,
		"editId":            c.editId,
		"apkVersionCode":    strconv.FormatInt(c.apkVersionCode, 10),
		"expansionFileType": c.expansionFileType,
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
	var ret *ExpansionFile
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates the APK's Expansion File configuration to reference another APK's Expansion Files. To add a new Expansion File use the Upload method. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "androidpublisher.edits.expansionfiles.patch",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "apkVersionCode",
	//     "expansionFileType"
	//   ],
	//   "parameters": {
	//     "apkVersionCode": {
	//       "description": "The version code of the APK whose Expansion File configuration is being read or modified.",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     },
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "expansionFileType": {
	//       "enum": [
	//         "main",
	//         "patch"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         ""
	//       ],
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/apks/{apkVersionCode}/expansionFiles/{expansionFileType}",
	//   "request": {
	//     "$ref": "ExpansionFile"
	//   },
	//   "response": {
	//     "$ref": "ExpansionFile"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.expansionfiles.update":

type EditsExpansionfilesUpdateCall struct {
	s                 *Service
	packageNameid     string
	editId            string
	apkVersionCode    int64
	expansionFileType string
	expansionfile     *ExpansionFile
	opt_              map[string]interface{}
}

// Update: Updates the APK's Expansion File configuration to reference
// another APK's Expansion Files. To add a new Expansion File use the
// Upload method.
func (r *EditsExpansionfilesService) Update(packageNameid string, editId string, apkVersionCode int64, expansionFileType string, expansionfile *ExpansionFile) *EditsExpansionfilesUpdateCall {
	c := &EditsExpansionfilesUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.apkVersionCode = apkVersionCode
	c.expansionFileType = expansionFileType
	c.expansionfile = expansionfile
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsExpansionfilesUpdateCall) Fields(s ...googleapi.Field) *EditsExpansionfilesUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsExpansionfilesUpdateCall) Do() (*ExpansionFile, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.expansionfile)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/apks/{apkVersionCode}/expansionFiles/{expansionFileType}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":       c.packageNameid,
		"editId":            c.editId,
		"apkVersionCode":    strconv.FormatInt(c.apkVersionCode, 10),
		"expansionFileType": c.expansionFileType,
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
	var ret *ExpansionFile
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates the APK's Expansion File configuration to reference another APK's Expansion Files. To add a new Expansion File use the Upload method.",
	//   "httpMethod": "PUT",
	//   "id": "androidpublisher.edits.expansionfiles.update",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "apkVersionCode",
	//     "expansionFileType"
	//   ],
	//   "parameters": {
	//     "apkVersionCode": {
	//       "description": "The version code of the APK whose Expansion File configuration is being read or modified.",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     },
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "expansionFileType": {
	//       "enum": [
	//         "main",
	//         "patch"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         ""
	//       ],
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/apks/{apkVersionCode}/expansionFiles/{expansionFileType}",
	//   "request": {
	//     "$ref": "ExpansionFile"
	//   },
	//   "response": {
	//     "$ref": "ExpansionFile"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.expansionfiles.upload":

type EditsExpansionfilesUploadCall struct {
	s                 *Service
	packageNameid     string
	editId            string
	apkVersionCode    int64
	expansionFileType string
	opt_              map[string]interface{}
	media_            io.Reader
}

// Upload: Uploads and attaches a new Expansion File to the APK
// specified.
func (r *EditsExpansionfilesService) Upload(packageNameid string, editId string, apkVersionCode int64, expansionFileType string) *EditsExpansionfilesUploadCall {
	c := &EditsExpansionfilesUploadCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.apkVersionCode = apkVersionCode
	c.expansionFileType = expansionFileType
	return c
}
func (c *EditsExpansionfilesUploadCall) Media(r io.Reader) *EditsExpansionfilesUploadCall {
	c.media_ = r
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsExpansionfilesUploadCall) Fields(s ...googleapi.Field) *EditsExpansionfilesUploadCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsExpansionfilesUploadCall) Do() (*ExpansionFilesUploadResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/apks/{apkVersionCode}/expansionFiles/{expansionFileType}")
	if c.media_ != nil {
		urls = strings.Replace(urls, "https://www.googleapis.com/", "https://www.googleapis.com/upload/", 1)
		params.Set("uploadType", "multipart")
	}
	urls += "?" + params.Encode()
	body = new(bytes.Buffer)
	ctype := "application/json"
	contentLength_, hasMedia_ := googleapi.ConditionallyIncludeMedia(c.media_, &body, &ctype)
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":       c.packageNameid,
		"editId":            c.editId,
		"apkVersionCode":    strconv.FormatInt(c.apkVersionCode, 10),
		"expansionFileType": c.expansionFileType,
	})
	if hasMedia_ {
		req.ContentLength = contentLength_
	}
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
	var ret *ExpansionFilesUploadResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Uploads and attaches a new Expansion File to the APK specified.",
	//   "httpMethod": "POST",
	//   "id": "androidpublisher.edits.expansionfiles.upload",
	//   "mediaUpload": {
	//     "accept": [
	//       "application/octet-stream"
	//     ],
	//     "maxSize": "2048MB",
	//     "protocols": {
	//       "resumable": {
	//         "multipart": true,
	//         "path": "/resumable/upload/androidpublisher/v2/applications/{packageName}/edits/{editId}/apks/{apkVersionCode}/expansionFiles/{expansionFileType}"
	//       },
	//       "simple": {
	//         "multipart": true,
	//         "path": "/upload/androidpublisher/v2/applications/{packageName}/edits/{editId}/apks/{apkVersionCode}/expansionFiles/{expansionFileType}"
	//       }
	//     }
	//   },
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "apkVersionCode",
	//     "expansionFileType"
	//   ],
	//   "parameters": {
	//     "apkVersionCode": {
	//       "description": "The version code of the APK whose Expansion File configuration is being read or modified.",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     },
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "expansionFileType": {
	//       "enum": [
	//         "main",
	//         "patch"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         ""
	//       ],
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/apks/{apkVersionCode}/expansionFiles/{expansionFileType}",
	//   "response": {
	//     "$ref": "ExpansionFilesUploadResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ],
	//   "supportsMediaUpload": true
	// }

}

// method id "androidpublisher.edits.images.delete":

type EditsImagesDeleteCall struct {
	s             *Service
	packageNameid string
	editId        string
	language      string
	imageType     string
	imageId       string
	opt_          map[string]interface{}
}

// Delete: Deletes the image (specified by id) from the edit.
func (r *EditsImagesService) Delete(packageNameid string, editId string, language string, imageType string, imageId string) *EditsImagesDeleteCall {
	c := &EditsImagesDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.language = language
	c.imageType = imageType
	c.imageId = imageId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsImagesDeleteCall) Fields(s ...googleapi.Field) *EditsImagesDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsImagesDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/listings/{language}/{imageType}/{imageId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
		"language":    c.language,
		"imageType":   c.imageType,
		"imageId":     c.imageId,
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
	//   "description": "Deletes the image (specified by id) from the edit.",
	//   "httpMethod": "DELETE",
	//   "id": "androidpublisher.edits.images.delete",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "language",
	//     "imageType",
	//     "imageId"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "imageId": {
	//       "description": "Unique identifier an image within the set of images attached to this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "imageType": {
	//       "enum": [
	//         "featureGraphic",
	//         "icon",
	//         "phoneScreenshots",
	//         "promoGraphic",
	//         "sevenInchScreenshots",
	//         "tenInchScreenshots"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         "",
	//         "",
	//         "",
	//         "",
	//         ""
	//       ],
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "language": {
	//       "description": "The language code (a BCP-47 language tag) of the localized listing whose images are to read or modified. For example, to select Austrian German, pass \"de-AT\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/listings/{language}/{imageType}/{imageId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.images.deleteall":

type EditsImagesDeleteallCall struct {
	s             *Service
	packageNameid string
	editId        string
	language      string
	imageType     string
	opt_          map[string]interface{}
}

// Deleteall: Deletes all images for the specified language and image
// type.
func (r *EditsImagesService) Deleteall(packageNameid string, editId string, language string, imageType string) *EditsImagesDeleteallCall {
	c := &EditsImagesDeleteallCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.language = language
	c.imageType = imageType
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsImagesDeleteallCall) Fields(s ...googleapi.Field) *EditsImagesDeleteallCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsImagesDeleteallCall) Do() (*ImagesDeleteAllResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/listings/{language}/{imageType}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
		"language":    c.language,
		"imageType":   c.imageType,
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
	var ret *ImagesDeleteAllResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Deletes all images for the specified language and image type.",
	//   "httpMethod": "DELETE",
	//   "id": "androidpublisher.edits.images.deleteall",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "language",
	//     "imageType"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "imageType": {
	//       "enum": [
	//         "featureGraphic",
	//         "icon",
	//         "phoneScreenshots",
	//         "promoGraphic",
	//         "sevenInchScreenshots",
	//         "tenInchScreenshots"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         "",
	//         "",
	//         "",
	//         "",
	//         ""
	//       ],
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "language": {
	//       "description": "The language code (a BCP-47 language tag) of the localized listing whose images are to read or modified. For example, to select Austrian German, pass \"de-AT\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/listings/{language}/{imageType}",
	//   "response": {
	//     "$ref": "ImagesDeleteAllResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.images.list":

type EditsImagesListCall struct {
	s             *Service
	packageNameid string
	editId        string
	language      string
	imageType     string
	opt_          map[string]interface{}
}

// List: Lists all images for the specified language and image type.
func (r *EditsImagesService) List(packageNameid string, editId string, language string, imageType string) *EditsImagesListCall {
	c := &EditsImagesListCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.language = language
	c.imageType = imageType
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsImagesListCall) Fields(s ...googleapi.Field) *EditsImagesListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsImagesListCall) Do() (*ImagesListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/listings/{language}/{imageType}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
		"language":    c.language,
		"imageType":   c.imageType,
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
	var ret *ImagesListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all images for the specified language and image type.",
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.edits.images.list",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "language",
	//     "imageType"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "imageType": {
	//       "enum": [
	//         "featureGraphic",
	//         "icon",
	//         "phoneScreenshots",
	//         "promoGraphic",
	//         "sevenInchScreenshots",
	//         "tenInchScreenshots"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         "",
	//         "",
	//         "",
	//         "",
	//         ""
	//       ],
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "language": {
	//       "description": "The language code (a BCP-47 language tag) of the localized listing whose images are to read or modified. For example, to select Austrian German, pass \"de-AT\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/listings/{language}/{imageType}",
	//   "response": {
	//     "$ref": "ImagesListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.images.upload":

type EditsImagesUploadCall struct {
	s             *Service
	packageNameid string
	editId        string
	language      string
	imageType     string
	opt_          map[string]interface{}
	media_        io.Reader
}

// Upload: Uploads a new image and adds it to the list of images for the
// specified language and image type.
func (r *EditsImagesService) Upload(packageNameid string, editId string, language string, imageType string) *EditsImagesUploadCall {
	c := &EditsImagesUploadCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.language = language
	c.imageType = imageType
	return c
}
func (c *EditsImagesUploadCall) Media(r io.Reader) *EditsImagesUploadCall {
	c.media_ = r
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsImagesUploadCall) Fields(s ...googleapi.Field) *EditsImagesUploadCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsImagesUploadCall) Do() (*ImagesUploadResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/listings/{language}/{imageType}")
	if c.media_ != nil {
		urls = strings.Replace(urls, "https://www.googleapis.com/", "https://www.googleapis.com/upload/", 1)
		params.Set("uploadType", "multipart")
	}
	urls += "?" + params.Encode()
	body = new(bytes.Buffer)
	ctype := "application/json"
	contentLength_, hasMedia_ := googleapi.ConditionallyIncludeMedia(c.media_, &body, &ctype)
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
		"language":    c.language,
		"imageType":   c.imageType,
	})
	if hasMedia_ {
		req.ContentLength = contentLength_
	}
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
	var ret *ImagesUploadResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Uploads a new image and adds it to the list of images for the specified language and image type.",
	//   "httpMethod": "POST",
	//   "id": "androidpublisher.edits.images.upload",
	//   "mediaUpload": {
	//     "accept": [
	//       "image/*"
	//     ],
	//     "maxSize": "15MB",
	//     "protocols": {
	//       "resumable": {
	//         "multipart": true,
	//         "path": "/resumable/upload/androidpublisher/v2/applications/{packageName}/edits/{editId}/listings/{language}/{imageType}"
	//       },
	//       "simple": {
	//         "multipart": true,
	//         "path": "/upload/androidpublisher/v2/applications/{packageName}/edits/{editId}/listings/{language}/{imageType}"
	//       }
	//     }
	//   },
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "language",
	//     "imageType"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "imageType": {
	//       "enum": [
	//         "featureGraphic",
	//         "icon",
	//         "phoneScreenshots",
	//         "promoGraphic",
	//         "sevenInchScreenshots",
	//         "tenInchScreenshots"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         "",
	//         "",
	//         "",
	//         "",
	//         ""
	//       ],
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "language": {
	//       "description": "The language code (a BCP-47 language tag) of the localized listing whose images are to read or modified. For example, to select Austrian German, pass \"de-AT\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/listings/{language}/{imageType}",
	//   "response": {
	//     "$ref": "ImagesUploadResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ],
	//   "supportsMediaUpload": true
	// }

}

// method id "androidpublisher.edits.listings.delete":

type EditsListingsDeleteCall struct {
	s             *Service
	packageNameid string
	editId        string
	language      string
	opt_          map[string]interface{}
}

// Delete: Deletes the specified localized store listing from an edit.
func (r *EditsListingsService) Delete(packageNameid string, editId string, language string) *EditsListingsDeleteCall {
	c := &EditsListingsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.language = language
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsListingsDeleteCall) Fields(s ...googleapi.Field) *EditsListingsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsListingsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/listings/{language}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
		"language":    c.language,
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
	//   "description": "Deletes the specified localized store listing from an edit.",
	//   "httpMethod": "DELETE",
	//   "id": "androidpublisher.edits.listings.delete",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "language"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "language": {
	//       "description": "The language code (a BCP-47 language tag) of the localized listing to read or modify. For example, to select Austrian German, pass \"de-AT\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/listings/{language}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.listings.deleteall":

type EditsListingsDeleteallCall struct {
	s             *Service
	packageNameid string
	editId        string
	opt_          map[string]interface{}
}

// Deleteall: Deletes all localized listings from an edit.
func (r *EditsListingsService) Deleteall(packageNameid string, editId string) *EditsListingsDeleteallCall {
	c := &EditsListingsDeleteallCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsListingsDeleteallCall) Fields(s ...googleapi.Field) *EditsListingsDeleteallCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsListingsDeleteallCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/listings")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
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
	//   "description": "Deletes all localized listings from an edit.",
	//   "httpMethod": "DELETE",
	//   "id": "androidpublisher.edits.listings.deleteall",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/listings",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.listings.get":

type EditsListingsGetCall struct {
	s             *Service
	packageNameid string
	editId        string
	language      string
	opt_          map[string]interface{}
}

// Get: Fetches information about a localized store listing.
func (r *EditsListingsService) Get(packageNameid string, editId string, language string) *EditsListingsGetCall {
	c := &EditsListingsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.language = language
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsListingsGetCall) Fields(s ...googleapi.Field) *EditsListingsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsListingsGetCall) Do() (*Listing, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/listings/{language}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
		"language":    c.language,
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
	var ret *Listing
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Fetches information about a localized store listing.",
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.edits.listings.get",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "language"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "language": {
	//       "description": "The language code (a BCP-47 language tag) of the localized listing to read or modify. For example, to select Austrian German, pass \"de-AT\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/listings/{language}",
	//   "response": {
	//     "$ref": "Listing"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.listings.list":

type EditsListingsListCall struct {
	s             *Service
	packageNameid string
	editId        string
	opt_          map[string]interface{}
}

// List: Returns all of the localized store listings attached to this
// edit.
func (r *EditsListingsService) List(packageNameid string, editId string) *EditsListingsListCall {
	c := &EditsListingsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsListingsListCall) Fields(s ...googleapi.Field) *EditsListingsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsListingsListCall) Do() (*ListingsListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/listings")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
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
	var ret *ListingsListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Returns all of the localized store listings attached to this edit.",
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.edits.listings.list",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/listings",
	//   "response": {
	//     "$ref": "ListingsListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.listings.patch":

type EditsListingsPatchCall struct {
	s             *Service
	packageNameid string
	editId        string
	language      string
	listing       *Listing
	opt_          map[string]interface{}
}

// Patch: Creates or updates a localized store listing. This method
// supports patch semantics.
func (r *EditsListingsService) Patch(packageNameid string, editId string, language string, listing *Listing) *EditsListingsPatchCall {
	c := &EditsListingsPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.language = language
	c.listing = listing
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsListingsPatchCall) Fields(s ...googleapi.Field) *EditsListingsPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsListingsPatchCall) Do() (*Listing, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.listing)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/listings/{language}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
		"language":    c.language,
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
	var ret *Listing
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates or updates a localized store listing. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "androidpublisher.edits.listings.patch",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "language"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "language": {
	//       "description": "The language code (a BCP-47 language tag) of the localized listing to read or modify. For example, to select Austrian German, pass \"de-AT\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/listings/{language}",
	//   "request": {
	//     "$ref": "Listing"
	//   },
	//   "response": {
	//     "$ref": "Listing"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.listings.update":

type EditsListingsUpdateCall struct {
	s             *Service
	packageNameid string
	editId        string
	language      string
	listing       *Listing
	opt_          map[string]interface{}
}

// Update: Creates or updates a localized store listing.
func (r *EditsListingsService) Update(packageNameid string, editId string, language string, listing *Listing) *EditsListingsUpdateCall {
	c := &EditsListingsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.language = language
	c.listing = listing
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsListingsUpdateCall) Fields(s ...googleapi.Field) *EditsListingsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsListingsUpdateCall) Do() (*Listing, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.listing)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/listings/{language}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
		"language":    c.language,
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
	var ret *Listing
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates or updates a localized store listing.",
	//   "httpMethod": "PUT",
	//   "id": "androidpublisher.edits.listings.update",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "language"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "language": {
	//       "description": "The language code (a BCP-47 language tag) of the localized listing to read or modify. For example, to select Austrian German, pass \"de-AT\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/listings/{language}",
	//   "request": {
	//     "$ref": "Listing"
	//   },
	//   "response": {
	//     "$ref": "Listing"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.testers.get":

type EditsTestersGetCall struct {
	s             *Service
	packageNameid string
	editId        string
	track         string
	opt_          map[string]interface{}
}

// Get:
func (r *EditsTestersService) Get(packageNameid string, editId string, track string) *EditsTestersGetCall {
	c := &EditsTestersGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.track = track
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsTestersGetCall) Fields(s ...googleapi.Field) *EditsTestersGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsTestersGetCall) Do() (*Testers, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/testers/{track}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
		"track":       c.track,
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
	var ret *Testers
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.edits.testers.get",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "track"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "track": {
	//       "enum": [
	//         "alpha",
	//         "beta",
	//         "production",
	//         "rollout"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         "",
	//         "",
	//         ""
	//       ],
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/testers/{track}",
	//   "response": {
	//     "$ref": "Testers"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.testers.patch":

type EditsTestersPatchCall struct {
	s             *Service
	packageNameid string
	editId        string
	track         string
	testers       *Testers
	opt_          map[string]interface{}
}

// Patch:
func (r *EditsTestersService) Patch(packageNameid string, editId string, track string, testers *Testers) *EditsTestersPatchCall {
	c := &EditsTestersPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.track = track
	c.testers = testers
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsTestersPatchCall) Fields(s ...googleapi.Field) *EditsTestersPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsTestersPatchCall) Do() (*Testers, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.testers)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/testers/{track}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
		"track":       c.track,
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
	var ret *Testers
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "httpMethod": "PATCH",
	//   "id": "androidpublisher.edits.testers.patch",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "track"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "track": {
	//       "enum": [
	//         "alpha",
	//         "beta",
	//         "production",
	//         "rollout"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         "",
	//         "",
	//         ""
	//       ],
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/testers/{track}",
	//   "request": {
	//     "$ref": "Testers"
	//   },
	//   "response": {
	//     "$ref": "Testers"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.testers.update":

type EditsTestersUpdateCall struct {
	s             *Service
	packageNameid string
	editId        string
	track         string
	testers       *Testers
	opt_          map[string]interface{}
}

// Update:
func (r *EditsTestersService) Update(packageNameid string, editId string, track string, testers *Testers) *EditsTestersUpdateCall {
	c := &EditsTestersUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.track = track
	c.testers = testers
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsTestersUpdateCall) Fields(s ...googleapi.Field) *EditsTestersUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsTestersUpdateCall) Do() (*Testers, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.testers)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/testers/{track}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
		"track":       c.track,
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
	var ret *Testers
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "httpMethod": "PUT",
	//   "id": "androidpublisher.edits.testers.update",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "track"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "track": {
	//       "enum": [
	//         "alpha",
	//         "beta",
	//         "production",
	//         "rollout"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         "",
	//         "",
	//         ""
	//       ],
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/testers/{track}",
	//   "request": {
	//     "$ref": "Testers"
	//   },
	//   "response": {
	//     "$ref": "Testers"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.tracks.get":

type EditsTracksGetCall struct {
	s             *Service
	packageNameid string
	editId        string
	track         string
	opt_          map[string]interface{}
}

// Get: Fetches the track configuration for the specified track type.
// Includes the APK version codes that are in this track.
func (r *EditsTracksService) Get(packageNameid string, editId string, track string) *EditsTracksGetCall {
	c := &EditsTracksGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.track = track
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsTracksGetCall) Fields(s ...googleapi.Field) *EditsTracksGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsTracksGetCall) Do() (*Track, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/tracks/{track}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
		"track":       c.track,
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
	var ret *Track
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Fetches the track configuration for the specified track type. Includes the APK version codes that are in this track.",
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.edits.tracks.get",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "track"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "track": {
	//       "description": "The track type to read or modify.",
	//       "enum": [
	//         "alpha",
	//         "beta",
	//         "production",
	//         "rollout"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         "",
	//         "",
	//         ""
	//       ],
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/tracks/{track}",
	//   "response": {
	//     "$ref": "Track"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.tracks.list":

type EditsTracksListCall struct {
	s             *Service
	packageNameid string
	editId        string
	opt_          map[string]interface{}
}

// List: Lists all the track configurations for this edit.
func (r *EditsTracksService) List(packageNameid string, editId string) *EditsTracksListCall {
	c := &EditsTracksListCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsTracksListCall) Fields(s ...googleapi.Field) *EditsTracksListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsTracksListCall) Do() (*TracksListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/tracks")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
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
	var ret *TracksListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all the track configurations for this edit.",
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.edits.tracks.list",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/tracks",
	//   "response": {
	//     "$ref": "TracksListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.tracks.patch":

type EditsTracksPatchCall struct {
	s             *Service
	packageNameid string
	editId        string
	track         string
	track2        *Track
	opt_          map[string]interface{}
}

// Patch: Updates the track configuration for the specified track type.
// This method supports patch semantics.
func (r *EditsTracksService) Patch(packageNameid string, editId string, track string, track2 *Track) *EditsTracksPatchCall {
	c := &EditsTracksPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.track = track
	c.track2 = track2
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsTracksPatchCall) Fields(s ...googleapi.Field) *EditsTracksPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsTracksPatchCall) Do() (*Track, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.track2)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/tracks/{track}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
		"track":       c.track,
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
	var ret *Track
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates the track configuration for the specified track type. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "androidpublisher.edits.tracks.patch",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "track"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "track": {
	//       "description": "The track type to read or modify.",
	//       "enum": [
	//         "alpha",
	//         "beta",
	//         "production",
	//         "rollout"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         "",
	//         "",
	//         ""
	//       ],
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/tracks/{track}",
	//   "request": {
	//     "$ref": "Track"
	//   },
	//   "response": {
	//     "$ref": "Track"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.edits.tracks.update":

type EditsTracksUpdateCall struct {
	s             *Service
	packageNameid string
	editId        string
	track         string
	track2        *Track
	opt_          map[string]interface{}
}

// Update: Updates the track configuration for the specified track type.
func (r *EditsTracksService) Update(packageNameid string, editId string, track string, track2 *Track) *EditsTracksUpdateCall {
	c := &EditsTracksUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.editId = editId
	c.track = track
	c.track2 = track2
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EditsTracksUpdateCall) Fields(s ...googleapi.Field) *EditsTracksUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EditsTracksUpdateCall) Do() (*Track, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.track2)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/edits/{editId}/tracks/{track}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"editId":      c.editId,
		"track":       c.track,
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
	var ret *Track
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates the track configuration for the specified track type.",
	//   "httpMethod": "PUT",
	//   "id": "androidpublisher.edits.tracks.update",
	//   "parameterOrder": [
	//     "packageName",
	//     "editId",
	//     "track"
	//   ],
	//   "parameters": {
	//     "editId": {
	//       "description": "Unique identifier for this edit.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app that is being updated; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "track": {
	//       "description": "The track type to read or modify.",
	//       "enum": [
	//         "alpha",
	//         "beta",
	//         "production",
	//         "rollout"
	//       ],
	//       "enumDescriptions": [
	//         "",
	//         "",
	//         "",
	//         ""
	//       ],
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/edits/{editId}/tracks/{track}",
	//   "request": {
	//     "$ref": "Track"
	//   },
	//   "response": {
	//     "$ref": "Track"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.inappproducts.batch":

type InappproductsBatchCall struct {
	s                         *Service
	inappproductsbatchrequest *InappproductsBatchRequest
	opt_                      map[string]interface{}
}

// Batch:
func (r *InappproductsService) Batch(inappproductsbatchrequest *InappproductsBatchRequest) *InappproductsBatchCall {
	c := &InappproductsBatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.inappproductsbatchrequest = inappproductsbatchrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *InappproductsBatchCall) Fields(s ...googleapi.Field) *InappproductsBatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *InappproductsBatchCall) Do() (*InappproductsBatchResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.inappproductsbatchrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "inappproducts/batch")
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
	var ret *InappproductsBatchResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "httpMethod": "POST",
	//   "id": "androidpublisher.inappproducts.batch",
	//   "path": "inappproducts/batch",
	//   "request": {
	//     "$ref": "InappproductsBatchRequest"
	//   },
	//   "response": {
	//     "$ref": "InappproductsBatchResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.inappproducts.delete":

type InappproductsDeleteCall struct {
	s             *Service
	packageNameid string
	skuid         string
	opt_          map[string]interface{}
}

// Delete: Delete an in-app product for an app.
func (r *InappproductsService) Delete(packageNameid string, skuid string) *InappproductsDeleteCall {
	c := &InappproductsDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.skuid = skuid
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *InappproductsDeleteCall) Fields(s ...googleapi.Field) *InappproductsDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *InappproductsDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/inappproducts/{sku}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"sku":         c.skuid,
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
	//   "description": "Delete an in-app product for an app.",
	//   "httpMethod": "DELETE",
	//   "id": "androidpublisher.inappproducts.delete",
	//   "parameterOrder": [
	//     "packageName",
	//     "sku"
	//   ],
	//   "parameters": {
	//     "packageName": {
	//       "description": "Unique identifier for the Android app with the in-app product; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "sku": {
	//       "description": "Unique identifier for the in-app product.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/inappproducts/{sku}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.inappproducts.get":

type InappproductsGetCall struct {
	s           *Service
	packageName string
	skuid       string
	opt_        map[string]interface{}
}

// Get: Returns information about the in-app product specified.
func (r *InappproductsService) Get(packageName string, skuid string) *InappproductsGetCall {
	c := &InappproductsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageName = packageName
	c.skuid = skuid
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *InappproductsGetCall) Fields(s ...googleapi.Field) *InappproductsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *InappproductsGetCall) Do() (*InAppProduct, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/inappproducts/{sku}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageName,
		"sku":         c.skuid,
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
	var ret *InAppProduct
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Returns information about the in-app product specified.",
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.inappproducts.get",
	//   "parameterOrder": [
	//     "packageName",
	//     "sku"
	//   ],
	//   "parameters": {
	//     "packageName": {
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "sku": {
	//       "description": "Unique identifier for the in-app product.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/inappproducts/{sku}",
	//   "response": {
	//     "$ref": "InAppProduct"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.inappproducts.insert":

type InappproductsInsertCall struct {
	s             *Service
	packageNameid string
	inappproduct  *InAppProduct
	opt_          map[string]interface{}
}

// Insert: Creates a new in-app product for an app.
func (r *InappproductsService) Insert(packageNameid string, inappproduct *InAppProduct) *InappproductsInsertCall {
	c := &InappproductsInsertCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.inappproduct = inappproduct
	return c
}

// AutoConvertMissingPrices sets the optional parameter
// "autoConvertMissingPrices": If true the prices for all regions
// targeted by the parent app that don't have a price specified for this
// in-app product will be auto converted to the target currency based on
// the default price. Defaults to false.
func (c *InappproductsInsertCall) AutoConvertMissingPrices(autoConvertMissingPrices bool) *InappproductsInsertCall {
	c.opt_["autoConvertMissingPrices"] = autoConvertMissingPrices
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *InappproductsInsertCall) Fields(s ...googleapi.Field) *InappproductsInsertCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *InappproductsInsertCall) Do() (*InAppProduct, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.inappproduct)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["autoConvertMissingPrices"]; ok {
		params.Set("autoConvertMissingPrices", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/inappproducts")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
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
	var ret *InAppProduct
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a new in-app product for an app.",
	//   "httpMethod": "POST",
	//   "id": "androidpublisher.inappproducts.insert",
	//   "parameterOrder": [
	//     "packageName"
	//   ],
	//   "parameters": {
	//     "autoConvertMissingPrices": {
	//       "description": "If true the prices for all regions targeted by the parent app that don't have a price specified for this in-app product will be auto converted to the target currency based on the default price. Defaults to false.",
	//       "location": "query",
	//       "type": "boolean"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/inappproducts",
	//   "request": {
	//     "$ref": "InAppProduct"
	//   },
	//   "response": {
	//     "$ref": "InAppProduct"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.inappproducts.list":

type InappproductsListCall struct {
	s             *Service
	packageNameid string
	opt_          map[string]interface{}
}

// List: List all the in-app products for an Android app, both
// subscriptions and managed in-app products..
func (r *InappproductsService) List(packageNameid string) *InappproductsListCall {
	c := &InappproductsListCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	return c
}

// MaxResults sets the optional parameter "maxResults":
func (c *InappproductsListCall) MaxResults(maxResults int64) *InappproductsListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// StartIndex sets the optional parameter "startIndex":
func (c *InappproductsListCall) StartIndex(startIndex int64) *InappproductsListCall {
	c.opt_["startIndex"] = startIndex
	return c
}

// Token sets the optional parameter "token":
func (c *InappproductsListCall) Token(token string) *InappproductsListCall {
	c.opt_["token"] = token
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *InappproductsListCall) Fields(s ...googleapi.Field) *InappproductsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *InappproductsListCall) Do() (*InappproductsListResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["startIndex"]; ok {
		params.Set("startIndex", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["token"]; ok {
		params.Set("token", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/inappproducts")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
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
	var ret *InappproductsListResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "List all the in-app products for an Android app, both subscriptions and managed in-app products..",
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.inappproducts.list",
	//   "parameterOrder": [
	//     "packageName"
	//   ],
	//   "parameters": {
	//     "maxResults": {
	//       "format": "uint32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app with in-app products; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "startIndex": {
	//       "format": "uint32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "token": {
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/inappproducts",
	//   "response": {
	//     "$ref": "InappproductsListResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.inappproducts.patch":

type InappproductsPatchCall struct {
	s             *Service
	packageNameid string
	skuid         string
	inappproduct  *InAppProduct
	opt_          map[string]interface{}
}

// Patch: Updates the details of an in-app product. This method supports
// patch semantics.
func (r *InappproductsService) Patch(packageNameid string, skuid string, inappproduct *InAppProduct) *InappproductsPatchCall {
	c := &InappproductsPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.skuid = skuid
	c.inappproduct = inappproduct
	return c
}

// AutoConvertMissingPrices sets the optional parameter
// "autoConvertMissingPrices": If true the prices for all regions
// targeted by the parent app that don't have a price specified for this
// in-app product will be auto converted to the target currency based on
// the default price. Defaults to false.
func (c *InappproductsPatchCall) AutoConvertMissingPrices(autoConvertMissingPrices bool) *InappproductsPatchCall {
	c.opt_["autoConvertMissingPrices"] = autoConvertMissingPrices
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *InappproductsPatchCall) Fields(s ...googleapi.Field) *InappproductsPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *InappproductsPatchCall) Do() (*InAppProduct, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.inappproduct)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["autoConvertMissingPrices"]; ok {
		params.Set("autoConvertMissingPrices", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/inappproducts/{sku}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"sku":         c.skuid,
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
	var ret *InAppProduct
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates the details of an in-app product. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "androidpublisher.inappproducts.patch",
	//   "parameterOrder": [
	//     "packageName",
	//     "sku"
	//   ],
	//   "parameters": {
	//     "autoConvertMissingPrices": {
	//       "description": "If true the prices for all regions targeted by the parent app that don't have a price specified for this in-app product will be auto converted to the target currency based on the default price. Defaults to false.",
	//       "location": "query",
	//       "type": "boolean"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app with the in-app product; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "sku": {
	//       "description": "Unique identifier for the in-app product.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/inappproducts/{sku}",
	//   "request": {
	//     "$ref": "InAppProduct"
	//   },
	//   "response": {
	//     "$ref": "InAppProduct"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.inappproducts.update":

type InappproductsUpdateCall struct {
	s             *Service
	packageNameid string
	skuid         string
	inappproduct  *InAppProduct
	opt_          map[string]interface{}
}

// Update: Updates the details of an in-app product.
func (r *InappproductsService) Update(packageNameid string, skuid string, inappproduct *InAppProduct) *InappproductsUpdateCall {
	c := &InappproductsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageNameid = packageNameid
	c.skuid = skuid
	c.inappproduct = inappproduct
	return c
}

// AutoConvertMissingPrices sets the optional parameter
// "autoConvertMissingPrices": If true the prices for all regions
// targeted by the parent app that don't have a price specified for this
// in-app product will be auto converted to the target currency based on
// the default price. Defaults to false.
func (c *InappproductsUpdateCall) AutoConvertMissingPrices(autoConvertMissingPrices bool) *InappproductsUpdateCall {
	c.opt_["autoConvertMissingPrices"] = autoConvertMissingPrices
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *InappproductsUpdateCall) Fields(s ...googleapi.Field) *InappproductsUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *InappproductsUpdateCall) Do() (*InAppProduct, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.inappproduct)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["autoConvertMissingPrices"]; ok {
		params.Set("autoConvertMissingPrices", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/inappproducts/{sku}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageNameid,
		"sku":         c.skuid,
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
	var ret *InAppProduct
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates the details of an in-app product.",
	//   "httpMethod": "PUT",
	//   "id": "androidpublisher.inappproducts.update",
	//   "parameterOrder": [
	//     "packageName",
	//     "sku"
	//   ],
	//   "parameters": {
	//     "autoConvertMissingPrices": {
	//       "description": "If true the prices for all regions targeted by the parent app that don't have a price specified for this in-app product will be auto converted to the target currency based on the default price. Defaults to false.",
	//       "location": "query",
	//       "type": "boolean"
	//     },
	//     "packageName": {
	//       "description": "Unique identifier for the Android app with the in-app product; for example, \"com.spiffygame\".",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "sku": {
	//       "description": "Unique identifier for the in-app product.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/inappproducts/{sku}",
	//   "request": {
	//     "$ref": "InAppProduct"
	//   },
	//   "response": {
	//     "$ref": "InAppProduct"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.purchases.products.get":

type PurchasesProductsGetCall struct {
	s           *Service
	packageName string
	productId   string
	token       string
	opt_        map[string]interface{}
}

// Get: Checks the purchase and consumption status of an inapp item.
func (r *PurchasesProductsService) Get(packageName string, productId string, token string) *PurchasesProductsGetCall {
	c := &PurchasesProductsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageName = packageName
	c.productId = productId
	c.token = token
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PurchasesProductsGetCall) Fields(s ...googleapi.Field) *PurchasesProductsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PurchasesProductsGetCall) Do() (*ProductPurchase, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/purchases/products/{productId}/tokens/{token}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName": c.packageName,
		"productId":   c.productId,
		"token":       c.token,
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
	var ret *ProductPurchase
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Checks the purchase and consumption status of an inapp item.",
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.purchases.products.get",
	//   "parameterOrder": [
	//     "packageName",
	//     "productId",
	//     "token"
	//   ],
	//   "parameters": {
	//     "packageName": {
	//       "description": "The package name of the application the inapp product was sold in (for example, 'com.some.thing').",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "productId": {
	//       "description": "The inapp product SKU (for example, 'com.some.thing.inapp1').",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "token": {
	//       "description": "The token provided to the user's device when the inapp product was purchased.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/purchases/products/{productId}/tokens/{token}",
	//   "response": {
	//     "$ref": "ProductPurchase"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.purchases.subscriptions.cancel":

type PurchasesSubscriptionsCancelCall struct {
	s              *Service
	packageName    string
	subscriptionId string
	token          string
	opt_           map[string]interface{}
}

// Cancel: Cancels a user's subscription purchase. The subscription
// remains valid until its expiration time.
func (r *PurchasesSubscriptionsService) Cancel(packageName string, subscriptionId string, token string) *PurchasesSubscriptionsCancelCall {
	c := &PurchasesSubscriptionsCancelCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageName = packageName
	c.subscriptionId = subscriptionId
	c.token = token
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PurchasesSubscriptionsCancelCall) Fields(s ...googleapi.Field) *PurchasesSubscriptionsCancelCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PurchasesSubscriptionsCancelCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}:cancel")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":    c.packageName,
		"subscriptionId": c.subscriptionId,
		"token":          c.token,
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
	//   "description": "Cancels a user's subscription purchase. The subscription remains valid until its expiration time.",
	//   "httpMethod": "POST",
	//   "id": "androidpublisher.purchases.subscriptions.cancel",
	//   "parameterOrder": [
	//     "packageName",
	//     "subscriptionId",
	//     "token"
	//   ],
	//   "parameters": {
	//     "packageName": {
	//       "description": "The package name of the application for which this subscription was purchased (for example, 'com.some.thing').",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "subscriptionId": {
	//       "description": "The purchased subscription ID (for example, 'monthly001').",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "token": {
	//       "description": "The token provided to the user's device when the subscription was purchased.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}:cancel",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.purchases.subscriptions.defer":

type PurchasesSubscriptionsDeferCall struct {
	s                                 *Service
	packageName                       string
	subscriptionId                    string
	token                             string
	subscriptionpurchasesdeferrequest *SubscriptionPurchasesDeferRequest
	opt_                              map[string]interface{}
}

// Defer: Defers a user's subscription purchase until a specified future
// expiration time.
func (r *PurchasesSubscriptionsService) Defer(packageName string, subscriptionId string, token string, subscriptionpurchasesdeferrequest *SubscriptionPurchasesDeferRequest) *PurchasesSubscriptionsDeferCall {
	c := &PurchasesSubscriptionsDeferCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageName = packageName
	c.subscriptionId = subscriptionId
	c.token = token
	c.subscriptionpurchasesdeferrequest = subscriptionpurchasesdeferrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PurchasesSubscriptionsDeferCall) Fields(s ...googleapi.Field) *PurchasesSubscriptionsDeferCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PurchasesSubscriptionsDeferCall) Do() (*SubscriptionPurchasesDeferResponse, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.subscriptionpurchasesdeferrequest)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}:defer")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":    c.packageName,
		"subscriptionId": c.subscriptionId,
		"token":          c.token,
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
	var ret *SubscriptionPurchasesDeferResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Defers a user's subscription purchase until a specified future expiration time.",
	//   "httpMethod": "POST",
	//   "id": "androidpublisher.purchases.subscriptions.defer",
	//   "parameterOrder": [
	//     "packageName",
	//     "subscriptionId",
	//     "token"
	//   ],
	//   "parameters": {
	//     "packageName": {
	//       "description": "The package name of the application for which this subscription was purchased (for example, 'com.some.thing').",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "subscriptionId": {
	//       "description": "The purchased subscription ID (for example, 'monthly001').",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "token": {
	//       "description": "The token provided to the user's device when the subscription was purchased.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}:defer",
	//   "request": {
	//     "$ref": "SubscriptionPurchasesDeferRequest"
	//   },
	//   "response": {
	//     "$ref": "SubscriptionPurchasesDeferResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.purchases.subscriptions.get":

type PurchasesSubscriptionsGetCall struct {
	s              *Service
	packageName    string
	subscriptionId string
	token          string
	opt_           map[string]interface{}
}

// Get: Checks whether a user's subscription purchase is valid and
// returns its expiry time.
func (r *PurchasesSubscriptionsService) Get(packageName string, subscriptionId string, token string) *PurchasesSubscriptionsGetCall {
	c := &PurchasesSubscriptionsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageName = packageName
	c.subscriptionId = subscriptionId
	c.token = token
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PurchasesSubscriptionsGetCall) Fields(s ...googleapi.Field) *PurchasesSubscriptionsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PurchasesSubscriptionsGetCall) Do() (*SubscriptionPurchase, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":    c.packageName,
		"subscriptionId": c.subscriptionId,
		"token":          c.token,
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
	var ret *SubscriptionPurchase
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Checks whether a user's subscription purchase is valid and returns its expiry time.",
	//   "httpMethod": "GET",
	//   "id": "androidpublisher.purchases.subscriptions.get",
	//   "parameterOrder": [
	//     "packageName",
	//     "subscriptionId",
	//     "token"
	//   ],
	//   "parameters": {
	//     "packageName": {
	//       "description": "The package name of the application for which this subscription was purchased (for example, 'com.some.thing').",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "subscriptionId": {
	//       "description": "The purchased subscription ID (for example, 'monthly001').",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "token": {
	//       "description": "The token provided to the user's device when the subscription was purchased.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}",
	//   "response": {
	//     "$ref": "SubscriptionPurchase"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.purchases.subscriptions.refund":

type PurchasesSubscriptionsRefundCall struct {
	s              *Service
	packageName    string
	subscriptionId string
	token          string
	opt_           map[string]interface{}
}

// Refund: Refunds a user's subscription purchase, but the subscription
// remains valid until its expiration time and it will continue to
// recur.
func (r *PurchasesSubscriptionsService) Refund(packageName string, subscriptionId string, token string) *PurchasesSubscriptionsRefundCall {
	c := &PurchasesSubscriptionsRefundCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageName = packageName
	c.subscriptionId = subscriptionId
	c.token = token
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PurchasesSubscriptionsRefundCall) Fields(s ...googleapi.Field) *PurchasesSubscriptionsRefundCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PurchasesSubscriptionsRefundCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}:refund")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":    c.packageName,
		"subscriptionId": c.subscriptionId,
		"token":          c.token,
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
	//   "description": "Refunds a user's subscription purchase, but the subscription remains valid until its expiration time and it will continue to recur.",
	//   "httpMethod": "POST",
	//   "id": "androidpublisher.purchases.subscriptions.refund",
	//   "parameterOrder": [
	//     "packageName",
	//     "subscriptionId",
	//     "token"
	//   ],
	//   "parameters": {
	//     "packageName": {
	//       "description": "The package name of the application for which this subscription was purchased (for example, 'com.some.thing').",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "subscriptionId": {
	//       "description": "The purchased subscription ID (for example, 'monthly001').",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "token": {
	//       "description": "The token provided to the user's device when the subscription was purchased.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}:refund",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}

// method id "androidpublisher.purchases.subscriptions.revoke":

type PurchasesSubscriptionsRevokeCall struct {
	s              *Service
	packageName    string
	subscriptionId string
	token          string
	opt_           map[string]interface{}
}

// Revoke: Refunds and immediately revokes a user's subscription
// purchase. Access to the subscription will be terminated immediately
// and it will stop recurring.
func (r *PurchasesSubscriptionsService) Revoke(packageName string, subscriptionId string, token string) *PurchasesSubscriptionsRevokeCall {
	c := &PurchasesSubscriptionsRevokeCall{s: r.s, opt_: make(map[string]interface{})}
	c.packageName = packageName
	c.subscriptionId = subscriptionId
	c.token = token
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PurchasesSubscriptionsRevokeCall) Fields(s ...googleapi.Field) *PurchasesSubscriptionsRevokeCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PurchasesSubscriptionsRevokeCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}:revoke")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"packageName":    c.packageName,
		"subscriptionId": c.subscriptionId,
		"token":          c.token,
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
	//   "description": "Refunds and immediately revokes a user's subscription purchase. Access to the subscription will be terminated immediately and it will stop recurring.",
	//   "httpMethod": "POST",
	//   "id": "androidpublisher.purchases.subscriptions.revoke",
	//   "parameterOrder": [
	//     "packageName",
	//     "subscriptionId",
	//     "token"
	//   ],
	//   "parameters": {
	//     "packageName": {
	//       "description": "The package name of the application for which this subscription was purchased (for example, 'com.some.thing').",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "subscriptionId": {
	//       "description": "The purchased subscription ID (for example, 'monthly001').",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "token": {
	//       "description": "The token provided to the user's device when the subscription was purchased.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "{packageName}/purchases/subscriptions/{subscriptionId}/tokens/{token}:revoke",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/androidpublisher"
	//   ]
	// }

}
