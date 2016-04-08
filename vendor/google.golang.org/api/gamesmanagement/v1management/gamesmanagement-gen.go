// Package gamesmanagement provides access to the Google Play Game Services Management API.
//
// See https://developers.google.com/games/services
//
// Usage example:
//
//   import "google.golang.org/api/gamesmanagement/v1management"
//   ...
//   gamesmanagementService, err := gamesmanagement.New(oauthHttpClient)
package gamesmanagement

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

const apiId = "gamesManagement:v1management"
const apiName = "gamesManagement"
const apiVersion = "v1management"
const basePath = "https://www.googleapis.com/games/v1management/"

// OAuth2 scopes used by this API.
const (
	// Share your Google+ profile information and view and manage your game
	// activity
	GamesScope = "https://www.googleapis.com/auth/games"

	// Know your basic profile info and list of people in your circles.
	PlusLoginScope = "https://www.googleapis.com/auth/plus.login"
)

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Achievements = NewAchievementsService(s)
	s.Applications = NewApplicationsService(s)
	s.Events = NewEventsService(s)
	s.Players = NewPlayersService(s)
	s.Quests = NewQuestsService(s)
	s.Rooms = NewRoomsService(s)
	s.Scores = NewScoresService(s)
	s.TurnBasedMatches = NewTurnBasedMatchesService(s)
	return s, nil
}

type Service struct {
	client   *http.Client
	BasePath string // API endpoint base URL

	Achievements *AchievementsService

	Applications *ApplicationsService

	Events *EventsService

	Players *PlayersService

	Quests *QuestsService

	Rooms *RoomsService

	Scores *ScoresService

	TurnBasedMatches *TurnBasedMatchesService
}

func NewAchievementsService(s *Service) *AchievementsService {
	rs := &AchievementsService{s: s}
	return rs
}

type AchievementsService struct {
	s *Service
}

func NewApplicationsService(s *Service) *ApplicationsService {
	rs := &ApplicationsService{s: s}
	return rs
}

type ApplicationsService struct {
	s *Service
}

func NewEventsService(s *Service) *EventsService {
	rs := &EventsService{s: s}
	return rs
}

type EventsService struct {
	s *Service
}

func NewPlayersService(s *Service) *PlayersService {
	rs := &PlayersService{s: s}
	return rs
}

type PlayersService struct {
	s *Service
}

func NewQuestsService(s *Service) *QuestsService {
	rs := &QuestsService{s: s}
	return rs
}

type QuestsService struct {
	s *Service
}

func NewRoomsService(s *Service) *RoomsService {
	rs := &RoomsService{s: s}
	return rs
}

type RoomsService struct {
	s *Service
}

func NewScoresService(s *Service) *ScoresService {
	rs := &ScoresService{s: s}
	return rs
}

type ScoresService struct {
	s *Service
}

func NewTurnBasedMatchesService(s *Service) *TurnBasedMatchesService {
	rs := &TurnBasedMatchesService{s: s}
	return rs
}

type TurnBasedMatchesService struct {
	s *Service
}

type AchievementResetAllResponse struct {
	// Kind: Uniquely identifies the type of this resource. Value is always
	// the fixed string gamesManagement#achievementResetAllResponse.
	Kind string `json:"kind,omitempty"`

	// Results: The achievement reset results.
	Results []*AchievementResetResponse `json:"results,omitempty"`
}

type AchievementResetMultipleForAllRequest struct {
	// Achievement_ids: The IDs of achievements to reset.
	Achievement_ids []string `json:"achievement_ids,omitempty"`

	// Kind: Uniquely identifies the type of this resource. Value is always
	// the fixed string
	// gamesManagement#achievementResetMultipleForAllRequest.
	Kind string `json:"kind,omitempty"`
}

type AchievementResetResponse struct {
	// CurrentState: The current state of the achievement. This is the same
	// as the initial state of the achievement.
	// Possible values are:
	// -
	// "HIDDEN"- Achievement is hidden.
	// - "REVEALED" - Achievement is
	// revealed.
	// - "UNLOCKED" - Achievement is unlocked.
	CurrentState string `json:"currentState,omitempty"`

	// DefinitionId: The ID of an achievement for which player state has
	// been updated.
	DefinitionId string `json:"definitionId,omitempty"`

	// Kind: Uniquely identifies the type of this resource. Value is always
	// the fixed string gamesManagement#achievementResetResponse.
	Kind string `json:"kind,omitempty"`

	// UpdateOccurred: Flag to indicate if the requested update actually
	// occurred.
	UpdateOccurred bool `json:"updateOccurred,omitempty"`
}

type EventsResetMultipleForAllRequest struct {
	// Event_ids: The IDs of events to reset.
	Event_ids []string `json:"event_ids,omitempty"`

	// Kind: Uniquely identifies the type of this resource. Value is always
	// the fixed string gamesManagement#eventsResetMultipleForAllRequest.
	Kind string `json:"kind,omitempty"`
}

type GamesPlayedResource struct {
	// AutoMatched: True if the player was auto-matched with the currently
	// authenticated user.
	AutoMatched bool `json:"autoMatched,omitempty"`

	// TimeMillis: The last time the player played the game in milliseconds
	// since the epoch in UTC.
	TimeMillis int64 `json:"timeMillis,omitempty,string"`
}

