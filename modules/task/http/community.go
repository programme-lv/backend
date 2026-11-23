package http

import submsrvc "github.com/programme-lv/backend/modules/subm/srvc"

// mapCommunityStats converts the service's batch result into the public task
// preview shape. Tasks without completed attempts are absent rather than
// rendered as zero, and zero-valued verdict keys are dropped so the payload
// only carries positive counts.
func mapCommunityStats(stats map[string]submsrvc.TaskCommunityStats) map[string]CommunityStats {
	if len(stats) == 0 {
		return nil
	}

	mapped := make(map[string]CommunityStats, len(stats))
	for taskID, stat := range stats {
		if stat.UniqueAttempted <= 0 {
			continue
		}
		verdicts := make(map[string]int, len(stat.Verdicts))
		for verdict, count := range stat.Verdicts {
			if count > 0 {
				verdicts[verdict] = count
			}
		}
		mapped[taskID] = CommunityStats{
			UniqueSolvers:   stat.UniqueSolvers,
			UniqueAttempted: stat.UniqueAttempted,
			Verdicts:        verdicts,
		}
	}
	if len(mapped) == 0 {
		return nil
	}
	return mapped
}
