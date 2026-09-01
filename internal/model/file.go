package model

type FileSnapshot struct {
	Index     int          `json:"index"`
	Path      string       `json:"path"`
	Length    int64        `json:"length"`
	Completed int64        `json:"completed"`
	Priority  FilePriority `json:"priority"`
}
