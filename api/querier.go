// Copyright 2018-2019 The logrange Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import "context"

type (

	// QueryRequest struct describes a request for reading records
	QueryRequest struct {
		// ReqId identifies the request on server side. The field should not be populated by client,
		// but it can be returned with the structure in QueryResult.
		ReqId uint64

		// the LQL line for selecting records
		Query string

		// Pos contains the next read record position.
		Pos string

		// WaitTimeout in seconds provide waiting new data timeout in case of the request starts from
		// the EOF. The timout cannot exceed 60 seconds. When the tiemout expires and no data is arrived
		// response with no data will be returned.
		WaitTimeout int

		// Limit defines the maximum number of records which could be read from the sources
		Limit int
	}

	// QeryResult is a result returned by the server in a response on LQL execution (see Querier.Query)
	QueryResult struct {
		// Events slice contains the result of the query execution
		Events []*LogEvent
		// NextQueryRequest contains the query for reading next porition of events. It makes sense only if Err is
		// nil
		NextQueryRequest QueryRequest
		// Err the operation error. If the Err is nil, the operation successfully executed
		Err error
	}

	// Source struct describes a source structure
	Source struct {
		// Tags contains tag for the source
		Tags string

		// Size contains data size (in bytes)
		Size uint64

		// Records contains number of records
		Records uint64
	}

	// SourcesResult struct contains the result of queries of sources
	SourcesResult struct {
		// Sources found by the request
		Sources []Source

		// Count number of sources which meet the TagsCond criteria
		Count int

		// Err the operaion error, if any
		Err error `json:"-"`
	}

	// Querier - executes a query agains logrange database
	Querier interface {
		// Query runs lql to collect the server data and return it in the QueryResult. It returns an error which indicates
		// that the query could not be delivered to the server, or it did not happen.
		Query(ctx context.Context, req *QueryRequest, res *QueryResult) error

		// Sources requests
		Sources(ctx context.Context, TagsCond string, res *SourcesResult) error
	}
)