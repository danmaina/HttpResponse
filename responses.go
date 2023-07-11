package handlers

import (
	"encoding/json"
	"github.com/danmaina/logger"
	"net/http"
)

type Response struct {
	Status int         `json:"status"`
	Error  error       `json:"error"`
	Body   interface{} `json:"body"`
}

func ReturnResponse(status int, err error, body interface{}, res http.ResponseWriter) {
	_ = Response{
		Status: status,
		Error:  err,
		Body:   body,
	}.returnResponse(res)
}

// returnResponses :- in json format
func (res Response) returnResponse(w http.ResponseWriter) error {

	logger.INFO("Setting the Response Header to json")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(res.Status)

	// Return the relevant response{error or actual response}
	var errE error

	if res.Error != nil {
		logger.ERR("Returning Error Body: ", res.Error)
		errE = json.NewEncoder(w).Encode(map[string]string{
			"error": res.Error.Error(),
		})
	} else {
		logger.DEBUG("Creating a new Json Encoder")
		errE = json.NewEncoder(w).Encode(res.Body)
	}

	if errE != nil {
		logger.ERR("Error while encoding the Response Body: ", errE)

		logger.DEBUG("Trying to Marshal the Response Body.")

		mbArr, errM := json.Marshal(res)

		if errM != nil {
			logger.ERR("Error while Marshalling the Response Body: ", errM)
			return errM
		}

		wRes, errW := w.Write(mbArr)
		if errW != nil {
			logger.ERR("Error while sending back the Marshaled Response Body: ", errW)
			return errW
		}

		logger.INFO("Generated the Marshaled JSON Response Body Successfully. Id: ", wRes)
	}

	return nil
}
