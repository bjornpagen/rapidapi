package twitter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"go.uber.org/ratelimit"
)

const (
	_pageLimit = 100
)

var (
	ErrNotImplemented = errors.New("not implemented")
)

type option func(option *options) error

type options struct {
	host       string
	rateLimit  *ratelimit.Limiter
	httpClient *http.Client
}

func WithHost(host string) option {
	return func(option *options) error {
		// Check if host is valid.
		_, err := http.NewRequest("GET", fmt.Sprintf("https://%s", host), nil)
		if err != nil {
			return fmt.Errorf("invalid host: %w", err)
		}

		option.host = host
		return nil
	}
}

func WithRateLimit(rl ratelimit.Limiter) option {
	return func(option *options) error {
		option.rateLimit = &rl
		return nil
	}
}

func WithHttpClient(hc http.Client) option {
	return func(option *options) error {
		option.httpClient = &hc
		return nil
	}
}

type Client struct {
	apiKey  string
	options *options
}

func New(apiKey string, opts ...option) (c Client, err error) {
	o := &options{}
	for _, opt := range opts {
		err := opt(o)
		if err != nil {
			return c, fmt.Errorf("bad option: %w", err)
		}
	}

	if o.host == "" {
		o.host = "twitter154.p.rapidapi.com"
	}

	if o.rateLimit == nil {
		o.rateLimit = new(ratelimit.Limiter)
		*o.rateLimit = ratelimit.NewUnlimited()
	}

	if o.httpClient == nil {
		o.httpClient = http.DefaultClient
	}

	return Client{
		apiKey:  apiKey,
		options: o,
	}, nil
}

type param struct {
	key   string
	value any
}

func (c *Client) buildUrl(p []string) string {
	return fmt.Sprintf("https://%s/%s", c.options.host, path.Join(p...))
}

func (c *Client) buildUrlWithParameters(path []string, params []param) string {
	uri := c.buildUrl(path)
	for i, p := range params {
		separator := "&"
		if i == 0 {
			separator = "?"
		}
		uri = fmt.Sprintf("%s%s%s=%s", uri, separator, p.key, url.QueryEscape(fmt.Sprintf("%v", p.value)))
	}
	return uri
}

func (c *Client) do(req *http.Request) (data []byte, err error) {
	req.Header.Add("X-RapidAPI-Key", c.apiKey)
	req.Header.Add("X-RapidAPI-Host", c.options.host)

	(*c.options.rateLimit).Take()
	resp, err := c.options.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status code %d", resp.StatusCode)
	}

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return data, nil
}

func (c *Client) get(path []string, params []param) (data []byte, err error) {
	url := c.buildUrlWithParameters(path, params)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	return c.do(req)
}

type result[T any] interface {
	Result() T
}

func getResult[T any, R result[T]](c *Client, path []string, params []param) (result T, err error) {
	data, err := c.get(path, params)
	if err != nil {
		return result, fmt.Errorf("get: %w", err)
	}

	var r R
	err = json.Unmarshal(data, &r)
	if err != nil {
		return result, fmt.Errorf("unmarshal response: %w", err)
	}

	return r.Result(), nil
}

type resultPaginated[T any] interface {
	Result() []T
	Token() string
}

func getResultPaginated[T any, R resultPaginated[T]](c *Client, path []string, params []param) (results []T, err error) {
	data, err := c.get(path, params)
	if err != nil {
		return nil, fmt.Errorf("get: %w", err)
	}

	var r R
	err = json.Unmarshal(data, &r)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	path = append(path, "continuation")
	params = append(params, param{"continuation_token", r.Token()})

	for len(r.Result()) != 0 {
		results = append(results, r.Result()...)
		data, err := c.get(path, params)
		if err != nil {
			return nil, fmt.Errorf("get: %w", err)
		}

		err = json.Unmarshal(data, &r)
		if err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}

		params[len(params)-1].value = r.Token()
	}

	return results, nil
}

type getUsernameResponse struct {
	UserId   string `json:"user_id"`
	Username string `json:"username"`
}

func (r getUsernameResponse) Result() string {
	return r.Username
}

var _ result[string] = (*getUsernameResponse)(nil)

// GetUsername returns a User's username given a user ID.
func (c *Client) GetUsername(userId string) (username string, err error) {
	path := []string{"user", "username"}
	params := []param{
		{"user_id", userId},
	}

	return getResult[string, getUsernameResponse](c, path, params)
}

type getUserResponse = User

func (r getUserResponse) Result() User {
	return r
}

var _ result[User] = (*getUserResponse)(nil)

