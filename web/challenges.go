package web

import "time"

type Challenges struct {
	UserId int
	First  bool
	Second bool
	Third  bool
	Fourth bool
	Fifth  bool
	Date   time.Time
}

type ChallengeWithUsername struct {
	Username string
	First    bool
	Second   bool
	Third    bool
	Fourth   bool
	Fifth    bool
	Date     time.Time
}

type ChallengeWithDateString struct {
	UserId int
	First  bool
	Second bool
	Third  bool
	Fourth bool
	Fifth  bool
	Month  string
	Day    string
}
