package twitter

type User struct {
	CreationDate     string        `json:"creation_date"`
	UserId           string        `json:"user_id"`
	Username         string        `json:"username"`
	Name             string        `json:"name"`
	FollowerCount    int           `json:"follower_count"`
	FollowingCount   int           `json:"following_count"`
	FavouritesCount  int           `json:"favourites_count"`
	IsPrivate        bool          `json:"is_private"`
	IsVerified       bool          `json:"is_verified"`
	IsBlueVerified   bool          `json:"is_blue_verified"`
	Location         string        `json:"location"`
	ProfilePicUrl    string        `json:"profile_pic_url"`
	ProfileBannerUrl string        `json:"profile_banner_url"`
	Description      string        `json:"description"`
	ExternalUrl      string        `json:"external_url"`
	NumberOfTweets   int           `json:"number_of_tweets"`
	Bot              bool          `json:"bot"`
	Timestamp        int           `json:"timestamp"`
	HasNftAvatar     bool          `json:"has_nft_avatar"`
	Category         *UserCategory `json:"category"`
	DefaultProfile   bool          `json:"default_profile"`
	DefaultImage     bool          `json:"default_profile_image"`
}

type UserCategory struct {
	Name string `json:"name"`
	Id   int    `json:"id"`
}

type Tweet struct {
	TweetId           string           `json:"tweet_id"`
	CreationDate      string           `json:"creation_date"`
	Text              string           `json:"text"`
	MediaUrl          []string         `json:"media_url"`
	VideoUrl          []VideoUrl       `json:"video_url"`
	User              User             `json:"user"`
	Language          string           `json:"language"`
	FavoriteCount     int              `json:"favorite_count"`
	RetweetCount      int              `json:"retweet_count"`
	ReplyCount        int              `json:"reply_count"`
	QuoteCount        int              `json:"quote_count"`
	Retweet           bool             `json:"retweet"`
	Views             int64            `json:"views"`
	Timestamp         int64            `json:"timestamp"`
	VideoViewCount    int64            `json:"video_view_count"`
	InReplyToStatusId any              `json:"in_reply_to_status_id"`
	QuotedStatusId    any              `json:"quoted_status_id"`
	BindingValues     []BindingValue   `json:"binding_values"`
	ExpandedUrl       string           `json:"expanded_url"`
	RetweetTweetId    any              `json:"retweet_tweet_id"`
	ExtendedEntities  ExtendedEntities `json:"extended_entities"`
	ConversationId    string           `json:"conversation_id"`
	RetweetStatus     any              `json:"retweet_status"`
}

type VideoUrl struct {
	Bitrate     int    `json:"bitrate"`
	ContentType string `json:"content_type"`
	Url         string `json:"url"`
}

type ExtendedEntities struct {
	Media []Media `json:"media"`
}

type Media struct {
	DisplayUrl          string `json:"display_url"`
	ExpandedUrl         string `json:"expanded_url"`
	IdStr               string `json:"id_str"`
	Indices             []int  `json:"indices"`
	MediaKey            string `json:"media_key"`
	MediaUrlHttps       string `json:"media_url_https"`
	Type                string `json:"type"`
	Url                 string `json:"url"`
	AdditionalMediaInfo struct {
		Monetizable bool `json:"monetizable"`
	} `json:"additional_media_info"`
	MediaStats struct {
		ViewCount int `json:"viewCount"`
	} `json:"mediaStats"`
	ExtMediaAvailability struct {
		Status string `json:"status"`
	} `json:"ext_media_availability"`
	Features struct {
	} `json:"features"`
	Sizes struct {
		Large struct {
			H      int    `json:"h"`
			W      int    `json:"w"`
			Resize string `json:"resize"`
		} `json:"large"`
		Medium struct {
			H      int    `json:"h"`
			W      int    `json:"w"`
			Resize string `json:"resize"`
		} `json:"medium"`
		Small struct {
			H      int    `json:"h"`
			W      int    `json:"w"`
			Resize string `json:"resize"`
		} `json:"small"`
		Thumb struct {
			H      int    `json:"h"`
			W      int    `json:"w"`
			Resize string `json:"resize"`
		} `json:"thumb"`
	} `json:"sizes"`
	OriginalInfo struct {
		Height int `json:"height"`
		Width  int `json:"width"`
	} `json:"original_info"`
	VideoInfo struct {
		AspectRatio    []int `json:"aspect_ratio"`
		DurationMillis int   `json:"duration_millis"`
		Variants       []struct {
			Bitrate     int    `json:"bitrate,omitempty"`
			ContentType string `json:"content_type"`
			Url         string `json:"url"`
		} `json:"variants"`
	} `json:"video_info"`
}

type BindingValue struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}
