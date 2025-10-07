package validation

import (
	"errors"
	"fmt"
	"mime/multipart"
	"reflect"
	"regexp"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-multierror"
	"github.com/shopspring/decimal"
)

var validate = validator.New()

func init() {
	registerNoSpecialCharacters()
	registerNoSpacesAtStartOrEnd()
	registerDate()
	registerDatetime()
	registerReconFileMustCSV()
	registerDecimalGreaterThan()
	registerISO8601DateTme()
}

func ValidateStruct(toValidate interface{}) error {
	// register function to get tag name from json tags.
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	var errs *multierror.Error
	if err := validate.Struct(toValidate); err != nil {
		// this check is only needed when your code could produce
		// an invalid value for validation such as interface with nil
		// value most including myself do not usually have code like this.
		if _, ok := err.(*validator.InvalidValidationError); ok {
			errs = multierror.Append(errs, ErrorValidateResponse{
				Message: err.Error(),
			})
			return errs.ErrorOrNil()
		}

		var valErrs validator.ValidationErrors
		if errors.As(err, &valErrs) {
			for _, valErr := range valErrs {
				key := fmt.Sprintf("%s_%s", valErr.Namespace(), valErr.Tag())
				data, found := models.MapErrors[key]
				if found {
					errorResponse := ErrorValidateResponse{
						Code:    data.Code,
						Field:   valErr.Field(),
						Message: data.ErrorMessage.Error(),
					}
					errs = multierror.Append(errs, errorResponse)
				} else {
					key := fmt.Sprintf("%s_%s", valErr.Field(), valErr.Tag())
					if data, found := models.MapErrors[key]; found {
						errorResponse := ErrorValidateResponse{
							Code:    data.Code,
							Field:   valErr.Field(),
							Message: data.ErrorMessage.Error(),
						}
						errs = multierror.Append(errs, errorResponse)
					} else {
						errorResponse := ErrorValidateResponse{
							Code:    "UNKNOW",
							Field:   valErr.Field(),
							Message: strings.TrimSpace(fmt.Sprintf("%s %s", valErr.Tag(), valErr.Param())),
						}
						errs = multierror.Append(errs, errorResponse)
					}
				}
			}
		}
	}

	return errs.ErrorOrNil()
}

func registerDecimalGreaterThan() {
	validate.RegisterCustomTypeFunc(func(field reflect.Value) interface{} {
		if valuer, ok := field.Interface().(models.Decimal); ok {
			return valuer.String()
		}
		return nil
	}, models.Decimal{})

	validate.RegisterValidation("decimalGreaterThan", func(fl validator.FieldLevel) bool {
		data, ok := fl.Field().Interface().(string)
		if !ok {
			return false
		}

		value, err := decimal.NewFromString(data)
		if err != nil {
			return false
		}
		inputUser := models.NewDecimalFromExternal(value)

		parameterValue, err := models.NewDecimal(fl.Param())
		if err != nil {
			return false
		}

		return inputUser.GreaterThan(parameterValue.Decimal)
	})

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

func registerDate() {
	validate.RegisterValidation("date", func(fl validator.FieldLevel) bool {
		input := fl.Field().String()
		pattern := `\d{4}-\d{2}-\d{2}`
		return regexp.MustCompile(pattern).MatchString(input)
	})
}

func registerDatetime() {
	validate.RegisterValidation("datetime", func(fl validator.FieldLevel) bool {
		input := fl.Field().String()
		pattern := `\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`
		return regexp.MustCompile(pattern).MatchString(input)
	})
}

func registerISO8601DateTme() {
	validate.RegisterValidation("iso8601datetime", func(fl validator.FieldLevel) bool {
		input := fl.Field().String()
		if input != "" {
			_, err := time.Parse(time.RFC3339, input)
			return err == nil
		}

		return true
	})
}

func registerReconFileMustCSV() {
	validate.RegisterStructValidation(func(sl validator.StructLevel) {
		// check that top is expected one
		if !(sl.Top().Type() == reflect.TypeOf((*models.UploadReconFileRequest)(nil))) {
			return
		}

		// Check
		fileRecon, ok := sl.Current().Interface().(multipart.FileHeader)
		if !ok {
			sl.ReportError(fileRecon.Filename, "reconFile", "TBD", "invalidType", "")
			return
		}
		if !strings.HasSuffix(strings.ToLower(fileRecon.Filename), ".csv") {
			sl.ReportError(fileRecon.Filename, "reconFile", "TBD", "mustCSV", "")
			return
		}
	}, &multipart.FileHeader{})
}
