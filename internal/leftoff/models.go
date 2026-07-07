package leftoff

import (
	"strings"
	"time"
)

type RecordType string

const (
	RecordTask          RecordType = "task"
	RecordIdea          RecordType = "idea"
	RecordDecision      RecordType = "decision"
	RecordProblem       RecordType = "problem"
	RecordSolution      RecordType = "solution"
	RecordOpenLoop      RecordType = "open_loop"
	RecordReleaseIntent RecordType = "release_intent"
	RecordFrictionEvent RecordType = "friction_event"
	RecordActivityEvent RecordType = "activity_event"
)

type TaskStatus string

const (
	StatusInbox     TaskStatus = "inbox"
	StatusActive    TaskStatus = "active"
	StatusBlocked   TaskStatus = "blocked"
	StatusWaiting   TaskStatus = "waiting"
	StatusParked    TaskStatus = "parked"
	StatusDone      TaskStatus = "done"
	StatusAbandoned TaskStatus = "abandoned"
)

func ParseRecordType(kind string) (RecordType, bool) {
	k := strings.ToLower(strings.TrimSpace(kind))
	k = strings.TrimPrefix(k, "/")
	k = strings.ReplaceAll(k, "_", "-")
	k = strings.ReplaceAll(k, " ", "-")

	switch k {
	case "task", "todo", "to-do":
		return RecordTask, true
	case "idea":
		return RecordIdea, true
	case "decision", "decide":
		return RecordDecision, true
	case "problem", "bug", "issue", "failure":
		return RecordProblem, true
	case "solution", "fix", "workaround", "recipe":
		return RecordSolution, true
	case "follow-up", "followup", "promise", "open-loop", "waiting-item":
		return RecordOpenLoop, true
	case "release", "release-intent", "ship-condition":
		return RecordReleaseIntent, true
	case "friction", "friction-event", "friction-observation":
		return RecordFrictionEvent, true
	case "activity", "activity-event":
		return RecordActivityEvent, true
	case "project-context":
		return RecordIdea, true
	default:
		return "", false
	}
}

func (t RecordType) Prefix() string {
	switch t {
	case RecordTask:
		return "TASK"
	case RecordIdea:
		return "IDEA"
	case RecordDecision:
		return "DECISION"
	case RecordProblem:
		return "PROBLEM"
	case RecordSolution:
		return "SOLUTION"
	case RecordOpenLoop:
		return "OPEN-LOOP"
	case RecordReleaseIntent:
		return "RELEASE"
	case RecordFrictionEvent:
		return "FRICTION"
	case RecordActivityEvent:
		return "ACTIVITY"
	default:
		return strings.ToUpper(string(t))
	}
}

func (t RecordType) DestinationFile() string {
	switch t {
	case RecordDecision:
		return "decisions.md"
	case RecordProblem, RecordSolution:
		return "solved-problems.md"
	case RecordReleaseIntent:
		return "releases.md"
	case RecordFrictionEvent:
		return "friction.md"
	default:
		return "open-loops.md"
	}
}

func (t RecordType) DefaultStatus(projectLinked bool) string {
	switch t {
	case RecordTask:
		if projectLinked {
			return string(StatusActive)
		}
		return string(StatusInbox)
	case RecordIdea:
		return string(StatusParked)
	case RecordDecision:
		return "accepted"
	case RecordProblem:
		return "open"
	case RecordSolution:
		return "tentative"
	case RecordOpenLoop:
		if projectLinked {
			return string(StatusActive)
		}
		return string(StatusInbox)
	case RecordReleaseIntent:
		return string(StatusActive)
	case RecordFrictionEvent:
		return "observed"
	default:
		return "recorded"
	}
}

type CaptureRequest struct {
	Kind     string `json:"kind"`
	Text     string `json:"text"`
	Project  string `json:"project,omitempty"`
	RepoPath string `json:"repo_path,omitempty"`
	Source   string `json:"source,omitempty"`
}

type CaptureResult struct {
	ID          string        `json:"id"`
	Type        RecordType    `json:"type"`
	ProjectSlug string        `json:"project_slug,omitempty"`
	Path        string        `json:"path"`
	Warnings    []string      `json:"warnings,omitempty"`
	Activity    ActivityEvent `json:"activity"`
}

type ActivityEvent struct {
	Timestamp  string `json:"timestamp"`
	Kind       string `json:"kind"`
	RecordID   string `json:"record_id"`
	RecordType string `json:"record_type"`
	Project    string `json:"project,omitempty"`
	Summary    string `json:"summary"`
	Evidence   string `json:"evidence"`
}

type ProjectMeta struct {
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Remote    string    `json:"remote,omitempty"`
	LocalPath string    `json:"local_path,omitempty"`
	Created   time.Time `json:"created"`
}

type ChangedFile struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

type Commit struct {
	Hash    string `json:"hash"`
	Summary string `json:"summary"`
}

type Worktree struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
	Head   string `json:"head"`
}

type StaleBranch struct {
	Name       string `json:"name"`
	LastCommit string `json:"last_commit"`
	AgeDays    int    `json:"age_days"`
}

type GitSnapshot struct {
	Available       bool          `json:"available"`
	IsRepo          bool          `json:"is_repo"`
	InspectedAt     time.Time     `json:"inspected_at"`
	Root            string        `json:"root"`
	RepoName        string        `json:"repo_name"`
	Remote          string        `json:"remote,omitempty"`
	Branch          string        `json:"branch"`
	Head            string        `json:"head"`
	Worktree        string        `json:"worktree"`
	WorktreeStatus  string        `json:"worktree_status"`
	Ahead           int           `json:"ahead"`
	Behind          int           `json:"behind"`
	UnpushedCommits int           `json:"unpushed_commits"`
	ChangedFiles    []ChangedFile `json:"changed_files,omitempty"`
	RecentCommits   []Commit      `json:"recent_commits,omitempty"`
	Worktrees       []Worktree    `json:"worktrees,omitempty"`
	StaleBranches   []StaleBranch `json:"stale_branches,omitempty"`
	HealthNotes     []string      `json:"health_notes,omitempty"`
	Commands        []string      `json:"commands,omitempty"`
}

type SavedState struct {
	Exists      bool   `json:"exists"`
	Path        string `json:"path"`
	LastUpdated string `json:"last_updated"`
	Repository  string `json:"repository"`
	Remote      string `json:"remote"`
	Worktree    string `json:"worktree"`
	Branch      string `json:"branch"`
	Head        string `json:"head"`
	DirtyFiles  string `json:"dirty_files"`
}

type ScanResult struct {
	Path     string      `json:"path,omitempty"`
	Snapshot GitSnapshot `json:"snapshot"`
}