// GetUser returns the public information about a Twitter profile.
func (c *Client) GetUser(userId string) (user User, err error) {
	path := []string{"user", "details"}
	params := []param{
		{"user_id", userId},
	}

	return getResult[User, getUserResponse](c, path, params)
}

// GetUserByUsername returns the public information about a Twitter profile.
func (c *Client) GetUserByUsername(username string) (user User, err error) {
	path := []string{"user", "details"}
	params := []param{
		{"username", username},
	}

	return getResult[User, getUserResponse](c, path, params)
}

type getUserTweetsOptions struct {
	includeReplies bool
	includePinned  bool
}

type getUserTweetsOption func(*getUserTweetsOptions)

func IncludeReplies() getUserTweetsOption {
	return func(o *getUserTweetsOptions) {
		o.includeReplies = true
	}
}

func IncludePinned() getUserTweetsOption {
	return func(o *getUserTweetsOptions) {
		o.includePinned = true
	}
}

type getUserTweetsResponse struct {
	Results           []Tweet `json:"results"`
	ContinuationToken string  `json:"continuation_token"`
}

func (g getUserTweetsResponse) Result() []Tweet {
	return g.Results
}

func (g getUserTweetsResponse) Token() string {
	return g.ContinuationToken
}

var _ resultPaginated[Tweet] = (*getUserTweetsResponse)(nil)

// GetUserTweets returns a list of user's tweets.
func (c *Client) GetUserTweets(userId string, opts ...getUserTweetsOption) (tweets []Tweet, err error) {
	path := []string{"user", "tweets"}
	params := []param{
		{"user_id", userId},
		{"limit", _pageLimit},
	}

	o := getUserTweetsOptions{}
	for _, opt := range opts {
		opt(&o)
	}

	if o.includeReplies {
		params = append(params, param{"include_replies", "true"})
	} else {
		params = append(params, param{"include_replies", "false"})
	}

	if o.includePinned {
		params = append(params, param{"include_pinned", "true"})
	} else {
		params = append(params, param{"include_pinned", "false"})
	}

	return getResultPaginated[Tweet, getUserTweetsResponse](c, path, params)
}

type getUserFollowsResponse struct {
	Results           []User `json:"results"`
	ContinuationToken string `json:"continuation_token"`
}

func (g getUserFollowsResponse) Result() []User {
	return g.Results
}

func (g getUserFollowsResponse) Token() string {
	return g.ContinuationToken
}

var _ resultPaginated[User] = (*getUserFollowsResponse)(nil)

// GetUserFollowing returns a list of user's following.
func (c *Client) GetUserFollowing(userId string) (following []User, err error) {
	path := []string{"user", "following"}
	params := []param{
		{"user_id", userId},
		{"limit", _pageLimit},
	}

	return getResultPaginated[User, getUserFollowsResponse](c, path, params)
}

// GetUserFollowers returns a list of user's followers.
func (c *Client) GetUserFollowers(userId string) (followers []User, err error) {
	path := []string{"user", "followers"}
	params := []param{
		{"user_id", userId},
		{"limit", _pageLimit},
	}

	return getResultPaginated[User, getUserFollowsResponse](c, path, params)
}

// GetUserLikes returns a list of user's likes given a user ID
func (c *Client) GetUserLikes(userId string) (likes []Tweet, err error) {
	return likes, ErrNotImplemented
}

// GetUserMedia returns a list of user's media given a user ID
func (c *Client) GetUserMedia(userId string) (media any, err error) {
	return media, ErrNotImplemented
}

type getTweetRepliesResponse struct {
	Replies           []Tweet `json:"replies"`
	ContinuationToken string  `json:"continuation_token"`
}

func (g getTweetRepliesResponse) Result() []Tweet {
	return g.Replies
}

func (g getTweetRepliesResponse) Token() string {
	return g.ContinuationToken
}

var _ resultPaginated[Tweet] = (*getTweetRepliesResponse)(nil)

// GetTweetReplies returns a list of replies to a tweet.
func (c *Client) GetTweetReplies(tweetId string) (replies []Tweet, err error) {
	path := []string{"tweet", "replies"}
	params := []param{
		{"tweet_id", tweetId},
	}

	return getResultPaginated[Tweet, getTweetRepliesResponse](c, path, params)
}

type getTweetDetailsResponse = Tweet

func (g getTweetDetailsResponse) Result() Tweet {
	return g
}

var _ result[Tweet] = (*getTweetDetailsResponse)(nil)

// GetTweetDetails returns general information about a tweet.
func (c *Client) GetTweetDetails(tweetId string) (tweet Tweet, err error) {
	path := []string{"tweet", "details"}
	params := []param{
		{"tweet_id", tweetId},
	}

	return getResult[Tweet, getTweetDetailsResponse](c, path, params)
}

