package comment

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/apache/trafficcontrol/lib/go-tc"
	"github.com/apache/trafficcontrol/lib/go-tc/tovalidate"
	"github.com/apache/trafficcontrol/lib/go-util"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/api"
	"github.com/apache/trafficcontrol/traffic_ops/traffic_ops_golang/dbhelpers"

	"github.com/go-ozzo/ozzo-validation"
)

//we need a type alias to define functions on
type TODeliveryServiceRequestComment struct {
	ReqInfo *api.APIInfo `json:"-"`
	tc.DeliveryServiceRequestCommentNullable
}

func (v *TODeliveryServiceRequestComment) APIInfo() *api.APIInfo         { return v.ReqInfo }
func (v *TODeliveryServiceRequestComment) SetLastUpdated(t tc.TimeNoMod) { v.LastUpdated = &t }
func (v *TODeliveryServiceRequestComment) InsertQuery() string           { return insertQuery() }
func (v *TODeliveryServiceRequestComment) NewReadObj() interface{} {
	return &tc.DeliveryServiceRequestCommentNullable{}
}
func (v *TODeliveryServiceRequestComment) SelectQuery() string { return selectQuery() }
func (v *TODeliveryServiceRequestComment) ParamColumns() map[string]dbhelpers.WhereColumnInfo {
	return map[string]dbhelpers.WhereColumnInfo{
		"authorId":                 dbhelpers.WhereColumnInfo{"dsrc.author_id", nil},
		"author":                   dbhelpers.WhereColumnInfo{"a.username", nil},
		"deliveryServiceRequestId": dbhelpers.WhereColumnInfo{"dsrc.deliveryservice_request_id", nil},
		"id": dbhelpers.WhereColumnInfo{"dsrc.id", api.IsInt},
	}
}
func (v *TODeliveryServiceRequestComment) UpdateQuery() string { return updateQuery() }
func (v *TODeliveryServiceRequestComment) DeleteQuery() string { return deleteQuery() }

func GetTypeSingleton() api.CRUDFactory {
	return func(reqInfo *api.APIInfo) api.CRUDer {
		toReturn := TODeliveryServiceRequestComment{reqInfo, tc.DeliveryServiceRequestCommentNullable{}}
		return &toReturn
	}
}

func (comment TODeliveryServiceRequestComment) GetKeyFieldsInfo() []api.KeyFieldInfo {
	return []api.KeyFieldInfo{{"id", api.GetIntKey}}
}

//Implementation of the Identifier, Validator interface functions
func (comment TODeliveryServiceRequestComment) GetKeys() (map[string]interface{}, bool) {
	if comment.ID == nil {
		return map[string]interface{}{"id": 0}, false
	}
	return map[string]interface{}{"id": *comment.ID}, true
}

func (comment *TODeliveryServiceRequestComment) SetKeys(keys map[string]interface{}) {
	i, _ := keys["id"].(int) //this utilizes the non panicking type assertion, if the thrown away ok variable is false i will be the zero of the type, 0 here.
	comment.ID = &i
}

func (comment TODeliveryServiceRequestComment) GetAuditName() string {
	if comment.ID != nil {
		return strconv.Itoa(*comment.ID)
	}
	return "unknown"
}

func (comment TODeliveryServiceRequestComment) GetType() string {
	return "deliveryservice_request_comment"
}

func (comment TODeliveryServiceRequestComment) Validate() error {
	errs := validation.Errors{
		"deliveryServiceRequestId": validation.Validate(comment.DeliveryServiceRequestID, validation.NotNil),
		"value":                    validation.Validate(comment.Value, validation.NotNil),
	}
	return util.JoinErrs(tovalidate.ToErrors(errs))
}

func (comment *TODeliveryServiceRequestComment) Create() (error, error, int) {
	au := tc.IDNoMod(comment.ReqInfo.User.ID)
	comment.AuthorID = &au
	return api.GenericCreate(comment)
}

func (comment *TODeliveryServiceRequestComment) Read() ([]interface{}, error, error, int) {
	return api.GenericRead(comment)
}

func (comment *TODeliveryServiceRequestComment) Update() (error, error, int) {
	current := TODeliveryServiceRequestComment{}
	err := comment.ReqInfo.Tx.QueryRowx(selectQuery() + `WHERE dsrc.id=` + strconv.Itoa(*comment.ID)).StructScan(&current)
	if err != nil {
		return api.ParseDBErr(err, comment.GetType())
	}

	userID := tc.IDNoMod(comment.ReqInfo.User.ID)
	if *current.AuthorID != userID {
		return errors.New("Comments can only be updated by the author"), nil, http.StatusBadRequest
	}

	return api.GenericUpdate(comment)
}

func (comment *TODeliveryServiceRequestComment) Delete() (error, error, int) {
	var current TODeliveryServiceRequestComment
	err := comment.ReqInfo.Tx.QueryRowx(selectQuery() + `WHERE dsrc.id=` + strconv.Itoa(*comment.ID)).StructScan(&current)
	if err != nil {
		return nil, errors.New("querying DeliveryServiceRequestComments: " + err.Error()), http.StatusInternalServerError
	}

	if userID := tc.IDNoMod(comment.ReqInfo.User.ID); *current.AuthorID != userID {
		// TODO determine if users should be able to delete sub-tenant users' comments? Else, a deleted user's comments can never be removed.
		return errors.New("Comments can only be deleted by the author"), nil, http.StatusBadRequest
	}

	return api.GenericDelete(comment)
}

func insertQuery() string {
	query := `INSERT INTO deliveryservice_request_comment (
author_id,
deliveryservice_request_id,
value) VALUES (
:author_id,
:deliveryservice_request_id,
:value) RETURNING id,last_updated`
	return query
}

func selectQuery() string {
	query := `SELECT
a.username AS author,
dsrc.author_id,
dsrc.deliveryservice_request_id,
dsr.deliveryservice->>'xmlId' as xml_id,
dsrc.id,
dsrc.last_updated,
dsrc.value
FROM deliveryservice_request_comment dsrc
JOIN tm_user a ON dsrc.author_id = a.id
JOIN deliveryservice_request dsr ON dsrc.deliveryservice_request_id = dsr.id
`
	return query
}

func updateQuery() string {
	query := `UPDATE
deliveryservice_request_comment SET
deliveryservice_request_id=:deliveryservice_request_id,
value=:value
WHERE id=:id RETURNING last_updated`
	return query
}

func deleteQuery() string {
	return `DELETE FROM deliveryservice_request_comment WHERE id = :id`
}
