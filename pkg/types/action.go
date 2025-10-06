package types

// RemediationAction은 HTTP POST Body로 수신되는 표준 JSON 구조체입니다.
// 여러 패키지에서 공유되므로 별도의 types 패키지로 분리합니다.
type RemediationAction struct {
	ActionType   string            `json:"actionType"`
	Namespace    string            `json:"namespace"`
	ResourceName string            `json:"resourceName"`
	Parameters   map[string]string `json:"parameters"`
	Reason       string            `json:"reason"`
	TriggeredBy  string            `json:"triggeredBy"`
}