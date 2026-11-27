package converter

import (
	"fmt"
	"strconv"
	"strings"

	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	domain "github.com/krackl1n/careeria/services/vacancy/internal/domain/vacancy"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func amount(value *string) (*domain.Amount, error) {
	if value == nil {
		return nil, nil
	}
	parts := strings.Split(*value, ".")
	if len(parts) > 2 || parts[0] == "" || len(parts[0]) > 16 {
		return nil, status.Error(codes.InvalidArgument, "invalid compensation")
	}
	fraction := "00"
	if len(parts) == 2 {
		if len(parts[1]) < 1 || len(parts[1]) > 2 {
			return nil, status.Error(codes.InvalidArgument, "invalid compensation")
		}
		fraction = (parts[1] + "0")[:2]
	}
	cents, err := strconv.ParseInt(parts[0]+fraction, 10, 64)
	if err != nil || cents < 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid compensation")
	}
	result := domain.Amount(cents)
	return &result, nil
}

func compensation(c *pb.Compensation) (*domain.Compensation, error) {
	if c == nil {
		return nil, nil
	}
	min, err := amount(c.Minimum)
	if err != nil {
		return nil, err
	}
	max, err := amount(c.Maximum)
	if err != nil {
		return nil, err
	}
	result := &domain.Compensation{Minimum: min, Maximum: max, Currency: c.GetCurrency(), Period: c.GetPeriod(), Bonus: c.Bonus}
	switch c.GetTaxBasis() {
	case pb.TaxBasis_TAX_BASIS_UNSPECIFIED:
	case pb.TaxBasis_TAX_BASIS_GROSS:
		result.TaxBasis = domain.TaxGross
	case pb.TaxBasis_TAX_BASIS_NET:
		result.TaxBasis = domain.TaxNet
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid tax basis")
	}
	return result, nil
}

func workFormat(value pb.WorkFormat) (domain.WorkFormat, error) {
	switch value {
	case pb.WorkFormat_WORK_FORMAT_OFFICE:
		return domain.WorkOffice, nil
	case pb.WorkFormat_WORK_FORMAT_REMOTE:
		return domain.WorkRemote, nil
	case pb.WorkFormat_WORK_FORMAT_HYBRID:
		return domain.WorkHybrid, nil
	default:
		return "", status.Error(codes.InvalidArgument, "invalid work format")
	}
}

func skills(input []*pb.VacancySkill) ([]domain.SkillRequirement, error) {
	result := make([]domain.SkillRequirement, 0, len(input))
	for _, item := range input {
		if item == nil {
			return nil, status.Error(codes.InvalidArgument, "invalid skill")
		}
		result = append(result, domain.SkillRequirement{Key: item.SkillKey, DisplayName: item.DisplayName})
	}
	return result, nil
}

func details(input *pb.VacancyDetails) (domain.Details, error) {
	if input == nil {
		return domain.Details{}, status.Error(codes.InvalidArgument, "details are required")
	}
	work, err := workFormat(input.WorkFormat)
	if err != nil {
		return domain.Details{}, err
	}
	comp, err := compensation(input.Compensation)
	if err != nil {
		return domain.Details{}, err
	}
	skillSet, err := skills(input.Skills)
	if err != nil {
		return domain.Details{}, err
	}
	return domain.Details{Title: input.Title, Description: input.Description, WorkFormat: work, Location: input.Location, RequiredExperience: input.RequiredExperience, EmploymentType: input.EmploymentType, Compensation: comp, Skills: skillSet}, nil
}

func statusValue(value domain.Status) pb.VacancyStatus {
	switch value {
	case domain.StatusDraft:
		return pb.VacancyStatus_VACANCY_STATUS_DRAFT
	case domain.StatusPublished:
		return pb.VacancyStatus_VACANCY_STATUS_PUBLISHED
	case domain.StatusArchived:
		return pb.VacancyStatus_VACANCY_STATUS_ARCHIVED
	case domain.StatusClosed:
		return pb.VacancyStatus_VACANCY_STATUS_CLOSED
	default:
		panic(fmt.Sprintf("unknown vacancy status %q", value))
	}
}
