package tui

import (
	"fmt"
	"log"
	"os"

	"github.com/looplab/fsm"
)

type machine struct {
	app *App
	fsm *fsm.FSM
}

const (
	// events
	evSelected    = "selected"
	evCancelled   = "cancelled"
	evConfirmed   = "confirmed"
	evDeleted     = "deleted"
	evFetched     = "fetched"
	evNothingToDo = "nothing_to_do"
	evSearch      = "search"
	evLocate      = "locate"

	// states
	stSelecting  = "selecting"
	stSearching  = "searching"
	stFetching   = "fetching"
	stConfirming = "confirming"
	stDeleting   = "deleting"
	stNothing    = "nothing"

	// metadata
	metaMessages = "messages"
	metaChat     = "chat"
)

func initFSM(app *App) *fsm.FSM {
	m := machine{app: app}
	sm := fsm.NewFSM(
		stSelecting,
		fsm.Events{
			{Name: evSelected, Src: []string{stSelecting}, Dst: stFetching},
			{Name: evFetched, Src: []string{stFetching}, Dst: stConfirming},
			{Name: evNothingToDo, Src: []string{stFetching}, Dst: stNothing},
			{Name: evConfirmed, Src: []string{stConfirming}, Dst: stDeleting},
			{Name: evDeleted, Src: []string{stDeleting}, Dst: stSelecting},
			// search
			{Name: evSearch, Src: []string{stSelecting}, Dst: stSearching},
			{Name: evLocate, Src: []string{stSearching}, Dst: stSelecting},
			// cancel
			{Name: evCancelled, Src: []string{stFetching, stConfirming, stNothing, stSearching}, Dst: stSelecting},
		},
		fsm.Callbacks{
			m.enter("state"): func(e *fsm.Event) {
				m.app.log.Debugf("*** transition: %q -> %q\n", e.Src, e.Dst)
				m.app.pages.ShowPage(e.Dst)
			},
			// states
			m.leave(stConfirming): m.hidePage,
			m.leave(stNothing):    m.hidePage,
			m.leave(stSearching):  m.hidePage,
			m.leave(stDeleting):   m.leaveDeleting,
			// events
			m.after(evCancelled): m.afterCancelled,
		},
	)
	m.fsm = sm

	return m.fsm
}

func (*machine) leave(state string) string {
	return "leave_" + state
}

func (*machine) enter(state string) string {
	return "enter_" + state
}

func (*machine) after(event string) string {
	return "after_" + event
}

//
// States
//

func (m *machine) hidePage(e *fsm.Event) {
	m.app.pages.HidePage(e.Src)
}

func (m *machine) leaveDeleting(e *fsm.Event) {
	m.cleanUp()
	m.hidePage(e)
}

//
// Events
//

func (m *machine) afterCancelled(*fsm.Event) {
	// clear metadata
	m.cleanUp()
	m.app.logf("Operation cancelled")
}

func (m *machine) cleanUp() {
	m.fsm.SetMetadata(metaChat, nil)
	m.fsm.SetMetadata(metaMessages, nil)
}

// eventValue allows to get an event value at idx.
func eventValue[T any](e *fsm.Event, idx int) (T, bool) {
	var ret T
	if len(e.Args)-1 < idx {
		return ret, false
	}
	ret, ok := e.Args[idx].(T)
	if !ok {
		return ret, false
	}
	return ret, true
}

func metadata[T any](fsm *fsm.FSM, key string) (T, error) {
	var ret T
	val, ok := fsm.Metadata(key)
	if !ok || val == nil {
		return ret, fmt.Errorf("value of type %T not present in metadata", ret)
	}
	ret, ok = val.(T)
	if !ok {
		return ret, fmt.Errorf("invalid type (metadata: %T, want %T)", val, ret)
	}
	return ret, nil
}

func visualise(m *fsm.FSM) {
	if err := os.WriteFile("fsm.dot", []byte(fsm.Visualize(m)), 0666); err != nil {
		log.Panicf("error writing fsm: %s", err)
	}
}
