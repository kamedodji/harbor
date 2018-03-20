package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/vmware/harbor/src/jobservice_v2/opm"

	"github.com/gorilla/mux"

	"github.com/vmware/harbor/src/jobservice_v2/core"
	"github.com/vmware/harbor/src/jobservice_v2/errs"
	"github.com/vmware/harbor/src/jobservice_v2/models"
)

//Handler defines approaches to handle the http requests.
type Handler interface {
	//HandleLaunchJobReq is used to handle the job submission request.
	HandleLaunchJobReq(w http.ResponseWriter, req *http.Request)

	//HandleGetJobReq is used to handle the job stats query request.
	HandleGetJobReq(w http.ResponseWriter, req *http.Request)

	//HandleJobActionReq is used to handle the job action requests (stop/retry).
	HandleJobActionReq(w http.ResponseWriter, req *http.Request)

	//HandleCheckStatusReq is used to handle the job service healthy status checking request.
	HandleCheckStatusReq(w http.ResponseWriter, req *http.Request)
}

//DefaultHandler is the default request handler which implements the Handler interface.
type DefaultHandler struct {
	controller *core.Controller
}

//NewDefaultHandler is constructor of DefaultHandler.
func NewDefaultHandler(ctl *core.Controller) *DefaultHandler {
	return &DefaultHandler{
		controller: ctl,
	}
}

//HandleLaunchJobReq is implementation of method defined in interface 'Handler'
func (dh *DefaultHandler) HandleLaunchJobReq(w http.ResponseWriter, req *http.Request) {
	if !dh.preCheck(w) {
		return
	}

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		dh.handleError(w, http.StatusInternalServerError, errs.ReadRequestBodyError(err))
		return
	}

	//unmarshal data
	jobReq := models.JobRequest{}
	if err = json.Unmarshal(data, &jobReq); err != nil {
		dh.handleError(w, http.StatusInternalServerError, errs.HandleJSONDataError(err))
		return
	}

	//Pass request to the controller for the follow-up.
	jobStats, err := dh.controller.LaunchJob(jobReq)
	if err != nil {
		dh.handleError(w, http.StatusInternalServerError, errs.LaunchJobError(err))
		return
	}

	data, ok := dh.handleJSONData(w, jobStats)
	if !ok {
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write(data)
}

//HandleGetJobReq is implementation of method defined in interface 'Handler'
func (dh *DefaultHandler) HandleGetJobReq(w http.ResponseWriter, req *http.Request) {
	if !dh.preCheck(w) {
		return
	}

	vars := mux.Vars(req)
	jobID := vars["job_id"]

	jobStats, err := dh.controller.GetJob(jobID)
	if err != nil {
		dh.handleError(w, http.StatusInternalServerError, errs.GetJobStatsError(err))
		return
	}

	data, ok := dh.handleJSONData(w, jobStats)
	if !ok {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

//HandleJobActionReq is implementation of method defined in interface 'Handler'
func (dh *DefaultHandler) HandleJobActionReq(w http.ResponseWriter, req *http.Request) {
	if !dh.preCheck(w) {
		return
	}

	vars := mux.Vars(req)
	jobID := vars["job_id"]

	data, err := ioutil.ReadAll(req.Body)
	if err != nil {
		dh.handleError(w, http.StatusInternalServerError, errs.ReadRequestBodyError(err))
		return
	}

	//unmarshal data
	jobActionReq := models.JobActionRequest{}
	if err = json.Unmarshal(data, &jobActionReq); err != nil {
		dh.handleError(w, http.StatusInternalServerError, errs.HandleJSONDataError(err))
		return
	}

	if jobActionReq.Action == opm.CtlCommandStop {
		if err := dh.controller.StopJob(jobID); err != nil {
			dh.handleError(w, http.StatusInternalServerError, errs.StopJobError(err))
			return
		}
	} else if jobActionReq.Action == opm.CtlCommandCancel {
		if err := dh.controller.StopJob(jobID); err != nil {
			dh.handleError(w, http.StatusInternalServerError, errs.StopJobError(err))
			return
		}
	} else {
		dh.handleError(w, http.StatusNotImplemented, errs.UnknownActionNameError(fmt.Errorf("%s", jobID)))
		return
	}

	w.WriteHeader(http.StatusOK)
}

//HandleCheckStatusReq is implementation of method defined in interface 'Handler'
func (dh *DefaultHandler) HandleCheckStatusReq(w http.ResponseWriter, req *http.Request) {
	if !dh.preCheck(w) {
		return
	}

	stats, err := dh.controller.CheckStatus()
	if err != nil {
		dh.handleError(w, http.StatusInternalServerError, errs.CheckStatsError(err))
		return
	}

	data, ok := dh.handleJSONData(w, stats)
	if !ok {
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (dh *DefaultHandler) handleJSONData(w http.ResponseWriter, object interface{}) ([]byte, bool) {
	data, err := json.Marshal(object)
	if err != nil {
		dh.handleError(w, http.StatusInternalServerError, errs.HandleJSONDataError(err))
		return nil, false
	}

	return data, true
}

func (dh *DefaultHandler) handleError(w http.ResponseWriter, code int, err error) {
	w.WriteHeader(code)
	w.Write([]byte(err.Error()))
}

func (dh *DefaultHandler) preCheck(w http.ResponseWriter) bool {
	if dh.controller == nil {
		dh.handleError(w, http.StatusInternalServerError, errs.MissingBackendHandlerError(fmt.Errorf("nil controller")))
		return false
	}

	return true
}
