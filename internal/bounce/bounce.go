package bounce

import (
	"errors"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/internal/bounce/mailbox"
	"github.com/knadh/listmonk/internal/bounce/webhooks"
	"github.com/knadh/listmonk/models"
)

// Mailbox represents a POP/IMAP mailbox client that can scan messages and pass
// them to a given channel.
type Mailbox interface {
	Scan(limit int, ch chan models.Bounce) error
}

// Opt represents bounce processing options.
type Opt struct {
	Mailboxes       []MailboxOpt `json:"mailboxes"`
	WebhooksEnabled bool         `json:"webhooks_enabled"`
	SESEnabled      bool         `json:"ses_enabled"`
	SendgridEnabled bool         `json:"sendgrid_enabled"`
	SendgridKey     string       `json:"sendgrid_key"`
	Postmark        struct {
		Enabled  bool
		Username string
		Password string
	}
	ForwardEmail struct {
		Enabled bool
		Key     string
	}
	Azure struct {
		Enabled bool
	}

	RecordBounceCB func(models.Bounce) error
}

// MailboxOpt represents a single mailbox configuration.
type MailboxOpt struct {
	UUID    string      `json:"uuid"`
	Name    string      `json:"name"`
	Enabled bool        `json:"enabled"`
	Type    string      `json:"type"`
	Opt     mailbox.Opt `json:"opt"`
}

// Manager handles e-mail bounces.
type Manager struct {
	queue        chan models.Bounce
	mailboxes    map[string]Mailbox // keyed by UUID
	SES          *webhooks.SES
	Sendgrid     *webhooks.Sendgrid
	Postmark     *webhooks.Postmark
	Forwardemail *webhooks.Forwardemail
	Azure        *webhooks.Azure
	queries      *Queries
	opt          Opt
	log          *log.Logger
}

// Queries contains the queries.
type Queries struct {
	DB          *sqlx.DB
	RecordQuery *sqlx.Stmt
}

// New returns a new instance of the bounce manager.
func New(opt Opt, q *Queries, lo *log.Logger) (*Manager, error) {
	m := &Manager{
		opt:       opt,
		queries:   q,
		queue:     make(chan models.Bounce, 1000),
		mailboxes: make(map[string]Mailbox),
		log:       lo,
	}

	// Initialize all enabled mailboxes.
	for _, boxOpt := range opt.Mailboxes {
		if !boxOpt.Enabled {
			continue
		}

		switch boxOpt.Type {
		case "pop":
			m.mailboxes[boxOpt.UUID] = mailbox.NewPOP(boxOpt.Opt)
			lo.Printf("initialized bounce mailbox: %s (%s)", boxOpt.Name, boxOpt.Type)
		default:
			return nil, errors.New("unknown bounce mailbox type: " + boxOpt.Type)
		}
	}

	if opt.WebhooksEnabled {
		if opt.SESEnabled {
			m.SES = webhooks.NewSES()
		}

		if opt.SendgridEnabled {
			sg, err := webhooks.NewSendgrid(opt.SendgridKey)
			if err != nil {
				lo.Printf("error initializing sendgrid webhooks: %v", err)
			} else {
				m.Sendgrid = sg
			}
		}

		if opt.Postmark.Enabled {
			m.Postmark = webhooks.NewPostmark(opt.Postmark.Username, opt.Postmark.Password)
		}

		if opt.ForwardEmail.Enabled {
			fe := webhooks.NewForwardemail([]byte(opt.ForwardEmail.Key))
			m.Forwardemail = fe
		}

		if opt.Azure.Enabled {
			m.Azure = webhooks.NewAzure()
		}
	}

	return m, nil
}

// Run is a blocking function that listens for bounce events from webhooks and or mailboxes
// and executes them on the DB.
func (m *Manager) Run() {
	// Start a scanner goroutine for each enabled mailbox.
	for uuid, mb := range m.mailboxes {
		go m.runMailboxScanner(uuid, mb)
	}

	for b := range m.queue {
		if b.CreatedAt.IsZero() {
			b.CreatedAt = time.Now()
		}

		if err := m.opt.RecordBounceCB(b); err != nil {
			continue
		}
	}
}

// runMailboxScanner runs a blocking loop that scans the mailbox at given intervals.
func (m *Manager) runMailboxScanner(uuid string, mb Mailbox) {
	// Find the mailbox configuration to get the scan interval and details for logging.
	var scanInterval time.Duration
	var mailboxName string
	var mailboxType string
	var mailboxHost string
	var mailboxUsername string
	var mailboxPort int

	for _, boxOpt := range m.opt.Mailboxes {
		if boxOpt.UUID == uuid {
			// Parse the scan interval string (e.g., "15m", "1h", "30s")
			if boxOpt.Opt.ScanInterval != "" {
				d, err := time.ParseDuration(boxOpt.Opt.ScanInterval)
				if err != nil {
					m.log.Printf("error parsing scan interval '%s' for mailbox '%s': %v, using default 15m",
						boxOpt.Opt.ScanInterval, boxOpt.Name, err)
					scanInterval = 15 * time.Minute
				} else {
					scanInterval = d
				}
			}
			mailboxName = boxOpt.Name
			mailboxType = boxOpt.Type
			mailboxHost = boxOpt.Opt.Host
			mailboxUsername = boxOpt.Opt.Username
			mailboxPort = boxOpt.Opt.Port
			break
		}
	}

	if scanInterval == 0 {
		scanInterval = 15 * time.Minute // default
	}

	m.log.Printf("bounce mailbox '%s' will scan every %v", mailboxName, scanInterval)

	for {
		if err := mb.Scan(1000, m.queue); err != nil {
			m.log.Printf("error scanning bounce mailbox '%s' (type=%s, host=%s:%d, username=%s): %v",
				mailboxName, mailboxType, mailboxHost, mailboxPort, mailboxUsername, err)
		}

		time.Sleep(scanInterval)
	}
}

// Record records a new bounce event given the subscriber's email or UUID.
func (m *Manager) Record(b models.Bounce) error {
	m.queue <- b
	return nil
}