type GamesPlayerExperienceInfoResource struct {
	// CurrentExperiencePoints: The current number of experience points for
	// the player.
	CurrentExperiencePoints int64 `json:"currentExperiencePoints,omitempty,string"`

	// CurrentLevel: The current level of the player.
	CurrentLevel *GamesPlayerLevelResource `json:"currentLevel,omitempty"`

	// LastLevelUpTimestampMillis: The timestamp when the player was leveled
	// up, in millis since Unix epoch UTC.
	LastLevelUpTimestampMillis int64 `json:"lastLevelUpTimestampMillis,omitempty,string"`

	// NextLevel: The next level of the player. If the current level is the
	// maximum level, this should be same as the current level.
	NextLevel *GamesPlayerLevelResource `json:"nextLevel,omitempty"`
}

type GamesPlayerLevelResource struct {
	// Level: The level for the user.
	Level int64 `json:"level,omitempty"`

	// MaxExperiencePoints: The maximum experience points for this level.
	MaxExperiencePoints int64 `json:"maxExperiencePoints,omitempty,string"`

	// MinExperiencePoints: The minimum experience points for this level.
	MinExperiencePoints int64 `json:"minExperiencePoints,omitempty,string"`
}

type HiddenPlayer struct {
	// HiddenTimeMillis: The time this player was hidden.
	HiddenTimeMillis int64 `json:"hiddenTimeMillis,omitempty,string"`

	// Kind: Uniquely identifies the type of this resource. Value is always
	// the fixed string gamesManagement#hiddenPlayer.
	Kind string `json:"kind,omitempty"`

	// Player: The player information.
	Player *Player `json:"player,omitempty"`
}

type HiddenPlayerList struct {
	// Items: The players.
	Items []*HiddenPlayer `json:"items,omitempty"`

	// Kind: Uniquely identifies the type of this resource. Value is always
	// the fixed string gamesManagement#hiddenPlayerList.
	Kind string `json:"kind,omitempty"`

	// NextPageToken: The pagination token for the next page of results.
	NextPageToken string `json:"nextPageToken,omitempty"`
}

type Player struct {
	// AvatarImageUrl: The base URL for the image that represents the
	// player.
	AvatarImageUrl string `json:"avatarImageUrl,omitempty"`

	// DisplayName: The name to display for the player.
	DisplayName string `json:"displayName,omitempty"`

	// ExperienceInfo: An object to represent Play Game experience
	// information for the player.
	ExperienceInfo *GamesPlayerExperienceInfoResource `json:"experienceInfo,omitempty"`

	// Kind: Uniquely identifies the type of this resource. Value is always
	// the fixed string gamesManagement#player.
	Kind string `json:"kind,omitempty"`

	// LastPlayedWith: Details about the last time this player played a
	// multiplayer game with the currently authenticated player. Populated
	// for PLAYED_WITH player collection members.
	LastPlayedWith *GamesPlayedResource `json:"lastPlayedWith,omitempty"`

	// Name: An object representation of the individual components of the
	// player's name. For some players, these fields may not be present.
	Name *PlayerName `json:"name,omitempty"`

	// PlayerId: The ID of the player.
	PlayerId string `json:"playerId,omitempty"`

	// Title: The player's title rewarded for their game activities.
	Title string `json:"title,omitempty"`
}

type PlayerName struct {
	// FamilyName: The family name of this player. In some places, this is
	// known as the last name.
	FamilyName string `json:"familyName,omitempty"`

	// GivenName: The given name of this player. In some places, this is
	// known as the first name.
	GivenName string `json:"givenName,omitempty"`
}

type PlayerScoreResetAllResponse struct {
	// Kind: Uniquely identifies the type of this resource. Value is always
	// the fixed string gamesManagement#playerScoreResetResponse.
	Kind string `json:"kind,omitempty"`

	// Results: The leaderboard reset results.
	Results []*PlayerScoreResetResponse `json:"results,omitempty"`
}

type PlayerScoreResetResponse struct {
	// DefinitionId: The ID of an leaderboard for which player state has
	// been updated.
	DefinitionId string `json:"definitionId,omitempty"`

	// Kind: Uniquely identifies the type of this resource. Value is always
	// the fixed string gamesManagement#playerScoreResetResponse.
	Kind string `json:"kind,omitempty"`

	// ResetScoreTimeSpans: The time spans of the updated score.
	// Possible
	// values are:
	// - "ALL_TIME" - The score is an all-time score.
	// -
	// "WEEKLY" - The score is a weekly score.
	// - "DAILY" - The score is a
	// daily score.
	ResetScoreTimeSpans []string `json:"resetScoreTimeSpans,omitempty"`
}

type QuestsResetMultipleForAllRequest struct {
	// Kind: Uniquely identifies the type of this resource. Value is always
	// the fixed string gamesManagement#questsResetMultipleForAllRequest.
	Kind string `json:"kind,omitempty"`

	// Quest_ids: The IDs of quests to reset.
	Quest_ids []string `json:"quest_ids,omitempty"`
}

