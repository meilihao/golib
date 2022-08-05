package event

import "time"

type Event struct {
	Id           string
	ParentId     string
	RootId       string
	AccountId    int64
	AccountName  string
	Action       string
	ActionName   string
	ResourceId   string
	ResourceType string
	ResourceName string
	In           string
	Out          string
	Status       string
	Percent      int16
	Timeout      int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
