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

package cursor

import (
	"bytes"
	"context"
	"fmt"
	"github.com/logrange/logrange/pkg/lql"
	"github.com/logrange/logrange/pkg/model"
	"github.com/logrange/logrange/pkg/tindex"
	"github.com/logrange/range/pkg/records/journal"
	"github.com/pkg/errors"
	"strings"
)

type (
	// State describes state of a cursor. This structure is used for creating new ones and
	// for providing current state of a cursor
	State struct {
		// Id the cursor state Id
		Id uint64

		// Sources contains tags expressions for selecting journal sources.
		Sources string

		// Where contains the expression over LogEvent fields (ts and msg) to filter them. Empty value
		// indicates all records
		Where string

		// Pos indicates the position of the record which must be read next.
		Pos string
	}

	// Cursor struct describes a context of a query execution. Cursor state could be expressed in cursor.State and
	// a new cursor could be created from the struct. Cursor supports model.Iterator interface which allows to access
	// to records the cursor selects
	Cursor struct {
		state  State
		it     model.Iterator
		jDescs map[string]*jrnlDesc
	}

	jrnlDesc struct {
		tags model.TagLine
		j    journal.Journal
		it   journal.Iterator
	}
)

const cMaxSources = 50

// newCursor creates the new cursor based on the state provided.
func newCursor(ctx context.Context, state State, tidx tindex.Service, jctrl journal.Controller) (*Cursor, error) {
	// figuring out the journals list
	se, err := lql.ParseExpr(state.Sources)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not parse expression \"%s\" ", state.Where)
	}

	srcs, _, err := tidx.GetJournals(se, cMaxSources+1, false)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get a list of journals for the expression \"%s\"", state.Sources)
	}

	if len(srcs) == 0 {
		return nil, errors.Errorf("No sources for the expression \"%s\"", state.Sources)
	}

	if len(srcs) > cMaxSources {
		return nil, errors.Errorf("too many sources (greater than %d) correspond to the expresion \"%s\", more concreate condition is needed to reduce the number. ", cMaxSources, state.Sources)
	}

	jd := make(map[string]*jrnlDesc, len(srcs))
	// create the iterators
	var it model.Iterator
	if len(srcs) == 1 {
		for tags, src := range srcs {
			jrnl, err := jctrl.GetOrCreate(ctx, src)
			if err != nil {
				return nil, errors.Wrapf(err, "Could not get the access to the journal %s for tags %s, which's got for the \"%s\" expression ", src, tags, state.Sources)
			}
			jit := jrnl.Iterator()
			it = (&model.LogEventIterator{}).Wrap(tags, jit)

			jd[src] = &jrnlDesc{tags, jrnl, jit}
		}
	} else {
		mxs := make([]model.Iterator, len(srcs))

		i := 0
		for tags, src := range srcs {
			jrnl, err := jctrl.GetOrCreate(ctx, src)
			if err != nil {
				return nil, errors.Wrapf(err, "Could not get the access to the journal %s, which's got for the \"%s\" expression ", src, state.Sources)
			}

			jit := jrnl.Iterator()
			jd[src] = &jrnlDesc{tags, jrnl, jit}
			mxs[i] = (&model.LogEventIterator{}).Wrap(tags, jit)
			i++
		}

		// mixing them
		for len(mxs) > 1 {
			for i := 0; i < len(mxs)-1; i += 2 {
				m := &model.Mixer{}
				m.Init(model.GetEarliest, mxs[i], mxs[i+1])
				mxs[i/2] = m
			}
			if len(mxs)&1 == 1 {
				mxs[len(mxs)/2] = mxs[len(mxs)-1]
				mxs = mxs[:len(mxs)/2+1]
			} else {
				mxs = mxs[:len(mxs)/2]
			}
		}

		it = mxs[0]
	}

	cur := new(Cursor)
	cur.state = state
	cur.it = it
	cur.jDescs = jd
	if err := cur.applyPos(); err != nil {
		return nil, errors.Wrapf(err, "The position %s could not be applied ", state.Pos)
	}

	return cur, nil
}

