// Package adexchangebuyer provides access to the Ad Exchange Buyer API.
//
// See https://developers.google.com/ad-exchange/buyer-rest
//
// Usage example:
//
//   import "google.golang.org/api/adexchangebuyer/v1.3"
//   ...
//   adexchangebuyerService, err := adexchangebuyer.New(oauthHttpClient)
package adexchangebuyer

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

const apiId = "adexchangebuyer:v1.3"
const apiName = "adexchangebuyer"
const apiVersion = "v1.3"
const basePath = "https://www.googleapis.com/adexchangebuyer/v1.3/"

// OAuth2 scopes used by this API.
const (
	// Manage your Ad Exchange buyer account configuration
	AdexchangeBuyerScope = "https://www.googleapis.com/auth/adexchange.buyer"
)

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Accounts = NewAccountsService(s)
	s.BillingInfo = NewBillingInfoService(s)
	s.Creatives = NewCreativesService(s)
	s.DirectDeals = NewDirectDealsService(s)
	s.PerformanceReport = NewPerformanceReportService(s)
	s.PretargetingConfig = NewPretargetingConfigService(s)
	return s, nil
}

type Service struct {
	client   *http.Client
	BasePath string // API endpoint base URL

	Accounts *AccountsService

	BillingInfo *BillingInfoService

	Creatives *CreativesService

	DirectDeals *DirectDealsService

	PerformanceReport *PerformanceReportService

	PretargetingConfig *PretargetingConfigService
}

func NewAccountsService(s *Service) *AccountsService {
	rs := &AccountsService{s: s}
	return rs
}

type AccountsService struct {
	s *Service
}

func NewBillingInfoService(s *Service) *BillingInfoService {
	rs := &BillingInfoService{s: s}
	return rs
}

type BillingInfoService struct {
	s *Service
}

func NewCreativesService(s *Service) *CreativesService {
	rs := &CreativesService{s: s}
	return rs
}

type CreativesService struct {
	s *Service
}

func NewDirectDealsService(s *Service) *DirectDealsService {
	rs := &DirectDealsService{s: s}
	return rs
}

type DirectDealsService struct {
	s *Service
}

func NewPerformanceReportService(s *Service) *PerformanceReportService {
	rs := &PerformanceReportService{s: s}
	return rs
}

type PerformanceReportService struct {
	s *Service
}

func NewPretargetingConfigService(s *Service) *PretargetingConfigService {
	rs := &PretargetingConfigService{s: s}
	return rs
}

type PretargetingConfigService struct {
	s *Service
}

type Account struct {
	// BidderLocation: Your bidder locations that have distinct URLs.
	BidderLocation []*AccountBidderLocation `json:"bidderLocation,omitempty"`

	// CookieMatchingNid: The nid parameter value used in cookie match
	// requests. Please contact your technical account manager if you need
	// to change this.
	CookieMatchingNid string `json:"cookieMatchingNid,omitempty"`

	// CookieMatchingUrl: The base URL used in cookie match requests.
	CookieMatchingUrl string `json:"cookieMatchingUrl,omitempty"`

	// Id: Account id.
	Id int64 `json:"id,omitempty"`

	// Kind: Resource type.
	Kind string `json:"kind,omitempty"`

	// MaximumActiveCreatives: The maximum number of active creatives that
	// an account can have, where a creative is active if it was inserted or
	// bid with in the last 30 days. Please contact your technical account
	// manager if you need to change this.
	MaximumActiveCreatives int64 `json:"maximumActiveCreatives,omitempty"`

	// MaximumTotalQps: The sum of all bidderLocation.maximumQps values
	// cannot exceed this. Please contact your technical account manager if
	// you need to change this.
	MaximumTotalQps int64 `json:"maximumTotalQps,omitempty"`

	// NumberActiveCreatives: The number of creatives that this account
	// inserted or bid with in the last 30 days.
	NumberActiveCreatives int64 `json:"numberActiveCreatives,omitempty"`
}

type AccountBidderLocation struct {
	// MaximumQps: The maximum queries per second the Ad Exchange will send.
	MaximumQps int64 `json:"maximumQps,omitempty"`

	// Region: The geographical region the Ad Exchange should send requests
	// from. Only used by some quota systems, but always setting the value
	// is recommended. Allowed values:
	// - ASIA
	// - EUROPE
	// - US_EAST
	// -
	// US_WEST
	Region string `json:"region,omitempty"`

	// Url: The URL to which the Ad Exchange will send bid requests.
	Url string `json:"url,omitempty"`
}

type AccountsList struct {
	// Items: A list of accounts.
	Items []*Account `json:"items,omitempty"`

	// Kind: Resource type.
	Kind string `json:"kind,omitempty"`
}

type BillingInfo struct {
	// AccountId: Account id.
	AccountId int64 `json:"accountId,omitempty"`

	// AccountName: Account name.
	AccountName string `json:"accountName,omitempty"`

	// BillingId: A list of adgroup IDs associated with this particular
	// account. These IDs may show up as part of a realtime bidding
	// BidRequest, which indicates a bid request for this account.
	BillingId []string `json:"billingId,omitempty"`

	// Kind: Resource type.
	Kind string `json:"kind,omitempty"`
}

type BillingInfoList struct {
	// Items: A list of billing info relevant for your account.
	Items []*BillingInfo `json:"items,omitempty"`

	// Kind: Resource type.
	Kind string `json:"kind,omitempty"`
}