type ScoresResetMultipleForAllRequest struct {
	// Kind: Uniquely identifies the type of this resource. Value is always
	// the fixed string gamesManagement#scoresResetMultipleForAllRequest.
	Kind string `json:"kind,omitempty"`

	// Leaderboard_ids: The IDs of leaderboards to reset.
	Leaderboard_ids []string `json:"leaderboard_ids,omitempty"`
}

// method id "gamesManagement.achievements.reset":

type AchievementsResetCall struct {
	s             *Service
	achievementId string
	opt_          map[string]interface{}
}

// Reset: Resets the achievement with the given ID for the currently
// authenticated player. This method is only accessible to whitelisted
// tester accounts for your application.
func (r *AchievementsService) Reset(achievementId string) *AchievementsResetCall {
	c := &AchievementsResetCall{s: r.s, opt_: make(map[string]interface{})}
	c.achievementId = achievementId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AchievementsResetCall) Fields(s ...googleapi.Field) *AchievementsResetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AchievementsResetCall) Do() (*AchievementResetResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "achievements/{achievementId}/reset")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"achievementId": c.achievementId,
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
	var ret *AchievementResetResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Resets the achievement with the given ID for the currently authenticated player. This method is only accessible to whitelisted tester accounts for your application.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.achievements.reset",
	//   "parameterOrder": [
	//     "achievementId"
	//   ],
	//   "parameters": {
	//     "achievementId": {
	//       "description": "The ID of the achievement used by this method.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "achievements/{achievementId}/reset",
	//   "response": {
	//     "$ref": "AchievementResetResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.achievements.resetAll":

type AchievementsResetAllCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// ResetAll: Resets all achievements for the currently authenticated
// player for your application. This method is only accessible to
// whitelisted tester accounts for your application.
func (r *AchievementsService) ResetAll() *AchievementsResetAllCall {
	c := &AchievementsResetAllCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AchievementsResetAllCall) Fields(s ...googleapi.Field) *AchievementsResetAllCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AchievementsResetAllCall) Do() (*AchievementResetAllResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "achievements/reset")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
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
	var ret *AchievementResetAllResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Resets all achievements for the currently authenticated player for your application. This method is only accessible to whitelisted tester accounts for your application.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.achievements.resetAll",
	//   "path": "achievements/reset",
	//   "response": {
	//     "$ref": "AchievementResetAllResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.achievements.resetAllForAllPlayers":

type AchievementsResetAllForAllPlayersCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// ResetAllForAllPlayers: Resets all draft achievements for all players.
// This method is only available to user accounts for your developer
// console.
func (r *AchievementsService) ResetAllForAllPlayers() *AchievementsResetAllForAllPlayersCall {
	c := &AchievementsResetAllForAllPlayersCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AchievementsResetAllForAllPlayersCall) Fields(s ...googleapi.Field) *AchievementsResetAllForAllPlayersCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AchievementsResetAllForAllPlayersCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "achievements/resetAllForAllPlayers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
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
	//   "description": "Resets all draft achievements for all players. This method is only available to user accounts for your developer console.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.achievements.resetAllForAllPlayers",
	//   "path": "achievements/resetAllForAllPlayers",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.achievements.resetForAllPlayers":

type AchievementsResetForAllPlayersCall struct {
	s             *Service
	achievementId string
	opt_          map[string]interface{}
}

// ResetForAllPlayers: Resets the achievement with the given ID for all
// players. This method is only available to user accounts for your
// developer console. Only draft achievements can be reset.
func (r *AchievementsService) ResetForAllPlayers(achievementId string) *AchievementsResetForAllPlayersCall {
	c := &AchievementsResetForAllPlayersCall{s: r.s, opt_: make(map[string]interface{})}
	c.achievementId = achievementId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AchievementsResetForAllPlayersCall) Fields(s ...googleapi.Field) *AchievementsResetForAllPlayersCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AchievementsResetForAllPlayersCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "achievements/{achievementId}/resetForAllPlayers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"achievementId": c.achievementId,
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
	//   "description": "Resets the achievement with the given ID for all players. This method is only available to user accounts for your developer console. Only draft achievements can be reset.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.achievements.resetForAllPlayers",
	//   "parameterOrder": [
	//     "achievementId"
	//   ],
	//   "parameters": {
	//     "achievementId": {
	//       "description": "The ID of the achievement used by this method.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "achievements/{achievementId}/resetForAllPlayers",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.achievements.resetMultipleForAllPlayers":

type AchievementsResetMultipleForAllPlayersCall struct {
	s                                     *Service
	achievementresetmultipleforallrequest *AchievementResetMultipleForAllRequest
	opt_                                  map[string]interface{}
}

