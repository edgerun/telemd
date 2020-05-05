package telemd

import "log"

const (
	Pause   Command = "pause"
	Unpause Command = "unpause"
)

type Command string

type commandChannel struct {
	channel chan Command
	stop    chan bool
}

func newCommandChannel() *commandChannel {
	tcc := &commandChannel{
		channel: make(chan Command),
		stop:    make(chan bool),
	}
	return tcc
}

func (daemon *Daemon) runCommandLoop() {
	for {
		select {
		case cmd := <-daemon.cmds.channel:
			daemon.handleCommand(cmd)
		case stop := <-daemon.cmds.stop:
			if stop {
				return
			}
		}
	}
}

func (daemon *Daemon) handleCommand(cmd Command) {
	switch cmd {
	case Pause:
		log.Printf("pausing %d tickers\n", len(daemon.tickers))
		daemon.pauseTickers()
	case Unpause:
		log.Printf("unpausing %d tickers\n", len(daemon.tickers))
		daemon.unpauseTickers()
	default:
		log.Println("unhandled command", cmd)
	}
}

func (daemon *Daemon) unpauseTickers() {
	for _, ticker := range daemon.tickers {
		ticker.Unpause()
	}
}

func (daemon *Daemon) pauseTickers() {
	for _, ticker := range daemon.tickers {
		ticker.Pause()
	}
}
