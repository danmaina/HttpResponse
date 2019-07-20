package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

type Response struct {
	Status int         `json:"status"`
	Error  error       `json:"error"`
	Body   interface{} `json:"body"`
}

// Return Responses in json format
func (res Response) ReturnResponse(w http.ResponseWriter) error {

	log.Println("Setting the Response Header to json")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(res.Status)

	// Return the relevant response{error or actual response}
	var errE error

	if res.Error != nil {
		log.Println("Returning Error Body: ", res.Error)
		errE = json.NewEncoder(w).Encode(map[string]string{
			"error": res.Error.Error(),
		})
	} else {
		log.Println("Returning Response Body")
		errE = json.NewEncoder(w).Encode(res.Body)
	}

	if errE != nil {
		log.Println("Got An Error while encoding the Response Body: ", errE)

		log.Println("Trying to Marshal the Response Body.")

		mbArr, errM := json.Marshal(res)

		if errM != nil {
			log.Println("Got an Error while Marshalling the Response Body: ", errM)
			return errM
		}

		wRes, errW := w.Write(mbArr)
		if errW != nil {
			log.Println("Got an error while sending back the Marshaled Response Body: ", errW)
			return errW
		}

		log.Println("Returned the Marshaled Response Body Successfully. Id: ", wRes)
	}

	return nil
}