type Creative struct {
	// HTMLSnippet: The HTML snippet that displays the ad when inserted in
	// the web page. If set, videoURL should not be set.
	HTMLSnippet string `json:"HTMLSnippet,omitempty"`

	// AccountId: Account id.
	AccountId int64 `json:"accountId,omitempty"`

	// AdvertiserId: Detected advertiser id, if any. Read-only. This field
	// should not be set in requests.
	AdvertiserId googleapi.Int64s `json:"advertiserId,omitempty"`

	// AdvertiserName: The name of the company being advertised in the
	// creative.
	AdvertiserName string `json:"advertiserName,omitempty"`

	// AgencyId: The agency id for this creative.
	AgencyId int64 `json:"agencyId,omitempty,string"`

	// Attribute: All attributes for the ads that may be shown from this
	// snippet.
	Attribute []int64 `json:"attribute,omitempty"`

	// BuyerCreativeId: A buyer-specific id identifying the creative in this
	// ad.
	BuyerCreativeId string `json:"buyerCreativeId,omitempty"`

	// ClickThroughUrl: The set of destination urls for the snippet.
	ClickThroughUrl []string `json:"clickThroughUrl,omitempty"`

	// Corrections: Shows any corrections that were applied to this
	// creative. Read-only. This field should not be set in requests.
	Corrections []*CreativeCorrections `json:"corrections,omitempty"`

	// DisapprovalReasons: The reasons for disapproval, if any. Note that
	// not all disapproval reasons may be categorized, so it is possible for
	// the creative to have a status of DISAPPROVED with an empty list for
	// disapproval_reasons. In this case, please reach out to your TAM to
	// help debug the issue. Read-only. This field should not be set in
	// requests.
	DisapprovalReasons []*CreativeDisapprovalReasons `json:"disapprovalReasons,omitempty"`

	// FilteringReasons: The filtering reasons for the creative. If this
	// feature is not enabled, please ask your technical account manager.
	// Read-only. This field should not be set in requests.
	FilteringReasons *CreativeFilteringReasons `json:"filteringReasons,omitempty"`

	// Height: Ad height.
	Height int64 `json:"height,omitempty"`

	// Kind: Resource type.
	Kind string `json:"kind,omitempty"`

	// ProductCategories: Detected product categories, if any. Read-only.
	// This field should not be set in requests.
	ProductCategories []int64 `json:"productCategories,omitempty"`

	// RestrictedCategories: All restricted categories for the ads that may
	// be shown from this snippet.
	RestrictedCategories []int64 `json:"restrictedCategories,omitempty"`

	// SensitiveCategories: Detected sensitive categories, if any.
	// Read-only. This field should not be set in requests.
	SensitiveCategories []int64 `json:"sensitiveCategories,omitempty"`

	// Status: Creative serving status. Read-only. This field should not be
	// set in requests.
	Status string `json:"status,omitempty"`

	// VendorType: All vendor types for the ads that may be shown from this
	// snippet.
	VendorType []int64 `json:"vendorType,omitempty"`

	// VideoURL: The url to fetch a video ad. If set, HTMLSnippet should not
	// be set.
	VideoURL string `json:"videoURL,omitempty"`

	// Width: Ad width.
	Width int64 `json:"width,omitempty"`
}

type CreativeCorrections struct {
	// Details: Additional details about the correction.
	Details []string `json:"details,omitempty"`

	// Reason: The type of correction that was applied to the creative.
	Reason string `json:"reason,omitempty"`
}

type CreativeDisapprovalReasons struct {
	// Details: Additional details about the reason for disapproval.
	Details []string `json:"details,omitempty"`

	// Reason: The categorized reason for disapproval.
	Reason string `json:"reason,omitempty"`
}

type CreativeFilteringReasons struct {
	// Date: The date in ISO 8601 format for the data. The data is collected
	// from 00:00:00 to 23:59:59 in PST.
	Date string `json:"date,omitempty"`

	// Reasons: The filtering reasons.
	Reasons []*CreativeFilteringReasonsReasons `json:"reasons,omitempty"`
}

type CreativeFilteringReasonsReasons struct {
	// FilteringCount: The number of times the creative was filtered for the
	// status. The count is aggregated across all publishers on the
	// exchange.
	FilteringCount int64 `json:"filteringCount,omitempty,string"`

	// FilteringStatus: The filtering status code. Please refer to the
	// creative-status-codes.txt file for different statuses.
	FilteringStatus int64 `json:"filteringStatus,omitempty"`
}

type CreativesList struct {
	// Items: A list of creatives.
	Items []*Creative `json:"items,omitempty"`

	// Kind: Resource type.
	Kind string `json:"kind,omitempty"`

	// NextPageToken: Continuation token used to page through creatives. To
	// retrieve the next page of results, set the next request's "pageToken"
	// value to this.
	NextPageToken string `json:"nextPageToken,omitempty"`
}

