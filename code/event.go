package main

type event struct {
	ServiceType     string       `json:"serviceType"`
	Product         string       `json:"product"`
	ResourceID      string       `json:"resourceId"`
	Ver             string       `json:"ver"`
	EventRealname   string       `json:"eventRealname"`
	InstanceName    string       `json:"instanceName"`
	Level           string       `json:"level"`
	Resource        string       `json:"resource"`
	RegionName      string       `json:"regionName"`
	GroupID         string       `json:"groupId"`
	EventRealnameEn string       `json:"eventRealnameEn"`
	EventType       string       `json:"eventType"`
	UserID          string       `json:"userId"`
	Content         eventContent `json:"content"`
	CurLevel        string       `json:"curLevel"`
	RegionID        string       `json:"regionId"`
	EventTime       string       `json:"eventTime"`
	Name            string       `json:"name"`
	RuleName        string       `json:"ruleName"`
	ID              string       `json:"id"`
	Status          string       `json:"status"`
}

type eventContent struct {
	InstanceID string `json:"instanceId"`
	Action     string `json:"action"`
}
