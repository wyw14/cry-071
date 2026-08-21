package application

import (
	"context"

	"github.com/wyw14/cry-071/internal/domain"
)

type CatalogRecords interface {
	GetArea(context.Context, string) (domain.PublicArea, error)
	ListAreas(context.Context) ([]domain.PublicArea, error)
	GetCategory(context.Context, string) (domain.FacilityCategory, error)
	GetSubject(context.Context, string) (domain.FeedbackSubject, error)
}

type CatalogService struct{ deps Dependencies }

func NewCatalogService(deps Dependencies) *CatalogService { return &CatalogService{deps: deps} }

func (s *CatalogService) Areas(ctx context.Context) ([]domain.PublicArea, error) {
	areas, err := s.deps.Repositories.ListAreas(ctx)
	if err != nil {
		return nil, err
	}
	active := make([]domain.PublicArea, 0, len(areas))
	for _, area := range areas {
		if area.Active {
			active = append(active, area)
		}
	}
	return active, nil
}

func (s *CatalogService) Project() domain.ProjectMetadata { return domain.Metadata() }