type DirectDeal struct {
	// AccountId: The account id of the buyer this deal is for.
	AccountId int64 `json:"accountId,omitempty"`

	// Advertiser: The name of the advertiser this deal is for.
	Advertiser string `json:"advertiser,omitempty"`

	// CurrencyCode: The currency code that applies to the fixed_cpm value.
	// If not set then assumed to be USD.
	CurrencyCode string `json:"currencyCode,omitempty"`

	// EndTime: End time for when this deal stops being active. If not set
	// then this deal is valid until manually disabled by the publisher. In
	// seconds since the epoch.
	EndTime int64 `json:"endTime,omitempty,string"`

	// FixedCpm: The fixed price for this direct deal. In cpm micros of
	// currency according to currency_code. If set, then this deal is
	// eligible for the fixed price tier of buying (highest priority, pay
	// exactly the configured fixed price).
	FixedCpm int64 `json:"fixedCpm,omitempty,string"`

	// Id: Deal id.
	Id int64 `json:"id,omitempty,string"`

	// Kind: Resource type.
	Kind string `json:"kind,omitempty"`

	// Name: Deal name.
	Name string `json:"name,omitempty"`

	// PrivateExchangeMinCpm: The minimum price for this direct deal. In cpm
	// micros of currency according to currency_code. If set, then this deal
	// is eligible for the private exchange tier of buying (below fixed
	// price priority, run as a second price auction).
	PrivateExchangeMinCpm int64 `json:"privateExchangeMinCpm,omitempty,string"`

	// PublisherBlocksOverriden: If true, the publisher has opted to have
	// their blocks ignored when a creative is bid with for this deal.
	PublisherBlocksOverriden bool `json:"publisherBlocksOverriden,omitempty"`

	// SellerNetwork: The name of the publisher offering this direct deal.
	SellerNetwork string `json:"sellerNetwork,omitempty"`

	// StartTime: Start time for when this deal becomes active. If not set
	// then this deal is active immediately upon creation. In seconds since
	// the epoch.
	StartTime int64 `json:"startTime,omitempty,string"`
}

type DirectDealsList struct {
	// DirectDeals: A list of direct deals relevant for your account.
	DirectDeals []*DirectDeal `json:"directDeals,omitempty"`

	// Kind: Resource type.
	Kind string `json:"kind,omitempty"`
}

type PerformanceReport struct {
	// CalloutStatusRate: Rate of various prefiltering statuses per match.
	// Please refer to the callout-status-codes.txt file for different
	// statuses.
	CalloutStatusRate []interface{} `json:"calloutStatusRate,omitempty"`

	// CookieMatcherStatusRate: Average QPS for cookie matcher operations.
	CookieMatcherStatusRate []interface{} `json:"cookieMatcherStatusRate,omitempty"`

	// CreativeStatusRate: Rate of ads with a given status. Please refer to
	// the creative-status-codes.txt file for different statuses.
	CreativeStatusRate []interface{} `json:"creativeStatusRate,omitempty"`

	// HostedMatchStatusRate: Average QPS for hosted match operations.
	HostedMatchStatusRate []interface{} `json:"hostedMatchStatusRate,omitempty"`

	// Kind: Resource type.
	Kind string `json:"kind,omitempty"`

	// Latency50thPercentile: The 50th percentile round trip latency(ms) as
	// perceived from Google servers for the duration period covered by the
	// report.
	Latency50thPercentile float64 `json:"latency50thPercentile,omitempty"`

	// Latency85thPercentile: The 85th percentile round trip latency(ms) as
	// perceived from Google servers for the duration period covered by the
	// report.
	Latency85thPercentile float64 `json:"latency85thPercentile,omitempty"`

	// Latency95thPercentile: The 95th percentile round trip latency(ms) as
	// perceived from Google servers for the duration period covered by the
	// report.
	Latency95thPercentile float64 `json:"latency95thPercentile,omitempty"`

	// NoQuotaInRegion: Rate of various quota account statuses per quota
	// check.
	NoQuotaInRegion float64 `json:"noQuotaInRegion,omitempty"`

	// OutOfQuota: Rate of various quota account statuses per quota check.
	OutOfQuota float64 `json:"outOfQuota,omitempty"`

	// PixelMatchRequests: Average QPS for pixel match requests from
	// clients.
	PixelMatchRequests float64 `json:"pixelMatchRequests,omitempty"`

	// PixelMatchResponses: Average QPS for pixel match responses from
	// clients.
	PixelMatchResponses float64 `json:"pixelMatchResponses,omitempty"`

	// QuotaConfiguredLimit: The configured quota limits for this account.
	QuotaConfiguredLimit float64 `json:"quotaConfiguredLimit,omitempty"`

	// QuotaThrottledLimit: The throttled quota limits for this account.
	QuotaThrottledLimit float64 `json:"quotaThrottledLimit,omitempty"`

	// Region: The trading location of this data.
	Region string `json:"region,omitempty"`

	// Timestamp: The unix timestamp of the starting time of this
	// performance data.
	Timestamp int64 `json:"timestamp,omitempty,string"`
}

type PerformanceReportList struct {
	// Kind: Resource type.
	Kind string `json:"kind,omitempty"`

	// PerformanceReport: A list of performance reports relevant for the
	// account.
	PerformanceReport []*PerformanceReport `json:"performanceReport,omitempty"`
}

