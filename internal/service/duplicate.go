package service

import (
	"context"
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/wyw14/cry-071/internal/application"
	"github.com/wyw14/cry-071/internal/domain"
)

type DuplicateMatcher struct {
	repository application.FeedbackSearch
}

func NewDuplicateMatcher(repository application.FeedbackSearch) *DuplicateMatcher {
	return &DuplicateMatcher{repository: repository}
}

func (m *DuplicateMatcher) Find(ctx context.Context, feedback *domain.Feedback, limit int) ([]application.DuplicateCandidate, error) {
	if limit <= 0 {
		return nil, nil
	}
	page, err := m.repository.ListFeedbacks(ctx, application.FeedbackFilter{
		AreaID: feedback.AreaID, SubjectCode: feedback.SubjectCode, Page: 1, PageSize: 200,
		Sort: "created_at", Descending: true, Now: feedback.CreatedAt,
	})
	if err != nil {
		return nil, err
	}
	candidates := make([]application.DuplicateCandidate, 0)
	for _, existing := range page.Items {
		if existing.ID == feedback.ID || existing.Status.Terminal() {
			continue
		}
		titleScore := jaccard(tokens(feedback.Title), tokens(existing.Title))
		descriptionScore := jaccard(tokens(feedback.Description), tokens(existing.Description))
		locationScore := jaccard(tokens(feedback.Location), tokens(existing.Location))
		score := titleScore*0.50 + descriptionScore*0.30 + locationScore*0.20
		if score < 0.42 {
			continue
		}
		candidates = append(candidates, application.DuplicateCandidate{
			FeedbackID: existing.ID, AcceptanceNo: existing.AcceptanceNumber,
			Title: existing.Title, SimilarityScore: math.Round(score*1000) / 1000,
		})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].SimilarityScore > candidates[j].SimilarityScore })
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates, nil
}

func tokens(value string) map[string]struct{} {
	result := map[string]struct{}{}
	var token []rune
	flush := func() {
		if len(token) > 1 {
			result[string(token)] = struct{}{}
		}
		token = token[:0]
	}
	for _, current := range strings.ToLower(value) {
		if unicode.IsLetter(current) || unicode.IsDigit(current) {
			token = append(token, current)
			continue
		}
		flush()
	}
	flush()
	for _, current := range []rune(strings.ReplaceAll(value, " ", "")) {
		if unicode.Is(unicode.Han, current) {
			result[string(current)] = struct{}{}
		}
	}
	return result
}

func jaccard(left, right map[string]struct{}) float64 {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}
	intersection := 0
	union := make(map[string]struct{}, len(left)+len(right))
	for token := range left {
		union[token] = struct{}{}
		if _, ok := right[token]; ok {
			intersection++
		}
	}
	for token := range right {
		union[token] = struct{}{}
	}
	return float64(intersection) / float64(len(union))
}
