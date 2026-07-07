package leftoff

import (
	"strings"
	"testing"
)

func FuzzParseCapture(f *testing.F) {
	for _, seed := range []struct {
		kind string
		text string
	}{
		{"", "task: Add release smoke test"},
		{"decision", "Use Markdown because records must stay editable"},
		{"solution", "Docker requires a running daemon before build commands work"},
		{"activity", "should be rejected as a direct capture kind"},
		{"", ""},
	} {
		f.Add(seed.kind, seed.text)
	}

	f.Fuzz(func(t *testing.T, kind string, text string) {
		parsed, err := ParseCapture(kind, text)
		if err != nil {
			return
		}
		if strings.TrimSpace(parsed.Text) == "" {
			t.Fatalf("successful parse returned empty text")
		}
		if parsed.Type == "" || parsed.Type == RecordActivityEvent {
			t.Fatalf("successful parse returned invalid capture type %q", parsed.Type)
		}
	})
}

func FuzzParseMarkdownRecords(f *testing.F) {
	for _, seed := range []string{
		"",
		"# inbox\n\n## TASK-2026-07-06-001 - Write tests\n\n- Type: task\n- Status: active\n- Project: sample\n",
		"## DECISION-2026-07-06-001 - Use Markdown\n\n- Type: decision\n- Status: accepted\n",
		"## not a record\n\n- malformed: yes\n",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, content string) {
		records := ParseMarkdownRecords(content, "fuzz.md", "")
		for _, record := range records {
			if record.ID == "" {
				t.Fatalf("record has empty ID: %#v", record)
			}
			if record.Type == "" {
				t.Fatalf("record has empty type: %#v", record)
			}
			if record.Project == "" {
				t.Fatalf("record has empty project: %#v", record)
			}
		}
	})
}

func FuzzRedactRemoteURL(f *testing.F) {
	for _, seed := range []string{
		"",
		"https://github.com/example/repo.git",
		"https://user:secret@example.com/org/repo.git",
		"git@example.com:org/repo.git",
		"https://token:ghp_abcdefghijklmnopqrstuvwxyzABCDE@example.com/org/repo.git",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, remote string) {
		redacted := RedactRemoteURL(remote)
		if remote == "https://user:secret@example.com/org/repo.git" && strings.Contains(redacted, "user:secret@") {
			t.Fatalf("redacted URL leaked userinfo: %q", redacted)
		}
	})
}