type PretargetingConfig struct {
	// BillingId: The id for billing purposes, provided for reference. Leave
	// this field blank for insert requests; the id will be generated
	// automatically.
	BillingId int64 `json:"billingId,omitempty,string"`

	// ConfigId: The config id; generated automatically. Leave this field
	// blank for insert requests.
	ConfigId int64 `json:"configId,omitempty,string"`

	// ConfigName: The name of the config. Must be unique. Required for all
	// requests.
	ConfigName string `json:"configName,omitempty"`

	// CreativeType: List must contain exactly one of
	// PRETARGETING_CREATIVE_TYPE_HTML or PRETARGETING_CREATIVE_TYPE_VIDEO.
	CreativeType []string `json:"creativeType,omitempty"`

	// Dimensions: Requests which allow one of these (width, height) pairs
	// will match. All pairs must be supported ad dimensions.
	Dimensions []*PretargetingConfigDimensions `json:"dimensions,omitempty"`

	// ExcludedContentLabels: Requests with any of these content labels will
	// not match. Values are from content-labels.txt in the downloadable
	// files section.
	ExcludedContentLabels googleapi.Int64s `json:"excludedContentLabels,omitempty"`

	// ExcludedGeoCriteriaIds: Requests containing any of these geo criteria
	// ids will not match.
	ExcludedGeoCriteriaIds googleapi.Int64s `json:"excludedGeoCriteriaIds,omitempty"`

	// ExcludedPlacements: Requests containing any of these placements will
	// not match.
	ExcludedPlacements []*PretargetingConfigExcludedPlacements `json:"excludedPlacements,omitempty"`

	// ExcludedUserLists: Requests containing any of these users list ids
	// will not match.
	ExcludedUserLists googleapi.Int64s `json:"excludedUserLists,omitempty"`

	// ExcludedVerticals: Requests containing any of these vertical ids will
	// not match. Values are from the publisher-verticals.txt file in the
	// downloadable files section.
	ExcludedVerticals googleapi.Int64s `json:"excludedVerticals,omitempty"`

	// GeoCriteriaIds: Requests containing any of these geo criteria ids
	// will match.
	GeoCriteriaIds googleapi.Int64s `json:"geoCriteriaIds,omitempty"`

	// IsActive: Whether this config is active. Required for all requests.
	IsActive bool `json:"isActive,omitempty"`

	// Kind: The kind of the resource, i.e.
	// "adexchangebuyer#pretargetingConfig".
	Kind string `json:"kind,omitempty"`

	// Languages: Request containing any of these language codes will match.
	Languages []string `json:"languages,omitempty"`

	// MobileCarriers: Requests containing any of these mobile carrier ids
	// will match. Values are from mobile-carriers.csv in the downloadable
	// files section.
	MobileCarriers googleapi.Int64s `json:"mobileCarriers,omitempty"`

	// MobileDevices: Requests containing any of these mobile device ids
	// will match. Values are from mobile-devices.csv in the downloadable
	// files section.
	MobileDevices googleapi.Int64s `json:"mobileDevices,omitempty"`

	// MobileOperatingSystemVersions: Requests containing any of these
	// mobile operating system version ids will match. Values are from
	// mobile-os.csv in the downloadable files section.
	MobileOperatingSystemVersions googleapi.Int64s `json:"mobileOperatingSystemVersions,omitempty"`

	// Placements: Requests containing any of these placements will match.
	Placements []*PretargetingConfigPlacements `json:"placements,omitempty"`

	// Platforms: Requests matching any of these platforms will match.
	// Possible values are PRETARGETING_PLATFORM_MOBILE,
	// PRETARGETING_PLATFORM_DESKTOP, and PRETARGETING_PLATFORM_TABLET.
	Platforms []string `json:"platforms,omitempty"`

	// SupportedCreativeAttributes: Creative attributes should be declared
	// here if all creatives corresponding to this pretargeting
	// configuration have that creative attribute. Values are from
	// pretargetable-creative-attributes.txt in the downloadable files
	// section.
	SupportedCreativeAttributes googleapi.Int64s `json:"supportedCreativeAttributes,omitempty"`

	// UserLists: Requests containing any of these user list ids will match.
	UserLists googleapi.Int64s `json:"userLists,omitempty"`

	// VendorTypes: Requests that allow any of these vendor ids will match.
	// Values are from vendors.txt in the downloadable files section.
	VendorTypes googleapi.Int64s `json:"vendorTypes,omitempty"`

	// Verticals: Requests containing any of these vertical ids will match.
	Verticals googleapi.Int64s `json:"verticals,omitempty"`
}

type PretargetingConfigDimensions struct {
	// Height: Height in pixels.
	Height int64 `json:"height,omitempty,string"`

	// Width: Width in pixels.
	Width int64 `json:"width,omitempty,string"`
}

type PretargetingConfigExcludedPlacements struct {
	// Token: The value of the placement. Interpretation depends on the
	// placement type, e.g. URL for a site placement, channel name for a
	// channel placement, app id for a mobile app placement.
	Token string `json:"token,omitempty"`

	// Type: The type of the placement.
	Type string `json:"type,omitempty"`
}

type PretargetingConfigPlacements struct {
	// Token: The value of the placement. Interpretation depends on the
	// placement type, e.g. URL for a site placement, channel name for a
	// channel placement, app id for a mobile app placement.
	Token string `json:"token,omitempty"`

	// Type: The type of the placement.
	Type string `json:"type,omitempty"`
}

type PretargetingConfigList struct {
	// Items: A list of pretargeting configs
	Items []*PretargetingConfig `json:"items,omitempty"`

	// Kind: Resource type.
	Kind string `json:"kind,omitempty"`
}

// method id "adexchangebuyer.accounts.get":

type AccountsGetCall struct {
	s    *Service
	id   int64
	opt_ map[string]interface{}
}

