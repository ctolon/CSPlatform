package utils

import (
	"github.com/go-playground/validator/v10"
)

// ValidateMultipleStructs validates multiple structs
func ValidateMultipleStructs(structs ...any) error {

	validate := validator.New()

	for _, s := range structs {
		err := validate.Struct(s)
		if err != nil {
			return err
		}
	}

	return nil
}
