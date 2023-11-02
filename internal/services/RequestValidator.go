package services

import (
	"git.ghpcard.local/csipitca/mistake"
	errorscs "github.com/healthchecker/internal/errorcs"
	"strings"
)

func newRequestValidator(servs *Services) *requestValidator {
	return &requestValidator{
		servs:        servs,
		RequestModel: r,
	}
}

type requestValFunc func(*Request) error

type requestValidator struct {
	RequestModel
	servs      *Services
	validators map[string]map[string][]requestValFunc
}

// defineContentTypeValidators is a function that return a slice of
// validator functions to each field of ContentType model
func defineRequestValidators(o *requestValidator) {
	o.validators = map[string]map[string][]requestValFunc{}
}

func runRequestValidation(request *Request, v *requestValidator, action string) *mistake.Mistake {
	mist := mistake.NewUserDefined()
	for keyField, valueField := range v.validators {
		for keyAction, valueAction := range valueField {
			if strings.Contains(keyAction, action) {
				err := runRequestValFuncs(request, valueAction...)
				if err != nil {
					if err == errorscs.ErrInternalServer {
						return mistake.NewInternalServer(err)
					}
					mist.AddField(keyField, err)
				}
			}
		}
	}
	return mist
}

func runRequestValFuncs(request *Request, fns ...requestValFunc) error {
	for _, fn := range fns {
		if err := fn(request); err != nil {
			return err
		}
	}
	return nil
}

// Validate is used to validate data before save into database
func (r *requestValidator) Validate(request *Request, action string) *mistake.Mistake {
	mist := runRequestValidation(request, r, action)
	if mist.IsInternalServer() {
		return mist
	} else if mist.IsUserDefined() {
		if len(mist.Fields) > 0 {
			return mist
		}
	}
	return nil
}
