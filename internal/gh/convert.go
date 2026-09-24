package gh

import "time"

// toPR converts a raw search hit into a PR.
//
// The search endpoint is typed as Issue|PullRequest, so hits that are not pull
// requests come back as zero-valued nodes; those report ok == false. Nullable
// fields (author, reviewDecision, statusCheckRollup) are all handled here so
// the rest of the program can treat PR as fully populated.
func (n searchNode) toPR() (PR, bool) {
	if n.Repository.NameWithOwner == "" || n.Number == 0 {
		return PR{}, false
	}

	pr := PR{
		Number:       n.Number,
		Title:        n.Title,
		Body:         n.BodyText,
		URL:          n.URL,
		Repo:         n.Repository.NameWithOwner,
		IsDraft:      n.IsDraft,
		CreatedAt:    parseTime(n.CreatedAt),
		UpdatedAt:    parseTime(n.UpdatedAt),
		Additions:    n.Additions,
		Deletions:    n.Deletions,
		ChangedFiles: n.ChangedFiles,
		Comments:     n.Comments.TotalCount,
		HeadRef:      n.HeadRefName,
		BaseRef:      n.BaseRefName,
	}

	if n.Author != nil {
		pr.Author = n.Author.Login
		pr.IsBot = n.Author.Typename == "Bot"
	}
	if n.ReviewDecision != nil {
		pr.Review = Review(*n.ReviewDecision)
	}
	if len(n.Commits.Nodes) > 0 {
		if rollup := n.Commits.Nodes[0].Commit.StatusCheckRollup; rollup != nil {
			pr.Check = Check(rollup.State)
		}
	}
	for _, l := range n.Labels.Nodes {
		if l.Name != "" {
			pr.Labels = append(pr.Labels, l.Name)
		}
	}
	for _, rr := range n.ReviewRequests.Nodes {
		if rr.RequestedReviewer == nil {
			continue
		}
		// A requested reviewer is either a User (login) or a Team (slug).
		switch {
		case rr.RequestedReviewer.Login != "":
			pr.Reviewers = append(pr.Reviewers, rr.RequestedReviewer.Login)
		case rr.RequestedReviewer.Slug != "":
			pr.Reviewers = append(pr.Reviewers, rr.RequestedReviewer.Slug)
		}
	}

	return pr, true
}

// parseTime accepts GitHub's RFC 3339 timestamps and yields the zero time for
// anything it cannot read, which the UI renders as an unknown age rather than
// failing the whole refresh.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
