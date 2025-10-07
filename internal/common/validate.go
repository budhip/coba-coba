package common

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

type ErrorValidateResponse struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message,omitempty"`
}

var validate = validator.New()

func init() {
	registerNoSpecialCharacters()
	registerNoSpacesAtStartOrEnd()
}

func ValidateStruct(toValidate interface{}) []*ErrorValidateResponse {
	// register function to get tag name from json tags.
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	var errorResponse []*ErrorValidateResponse
	if err := validate.Struct(toValidate); err != nil {
		// this check is only needed when your code could produce
		// an invalid value for validation such as interface with nil
		// value most including myself do not usually have code like this.
		if _, ok := err.(*validator.InvalidValidationError); ok {
			errorResponse = append(errorResponse, &ErrorValidateResponse{
				Message: err.Error(),
			})
			return errorResponse
		}

		var valErrs validator.ValidationErrors
		if errors.As(err, &valErrs) {
			for _, valErr := range valErrs {
				errorResponse = append(errorResponse, &ErrorValidateResponse{
					Field:   valErr.Field(),
					Message: strings.TrimSpace(fmt.Sprintf("%s %s", valErr.Tag(), valErr.Param())),
				})
			}
		}
	}
	return errorResponse
}

func registerNoSpecialCharacters() {
	validate.RegisterValidation("nospecial", func(fl validator.FieldLevel) bool {
		input := fl.Field().String()
		// Define a regular expression pattern that allows only letters and digits.
		// Allow space
		pattern := "^[a-zA-Z0-9 ]*$"
		return regexp.MustCompile(pattern).MatchString(input)
	})
}

func registerNoSpacesAtStartOrEnd() {
	validate.RegisterValidation("noStartEndSpaces", func(fl validator.FieldLevel) bool {
		str := fl.Field().String()
		return str == "" || (str[0] != ' ' && str[len(str)-1] != ' ')
	})
}
