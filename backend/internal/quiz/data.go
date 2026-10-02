// Package quiz parses and validates the uploaded data.json question set.
package quiz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const (
	MaxQuestions = 200
	MinOptions   = 2
	MaxOptions   = 6
)

var AllowedDurations = []int{10, 15, 30, 60}

type Option struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

type Question struct {
	ID                 string   `json:"id"`
	Text               string   `json:"text"`
	Options            []Option `json:"options"`
	Points             *float64 `json:"points,omitempty"`
	DefaultDurationSec *int     `json:"defaultDurationSec,omitempty"`
}

type Set struct {
	Questions []Question `json:"questions"`
}

// FieldError names the offending field with a JSON-path-style path.
type FieldError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (e FieldError) Error() string { return e.Path + ": " + e.Message }

func ValidDuration(sec int) bool {
	for _, d := range AllowedDurations {
		if d == sec {
			return true
		}
	}
	return false
}

// Parse decodes and validates data.json. It returns every error found, not just the first.
func Parse(r io.Reader) (*Set, []FieldError) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, []FieldError{{Path: "$", Message: "could not read file"}}
	}
	return ParseBytes(raw)
}

func ParseBytes(raw []byte) (*Set, []FieldError) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil, []FieldError{{Path: "$", Message: "not valid JSON object: " + err.Error()}}
	}
	qraw, ok := top["questions"]
	if !ok {
		return nil, []FieldError{{Path: "questions", Message: "is required"}}
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(qraw, &items); err != nil {
		return nil, []FieldError{{Path: "questions", Message: "must be an array of objects"}}
	}
	var errs []FieldError
	add := func(path, msg string) { errs = append(errs, FieldError{path, msg}) }
	if len(items) == 0 {
		add("questions", "must contain at least 1 question")
	}
	if len(items) > MaxQuestions {
		add("questions", fmt.Sprintf("must contain at most %d questions", MaxQuestions))
	}

	set := &Set{}
	seen := map[string]int{}
	for i, it := range items {
		p := fmt.Sprintf("questions[%d]", i)
		var q Question
		if s, ok := str(it["id"]); !ok || s == "" {
			add(p+".id", "is required and must be a non-empty string")
		} else {
			q.ID = s
			if j, dup := seen[s]; dup {
				add(p+".id", fmt.Sprintf("duplicate id %q (also questions[%d])", s, j))
			}
			seen[s] = i
		}
		if s, ok := str(it["text"]); !ok || s == "" {
			add(p+".text", "is required and must be a non-empty string")
		} else {
			q.Text = s
		}
		q.Options = parseOptions(it["options"], p+".options", add)
		if v, ok := it["points"]; ok && !isNull(v) {
			var f float64
			if err := json.Unmarshal(v, &f); err != nil || f < 0 {
				add(p+".points", "must be a number >= 0")
			} else {
				q.Points = &f
			}
		}
		if v, ok := it["defaultDurationSec"]; ok && !isNull(v) {
			var d int
			if err := json.Unmarshal(v, &d); err != nil || !ValidDuration(d) {
				add(p+".defaultDurationSec", "must be one of 10, 15, 30, 60")
			} else {
				q.DefaultDurationSec = &d
			}
		}
		set.Questions = append(set.Questions, q)
	}
	if len(errs) > 0 {
		return nil, errs
	}
	return set, nil
}

func parseOptions(raw json.RawMessage, path string, add func(string, string)) []Option {
	if raw == nil || isNull(raw) {
		add(path, "is required")
		return nil
	}
	var items []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		add(path, "must be an array of objects")
		return nil
	}
	if len(items) < MinOptions || len(items) > MaxOptions {
		add(path, fmt.Sprintf("must have %d–%d items (got %d)", MinOptions, MaxOptions, len(items)))
	}
	var opts []Option
	keys := map[string]bool{}
	for i, it := range items {
		op := fmt.Sprintf("%s[%d]", path, i)
		var o Option
		if s, ok := str(it["key"]); !ok || s == "" {
			add(op+".key", "is required and must be a non-empty string")
		} else if keys[s] {
			add(op+".key", fmt.Sprintf("duplicate key %q", s))
		} else {
			keys[s] = true
			o.Key = s
		}
		if s, ok := str(it["text"]); !ok || s == "" {
			add(op+".text", "is required and must be a non-empty string")
		} else {
			o.Text = s
		}
		opts = append(opts, o)
	}
	return opts
}

func str(raw json.RawMessage) (string, bool) {
	if raw == nil {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

func isNull(raw json.RawMessage) bool { return bytes.Equal(bytes.TrimSpace(raw), []byte("null")) }
