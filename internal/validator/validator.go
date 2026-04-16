// Package validator provides request validation decorators.
package validator

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
)

var (
	ErrEmailRequired     = errors.New("email is required")
	ErrEmailInvalid      = errors.New("invalid email format")
	ErrPasswordRequired  = errors.New("password is required")
	ErrPasswordTooShort  = errors.New("password must be at least 8 characters")
	ErrPasswordNoUpper   = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLower   = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoDigit   = errors.New("password must contain at least one digit")
	ErrTitleRequired     = errors.New("title is required")
	ErrTitleTooLong      = errors.New("title must be less than 500 characters")
	ErrFirstNameRequired = errors.New("first name is required")
	ErrFirstNameInvalid  = errors.New("first name contains invalid characters")
)

// emailRegex validates email format
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Validator interface for request validation
type Validator interface {
	Validate() error
}

// RegisterRequestValidator validates registration requests
type RegisterRequestValidator struct {
	Email      string
	Password   string
	ApprovedID string
}

func (v *RegisterRequestValidator) Validate() error {
	if err := ValidateEmail(v.Email); err != nil {
		return err
	}
	if err := ValidatePassword(v.Password); err != nil {
		return err
	}
	if v.ApprovedID == "" {
		return errors.New("approved_id is required")
	}
	return nil
}

// LoginRequestValidator validates login requests
type LoginRequestValidator struct {
	Email    string
	Password string
}

func (v *LoginRequestValidator) Validate() error {
	if err := ValidateEmail(v.Email); err != nil {
		return err
	}
	if v.Password == "" {
		return ErrPasswordRequired
	}
	return nil
}

// CreateTodoRequestValidator validates todo creation requests
type CreateTodoRequestValidator struct {
	Title string
}

func (v *CreateTodoRequestValidator) Validate() error {
	return ValidateTitle(v.Title)
}

// UpdateTodoRequestValidator validates todo update requests
type UpdateTodoRequestValidator struct {
	Title string
}

func (v *UpdateTodoRequestValidator) Validate() error {
	return ValidateTitle(v.Title)
}

// ApprovedUserRequestValidator validates approved user creation requests
type ApprovedUserRequestValidator struct {
	Email     string
	FirstName string
}

func (v *ApprovedUserRequestValidator) Validate() error {
	if err := ValidateEmail(v.Email); err != nil {
		return err
	}
	if err := ValidateFirstName(v.FirstName); err != nil {
		return err
	}
	return nil
}

// ValidateEmail checks email format
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return ErrEmailRequired
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrEmailInvalid
	}
	if !emailRegex.MatchString(email) {
		return ErrEmailInvalid
	}
	return nil
}

// ValidatePassword checks password strength
func ValidatePassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}
	if len(password) < 8 {
		return ErrPasswordTooShort
	}

	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}

	if !hasUpper {
		return ErrPasswordNoUpper
	}
	if !hasLower {
		return ErrPasswordNoLower
	}
	if !hasDigit {
		return ErrPasswordNoDigit
	}

	return nil
}

// ValidateTitle checks title validity
func ValidateTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrTitleRequired
	}
	if len(title) > 500 {
		return ErrTitleTooLong
	}
	return nil
}

// ValidateFirstName checks first name validity
func ValidateFirstName(firstName string) error {
	firstName = strings.TrimSpace(firstName)
	if firstName == "" {
		return ErrFirstNameRequired
	}
	// Allow letters, spaces, hyphens, and apostrophes
	for _, r := range firstName {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' && r != '\'' {
			return ErrFirstNameInvalid
		}
	}
	return nil
}
