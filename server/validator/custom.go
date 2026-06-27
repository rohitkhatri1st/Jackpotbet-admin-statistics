package validator

import (
	playgroundValidator "github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func registerCustomValidations(v *playgroundValidator.Validate) {
	v.RegisterValidation("decimal128", validateDecimal128)
}

// validateDecimal128 checks that the field is parseable as a Decimal128 number.
func validateDecimal128(fl playgroundValidator.FieldLevel) bool {
	_, err := bson.ParseDecimal128(fl.Field().String())
	return err == nil
}