// ResetMultipleForAllPlayers: Resets achievements with the given IDs
// for all players. This method is only available to user accounts for
// your developer console. Only draft achievements may be reset.
func (r *AchievementsService) ResetMultipleForAllPlayers(achievementresetmultipleforallrequest *AchievementResetMultipleForAllRequest) *AchievementsResetMultipleForAllPlayersCall {
	c := &AchievementsResetMultipleForAllPlayersCall{s: r.s, opt_: make(map[string]interface{})}
	c.achievementresetmultipleforallrequest = achievementresetmultipleforallrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AchievementsResetMultipleForAllPlayersCall) Fields(s ...googleapi.Field) *AchievementsResetMultipleForAllPlayersCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *AchievementsResetMultipleForAllPlayersCall) Do() error {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.achievementresetmultipleforallrequest)
	if err != nil {
		return err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "achievements/resetMultipleForAllPlayers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
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
	//   "description": "Resets achievements with the given IDs for all players. This method is only available to user accounts for your developer console. Only draft achievements may be reset.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.achievements.resetMultipleForAllPlayers",
	//   "path": "achievements/resetMultipleForAllPlayers",
	//   "request": {
	//     "$ref": "AchievementResetMultipleForAllRequest"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.applications.listHidden":

type ApplicationsListHiddenCall struct {
	s             *Service
	applicationId string
	opt_          map[string]interface{}
}

// ListHidden: Get the list of players hidden from the given
// application. This method is only available to user accounts for your
// developer console.
func (r *ApplicationsService) ListHidden(applicationId string) *ApplicationsListHiddenCall {
	c := &ApplicationsListHiddenCall{s: r.s, opt_: make(map[string]interface{})}
	c.applicationId = applicationId
	return c
}

// MaxResults sets the optional parameter "maxResults": The maximum
// number of player resources to return in the response, used for
// paging. For any response, the actual number of player resources
// returned may be less than the specified maxResults.
func (c *ApplicationsListHiddenCall) MaxResults(maxResults int64) *ApplicationsListHiddenCall {
	c.opt_["maxResults"] = maxResults
	return c
}

// PageToken sets the optional parameter "pageToken": The token returned
// by the previous request.
func (c *ApplicationsListHiddenCall) PageToken(pageToken string) *ApplicationsListHiddenCall {
	c.opt_["pageToken"] = pageToken
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ApplicationsListHiddenCall) Fields(s ...googleapi.Field) *ApplicationsListHiddenCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ApplicationsListHiddenCall) Do() (*HiddenPlayerList, error) {
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
	urls := googleapi.ResolveRelative(c.s.BasePath, "applications/{applicationId}/players/hidden")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("GET", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"applicationId": c.applicationId,
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
	var ret *HiddenPlayerList
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Get the list of players hidden from the given application. This method is only available to user accounts for your developer console.",
	//   "httpMethod": "GET",
	//   "id": "gamesManagement.applications.listHidden",
	//   "parameterOrder": [
	//     "applicationId"
	//   ],
	//   "parameters": {
	//     "applicationId": {
	//       "description": "The application ID from the Google Play developer console.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "maxResults": {
	//       "description": "The maximum number of player resources to return in the response, used for paging. For any response, the actual number of player resources returned may be less than the specified maxResults.",
	//       "format": "int32",
	//       "location": "query",
	//       "maximum": "15",
	//       "minimum": "1",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "The token returned by the previous request.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "applications/{applicationId}/players/hidden",
	//   "response": {
	//     "$ref": "HiddenPlayerList"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.events.reset":

type EventsResetCall struct {
	s       *Service
	eventId string
	opt_    map[string]interface{}
}

// Reset: Resets all player progress on the event with the given ID for
// the currently authenticated player. This method is only accessible to
// whitelisted tester accounts for your application. All quests for this
// player that use the event will also be reset.
func (r *EventsService) Reset(eventId string) *EventsResetCall {
	c := &EventsResetCall{s: r.s, opt_: make(map[string]interface{})}
	c.eventId = eventId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EventsResetCall) Fields(s ...googleapi.Field) *EventsResetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EventsResetCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "events/{eventId}/reset")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"eventId": c.eventId,
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
	//   "description": "Resets all player progress on the event with the given ID for the currently authenticated player. This method is only accessible to whitelisted tester accounts for your application. All quests for this player that use the event will also be reset.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.events.reset",
	//   "parameterOrder": [
	//     "eventId"
	//   ],
	//   "parameters": {
	//     "eventId": {
	//       "description": "The ID of the event.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "events/{eventId}/reset",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.events.resetAll":

type EventsResetAllCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// ResetAll: Resets all player progress on all events for the currently
// authenticated player. This method is only accessible to whitelisted
// tester accounts for your application. All quests for this player will
// also be reset.
func (r *EventsService) ResetAll() *EventsResetAllCall {
	c := &EventsResetAllCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EventsResetAllCall) Fields(s ...googleapi.Field) *EventsResetAllCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EventsResetAllCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "events/reset")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
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
	//   "description": "Resets all player progress on all events for the currently authenticated player. This method is only accessible to whitelisted tester accounts for your application. All quests for this player will also be reset.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.events.resetAll",
	//   "path": "events/reset",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.events.resetAllForAllPlayers":

type EventsResetAllForAllPlayersCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// ResetAllForAllPlayers: Resets all draft events for all players. This
// method is only available to user accounts for your developer console.
// All quests that use any of these events will also be reset.
func (r *EventsService) ResetAllForAllPlayers() *EventsResetAllForAllPlayersCall {
	c := &EventsResetAllForAllPlayersCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EventsResetAllForAllPlayersCall) Fields(s ...googleapi.Field) *EventsResetAllForAllPlayersCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EventsResetAllForAllPlayersCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "events/resetAllForAllPlayers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
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
	//   "description": "Resets all draft events for all players. This method is only available to user accounts for your developer console. All quests that use any of these events will also be reset.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.events.resetAllForAllPlayers",
	//   "path": "events/resetAllForAllPlayers",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.events.resetForAllPlayers":

