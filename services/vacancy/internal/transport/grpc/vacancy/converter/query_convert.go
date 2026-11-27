package converter

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	usecase "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancyquery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func view(value pb.VacancyView) (usecase.View, error) {
	switch value {
	case pb.VacancyView_VACANCY_VIEW_UNSPECIFIED, pb.VacancyView_VACANCY_VIEW_PUBLIC:
		return usecase.ViewPublic, nil
	case pb.VacancyView_VACANCY_VIEW_COMPANY:
		return usecase.ViewCompany, nil
	default:
		return 0, status.Error(codes.InvalidArgument, "invalid view")
	}
}

func timestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func money(value *domain.Amount) *string {
	if value == nil {
		return nil
	}
	result := fmt.Sprintf("%d.%02d", int64(*value)/100, int64(*value)%100)
	return &result
}

func toProtoDetails(d domain.Details) *pb.VacancyDetails {
	result := &pb.VacancyDetails{Title: d.Title, Description: d.Description, Location: d.Location, RequiredExperience: d.RequiredExperience, EmploymentType: d.EmploymentType}
	switch d.WorkFormat {
	case domain.WorkOffice:
		result.WorkFormat = pb.WorkFormat_WORK_FORMAT_OFFICE
	case domain.WorkRemote:
		result.WorkFormat = pb.WorkFormat_WORK_FORMAT_REMOTE
	case domain.WorkHybrid:
		result.WorkFormat = pb.WorkFormat_WORK_FORMAT_HYBRID
	}
	for _, skill := range d.Skills {
		result.Skills = append(result.Skills, &pb.VacancySkill{SkillKey: skill.Key, DisplayName: skill.DisplayName})
	}
	if c := d.Compensation; c != nil {
		comp := &pb.Compensation{Minimum: money(c.Minimum), Maximum: money(c.Maximum), Bonus: c.Bonus}
		if c.Currency != "" {
			comp.Currency = &c.Currency
		}
		if c.Period != "" {
			comp.Period = &c.Period
		}
		switch c.TaxBasis {
		case domain.TaxGross:
			comp.TaxBasis = pb.TaxBasis_TAX_BASIS_GROSS
		case domain.TaxNet:
			comp.TaxBasis = pb.TaxBasis_TAX_BASIS_NET
		}
		result.Compensation = comp
	}
	return result
}

func vacancy(entity *domain.Vacancy, mode usecase.View) *pb.Vacancy {
	if entity == nil {
		return nil
	}
	stateSnapshot := entity.Snapshot()
	s := &stateSnapshot
	if mode == usecase.ViewPublic {
		return &pb.Vacancy{Representation: &pb.Vacancy_PublicVacancy{PublicVacancy: &pb.VacancyCard{VacancyId: s.ID.String(), CompanyId: s.CompanyID.String(), Details: toProtoDetails(s.Details), PublishedAt: timestamp(s.PublishedAt)}}}
	}
	var state pb.VacancyStatus
	switch s.Status {
	case domain.StatusDraft:
		state = pb.VacancyStatus_VACANCY_STATUS_DRAFT
	case domain.StatusPublished:
		state = pb.VacancyStatus_VACANCY_STATUS_PUBLISHED
	case domain.StatusArchived:
		state = pb.VacancyStatus_VACANCY_STATUS_ARCHIVED
	case domain.StatusClosed:
		state = pb.VacancyStatus_VACANCY_STATUS_CLOSED
	}
	return &pb.Vacancy{Representation: &pb.Vacancy_CompanyVacancy{CompanyVacancy: &pb.CompanyVacancy{
		Card:     &pb.VacancyCard{VacancyId: s.ID.String(), CompanyId: s.CompanyID.String(), Details: toProtoDetails(s.Details), PublishedAt: timestamp(s.PublishedAt)},
		AuthorId: s.AuthorID.String(), Status: state, Version: s.Version, CreatedAt: timestamp(s.CreatedAt),
		UpdatedAt: timestamp(s.UpdatedAt), FirstPublishedAt: timestamp(s.FirstPublishedAt), ClosedAt: timestamp(s.ClosedAt),
	}}}
}

func ToGetVacancyInput(req *pb.GetVacancyByIDRequest) (uuid.UUID, usecase.View, error) {
	id, err := ParseVacancyID(req.GetVacancyId())
	if err != nil {
		return uuid.Nil, 0, err
	}
	mode, err := view(req.GetView())
	if err != nil {
		return uuid.Nil, 0, err
	}
	return id, mode, nil
}

func ToListCompanyInput(req *pb.ListCompanyVacanciesRequest) (usecase.ListInput, error) {
	id, err := uuid.Parse(req.GetCompanyId())
	if err != nil {
		return usecase.ListInput{}, status.Error(codes.InvalidArgument, "invalid company id")
	}
	mode, err := view(req.GetView())
	if err != nil {
		return usecase.ListInput{}, err
	}
	pageSize := int(req.GetPageSize())
	if pageSize == 0 {
		pageSize = 20
	}
	cursor, err := decodePageToken(req.GetPageToken(), id, mode)
	if err != nil {
		return usecase.ListInput{}, err
	}
	return usecase.ListInput{CompanyID: id, View: mode, PageSize: pageSize, Cursor: cursor}, nil
}

func ToListCompanyResponse(result domain.Page, input usecase.ListInput) (*pb.ListCompanyVacanciesResponse, error) {
	nextPageToken, err := encodePageToken(result.Next, input.CompanyID, input.View)
	if err != nil {
		return nil, err
	}
	response := &pb.ListCompanyVacanciesResponse{NextPageToken: nextPageToken}
	for _, item := range result.Vacancies {
		response.Vacancies = append(response.Vacancies, vacancy(item, input.View))
	}
	return response, nil
}

func ToGetVacancyResponse(entity *domain.Vacancy, mode usecase.View) *pb.GetVacancyByIDResponse {
	return &pb.GetVacancyByIDResponse{Vacancy: vacancy(entity, mode)}
}