// GetTweetUserRetweets returns a list of users who retweeted the tweet
func (c *Client) GetTweetUserRetweets(tweetId string) (users []User, err error) {
	return users, ErrNotImplemented
}

type getUserFavoritesResponse struct {
	Favoriters        []User `json:"favoriters"`
	ContinuationToken string `json:"continuation_token"`
}

func (g getUserFavoritesResponse) Result() []User {
	return g.Favoriters
}

func (g getUserFavoritesResponse) Token() string {
	return g.ContinuationToken
}

var _ resultPaginated[User] = (*getUserFavoritesResponse)(nil)

// GetTweetUserFavorites returns a list of users who favorited the tweet
func (c *Client) GetTweetUserFavorites(tweetId string) (users []User, err error) {
	path := []string{"tweet", "favoriters"}
	params := []param{
		{"tweet_id", tweetId},
	}

	return getResultPaginated[User, getUserFavoritesResponse](c, path, params)
}

type getSearchResponse struct {
	Results           []Tweet `json:"results"`
	ContinuationToken string  `json:"continuation_token"`
}

// Search returns a list of tweets matching a query.
func (c *Client) Search(query string) (tweets []Tweet, err error) {
	return tweets, ErrNotImplemented
}

type geoSearchOptions struct {
	latitude  float64
	longitude float64
	radius    int
	language  string
}

type geoSearchOption func(*geoSearchOptions)

// GeoSearch returns a list of tweets matching a query and a geolocation.
func (c *Client) GeoSearch(query string, opts ...geoSearchOption) (tweets []Tweet, err error) {
	return tweets, ErrNotImplemented
}

func (c *Client) Hashtag(hashtag string) (tweets []Tweet, err error) {
	return tweets, ErrNotImplemented
}

/*
	{
	  "list_id": "1591033111726391297",
	  "list_id_str": "TGlzdDoxNTkxMDMzMTExNzI2MzkxMjk3",
	  "member_count": 8,
	  "name": "testing",
	  "subscriber_count": 0,
	  "creation_date": "1668166828000",
	  "mode": "Public",
	  "default_banner_media": {
	    "media_info": {
	      "original_img_url": "https://pbs.twimg.com/media/EXZ2mJCUEAEbJb3.png",
	      "original_img_width": 1125,
	      "original_img_height": 375,
	      "salient_rect": {
	        "left": 562,
	        "top": 187,
	        "width": 1,
	        "height": 1
	      }
	    }
	  },
	  "default_banner_media_results": {
	    "result": {
	      "id": "QXBpTWVkaWE6DAABCgABEXZ2mJCUEAEKAAIQF4A+eBQgAAAA",
	      "media_key": "3_1258323543529361409",
	      "media_id": "1258323543529361409",
	      "media_info": {
	        "__typename": "ApiImage",
	        "original_img_height": 375,
	        "original_img_width": 1125,
	        "original_img_url": "https://pbs.twimg.com/media/EXZ2mJCUEAEbJb3.png",
	        "salient_rect": {
	          "height": 1,
	          "left": 562,
	          "top": 187,
	          "width": 1
	        }
	      },
	      "__typename": "ApiMedia"
	    }
	  },
	  "user": {
	    "creation_date": "Mon Jan 13 18:44:09 +0000 2014",
	    "user_id": "2290075459",
	    "username": "previewuser",
	    "name": "Userbet preview",
	    "follower_count": 44,
	    "following_count": 53,
	    "favourites_count": 0,
	    "is_private": null,
	    "is_verified": false,
	    "is_blue_verified": false,
	    "location": "Germany",
	    "profile_pic_url": "https://pbs.twimg.com/profile_images/1553007348213600256/K3DnFMLD_normal.jpg",
	    "profile_banner_url": null,
	    "description": "",
	    "external_url": null,
	    "number_of_tweets": 65958,
	    "bot": false,
	    "timestamp": 1389638649,
	    "has_nft_avatar": false,
	    "category": null,
	    "default_profile": null,
	    "default_profile_image": null
	  },
	  "description": null
	}
*/
type List = any

func (c *Client) GetListDetails(listId string) (list List, err error) {
	return list, ErrNotImplemented
}

func (c *Client) GetListTweets(listId string) (tweets []Tweet, err error) {
	return tweets, ErrNotImplemented
}

type Trend = any

func (c *Client) GetTrends(woeId int) (trends []Trend, err error) {
	return trends, ErrNotImplemented
}

type Location = any

func (c *Client) GetLocations() (locations []Location, err error) {
	return locations, ErrNotImplemented
}