// Get: Gets one account by ID.
func (r *AccountsService) Get(id int64) *AccountsGetCall {
	c := &AccountsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.id = id
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
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"id": strconv.FormatInt(c.id, 10),
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
	//   "description": "Gets one account by ID.",
	//   "httpMethod": "GET",
	//   "id": "adexchangebuyer.accounts.get",
	//   "parameterOrder": [
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The account id",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     }
	//   },
	//   "path": "accounts/{id}",
	//   "response": {
	//     "$ref": "Account"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.accounts.list":

type AccountsListCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// List: Retrieves the authenticated user's list of accounts.
func (r *AccountsService) List() *AccountsListCall {
	c := &AccountsListCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AccountsListCall) Fields(s ...googleapi.Field) *AccountsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AccountsListCall) Do() (*AccountsList, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *AccountsList
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves the authenticated user's list of accounts.",
	//   "httpMethod": "GET",
	//   "id": "adexchangebuyer.accounts.list",
	//   "path": "accounts",
	//   "response": {
	//     "$ref": "AccountsList"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.accounts.patch":

type AccountsPatchCall struct {
	s       *Service
	id      int64
	account *Account
	opt_    map[string]interface{}
}

// Patch: Updates an existing account. This method supports patch
// semantics.
func (r *AccountsService) Patch(id int64, account *Account) *AccountsPatchCall {
	c := &AccountsPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.id = id
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
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"id": strconv.FormatInt(c.id, 10),
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
	//   "description": "Updates an existing account. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "adexchangebuyer.accounts.patch",
	//   "parameterOrder": [
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The account id",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     }
	//   },
	//   "path": "accounts/{id}",
	//   "request": {
	//     "$ref": "Account"
	//   },
	//   "response": {
	//     "$ref": "Account"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.accounts.update":

type AccountsUpdateCall struct {
	s       *Service
	id      int64
	account *Account
	opt_    map[string]interface{}
}

// Update: Updates an existing account.
func (r *AccountsService) Update(id int64, account *Account) *AccountsUpdateCall {
	c := &AccountsUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.id = id
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
	urls := googleapi.ResolveRelative(c.s.BasePath, "accounts/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"id": strconv.FormatInt(c.id, 10),
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
	//   "description": "Updates an existing account.",
	//   "httpMethod": "PUT",
	//   "id": "adexchangebuyer.accounts.update",
	//   "parameterOrder": [
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The account id",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     }
	//   },
	//   "path": "accounts/{id}",
	//   "request": {
	//     "$ref": "Account"
	//   },
	//   "response": {
	//     "$ref": "Account"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.billingInfo.get":

type BillingInfoGetCall struct {
	s         *Service
	accountId int64
	opt_      map[string]interface{}
}

// Get: Returns the billing information for one account specified by
// account ID.
func (r *BillingInfoService) Get(accountId int64) *BillingInfoGetCall {
	c := &BillingInfoGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *BillingInfoGetCall) Fields(s ...googleapi.Field) *BillingInfoGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *BillingInfoGetCall) Do() (*BillingInfo, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "billinginfo/{accountId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId": strconv.FormatInt(c.accountId, 10),
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
	var ret *BillingInfo
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Returns the billing information for one account specified by account ID.",
	//   "httpMethod": "GET",
	//   "id": "adexchangebuyer.billingInfo.get",
	//   "parameterOrder": [
	//     "accountId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The account id.",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     }
	//   },
	//   "path": "billinginfo/{accountId}",
	//   "response": {
	//     "$ref": "BillingInfo"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.billingInfo.list":

type BillingInfoListCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// List: Retrieves a list of billing information for all accounts of the
// authenticated user.
func (r *BillingInfoService) List() *BillingInfoListCall {
	c := &BillingInfoListCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *BillingInfoListCall) Fields(s ...googleapi.Field) *BillingInfoListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *BillingInfoListCall) Do() (*BillingInfoList, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "billinginfo")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *BillingInfoList
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves a list of billing information for all accounts of the authenticated user.",
	//   "httpMethod": "GET",
	//   "id": "adexchangebuyer.billingInfo.list",
	//   "path": "billinginfo",
	//   "response": {
	//     "$ref": "BillingInfoList"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.creatives.get":

type CreativesGetCall struct {
	s               *Service
	accountId       int64
	buyerCreativeId string
	opt_            map[string]interface{}
}

// Get: Gets the status for a single creative. A creative will be
// available 30-40 minutes after submission.
func (r *CreativesService) Get(accountId int64, buyerCreativeId string) *CreativesGetCall {
	c := &CreativesGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.buyerCreativeId = buyerCreativeId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *CreativesGetCall) Fields(s ...googleapi.Field) *CreativesGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *CreativesGetCall) Do() (*Creative, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "creatives/{accountId}/{buyerCreativeId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId":       strconv.FormatInt(c.accountId, 10),
		"buyerCreativeId": c.buyerCreativeId,
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
	var ret *Creative
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets the status for a single creative. A creative will be available 30-40 minutes after submission.",
	//   "httpMethod": "GET",
	//   "id": "adexchangebuyer.creatives.get",
	//   "parameterOrder": [
	//     "accountId",
	//     "buyerCreativeId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The id for the account that will serve this creative.",
	//       "format": "int32",
	//       "location": "path",
	//       "required": true,
	//       "type": "integer"
	//     },
	//     "buyerCreativeId": {
	//       "description": "The buyer-specific id for this creative.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "creatives/{accountId}/{buyerCreativeId}",
	//   "response": {
	//     "$ref": "Creative"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.creatives.insert":

type CreativesInsertCall struct {
	s        *Service
	creative *Creative
	opt_     map[string]interface{}
}

// Insert: Submit a new creative.
func (r *CreativesService) Insert(creative *Creative) *CreativesInsertCall {
	c := &CreativesInsertCall{s: r.s, opt_: make(map[string]interface{})}
	c.creative = creative
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *CreativesInsertCall) Fields(s ...googleapi.Field) *CreativesInsertCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *CreativesInsertCall) Do() (*Creative, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.creative)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "creatives")
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
	var ret *Creative
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Submit a new creative.",
	//   "httpMethod": "POST",
	//   "id": "adexchangebuyer.creatives.insert",
	//   "path": "creatives",
	//   "request": {
	//     "$ref": "Creative"
	//   },
	//   "response": {
	//     "$ref": "Creative"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.creatives.list":

type CreativesListCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// List: Retrieves a list of the authenticated user's active creatives.
// A creative will be available 30-40 minutes after submission.
func (r *CreativesService) List() *CreativesListCall {
	c := &CreativesListCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// AccountId sets the optional parameter "accountId": When specified,
// only creatives for the given account ids are returned.
func (c *CreativesListCall) AccountId(accountId int64) *CreativesListCall {
	c.opt_["accountId"] = accountId
	return c
}

// BuyerCreativeId sets the optional parameter "buyerCreativeId": When
// specified, only creatives for the given buyer creative ids are
// returned.
func (c *CreativesListCall) BuyerCreativeId(buyerCreativeId string) *CreativesListCall {
	c.opt_["buyerCreativeId"] = buyerCreativeId
	return c
}

// MaxResults sets the optional parameter "maxResults": Maximum number
// of entries returned on one result page. If not set, the default is
// 100.
func (c *CreativesListCall) MaxResults(maxResults int64) *CreativesListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": A continuation
// token, used to page through ad clients. To retrieve the next page,
// set this parameter to the value of "nextPageToken" from the previous
// response.
func (c *CreativesListCall) PageToken(pageToken string) *CreativesListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// StatusFilter sets the optional parameter "statusFilter": When
// specified, only creatives having the given status are returned.
func (c *CreativesListCall) StatusFilter(statusFilter string) *CreativesListCall {
	c.opt_["statusFilter"] = statusFilter
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *CreativesListCall) Fields(s ...googleapi.Field) *CreativesListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *CreativesListCall) Do() (*CreativesList, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["accountId"]; ok {
		params.Set("accountId", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["buyerCreativeId"]; ok {
		params.Set("buyerCreativeId", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["statusFilter"]; ok {
		params.Set("statusFilter", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "creatives")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *CreativesList
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves a list of the authenticated user's active creatives. A creative will be available 30-40 minutes after submission.",
	//   "httpMethod": "GET",
	//   "id": "adexchangebuyer.creatives.list",
	//   "parameters": {
	//     "accountId": {
	//       "description": "When specified, only creatives for the given account ids are returned.",
	//       "format": "int32",
	//       "location": "query",
	//       "repeated": true,
	//       "type": "integer"
	//     },
	//     "buyerCreativeId": {
	//       "description": "When specified, only creatives for the given buyer creative ids are returned.",
	//       "location": "query",
	//       "repeated": true,
	//       "type": "string"
	//     },
	//     "maxResults": {
	//       "description": "Maximum number of entries returned on one result page. If not set, the default is 100. Optional.",
	//       "format": "uint32",
	//       "location": "query",
	//       "maximum": "1000",
	//       "minimum": "1",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "A continuation token, used to page through ad clients. To retrieve the next page, set this parameter to the value of \"nextPageToken\" from the previous response. Optional.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "statusFilter": {
	//       "description": "When specified, only creatives having the given status are returned.",
	//       "enum": [
	//         "approved",
	//         "disapproved",
	//         "not_checked"
	//       ],
	//       "enumDescriptions": [
	//         "Creatives which have been approved.",
	//         "Creatives which have been disapproved.",
	//         "Creatives whose status is not yet checked."
	//       ],
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "creatives",
	//   "response": {
	//     "$ref": "CreativesList"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.directDeals.get":

type DirectDealsGetCall struct {
	s    *Service
	id   int64
	opt_ map[string]interface{}
}

// Get: Gets one direct deal by ID.
func (r *DirectDealsService) Get(id int64) *DirectDealsGetCall {
	c := &DirectDealsGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.id = id
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DirectDealsGetCall) Fields(s ...googleapi.Field) *DirectDealsGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DirectDealsGetCall) Do() (*DirectDeal, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "directdeals/{id}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"id": strconv.FormatInt(c.id, 10),
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
	var ret *DirectDeal
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets one direct deal by ID.",
	//   "httpMethod": "GET",
	//   "id": "adexchangebuyer.directDeals.get",
	//   "parameterOrder": [
	//     "id"
	//   ],
	//   "parameters": {
	//     "id": {
	//       "description": "The direct deal id",
	//       "format": "int64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "directdeals/{id}",
	//   "response": {
	//     "$ref": "DirectDeal"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.directDeals.list":

type DirectDealsListCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// List: Retrieves the authenticated user's list of direct deals.
func (r *DirectDealsService) List() *DirectDealsListCall {
	c := &DirectDealsListCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *DirectDealsListCall) Fields(s ...googleapi.Field) *DirectDealsListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *DirectDealsListCall) Do() (*DirectDealsList, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "directdeals")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *DirectDealsList
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves the authenticated user's list of direct deals.",
	//   "httpMethod": "GET",
	//   "id": "adexchangebuyer.directDeals.list",
	//   "path": "directdeals",
	//   "response": {
	//     "$ref": "DirectDealsList"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.performanceReport.list":

type PerformanceReportListCall struct {
	s             *Service
	accountId     int64
	endDateTime   string
	startDateTime string
	opt_          map[string]interface{}
}

// List: Retrieves the authenticated user's list of performance metrics.
func (r *PerformanceReportService) List(accountId int64, endDateTime string, startDateTime string) *PerformanceReportListCall {
	c := &PerformanceReportListCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.endDateTime = endDateTime
	c.startDateTime = startDateTime
	return c
}

// MaxResults sets the optional parameter "maxResults": Maximum number
// of entries returned on one result page. If not set, the default is
// 100.
func (c *PerformanceReportListCall) MaxResults(maxResults int64) *PerformanceReportListCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": A continuation
// token, used to page through performance reports. To retrieve the next
// page, set this parameter to the value of "nextPageToken" from the
// previous response.
func (c *PerformanceReportListCall) PageToken(pageToken string) *PerformanceReportListCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PerformanceReportListCall) Fields(s ...googleapi.Field) *PerformanceReportListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PerformanceReportListCall) Do() (*PerformanceReportList, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	params.Set("accountId", fmt.Sprintf("%v", c.accountId))
	params.Set("endDateTime", fmt.Sprintf("%v", c.endDateTime))
	params.Set("startDateTime", fmt.Sprintf("%v", c.startDateTime))
	if v, ok := c.opt_["maxResults"]; ok {
		params.Set("maxResults", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["pageToken"]; ok {
		params.Set("pageToken", fmt.Sprintf("%v", v))
	}
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "performancereport")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("User-Agent", "google-api-go-client/0.5")
	res, err := c.s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	var ret *PerformanceReportList
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves the authenticated user's list of performance metrics.",
	//   "httpMethod": "GET",
	//   "id": "adexchangebuyer.performanceReport.list",
	//   "parameterOrder": [
	//     "accountId",
	//     "endDateTime",
	//     "startDateTime"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The account id to get the reports.",
	//       "format": "int64",
	//       "location": "query",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "endDateTime": {
	//       "description": "The end time of the report in ISO 8601 timestamp format using UTC.",
	//       "location": "query",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "maxResults": {
	//       "description": "Maximum number of entries returned on one result page. If not set, the default is 100. Optional.",
	//       "format": "uint32",
	//       "location": "query",
	//       "maximum": "1000",
	//       "minimum": "1",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "A continuation token, used to page through performance reports. To retrieve the next page, set this parameter to the value of \"nextPageToken\" from the previous response. Optional.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "startDateTime": {
	//       "description": "The start time of the report in ISO 8601 timestamp format using UTC.",
	//       "location": "query",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "performancereport",
	//   "response": {
	//     "$ref": "PerformanceReportList"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.pretargetingConfig.delete":

type PretargetingConfigDeleteCall struct {
	s         *Service
	accountId int64
	configId  int64
	opt_      map[string]interface{}
}

// Delete: Deletes an existing pretargeting config.
func (r *PretargetingConfigService) Delete(accountId int64, configId int64) *PretargetingConfigDeleteCall {
	c := &PretargetingConfigDeleteCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.configId = configId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PretargetingConfigDeleteCall) Fields(s ...googleapi.Field) *PretargetingConfigDeleteCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PretargetingConfigDeleteCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "pretargetingconfigs/{accountId}/{configId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId": strconv.FormatInt(c.accountId, 10),
		"configId":  strconv.FormatInt(c.configId, 10),
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
	//   "description": "Deletes an existing pretargeting config.",
	//   "httpMethod": "DELETE",
	//   "id": "adexchangebuyer.pretargetingConfig.delete",
	//   "parameterOrder": [
	//     "accountId",
	//     "configId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The account id to delete the pretargeting config for.",
	//       "format": "int64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "configId": {
	//       "description": "The specific id of the configuration to delete.",
	//       "format": "int64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "pretargetingconfigs/{accountId}/{configId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.pretargetingConfig.get":

type PretargetingConfigGetCall struct {
	s         *Service
	accountId int64
	configId  int64
	opt_      map[string]interface{}
}

// Get: Gets a specific pretargeting configuration
func (r *PretargetingConfigService) Get(accountId int64, configId int64) *PretargetingConfigGetCall {
	c := &PretargetingConfigGetCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.configId = configId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PretargetingConfigGetCall) Fields(s ...googleapi.Field) *PretargetingConfigGetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PretargetingConfigGetCall) Do() (*PretargetingConfig, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "pretargetingconfigs/{accountId}/{configId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId": strconv.FormatInt(c.accountId, 10),
		"configId":  strconv.FormatInt(c.configId, 10),
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
	var ret *PretargetingConfig
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets a specific pretargeting configuration",
	//   "httpMethod": "GET",
	//   "id": "adexchangebuyer.pretargetingConfig.get",
	//   "parameterOrder": [
	//     "accountId",
	//     "configId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The account id to get the pretargeting config for.",
	//       "format": "int64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "configId": {
	//       "description": "The specific id of the configuration to retrieve.",
	//       "format": "int64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "pretargetingconfigs/{accountId}/{configId}",
	//   "response": {
	//     "$ref": "PretargetingConfig"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.pretargetingConfig.insert":

type PretargetingConfigInsertCall struct {
	s                  *Service
	accountId          int64
	pretargetingconfig *PretargetingConfig
	opt_               map[string]interface{}
}

// Insert: Inserts a new pretargeting configuration.
func (r *PretargetingConfigService) Insert(accountId int64, pretargetingconfig *PretargetingConfig) *PretargetingConfigInsertCall {
	c := &PretargetingConfigInsertCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.pretargetingconfig = pretargetingconfig
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PretargetingConfigInsertCall) Fields(s ...googleapi.Field) *PretargetingConfigInsertCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PretargetingConfigInsertCall) Do() (*PretargetingConfig, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.pretargetingconfig)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "pretargetingconfigs/{accountId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId": strconv.FormatInt(c.accountId, 10),
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
	var ret *PretargetingConfig
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Inserts a new pretargeting configuration.",
	//   "httpMethod": "POST",
	//   "id": "adexchangebuyer.pretargetingConfig.insert",
	//   "parameterOrder": [
	//     "accountId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The account id to insert the pretargeting config for.",
	//       "format": "int64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "pretargetingconfigs/{accountId}",
	//   "request": {
	//     "$ref": "PretargetingConfig"
	//   },
	//   "response": {
	//     "$ref": "PretargetingConfig"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.pretargetingConfig.list":

type PretargetingConfigListCall struct {
	s         *Service
	accountId int64
	opt_      map[string]interface{}
}

// List: Retrieves a list of the authenticated user's pretargeting
// configurations.
func (r *PretargetingConfigService) List(accountId int64) *PretargetingConfigListCall {
	c := &PretargetingConfigListCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PretargetingConfigListCall) Fields(s ...googleapi.Field) *PretargetingConfigListCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PretargetingConfigListCall) Do() (*PretargetingConfigList, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "pretargetingconfigs/{accountId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId": strconv.FormatInt(c.accountId, 10),
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
	var ret *PretargetingConfigList
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Retrieves a list of the authenticated user's pretargeting configurations.",
	//   "httpMethod": "GET",
	//   "id": "adexchangebuyer.pretargetingConfig.list",
	//   "parameterOrder": [
	//     "accountId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The account id to get the pretargeting configs for.",
	//       "format": "int64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "pretargetingconfigs/{accountId}",
	//   "response": {
	//     "$ref": "PretargetingConfigList"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.pretargetingConfig.patch":

type PretargetingConfigPatchCall struct {
	s                  *Service
	accountId          int64
	configId           int64
	pretargetingconfig *PretargetingConfig
	opt_               map[string]interface{}
}

// Patch: Updates an existing pretargeting config. This method supports
// patch semantics.
func (r *PretargetingConfigService) Patch(accountId int64, configId int64, pretargetingconfig *PretargetingConfig) *PretargetingConfigPatchCall {
	c := &PretargetingConfigPatchCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.configId = configId
	c.pretargetingconfig = pretargetingconfig
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PretargetingConfigPatchCall) Fields(s ...googleapi.Field) *PretargetingConfigPatchCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PretargetingConfigPatchCall) Do() (*PretargetingConfig, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.pretargetingconfig)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "pretargetingconfigs/{accountId}/{configId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PATCH", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId": strconv.FormatInt(c.accountId, 10),
		"configId":  strconv.FormatInt(c.configId, 10),
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
	var ret *PretargetingConfig
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates an existing pretargeting config. This method supports patch semantics.",
	//   "httpMethod": "PATCH",
	//   "id": "adexchangebuyer.pretargetingConfig.patch",
	//   "parameterOrder": [
	//     "accountId",
	//     "configId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The account id to update the pretargeting config for.",
	//       "format": "int64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "configId": {
	//       "description": "The specific id of the configuration to update.",
	//       "format": "int64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "pretargetingconfigs/{accountId}/{configId}",
	//   "request": {
	//     "$ref": "PretargetingConfig"
	//   },
	//   "response": {
	//     "$ref": "PretargetingConfig"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}

// method id "adexchangebuyer.pretargetingConfig.update":

type PretargetingConfigUpdateCall struct {
	s                  *Service
	accountId          int64
	configId           int64
	pretargetingconfig *PretargetingConfig
	opt_               map[string]interface{}
}

// Update: Updates an existing pretargeting config.
func (r *PretargetingConfigService) Update(accountId int64, configId int64, pretargetingconfig *PretargetingConfig) *PretargetingConfigUpdateCall {
	c := &PretargetingConfigUpdateCall{s: r.s, opt_: make(map[string]interface{})}
	c.accountId = accountId
	c.configId = configId
	c.pretargetingconfig = pretargetingconfig
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PretargetingConfigUpdateCall) Fields(s ...googleapi.Field) *PretargetingConfigUpdateCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PretargetingConfigUpdateCall) Do() (*PretargetingConfig, error) {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.pretargetingconfig)
	if err != nil {
		return nil, err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "pretargetingconfigs/{accountId}/{configId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("PUT", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"accountId": strconv.FormatInt(c.accountId, 10),
		"configId":  strconv.FormatInt(c.configId, 10),
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
	var ret *PretargetingConfig
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Updates an existing pretargeting config.",
	//   "httpMethod": "PUT",
	//   "id": "adexchangebuyer.pretargetingConfig.update",
	//   "parameterOrder": [
	//     "accountId",
	//     "configId"
	//   ],
	//   "parameters": {
	//     "accountId": {
	//       "description": "The account id to update the pretargeting config for.",
	//       "format": "int64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "configId": {
	//       "description": "The specific id of the configuration to update.",
	//       "format": "int64",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "pretargetingconfigs/{accountId}/{configId}",
	//   "request": {
	//     "$ref": "PretargetingConfig"
	//   },
	//   "response": {
	//     "$ref": "PretargetingConfig"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/adexchange.buyer"
	//   ]
	// }

}