// String returns the cursor description
func (cur *Cursor) String() string {
	return fmt.Sprintf("{descs:%d, state:%s}", len(cur.jDescs), cur.state.String())
}

// Id returns the cursor Id
func (cur *Cursor) Id() uint64 {
	return cur.state.Id
}

// Next part of the model.Iterator
func (cur *Cursor) Next(ctx context.Context) {
	cur.it.Next(ctx)
}

// Get part of the model.Iterator
func (cur *Cursor) Get(ctx context.Context) (model.LogEvent, model.TagLine, error) {
	return cur.it.Get(ctx)
}

// ApplyState tries to apply state to the cursor. Returns an error, if the operation could not be completed.
// Current implementation allows to apply position only
func (cur *Cursor) ApplyState(state State) error {
	if cur.state.Where != state.Where || cur.state.Sources != state.Sources || cur.state.Id != state.Id {
		return errors.Errorf("Could not apply state %s to the current cursor state %s", state, cur.state)
	}

	if cur.state.Pos != state.Pos {
		oldPos := cur.state.Pos
		cur.state.Pos = state.Pos
		err := cur.applyStatePos()
		if err != nil {
			cur.state.Pos = oldPos
			return errors.Wrapf(err, "Could not apply position %s to the cursor state %s ", state.Pos, cur.state)
		}
	}
	return nil
}

const cPosJrnlSplit = ":"
const cPosJrnlVal = "="

// Commit is called by the cursor reader to indicate that the reading process is over and return the current state
func (cur *Cursor) commit(ctx context.Context) State {
	// calling cur.Get(ctx) to fix the cursor position in case of last call was cur.Next()
	cur.Get(ctx)
	cur.state.Pos = cur.collectPos()
	return cur.state
}

// collectPos walks over all iterators to collect their current position
func (cur *Cursor) collectPos() string {
	var buf bytes.Buffer
	first := true
	for jn, jd := range cur.jDescs {
		if !first {
			buf.WriteString(cPosJrnlSplit)
		} else {
			first = false
		}
		buf.WriteString(jn)
		buf.WriteString(cPosJrnlVal)
		buf.WriteString(jd.it.Pos().String())

		jd.it.Release()
	}
	return buf.String()
}

func (cur *Cursor) applyPos() error {
	if !cur.applyCornerPos(cur.state.Pos) {
		err := cur.applyStatePos()
		if err != nil {
			return err
		}
	}
	return nil
}

func (cur *Cursor) applyCornerPos(pstr string) bool {
	ps := strings.ToLower(pstr)
	var p journal.Pos
	if ps == "tail" {
		p.CId = 0xFFFFFFFFFFFFFFFF
		p.Idx = 0xFFFFFFFF
	} else if ps != "head" && ps != "" {
		return false
	}

	for _, jd := range cur.jDescs {
		jd.it.SetPos(p)
	}
	return true
}

func (cur *Cursor) applyStatePos() error {
	vals := strings.Split(cur.state.Pos, cPosJrnlSplit)
	m := make(map[string]journal.Pos, len(vals))
	for _, v := range vals {
		kv := strings.Split(v, cPosJrnlVal)
		if len(kv) != 2 {
			return errors.Errorf(
				"Could not parse position=%s, value the %s sub-string doesn't look like journal pos. Expecting <jrnlId>%s<jrnlPos>",
				cur.state.Pos, v, cPosJrnlVal)
		}

		jrnl := kv[0]
		pos, err := journal.ParsePos(kv[1])
		if err != nil {
			return errors.Wrapf(err, "Could not parse pos %s to journal.Pos", kv[1])
		}
		m[jrnl] = pos
	}

	for j, pos := range m {
		if jd, ok := cur.jDescs[j]; ok {
			jd.it.SetPos(pos)
		}
	}
	return nil
}

func (s State) String() string {
	return fmt.Sprintf("{Id: %d, Sources:\"%s\", Where:\"%s\", Pos:%s}", s.Id, s.Sources, s.Where, s.Pos)
}