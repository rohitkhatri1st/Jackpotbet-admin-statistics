package validator

import (
	"errors"
	"fmt"
	"strings"

	playgroundValidator "github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *playgroundValidator.Validate
}

func NewValidator() *Validator {
	v := playgroundValidator.New(playgroundValidator.WithRequiredStructEnabled())
	registerCustomValidations(v)
	return &Validator{validate: v}
}

// Validate validates a struct based on its `validate` tags.
// Returns nil if valid, or a formatted error describing all validation failures.
func (v *Validator) Validate(s any) error {
	err := v.validate.Struct(s)
	if err == nil {
		return nil
	}

	var invalidErr *playgroundValidator.InvalidValidationError
	if errors.As(err, &invalidErr) {
		return err
	}

	var validationErrs playgroundValidator.ValidationErrors
	if errors.As(err, &validationErrs) {
		msgs := make([]string, 0, len(validationErrs))
		for _, e := range validationErrs {
			msgs = append(msgs, fmt.Sprintf("field '%s' failed on '%s'", e.Field(), e.Tag()))
		}
		return fmt.Errorf("validation failed: %s", strings.Join(msgs, "; "))
	}

	return err
}
