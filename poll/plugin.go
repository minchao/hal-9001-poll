package poll

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/netflix/hal-9001/hal"
)

const usage = `Usage: !poll <command> [arg...]

Poll.

Commands:

!poll show
    Show the poll
!poll new <title>
    Create a new poll
!poll remove
    Remove the poll
!poll option <option>
    Add an option to the poll
!poll start
    Start the poll
!poll end
    Stop the currently running poll
!poll vote <index>
    Vote for the currently running poll
`

var (
	polls map[string]*pollEntry
	mutex sync.Mutex
)

func init() {
	polls = make(map[string]*pollEntry)
}

type pollOption struct {
	Text  string
	Votes int
}

type pollEntry struct {
	Title    string
	Options  []pollOption
	HasVoted []string
	IsActive bool
}

func (p pollEntry) Result() string {
	options := ""
	for k, o := range p.Options {
		options = fmt.Sprintf("%s %d. %s (%d votes)\n", options, k+1, o.Text, o.Votes)
	}
	return fmt.Sprintf("%s\n%s", p.Title, strings.Trim(options, "\n"))
}

func Register() {
	p := hal.Plugin{
		Name:  "poll",
		Func:  poll,
		Regex: "^[[:space:]]*!poll",
	}
	p.Register()
}

func poll(evt hal.Evt) {
	argv := evt.BodyAsArgv()
	if len(argv) < 2 {
		evt.Reply(usage)
		return
	}

	switch argv[1] {
	case "show":
		evt.Reply(pollShow(evt.RoomId))
		return
	case "new":
		if len(argv) < 3 {
			evt.Reply("Usage: !poll new <title>")
			return
		}
		evt.Reply(pollNew(evt.RoomId, strings.Join(argv[2:], " ")))
		return
	case "remove":
		evt.Reply(pollRemove(evt.RoomId))
		return
	case "option":
		if len(argv) < 3 {
			evt.Reply("Usage: !poll option <option>")
			return
		}
		evt.Reply(pollAddOption(evt.RoomId, strings.Join(argv[2:], " ")))
		return
	case "start":
		evt.Reply(pollStart(evt.RoomId))
		return
	case "end":
		evt.Reply(pollEnd(evt.RoomId))
		return
	case "vote":
		if len(argv) < 3 {
			evt.Reply("Usage: !poll vote <index>")
			return
		}
		index, err := strconv.Atoi(argv[2])
		if err != nil {
			evt.Reply("Please vote using the numerical index of the option.")
		}
		evt.Reply(pollVote(evt.RoomId, evt.UserId, index))
		return
	default:
		evt.Reply("Wrong command.")
		evt.Reply(usage)
		return
	}
}

func pollShow(roomId string) string {
	mutex.Lock()
	defer mutex.Unlock()

	poll, ok := polls[roomId]
	if !ok {
		return "There is no poll."
	}

	status := ""
	if !poll.IsActive {
		status = " (Inactive)"
	}

	return fmt.Sprintf("Poll%s:\n%s", status, poll.Result())
}

func pollNew(roomId, title string) string {
	mutex.Lock()
	defer mutex.Unlock()

	if poll, ok := polls[roomId]; ok {
		return fmt.Sprintf("The poll '%s' already exists.", poll.Title)
	}

	polls[roomId] = &pollEntry{Title: title}

	return fmt.Sprintf("Poll '%s' created.\nUse !poll option <option> to add options.", title)
}

func pollRemove(roomId string) string {
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := polls[roomId]; !ok {
		return "There is no poll."
	}

	delete(polls, roomId)

	return "Poll removed."
}

func pollAddOption(roomId, option string) string {
	mutex.Lock()
	defer mutex.Unlock()

	poll, ok := polls[roomId]
	if !ok {
		return "There is no poll."
	}

	op := pollOption{
		Text:  option,
		Votes: 0,
	}
	poll.Options = append(poll.Options, op)
	return fmt.Sprintf("Added option: %s", op.Text)
}

func pollStart(roomId string) string {
	mutex.Lock()
	defer mutex.Unlock()

	poll, ok := polls[roomId]
	if !ok {
		return "There is no poll."
	}
	if poll.IsActive {
		return "The poll is currently running."
	}
	if len(poll.Options) < 2 {
		return "Use !poll option <option> to add options."
	}

	poll.IsActive = true

	return fmt.Sprintf("Poll:\n%s", poll.Result())
}

func pollEnd(roomId string) string {
	mutex.Lock()
	defer mutex.Unlock()

	poll, ok := polls[roomId]
	if !ok {
		return "There is no poll."
	}
	if !poll.IsActive {
		return "There is no active poll."
	}

	delete(polls, roomId)

	return fmt.Sprintf("Poll finished, final results:\n%s", poll.Result())
}

func pollVote(roomId, userId string, index int) string {
	mutex.Lock()
	defer mutex.Unlock()

	poll, ok := polls[roomId]
	if !ok {
		return "There is no poll."
	}
	if !poll.IsActive {
		return "There is no active poll. Use !poll start to start the poll."
	}
	if index <= 0 || index > len(poll.Options) {
		return fmt.Sprintf("Please choose a number between 1 to %d", len(poll.Options))
	}
	hasVoted := false
	for _, uId := range poll.HasVoted {
		if userId == uId {
			hasVoted = true
			break
		}
	}
	if hasVoted {
		return "You have already voted."
	}

	poll.Options[index-1].Votes += 1
	poll.HasVoted = append(poll.HasVoted, userId)

	return fmt.Sprintf("Poll:\n%s", poll.Result())
}
