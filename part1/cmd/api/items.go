//Filename:cmd/api/items.go
package main

import	(
	"fmt"
	"errors"
	"net/http"
	"quiz3.castillojadah.net/internals/data"
	"quiz3.castillojadah.net/internals/validator"
)
//create entry handler for the POST items endpoint
func (app *application) createToDoHandler(w http.ResponseWriter, r *http.Request){
	
	//our target decode destination
	var input struct{
		Name string `json:"name"`
		Description string `json:"description"`
		Status string `json:"status"`
		Mode []string `json:"mode"`
	}

	err := app.readJSON(w, r, &input )
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	items:= &data.Items{
		Name: input.Name,
		Description: input.Description,
		Status: input.Status,
		Mode: input.Mode,
	}
	//initialize a new validator instance
	v := validator.New()
	//check the map to determine if there were any validation errors
	if data.ValidateItems(v,items); !v.Valid(){
		app.failedValidationResponse(w,r,v.Errors)
		return
	}
	//create a to do list
	err = app.models.Items.Insert(items)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
	// Create a Location header for the newly created resource/todo
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/todo/%d", items.ID))
	// Write the JSON response with 201 - Created status code with the body
	// being the item data and the header being the headers map
	err = app.writeJSON(w, http.StatusCreated, envelope{"items": items}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) showToDoHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Fetch the specific items
	items, err := app.models.Items.Get(id)
	// Handle errors
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Write the data returned by Get()
	err = app.writeJSON(w, http.StatusOK, envelope{"items": items}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
func (app *application) updateToDoHandler(w http.ResponseWriter, r *http.Request) {
	// This method does a partial replacement
	// Get the id for the item that needs updating
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	// Fetch the orginal record from the database
	items, err := app.models.Items.Get(id)
	// Handle errors
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Create an input struct to hold data read in from the client
	// We update input struct to use pointers because pointers have a
	// default value of nil
	// If a field remains nil then we know that the client did not update it
	var input struct {
		Name    *string  `json:"name"`
		Description  *string  `json:"description"`
		Status *string  `json:"status"`
		Mode []*string  `json:"mode"`
	}

	// Initialize a new json.Decoder instance
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	// Check for updates
	if input.Name != nil {
		items.Name = *input.Name
	}
	if input.Description != nil {
		items.Description= *input.Description
	}
	if input.Status!= nil {
		items.Status = *input.Status
	}
	//if input.Mode!= nil {
	//	items.Mode= *input.Mode
	//}

	// Perform validation on the updated items. If validation fails, then
	// we send a 422 - Unprocessable Entity respose to the client
	// Initialize a new Validator instance
	v := validator.New()

	// Check the map to determine if there were any validation errors
	if data.ValidateItems(v, items); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// Pass the updated item record to the Update() method
	err = app.models.Items.Update(items)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Write the data returned by Get()
	err = app.writeJSON(w, http.StatusOK, envelope{"items": items}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteToDoHandler(w http.ResponseWriter, r *http.Request) {
	// Get the id for the itesm that needs updating
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	// Delete the items from the database. Send a 404 Not Found status code to the
	// client if there is no matching record
	err = app.models.Items.Delete(id)
	// Handle errors
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Return 200 Status OK to the client with a success message
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "items successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}
// The listToDoHandler() allows the client to see a listing of items based on a set of criteria
func (app *application) listToDoHandler(w http.ResponseWriter, r *http.Request) {
	// Create an input struct to hold our query parameters
	var input struct {
		Name  string
		Description string
		Status string
		Mode  []string
		data.Filters
	}
	// Initialize a validator
	v := validator.New()
	// Get the URL values map
	qs := r.URL.Query()
	// Use the helper methods to extract the values
	input.Name = app.readString(qs, "name", "")
	input.Description = app.readString(qs, "description", "")
	input.Status = app.readString(qs, "status", "")
	input.Mode = app.readCSV(qs, "mode", []string{})
	// Get the page information
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	// Get the sort information
	input.Filters.Sort = app.readString(qs, "sort", "id")
	// Specific the allowed sort values
	input.Filters.SortList = []string{"id", "name", "description", "status", "-id", "-name", "-description", "-status"}
	// Check for validation errors
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// Get a listing of all items
	items, metadata, err := app.models.Items.GetAll(input.Name, input.Description, input.Status, input.Mode, input.Filters )
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// Send a JSON response containg all the items
	err = app.writeJSON(w, http.StatusOK, envelope{"items": items, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}