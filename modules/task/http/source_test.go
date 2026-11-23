package http

import (
	"encoding/json"
	"testing"

	"github.com/programme-lv/backend/modules/task/srvc"
	"github.com/stretchr/testify/require"
)

func TestSourceMapping(t *testing.T) {
	h := &taskHttpHandler{}
	for _, tc := range []struct{ olympiad, org, source, legacyPreview string }{
		{"LIO", "PPS", "LIO", "LIO"}, {"", " pps ", "PPS", "other"}, {"", "", "other", "other"},
	} {
		preview := h.mapTaskPreview(srvc.TaskPreview{OriginOlympiad: tc.olympiad, OriginOrg: tc.org})
		require.Equal(t, tc.source, preview.SourceID)
		require.Equal(t, tc.legacyPreview, preview.OriginOlympiad)
		detail := h.mapTaskResponse(srvc.Task{OriginOlympiad: tc.olympiad, OriginOrg: tc.org, MdStatements: []srvc.MarkdownStatement{{}}})
		require.Equal(t, tc.source, detail.SourceID)
		require.Equal(t, tc.olympiad, detail.OriginOlympiad)
	}
	tree := mapFilterTree(srvc.BuildFilterTree([]srvc.OriginCount{{Org: "PPS", Count: 3}, {Count: 2}}))
	b, err := json.Marshal(tree)
	require.NoError(t, err)
	require.JSONEq(t, `{"olympiads":[{"id":"other","count":5,"years":[]}],"sources":[{"id":"PPS","kind":"institution","count":3,"years":[]},{"id":"other","kind":"other","count":2,"years":[]}]}`, string(b))
}
