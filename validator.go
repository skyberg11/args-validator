package validator

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var ErrNotStruct = errors.New("wrong argument given, should be a struct")
var ErrInvalidValidatorSyntax = errors.New("invalid validator syntax")
var ErrValidateForUnexportedFields = errors.New("validation for unexported field is not allowed")
var ErrInvalidatedField = errors.New("field invalidated")
var ErrUnsupportedType = errors.New("type not supported")

type ValidationError struct {
	FieldName string
	Err       error
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	var sb strings.Builder
	for _, err := range v {
		if errors.Is(err.Err, ErrInvalidValidatorSyntax) {
			sb.WriteString(err.Err.Error())
		} else if errors.Is(err.Err, ErrValidateForUnexportedFields) {
			sb.WriteString(err.Err.Error())
		} else {
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", err.FieldName, err.Err.Error()))
		}
	}
	return sb.String()
}

func validateStringLen(str string, validateTag string) error {
	splitted := strings.Split(validateTag, ":")
	length, _ := strconv.Atoi(splitted[1])
	if len(str) != length {
		return ErrInvalidatedField
	}
	return nil
}

func validateStringMinMax(str string, validateTag string) error {
	splitted := strings.Split(validateTag, ":")
	length, _ := strconv.Atoi(splitted[1])
	switch splitted[0] {
	case "min":
		if len(str) < length {
			return ErrInvalidatedField
		}
	case "max":
		if len(str) > length {
			return ErrInvalidatedField
		}
	}
	return nil
}

func validateStringIn(str string, validateTag string) error {
	splitted := strings.Split(validateTag, ":")
	allowed := strings.Split(splitted[1], ",")
	for _, s := range allowed {
		if s == str {
			return nil
		}
	}
	return ErrInvalidatedField
}

func validateIntIn(num int, validateTag string) error {
	splitted := strings.Split(validateTag, ":")
	allowed := strings.Split(splitted[1], ",")
	for _, s := range allowed {
		i, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		if i == num {
			return nil
		}
	}
	return ErrInvalidatedField
}

func validateSyntax(validateTag string) bool {
	tags := strings.Split(validateTag, ";")
	for _, tag := range tags {
		splitted := strings.Split(tag, ":")
		if len(splitted) < 2 {
			return true
		}
		switch splitted[0] {
		case "in":
			if len(splitted) < 2 || len(splitted[1]) == 0 {
				return true
			}
		case "len", "min", "max":
			if len(splitted) != 2 || len(splitted[1]) == 0 {
				return true
			}
			if _, err := strconv.Atoi(splitted[1]); err != nil {
				return true
			}
		}
	}
	return false
}

func validateIntMinMax(num int, validateTag string) error {
	splitted := strings.Split(validateTag, ":")
	length, _ := strconv.Atoi(splitted[1])
	switch splitted[0] {
	case "min":
		if num < length {
			return ErrInvalidatedField
		}
	case "max":
		if num > length {
			return ErrInvalidatedField
		}
	}
	return nil
}

func validateString(str string, validateTag string) error {
	switch strings.Split(validateTag, ":")[0] {
	case "in":
		if err := validateStringIn(str, validateTag); err != nil {
			return err
		}
	case "len":
		if err := validateStringLen(str, validateTag); err != nil {
			return err
		}
	case "min", "max":
		if err := validateStringMinMax(str, validateTag); err != nil {
			return err
		}
	}
	return nil
}

func validateInt(num int, validateTag string) error {
	switch strings.Split(validateTag, ":")[0] {
	case "in":
		if err := validateIntIn(num, validateTag); err != nil {
			return err
		}
	case "min", "max":
		if err := validateIntMinMax(num, validateTag); err != nil {
			return err
		}
	}
	return nil
}

func Validate(v any) error {
	valueStruct := reflect.ValueOf(v)
	typeStruct := reflect.TypeOf(v)
	if valueStruct.Kind() != reflect.Struct {
		return ErrNotStruct
	}

	var errs ValidationErrors

	for i := 0; i < valueStruct.NumField(); i++ {
		valueField := valueStruct.Field(i)
		typeField := typeStruct.Field(i)

		validateTag := typeField.Tag.Get("validate")

		if validateTag == "" {
			continue
		}

		if !typeField.IsExported() {
			errs = append(errs, ValidationError{FieldName: valueField.Type().Name(), Err: ErrValidateForUnexportedFields})
			continue
		}

		if validateSyntax(validateTag) {
			errs = append(errs, ValidationError{FieldName: valueField.Type().Name(), Err: ErrInvalidValidatorSyntax})
			continue
		}

		for _, tags := range strings.Split(validateTag, ";") {
			switch typeField.Type.Kind() {
			case reflect.String:
				if err := validateString(valueField.String(), tags); err != nil {
					errs = append(errs, ValidationError{FieldName: valueField.Type().Name(), Err: err})
				}
			case reflect.Int:
				if err := validateInt(int(valueField.Int()), tags); err != nil {
					errs = append(errs, ValidationError{FieldName: valueField.Type().Name(), Err: err})
				}
			case reflect.Slice:
				if valueField.Type().Elem().Kind() == reflect.Int {
					for _, num := range valueField.Interface().([]int) {
						if err := validateInt(num, tags); err != nil {
							errs = append(errs, ValidationError{FieldName: valueField.Type().Name(), Err: err})
						}
					}
				} else if valueField.Type().Elem().Kind() == reflect.String {
					for _, str := range valueField.Interface().([]string) {
						if err := validateString(str, tags); err != nil {
							errs = append(errs, ValidationError{FieldName: valueField.Type().Name(), Err: err})
						}
					}
				} else {
					errs = append(errs, ValidationError{FieldName: valueField.Type().Name(), Err: ErrUnsupportedType})
				}
			default:
				errs = append(errs, ValidationError{FieldName: valueField.Type().Name(), Err: ErrUnsupportedType})
			}
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}
