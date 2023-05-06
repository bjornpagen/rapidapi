package twitter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

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
	url := c.buildUrl(path)
	for i, p := range params {
		separator := "&"
		if i == 0 {
			separator = "?"
		}
		url = fmt.Sprintf("%s%s%s=%v", url, separator, p.key, p.value)
	}
	return url
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

// GetUsername returns a User's username given a user ID.
func (c *Client) GetUsername(userId string) (username string, err error) {
	data, err := c.get([]string{"user", "id"}, []param{
		{"user_id", userId},
	})
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}

	type response struct {
		UserId   string `json:"user_id"`
		Username string `json:"username"`
	}

	var r response
	err = json.Unmarshal(data, &r)
	if err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	return r.Username, nil
}

type User struct {
	CreationDate     time.Time `json:"creation_date"`
	UserId           string    `json:"user_id"`
	Username         string    `json:"username"`
	Name             string    `json:"name"`
	FollowerCount    int       `json:"follower_count"`
	FollowingCount   int       `json:"following_count"`
	FavouritesCount  int       `json:"favourites_count"`
	IsPrivate        bool      `json:"is_private"`
	IsVerified       bool      `json:"is_verified"`
	Location         string    `json:"location"`
	ProfilePicUrl    string    `json:"profile_pic_url"`
	ProfileBannerUrl string    `json:"profile_banner_url"`
	Description      string    `json:"description"`
	ExternalUrl      string    `json:"external_url"`
	NumberOfTweets   int       `json:"number_of_tweets"`
	Bot              bool      `json:"bot"`
	Timestamp        int       `json:"timestamp"`
	HasNftAvatar     bool      `json:"has_nft_avatar"`
	Category         string    `json:"category"`
	DefaultProfile   bool      `json:"default_profile"`
	DefaultImage     bool      `json:"default_profile_image"`
}

// GetUser returns the public information about a Twitter profile.
func (c *Client) GetUser(userId string) (user User, err error) {
	data, err := c.get([]string{"user", "details"}, []param{
		{"user_id", userId},
	})
	if err != nil {
		return user, fmt.Errorf("get user: %w", err)
	}

	err = json.Unmarshal(data, &user)
	if err != nil {
		return user, fmt.Errorf("unmarshal response: %w", err)
	}

	return user, nil
}

// GetUserByUsername returns the public information about a Twitter profile.
func (c *Client) GetUserByUsername(username string) (user User, err error) {
	data, err := c.get([]string{"user", "details"}, []param{
		{"username", username},
	})
	if err != nil {
		return user, fmt.Errorf("get user: %w", err)
	}

	err = json.Unmarshal(data, &user)
	if err != nil {
		return user, fmt.Errorf("unmarshal response: %w", err)
	}

	return user, nil
}

type getUserTweetsResponse struct {
	Results           []Tweet `json:"results"`
	ContinuationToken string  `json:"continuation_token"`
}

type Tweet struct {
	TweetId           string         `json:"tweet_id"`
	CreationDate      string         `json:"creation_date"`
	Text              string         `json:"text"`
	MediaUrl          []string       `json:"media_url"`
	VideoUrl          []string       `json:"video_url"`
	User              User           `json:"user"`
	Language          string         `json:"language"`
	FavoriteCount     int            `json:"favorite_count"`
	RetweetCount      int            `json:"retweet_count"`
	ReplyCount        int            `json:"reply_count"`
	QuoteCount        int            `json:"quote_count"`
	Retweet           bool           `json:"retweet"`
	Views             int            `json:"views"`
	Timestamp         int64          `json:"timestamp"`
	VideoViewCount    any            `json:"video_view_count"`
	InReplyToStatusId string         `json:"in_reply_to_status_id"`
	QuotedStatusId    any            `json:"quoted_status_id"`
	BindingValues     []BindingValue `json:"binding_values"`
	ExpandedUrl       string         `json:"expanded_url"`
	RetweetTweetId    string         `json:"retweet_tweet_id"`
	ExtendedEntities  any            `json:"extended_entities"`
	ConversationId    string         `json:"conversation_id"`
	RetweetStatus     any            `json:"retweet_status"`
}

type BindingValue struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
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

// GetUserTweets returns a list of user's tweets.
func (c *Client) GetUserTweets(userId string, opts ...getUserTweetsOption) (tweets []Tweet, err error) {
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

	tweets = make([]Tweet, 0)
	data, err := c.get([]string{"user", "tweets"}, params)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	var r getUserTweetsResponse
	err = json.Unmarshal(data, &r)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	token := r.ContinuationToken
	continutationParams := append(params, param{"continuation_token", token})
	for token != "" {
		data, err := c.get([]string{"user", "tweets", "continuation"}, continutationParams)
		if err != nil {
			return nil, fmt.Errorf("get user: %w", err)
		}

		var r getUserTweetsResponse
		err = json.Unmarshal(data, &r)
		if err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}

		tweets = append(tweets, r.Results...)
		token = r.ContinuationToken
		continutationParams = append(continutationParams[:len(continutationParams)-1], param{"continuation_token", token})
	}

	return tweets, nil
}

type getUserFollowingResponse struct {
	Results           []User `json:"results"`
	ContinuationToken string `json:"continuation_token"`
}

