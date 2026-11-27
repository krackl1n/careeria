package converter

import (
	"github.com/google/uuid"
	pb "github.com/krackl1n/careeria/contracts/gen/careeria/vacancy/v1"
	usecase "github.com/krackl1n/careeria/services/vacancy/internal/usecase/vacancy"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ToUpdateDetailsInput(req *pb.UpdateVacancyDetailsRequest) (uuid.UUID, usecase.DetailsPatch, error) {
	id, err := uuid.Parse(req.VacancyId)
	if err != nil {
		return uuid.Nil, usecase.DetailsPatch{}, status.Error(codes.InvalidArgument, "invalid vacancy id")
	}
	if req.UpdateMask == nil || len(req.UpdateMask.Paths) == 0 {
		return uuid.Nil, usecase.DetailsPatch{}, status.Error(codes.InvalidArgument, "update mask is required")
	}
	var patch usecase.DetailsPatch
	seen := map[string]bool{}
	for _, path := range req.UpdateMask.Paths {
		if seen[path] {
			return uuid.Nil, usecase.DetailsPatch{}, status.Error(codes.InvalidArgument, "duplicate update mask path")
		}
		seen[path] = true
		switch path {
		case "title":
			if req.Title == nil {
				return uuid.Nil, usecase.DetailsPatch{}, status.Error(codes.InvalidArgument, "title is required")
			}
			patch.Title.Set, patch.Title.Value = true, *req.Title
		case "description":
			if req.Description == nil {
				return uuid.Nil, usecase.DetailsPatch{}, status.Error(codes.InvalidArgument, "description is required")
			}
			patch.Description.Set, patch.Description.Value = true, *req.Description
		case "work_format":
			if req.WorkFormat == nil {
				return uuid.Nil, usecase.DetailsPatch{}, status.Error(codes.InvalidArgument, "work format is required")
			}
			patch.WorkFormat.Set = true
			patch.WorkFormat.Value, err = workFormat(*req.WorkFormat)
		case "location":
			patch.Location.Set, patch.Location.Value = true, req.Location
		case "required_experience":
			patch.RequiredExperience.Set, patch.RequiredExperience.Value = true, req.RequiredExperience
		case "employment_type":
			patch.EmploymentType.Set, patch.EmploymentType.Value = true, req.EmploymentType
		case "compensation":
			patch.Compensation.Set = true
			patch.Compensation.Value, err = compensation(req.Compensation)
		case "skills":
			patch.Skills.Set = true
			patch.Skills.Value, err = skills(req.Skills)
		default:
			return uuid.Nil, usecase.DetailsPatch{}, status.Error(codes.InvalidArgument, "unknown update mask path")
		}
		if err != nil {
			return uuid.Nil, usecase.DetailsPatch{}, err
		}
	}
	return id, patch, nil
}
