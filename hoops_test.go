package hoops

import (
        "encoding/json"
        "strings"
        "testing"
)

func TestContributedHoopSetField(t *testing.T) {
        var story string = "This is a test story"
        h := NewContributedHoop()
        if h.setField("Story", story); h.Attributes().Story != story {
                t.Errorf("h.Attributes().Story = %v, want %v", h.Attributes().Story, story)
        }
}

func TestContributedHoopMarshalJSON(t *testing.T) {
        var story string = "This is a test story"
        h := NewContributedHoop()
        h.setField("Story", story);
        if j, _ := json.Marshal(h); !strings.Contains(string(j), story) {
                t.Errorf("Expected json.Marshal(h) = \"%v\", expected it to contain \"%v\"", string(j), story)
        }
}
