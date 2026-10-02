package session

import "quizlive/internal/quiz"

type QuestionView struct {
	ID      string        `json:"id"`
	Text    string        `json:"text"`
	Options []quiz.Option `json:"options"`
}

type ListItem struct {
	ID     string  `json:"id"`
	Text   string  `json:"text"`
	Status QStatus `json:"status"`
	// Options and DefaultDurationSec are only sent to the admin.
	Options            []quiz.Option `json:"options,omitempty"`
	DefaultDurationSec *int          `json:"defaultDurationSec,omitempty"`
}

// View is the role-specific snapshot sent on connect and after every mutation.
type View struct {
	SessionID   string                    `json:"sessionId"`
	Role        string                    `json:"role"`
	ServerNow   int64                     `json:"serverNow"`
	Phase       Phase                     `json:"phase"`
	Index       int                       `json:"currentQuestionIndex"`
	QuestionCnt int                       `json:"questionCount"`
	IsLast      bool                      `json:"isLast"`
	Question    *QuestionView             `json:"question,omitempty"`
	Timer       Timer                     `json:"timer"`
	Answered    int                       `json:"answeredCount"`
	ClientCount int                       `json:"clientCount"`
	Joined      bool                      `json:"joined,omitempty"`
	MyAnswer    string                    `json:"myAnswer,omitempty"`
	Tally       map[string]int            `json:"tally,omitempty"`
	Questions   []ListItem                `json:"questions,omitempty"`   // admin only
	PastTallies map[string]map[string]int `json:"pastTallies,omitempty"` // admin only
}