type EventsResetForAllPlayersCall struct {
	s       *Service
	eventId string
	opt_    map[string]interface{}
}

// ResetForAllPlayers: Resets the event with the given ID for all
// players. This method is only available to user accounts for your
// developer console. Only draft events can be reset. All quests that
// use the event will also be reset.
func (r *EventsService) ResetForAllPlayers(eventId string) *EventsResetForAllPlayersCall {
	c := &EventsResetForAllPlayersCall{s: r.s, opt_: make(map[string]interface{})}
	c.eventId = eventId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EventsResetForAllPlayersCall) Fields(s ...googleapi.Field) *EventsResetForAllPlayersCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EventsResetForAllPlayersCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "events/{eventId}/resetForAllPlayers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"eventId": c.eventId,
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
	//   "description": "Resets the event with the given ID for all players. This method is only available to user accounts for your developer console. Only draft events can be reset. All quests that use the event will also be reset.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.events.resetForAllPlayers",
	//   "parameterOrder": [
	//     "eventId"
	//   ],
	//   "parameters": {
	//     "eventId": {
	//       "description": "The ID of the event.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "events/{eventId}/resetForAllPlayers",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.events.resetMultipleForAllPlayers":

type EventsResetMultipleForAllPlayersCall struct {
	s                                *Service
	eventsresetmultipleforallrequest *EventsResetMultipleForAllRequest
	opt_                             map[string]interface{}
}

// ResetMultipleForAllPlayers: Resets events with the given IDs for all
// players. This method is only available to user accounts for your
// developer console. Only draft events may be reset. All quests that
// use any of the events will also be reset.
func (r *EventsService) ResetMultipleForAllPlayers(eventsresetmultipleforallrequest *EventsResetMultipleForAllRequest) *EventsResetMultipleForAllPlayersCall {
	c := &EventsResetMultipleForAllPlayersCall{s: r.s, opt_: make(map[string]interface{})}
	c.eventsresetmultipleforallrequest = eventsresetmultipleforallrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *EventsResetMultipleForAllPlayersCall) Fields(s ...googleapi.Field) *EventsResetMultipleForAllPlayersCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *EventsResetMultipleForAllPlayersCall) Do() error {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.eventsresetmultipleforallrequest)
	if err != nil {
		return err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "events/resetMultipleForAllPlayers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
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
	//   "description": "Resets events with the given IDs for all players. This method is only available to user accounts for your developer console. Only draft events may be reset. All quests that use any of the events will also be reset.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.events.resetMultipleForAllPlayers",
	//   "path": "events/resetMultipleForAllPlayers",
	//   "request": {
	//     "$ref": "EventsResetMultipleForAllRequest"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.players.hide":

type PlayersHideCall struct {
	s             *Service
	applicationId string
	playerId      string
	opt_          map[string]interface{}
}

// Hide: Hide the given player's leaderboard scores from the given
// application. This method is only available to user accounts for your
// developer console.
func (r *PlayersService) Hide(applicationId string, playerId string) *PlayersHideCall {
	c := &PlayersHideCall{s: r.s, opt_: make(map[string]interface{})}
	c.applicationId = applicationId
	c.playerId = playerId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PlayersHideCall) Fields(s ...googleapi.Field) *PlayersHideCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PlayersHideCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "applications/{applicationId}/players/hidden/{playerId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"applicationId": c.applicationId,
		"playerId":      c.playerId,
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
	//   "description": "Hide the given player's leaderboard scores from the given application. This method is only available to user accounts for your developer console.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.players.hide",
	//   "parameterOrder": [
	//     "applicationId",
	//     "playerId"
	//   ],
	//   "parameters": {
	//     "applicationId": {
	//       "description": "The application ID from the Google Play developer console.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "playerId": {
	//       "description": "A player ID. A value of me may be used in place of the authenticated player's ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "applications/{applicationId}/players/hidden/{playerId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.players.unhide":

type PlayersUnhideCall struct {
	s             *Service
	applicationId string
	playerId      string
	opt_          map[string]interface{}
}

// Unhide: Unhide the given player's leaderboard scores from the given
// application. This method is only available to user accounts for your
// developer console.
func (r *PlayersService) Unhide(applicationId string, playerId string) *PlayersUnhideCall {
	c := &PlayersUnhideCall{s: r.s, opt_: make(map[string]interface{})}
	c.applicationId = applicationId
	c.playerId = playerId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *PlayersUnhideCall) Fields(s ...googleapi.Field) *PlayersUnhideCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *PlayersUnhideCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "applications/{applicationId}/players/hidden/{playerId}")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("DELETE", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"applicationId": c.applicationId,
		"playerId":      c.playerId,
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
	//   "description": "Unhide the given player's leaderboard scores from the given application. This method is only available to user accounts for your developer console.",
	//   "httpMethod": "DELETE",
	//   "id": "gamesManagement.players.unhide",
	//   "parameterOrder": [
	//     "applicationId",
	//     "playerId"
	//   ],
	//   "parameters": {
	//     "applicationId": {
	//       "description": "The application ID from the Google Play developer console.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "playerId": {
	//       "description": "A player ID. A value of me may be used in place of the authenticated player's ID.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "applications/{applicationId}/players/hidden/{playerId}",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.quests.reset":

type QuestsResetCall struct {
	s       *Service
	questId string
	opt_    map[string]interface{}
}

// Reset: Resets all player progress on the quest with the given ID for
// the currently authenticated player. This method is only accessible to
// whitelisted tester accounts for your application.
func (r *QuestsService) Reset(questId string) *QuestsResetCall {
	c := &QuestsResetCall{s: r.s, opt_: make(map[string]interface{})}
	c.questId = questId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *QuestsResetCall) Fields(s ...googleapi.Field) *QuestsResetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *QuestsResetCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "quests/{questId}/reset")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"questId": c.questId,
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
	//   "description": "Resets all player progress on the quest with the given ID for the currently authenticated player. This method is only accessible to whitelisted tester accounts for your application.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.quests.reset",
	//   "parameterOrder": [
	//     "questId"
	//   ],
	//   "parameters": {
	//     "questId": {
	//       "description": "The ID of the quest.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "quests/{questId}/reset",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.quests.resetAll":

type QuestsResetAllCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// ResetAll: Resets all player progress on all quests for the currently
// authenticated player. This method is only accessible to whitelisted
// tester accounts for your application.
func (r *QuestsService) ResetAll() *QuestsResetAllCall {
	c := &QuestsResetAllCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *QuestsResetAllCall) Fields(s ...googleapi.Field) *QuestsResetAllCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *QuestsResetAllCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "quests/reset")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
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
	//   "description": "Resets all player progress on all quests for the currently authenticated player. This method is only accessible to whitelisted tester accounts for your application.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.quests.resetAll",
	//   "path": "quests/reset",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.quests.resetAllForAllPlayers":

type QuestsResetAllForAllPlayersCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// ResetAllForAllPlayers: Resets all draft quests for all players. This
// method is only available to user accounts for your developer console.
func (r *QuestsService) ResetAllForAllPlayers() *QuestsResetAllForAllPlayersCall {
	c := &QuestsResetAllForAllPlayersCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *QuestsResetAllForAllPlayersCall) Fields(s ...googleapi.Field) *QuestsResetAllForAllPlayersCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *QuestsResetAllForAllPlayersCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "quests/resetAllForAllPlayers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
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
	//   "description": "Resets all draft quests for all players. This method is only available to user accounts for your developer console.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.quests.resetAllForAllPlayers",
	//   "path": "quests/resetAllForAllPlayers",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.quests.resetForAllPlayers":

type QuestsResetForAllPlayersCall struct {
	s       *Service
	questId string
	opt_    map[string]interface{}
}

// ResetForAllPlayers: Resets all player progress on the quest with the
// given ID for all players. This method is only available to user
// accounts for your developer console. Only draft quests can be reset.
func (r *QuestsService) ResetForAllPlayers(questId string) *QuestsResetForAllPlayersCall {
	c := &QuestsResetForAllPlayersCall{s: r.s, opt_: make(map[string]interface{})}
	c.questId = questId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *QuestsResetForAllPlayersCall) Fields(s ...googleapi.Field) *QuestsResetForAllPlayersCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *QuestsResetForAllPlayersCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "quests/{questId}/resetForAllPlayers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"questId": c.questId,
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
	//   "description": "Resets all player progress on the quest with the given ID for all players. This method is only available to user accounts for your developer console. Only draft quests can be reset.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.quests.resetForAllPlayers",
	//   "parameterOrder": [
	//     "questId"
	//   ],
	//   "parameters": {
	//     "questId": {
	//       "description": "The ID of the quest.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "quests/{questId}/resetForAllPlayers",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.quests.resetMultipleForAllPlayers":

type QuestsResetMultipleForAllPlayersCall struct {
	s                                *Service
	questsresetmultipleforallrequest *QuestsResetMultipleForAllRequest
	opt_                             map[string]interface{}
}

