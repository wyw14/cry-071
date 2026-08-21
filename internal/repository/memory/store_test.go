package memory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wyw14/cry-071/internal/application"
	"github.com/wyw14/cry-071/internal/domain"
	"github.com/wyw14/cry-071/internal/repository/memory"
)

func TestTransactionRollbackDoesNotExposePartialFeedback(t *testing.T) {
	store := memory.NewStore()
	feedback, err := domain.CreateFeedback(domain.NewFeedback{ID: "f-rollback", AcceptanceNumber: "PSF-R", AreaID: "area", FacilityCategoryID: "lighting", SubjectCode: "damaged", Priority: domain.PriorityNormal, Title: "广场座椅出现破损", Description: "广场北侧座椅表面破损可能划伤使用者", Location: "广场北侧", SubmitterName: "市民", CreatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	expected := errors.New("force rollback")
	err = store.WithinTransaction(context.Background(), func(ctx context.Context, repos application.Repositories) error {
		if err := repos.Create(ctx, feedback); err != nil {
			return err
		}
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected rollback cause, got %v", err)
	}
	if _, err := store.Get(context.Background(), feedback.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("rolled back feedback became visible: %v", err)
	}
}

func TestOptimisticUpdateAllowsOnlyOneConcurrentWriter(t *testing.T) {
	store := memory.NewStore()
	now := time.Now().UTC()
	feedback, err := domain.CreateFeedback(domain.NewFeedback{ID: "f-race", AcceptanceNumber: "PSF-C", AreaID: "area", FacilityCategoryID: "lighting", SubjectCode: "safety", Priority: domain.PriorityHigh, Title: "步道护栏存在松动", Description: "滨河步道转角处护栏晃动存在跌落风险", Location: "滨河步道转角", SubmitterName: "市民", CreatedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Create(context.Background(), feedback); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, assignee := range []string{"agent-a", "agent-b"} {
		go func(assignee string) {
			current, err := store.Get(context.Background(), feedback.ID)
			if err != nil {
				results <- err
				return
			}
			<-start
			if err := current.Assign(domain.Actor{ID: "manager", Role: domain.RoleManager}, assignee, now.Add(time.Minute)); err != nil {
				results <- err
				return
			}
			results <- store.Update(context.Background(), current, feedback.Version)
		}(assignee)
	}
	close(start)
	success, conflict := 0, 0
	for range 2 {
		err := <-results
		if err == nil {
			success++
		} else if errors.Is(err, domain.ErrVersionConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflict)
	}
}