// GetUserFollowing returns a list of user's following.
func (c *Client) GetUserFollowing(userId string) (following []User, err error) {
	params := []param{
		{"user_id", userId},
		{"limit", _pageLimit},
	}

	following = make([]User, 0)
	data, err := c.get([]string{"user", "following"}, params)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	var r getUserFollowingResponse
	err = json.Unmarshal(data, &r)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	token := r.ContinuationToken
	continutationParams := append(params, param{"continuation_token", token})
	for token != "" {
		data, err := c.get([]string{"user", "following", "continuation"}, continutationParams)
		if err != nil {
			return nil, fmt.Errorf("get user: %w", err)
		}

		var r getUserFollowingResponse
		err = json.Unmarshal(data, &r)
		if err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}

		following = append(following, r.Results...)
		token = r.ContinuationToken
		continutationParams = append(continutationParams[:len(continutationParams)-1], param{"continuation_token", token})
	}

	return following, nil
}

type getUserFollowersResponse struct {
	Results           []User `json:"results"`
	ContinuationToken string `json:"continuation_token"`
}

// GetUserFollowers returns a list of user's followers.
func (c *Client) GetUserFollowers(userId string) (followers []User, err error) {
	params := []param{
		{"user_id", userId},
		{"limit", _pageLimit},
	}

	followers = make([]User, 0)
	data, err := c.get([]string{"user", "followers"}, params)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	var r getUserFollowersResponse
	err = json.Unmarshal(data, &r)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	token := r.ContinuationToken
	continutationParams := append(params, param{"continuation_token", token})
	for token != "" {
		data, err := c.get([]string{"user", "followers", "continuation"}, continutationParams)
		if err != nil {
			return nil, fmt.Errorf("get user: %w", err)
		}

		var r getUserFollowersResponse
		err = json.Unmarshal(data, &r)
		if err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}

		followers = append(followers, r.Results...)
		token = r.ContinuationToken
		continutationParams = append(continutationParams[:len(continutationParams)-1], param{"continuation_token", token})
	}

	return followers, nil
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

// GetTweetReplies returns a list of replies to a tweet.
func (c *Client) GetTweetReplies(tweetId string) (replies []Tweet, err error) {
	replies = make([]Tweet, 0)
	params := []param{
		{"tweet_id", tweetId},
	}

	data, err := c.get([]string{"tweet", "replies"}, params)
	if err != nil {
		return nil, fmt.Errorf("get tweet replies: %w", err)
	}

	var r getTweetRepliesResponse
	err = json.Unmarshal(data, &r)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	replies = append(replies, r.Replies...)
	token := r.ContinuationToken
	continutationParams := append(params, param{"continuation_token", token})
	for token != "" {
		data, err := c.get([]string{"tweet", "replies", "continuation"}, continutationParams)
		if err != nil {
			return nil, fmt.Errorf("get tweet replies: %w", err)
		}

		var r getTweetRepliesResponse
		err = json.Unmarshal(data, &r)
		if err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}

		replies = append(replies, r.Replies...)
		token = r.ContinuationToken
		continutationParams = append(continutationParams[:len(continutationParams)-1], param{"continuation_token", token})
	}

	return replies, nil
}

// GetTweetDetails returns general information about a tweet.
func (c *Client) GetTweetDetails(tweetId string) (tweet Tweet, err error) {
	params := []param{
		{"tweet_id", tweetId},
	}

	data, err := c.get([]string{"tweet", "details"}, params)
	if err != nil {
		return tweet, fmt.Errorf("get tweet details: %w", err)
	}

	err = json.Unmarshal(data, &tweet)
	if err != nil {
		return tweet, fmt.Errorf("unmarshal response: %w", err)
	}

	return tweet, nil
}

// GetTweetUserRetweets returns a list of users who retweeted the tweet
func (c *Client) GetTweetUserRetweets(tweetId string) (users []User, err error) {
	return users, ErrNotImplemented
}

type getUserFavoritesResponse struct {
	Favorites         []User `json:"favorites"`
	ContinuationToken string `json:"continuation_token"`
}

// GetTweetUserFavorites returns a list of users who favorited the tweet
func (c *Client) GetTweetUserFavorites(tweetId string) (users []User, err error) {
	params := []param{
		{"tweet_id", tweetId},
	}

	data, err := c.get([]string{"tweet", "favoriters"}, params)
	if err != nil {
		return nil, fmt.Errorf("get tweet favorites: %w", err)
	}

	var r getUserFavoritesResponse
	err = json.Unmarshal(data, &r)
	if err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	users = append(users, r.Favorites...)
	token := r.ContinuationToken
	continutationParams := append(params, param{"continuation_token", token})
	for token != "" {
		data, err := c.get([]string{"tweet", "favoriters", "continuation"}, continutationParams)
		if err != nil {
			return nil, fmt.Errorf("get tweet favorites: %w", err)
		}

		var r getUserFavoritesResponse
		err = json.Unmarshal(data, &r)
		if err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}

		users = append(users, r.Favorites...)
		token = r.ContinuationToken
		continutationParams = append(continutationParams[:len(continutationParams)-1], param{"continuation_token", token})
	}

	return users, nil
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
