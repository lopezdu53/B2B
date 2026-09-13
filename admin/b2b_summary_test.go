package admin

import "testing"

func TestReconcileLeadSummaryCapsOrphanProspects(t *testing.T) {
	t.Parallel()

	sum := B2BSummary{Total: 28, Prospects: 900, InProgress: 2}
	ReconcileLeadSummary(&sum, false)

	if sum.Total != 28 {
		t.Fatalf("total: got %d", sum.Total)
	}

	if sum.Prospects != 26 {
		t.Fatalf("prospects should follow the scrape pool, got %d", sum.Prospects)
	}

	if sum.InProgress != 2 {
		t.Fatalf("in progress: got %d", sum.InProgress)
	}
}

func TestReconcileLeadSummaryFillsMissingProspects(t *testing.T) {
	t.Parallel()

	sum := B2BSummary{Total: 40, Prospects: 3, Clients: 2}
	ReconcileLeadSummary(&sum, false)

	if sum.Prospects != 38 {
		t.Fatalf("untracked scrapes should count as prospects, got %d", sum.Prospects)
	}
}

func TestReconcileLeadSummaryAdvisorScope(t *testing.T) {
	t.Parallel()

	sum := B2BSummary{Total: 99, Prospects: 4, Clients: 1, InProgress: 1}
	ReconcileLeadSummary(&sum, true)

	if sum.Total != 6 {
		t.Fatalf("advisor total should be the status sum, got %d", sum.Total)
	}
}