// ResetMultipleForAllPlayers: Resets quests with the given IDs for all
// players. This method is only available to user accounts for your
// developer console. Only draft quests may be reset.
func (r *QuestsService) ResetMultipleForAllPlayers(questsresetmultipleforallrequest *QuestsResetMultipleForAllRequest) *QuestsResetMultipleForAllPlayersCall {
	c := &QuestsResetMultipleForAllPlayersCall{s: r.s, opt_: make(map[string]interface{})}
	c.questsresetmultipleforallrequest = questsresetmultipleforallrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *QuestsResetMultipleForAllPlayersCall) Fields(s ...googleapi.Field) *QuestsResetMultipleForAllPlayersCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *QuestsResetMultipleForAllPlayersCall) Do() error {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.questsresetmultipleforallrequest)
	if err != nil {
		return err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "quests/resetMultipleForAllPlayers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
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
	//   "description": "Resets quests with the given IDs for all players. This method is only available to user accounts for your developer console. Only draft quests may be reset.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.quests.resetMultipleForAllPlayers",
	//   "path": "quests/resetMultipleForAllPlayers",
	//   "request": {
	//     "$ref": "QuestsResetMultipleForAllRequest"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.rooms.reset":

type RoomsResetCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// Reset: Reset all rooms for the currently authenticated player for
// your application. This method is only accessible to whitelisted
// tester accounts for your application.
func (r *RoomsService) Reset() *RoomsResetCall {
	c := &RoomsResetCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *RoomsResetCall) Fields(s ...googleapi.Field) *RoomsResetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *RoomsResetCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "rooms/reset")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
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
	//   "description": "Reset all rooms for the currently authenticated player for your application. This method is only accessible to whitelisted tester accounts for your application.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.rooms.reset",
	//   "path": "rooms/reset",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.rooms.resetForAllPlayers":

type RoomsResetForAllPlayersCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// ResetForAllPlayers: Deletes rooms where the only room participants
// are from whitelisted tester accounts for your application. This
// method is only available to user accounts for your developer console.
func (r *RoomsService) ResetForAllPlayers() *RoomsResetForAllPlayersCall {
	c := &RoomsResetForAllPlayersCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *RoomsResetForAllPlayersCall) Fields(s ...googleapi.Field) *RoomsResetForAllPlayersCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *RoomsResetForAllPlayersCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "rooms/resetForAllPlayers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
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
	//   "description": "Deletes rooms where the only room participants are from whitelisted tester accounts for your application. This method is only available to user accounts for your developer console.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.rooms.resetForAllPlayers",
	//   "path": "rooms/resetForAllPlayers",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.scores.reset":

type ScoresResetCall struct {
	s             *Service
	leaderboardId string
	opt_          map[string]interface{}
}

// Reset: Resets scores for the leaderboard with the given ID for the
// currently authenticated player. This method is only accessible to
// whitelisted tester accounts for your application.
func (r *ScoresService) Reset(leaderboardId string) *ScoresResetCall {
	c := &ScoresResetCall{s: r.s, opt_: make(map[string]interface{})}
	c.leaderboardId = leaderboardId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ScoresResetCall) Fields(s ...googleapi.Field) *ScoresResetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ScoresResetCall) Do() (*PlayerScoreResetResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "leaderboards/{leaderboardId}/scores/reset")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"leaderboardId": c.leaderboardId,
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
	var ret *PlayerScoreResetResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Resets scores for the leaderboard with the given ID for the currently authenticated player. This method is only accessible to whitelisted tester accounts for your application.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.scores.reset",
	//   "parameterOrder": [
	//     "leaderboardId"
	//   ],
	//   "parameters": {
	//     "leaderboardId": {
	//       "description": "The ID of the leaderboard.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "leaderboards/{leaderboardId}/scores/reset",
	//   "response": {
	//     "$ref": "PlayerScoreResetResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.scores.resetAll":

type ScoresResetAllCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// ResetAll: Resets all scores for all leaderboards for the currently
// authenticated players. This method is only accessible to whitelisted
// tester accounts for your application.
func (r *ScoresService) ResetAll() *ScoresResetAllCall {
	c := &ScoresResetAllCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ScoresResetAllCall) Fields(s ...googleapi.Field) *ScoresResetAllCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ScoresResetAllCall) Do() (*PlayerScoreResetAllResponse, error) {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "scores/reset")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
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
	var ret *PlayerScoreResetAllResponse
	if err := json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Resets all scores for all leaderboards for the currently authenticated players. This method is only accessible to whitelisted tester accounts for your application.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.scores.resetAll",
	//   "path": "scores/reset",
	//   "response": {
	//     "$ref": "PlayerScoreResetAllResponse"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.scores.resetAllForAllPlayers":

type ScoresResetAllForAllPlayersCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// ResetAllForAllPlayers: Resets scores for all draft leaderboards for
// all players. This method is only available to user accounts for your
// developer console.
func (r *ScoresService) ResetAllForAllPlayers() *ScoresResetAllForAllPlayersCall {
	c := &ScoresResetAllForAllPlayersCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ScoresResetAllForAllPlayersCall) Fields(s ...googleapi.Field) *ScoresResetAllForAllPlayersCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ScoresResetAllForAllPlayersCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "scores/resetAllForAllPlayers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
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
	//   "description": "Resets scores for all draft leaderboards for all players. This method is only available to user accounts for your developer console.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.scores.resetAllForAllPlayers",
	//   "path": "scores/resetAllForAllPlayers",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.scores.resetForAllPlayers":

type ScoresResetForAllPlayersCall struct {
	s             *Service
	leaderboardId string
	opt_          map[string]interface{}
}

