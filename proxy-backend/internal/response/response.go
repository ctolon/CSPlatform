package response

import (
	"encoding/json"
	"net/http"
)

// WriteJSON writes a JSON response with the given status code.
func WriteJSON(w http.ResponseWriter, status int, data ...interface{}) error {
	if status == http.StatusNoContent {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		return nil
	}

	js, err := json.Marshal(data[0])
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

// ErrorJSON writes an error JSON response with the given status code.
func ErrorJSON(w http.ResponseWriter, err error, status ...int) {
	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	type jsonError struct {
		Message string `json:"message"`
	}

	theError := jsonError{
		Message: err.Error(),
	}

	WriteJSON(w, statusCode, theError)
}

func SuccessResponse(w http.ResponseWriter, data interface{}) interface{} {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "success",
		"data":    &data,
	})
}

func BadRequest(w http.ResponseWriter, data interface{}) interface{} {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	return json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "data validation failed",
		"data":    &data,
	})
}

func InternalServerError(w http.ResponseWriter, data interface{}) interface{} {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	return json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "internal server error",
		"data":    &data,
	})
}

type ApiResponse[T any] struct {
	ResponseKey     string `json:"response_key"`
	ResponseMessage string `json:"response_message"`
	Data            T      `json:"data"`
}

func Null() interface{} {
	return nil
}

type ResponseStatus int
type Headers int
type General int

const (

	// Success represents a successful response.
	Success ResponseStatus = iota + 1

	// DataNotFound represents a response when the data is not found.
	DataNotFound

	// UnknownError represents a response when an unknown error occurs.
	UnknownError

	// InvalidRequest represents a response when the request is invalid.
	InvalidRequest

	// Unauthorized represents a response when the request is unauthorized.
	Unauthorized
)

// GetResponseStatus returns the status of the response.
func (r ResponseStatus) GetResponseStatus() string {
	return [...]string{"SUCCESS", "DATA_NOT_FOUND", "UNKNOWN_ERROR", "INVALID_REQUEST", "UNAUTHORIZED"}[r-1]
}

// GetResponseMessage returns the message of the response status.
func (r ResponseStatus) GetResponseMessage() string {
	return [...]string{"Success", "Data Not Found", "Unknown Error", "Invalid Request", "Unauthorized"}[r-1]
}

// BuildResponse builds a response with the given status.
func BuildResponse[T any](responseStatus ResponseStatus, data T) ApiResponse[T] {
	return BuildResponse_(responseStatus.GetResponseStatus(), responseStatus.GetResponseMessage(), data)
}

// BuildResponse_ builds a response with the given status and message.
func BuildResponse_[T any](status string, message string, data T) ApiResponse[T] {
	return ApiResponse[T]{
		ResponseKey:     status,
		ResponseMessage: message,
		Data:            data,
	}
}

// JsonStringToSlice converts a json string to a slice of strings
func JsonStringToSlice(jsonString string) ([]string, error) {

	var data map[string]interface{}
	err := json.Unmarshal([]byte(jsonString), &data)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}

	return keys, nil
}
