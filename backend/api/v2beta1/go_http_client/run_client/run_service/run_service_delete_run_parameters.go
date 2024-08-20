// Code generated by go-swagger; DO NOT EDIT.

package run_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"

	strfmt "github.com/go-openapi/strfmt"
)

// NewRunServiceDeleteRunParams creates a new RunServiceDeleteRunParams object
// with the default values initialized.
func NewRunServiceDeleteRunParams() *RunServiceDeleteRunParams {
	var ()
	return &RunServiceDeleteRunParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewRunServiceDeleteRunParamsWithTimeout creates a new RunServiceDeleteRunParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewRunServiceDeleteRunParamsWithTimeout(timeout time.Duration) *RunServiceDeleteRunParams {
	var ()
	return &RunServiceDeleteRunParams{

		timeout: timeout,
	}
}

// NewRunServiceDeleteRunParamsWithContext creates a new RunServiceDeleteRunParams object
// with the default values initialized, and the ability to set a context for a request
func NewRunServiceDeleteRunParamsWithContext(ctx context.Context) *RunServiceDeleteRunParams {
	var ()
	return &RunServiceDeleteRunParams{

		Context: ctx,
	}
}

// NewRunServiceDeleteRunParamsWithHTTPClient creates a new RunServiceDeleteRunParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewRunServiceDeleteRunParamsWithHTTPClient(client *http.Client) *RunServiceDeleteRunParams {
	var ()
	return &RunServiceDeleteRunParams{
		HTTPClient: client,
	}
}

/*RunServiceDeleteRunParams contains all the parameters to send to the API endpoint
for the run service delete run operation typically these are written to a http.Request
*/
type RunServiceDeleteRunParams struct {

	/*ExperimentID
	  The ID of the parent experiment.

	*/
	ExperimentID *string
	/*RunID
	  The ID of the run to be deleted.

	*/
	RunID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the run service delete run params
func (o *RunServiceDeleteRunParams) WithTimeout(timeout time.Duration) *RunServiceDeleteRunParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the run service delete run params
func (o *RunServiceDeleteRunParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the run service delete run params
func (o *RunServiceDeleteRunParams) WithContext(ctx context.Context) *RunServiceDeleteRunParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the run service delete run params
func (o *RunServiceDeleteRunParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the run service delete run params
func (o *RunServiceDeleteRunParams) WithHTTPClient(client *http.Client) *RunServiceDeleteRunParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the run service delete run params
func (o *RunServiceDeleteRunParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithExperimentID adds the experimentID to the run service delete run params
func (o *RunServiceDeleteRunParams) WithExperimentID(experimentID *string) *RunServiceDeleteRunParams {
	o.SetExperimentID(experimentID)
	return o
}

// SetExperimentID adds the experimentId to the run service delete run params
func (o *RunServiceDeleteRunParams) SetExperimentID(experimentID *string) {
	o.ExperimentID = experimentID
}

// WithRunID adds the runID to the run service delete run params
func (o *RunServiceDeleteRunParams) WithRunID(runID string) *RunServiceDeleteRunParams {
	o.SetRunID(runID)
	return o
}

// SetRunID adds the runId to the run service delete run params
func (o *RunServiceDeleteRunParams) SetRunID(runID string) {
	o.RunID = runID
}

// WriteToRequest writes these params to a swagger request
func (o *RunServiceDeleteRunParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.ExperimentID != nil {

		// query param experiment_id
		var qrExperimentID string
		if o.ExperimentID != nil {
			qrExperimentID = *o.ExperimentID
		}
		qExperimentID := qrExperimentID
		if qExperimentID != "" {
			if err := r.SetQueryParam("experiment_id", qExperimentID); err != nil {
				return err
			}
		}

	}

	// path param run_id
	if err := r.SetPathParam("run_id", o.RunID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}