// ResetForAllPlayers: Resets scores for the leaderboard with the given
// ID for all players. This method is only available to user accounts
// for your developer console. Only draft leaderboards can be reset.
func (r *ScoresService) ResetForAllPlayers(leaderboardId string) *ScoresResetForAllPlayersCall {
	c := &ScoresResetForAllPlayersCall{s: r.s, opt_: make(map[string]interface{})}
	c.leaderboardId = leaderboardId
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ScoresResetForAllPlayersCall) Fields(s ...googleapi.Field) *ScoresResetForAllPlayersCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ScoresResetForAllPlayersCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "leaderboards/{leaderboardId}/scores/resetForAllPlayers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.Expand(req.URL, map[string]string{
		"leaderboardId": c.leaderboardId,
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
	//   "description": "Resets scores for the leaderboard with the given ID for all players. This method is only available to user accounts for your developer console. Only draft leaderboards can be reset.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.scores.resetForAllPlayers",
	//   "parameterOrder": [
	//     "leaderboardId"
	//   ],
	//   "parameters": {
	//     "leaderboardId": {
	//       "description": "The ID of the leaderboard.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     }
	//   },
	//   "path": "leaderboards/{leaderboardId}/scores/resetForAllPlayers",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.scores.resetMultipleForAllPlayers":

type ScoresResetMultipleForAllPlayersCall struct {
	s                                *Service
	scoresresetmultipleforallrequest *ScoresResetMultipleForAllRequest
	opt_                             map[string]interface{}
}

// ResetMultipleForAllPlayers: Resets scores for the leaderboards with
// the given IDs for all players. This method is only available to user
// accounts for your developer console. Only draft leaderboards may be
// reset.
func (r *ScoresService) ResetMultipleForAllPlayers(scoresresetmultipleforallrequest *ScoresResetMultipleForAllRequest) *ScoresResetMultipleForAllPlayersCall {
	c := &ScoresResetMultipleForAllPlayersCall{s: r.s, opt_: make(map[string]interface{})}
	c.scoresresetmultipleforallrequest = scoresresetmultipleforallrequest
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *ScoresResetMultipleForAllPlayersCall) Fields(s ...googleapi.Field) *ScoresResetMultipleForAllPlayersCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *ScoresResetMultipleForAllPlayersCall) Do() error {
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.scoresresetmultipleforallrequest)
	if err != nil {
		return err
	}
	ctype := "application/json"
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "scores/resetMultipleForAllPlayers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
	req.Header.Set("Content-Type", ctype)
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
	//   "description": "Resets scores for the leaderboards with the given IDs for all players. This method is only available to user accounts for your developer console. Only draft leaderboards may be reset.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.scores.resetMultipleForAllPlayers",
	//   "path": "scores/resetMultipleForAllPlayers",
	//   "request": {
	//     "$ref": "ScoresResetMultipleForAllRequest"
	//   },
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.turnBasedMatches.reset":

type TurnBasedMatchesResetCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// Reset: Reset all turn-based match data for a user. This method is
// only accessible to whitelisted tester accounts for your application.
func (r *TurnBasedMatchesService) Reset() *TurnBasedMatchesResetCall {
	c := &TurnBasedMatchesResetCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *TurnBasedMatchesResetCall) Fields(s ...googleapi.Field) *TurnBasedMatchesResetCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *TurnBasedMatchesResetCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "turnbasedmatches/reset")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
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
	//   "description": "Reset all turn-based match data for a user. This method is only accessible to whitelisted tester accounts for your application.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.turnBasedMatches.reset",
	//   "path": "turnbasedmatches/reset",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}

// method id "gamesManagement.turnBasedMatches.resetForAllPlayers":

type TurnBasedMatchesResetForAllPlayersCall struct {
	s    *Service
	opt_ map[string]interface{}
}

// ResetForAllPlayers: Deletes turn-based matches where the only match
// participants are from whitelisted tester accounts for your
// application. This method is only available to user accounts for your
// developer console.
func (r *TurnBasedMatchesService) ResetForAllPlayers() *TurnBasedMatchesResetForAllPlayersCall {
	c := &TurnBasedMatchesResetForAllPlayersCall{s: r.s, opt_: make(map[string]interface{})}
	return c
}

// Fields allows partial responses to be retrieved.
// See https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *TurnBasedMatchesResetForAllPlayersCall) Fields(s ...googleapi.Field) *TurnBasedMatchesResetForAllPlayersCall {
	c.opt_["fields"] = googleapi.CombineFields(s)
	return c
}

func (c *TurnBasedMatchesResetForAllPlayersCall) Do() error {
	var body io.Reader = nil
	params := make(url.Values)
	params.Set("alt", "json")
	if v, ok := c.opt_["fields"]; ok {
		params.Set("fields", fmt.Sprintf("%v", v))
	}
	urls := googleapi.ResolveRelative(c.s.BasePath, "turnbasedmatches/resetForAllPlayers")
	urls += "?" + params.Encode()
	req, _ := http.NewRequest("POST", urls, body)
	googleapi.SetOpaque(req.URL)
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
	//   "description": "Deletes turn-based matches where the only match participants are from whitelisted tester accounts for your application. This method is only available to user accounts for your developer console.",
	//   "httpMethod": "POST",
	//   "id": "gamesManagement.turnBasedMatches.resetForAllPlayers",
	//   "path": "turnbasedmatches/resetForAllPlayers",
	//   "scopes": [
	//     "https://www.googleapis.com/auth/games",
	//     "https://www.googleapis.com/auth/plus.login"
	//   ]
	// }

}
