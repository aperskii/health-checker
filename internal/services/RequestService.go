package services

func NewRequestService(servs *Services) RequestService {
	v := newRequestValidator(g, servs)
	defineRequestValidators(v)
	return &requestService{
		servs: servs,
	}
}

type RequestService interface {
	RequestModel
}

var _ RequestService = &requestService{}

type requestService struct {
	servs *Services
}
