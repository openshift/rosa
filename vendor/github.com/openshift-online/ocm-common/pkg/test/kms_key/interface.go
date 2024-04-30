package kms_key

type KMSKeyPolicy struct {
	Version   string      `json:"Version,omitempty"`
	ID        string      `json:"Id,omitempty"`
	Statement []Statement `json:"Statement,omitempty"`
}
type Principal struct {
	Aws interface{} `json:"AWS,omitempty"`
}
type Statement struct {
	Sid       string      `json:"Sid,omitempty"`
	Effect    string      `json:"Effect,omitempty"`
	Principal Principal   `json:"Principal,omitempty"`
	Action    interface{} `json:"Action,omitempty"`
	Resource  string      `json:"Resource,omitempty"`
	Condition Condition   `json:"Condition,omitempty"`
}
type Condition struct {
	Bool interface{} `json:"Bool,omitempty"`
}
